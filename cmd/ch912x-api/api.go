package main

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"

	"github.com/CursedHardware/ch912x"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
)

type CustomizedContext struct {
	echo.Context
	Product ch912x.Product
	Address net.HardwareAddr
}

func makeAPIService() http.Handler {
	mux := echo.New()
	mux.Use(middleware.Secure())
	mux.GET("/discovery", onDiscovery)                      // discovery all type
	mux.GET("/:product/:address", onPullModule, onBind)     // pull
	mux.POST("/:product/:address", onPushModule, onBind)    // push
	mux.DELETE("/:product/:address", onResetModule, onBind) // reset
	return mux
}

func onDiscovery(ctx echo.Context) (err error) {
	ctx.Response().WriteHeader(http.StatusNoContent)
	var group errgroup.Group
	group.Go(func() error { return plane.SendDiscovery(ch912x.ProductCH9120) })
	group.Go(func() error { return plane.SendDiscovery(ch912x.ProductCH9121) })
	group.Go(func() error { return plane.SendDiscovery(ch912x.ProductCH9126) })
	return group.Wait()
}

func onPullModule(ctx echo.Context) (err error) {
	product := ctx.(*CustomizedContext).Product
	address := ctx.(*CustomizedContext).Address
	module, err := plane.Pull(context.Background(), product, address)
	if err == nil {
		return ctx.JSON(http.StatusOK, module)
	}
	return
}

func onResetModule(ctx echo.Context) (err error) {
	product := ctx.(*CustomizedContext).Product
	address := ctx.(*CustomizedContext).Address
	module, err := plane.Reset(context.Background(), product, address)
	if err == nil {
		return ctx.JSON(http.StatusOK, module)
	}
	return
}

func onPushModule(ctx echo.Context) (err error) {
	address := ctx.(*CustomizedContext).Address
	var module ch912x.Module
	switch ctx.(*CustomizedContext).Product {
	case ch912x.ProductCH9120:
		module = &ch912x.CH9120{Kind: ch912x.KindPushRequest, ModuleMAC: address}
	case ch912x.ProductCH9121:
		module = &ch912x.CH9121{Kind: ch912x.KindPushRequest, ModuleMAC: address}
	case ch912x.ProductCH9126:
		module = &ch912x.CH9126{Kind: ch912x.KindPushRequest, ModuleMAC: address}
	}
	if err = ctx.Bind(module); err != nil {
		return
	}
	module, err = plane.Push(context.Background(), module)
	if err == nil {
		return ctx.JSON(http.StatusOK, module)
	}
	return
}

func onBind(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) (err error) {
		address, err := base64.StdEncoding.DecodeString(ctx.Param("address"))
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, err.Error())
			return
		}
		product := ch912x.Product(ctx.Param("product"))
		switch product {
		case ch912x.ProductCH9120:
		case ch912x.ProductCH9121:
		case ch912x.ProductCH9126:
		default:
			err = echo.NewHTTPError(http.StatusBadRequest, ch912x.ErrUnknownModuleType)
			return
		}
		err = next(&CustomizedContext{
			Context: ctx,
			Product: product,
			Address: address[0:6],
		})
		if err != nil {
			if _, ok := err.(*echo.HTTPError); !ok {
				err = echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
		}
		return
	}
}
