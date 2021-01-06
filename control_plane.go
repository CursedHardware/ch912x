package ch912x

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/mdlayher/arp"
)

const (
	listenPort  = 60000
	controlPort = 50000
)

type ControlPlane struct {
	udpClient   net.PacketConn
	arpClient   *arp.Client
	clientMAC   net.HardwareAddr
	pairs       map[string]func(Module)
	arpTable    map[string]func()
	discovery   chan Module
	Timeout     time.Duration
	SendTimeout time.Duration
}

func ListenCH912XByName(name string) (*ControlPlane, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	return ListenCH912X(ifi)
}

func ListenCH912X(ifi *net.Interface) (plane *ControlPlane, err error) {
	if ifi == nil {
		err = ErrInvalidNetworkInterface
		return
	}
	addr := &net.UDPAddr{Port: listenPort}
	plane = &ControlPlane{
		discovery:   make(chan Module),
		pairs:       make(map[string]func(Module)),
		arpTable:    make(map[string]func()),
		clientMAC:   ifi.HardwareAddr,
		Timeout:     15 * time.Second,
		SendTimeout: time.Second,
	}
	plane.udpClient, err = net.ListenUDP("udp", addr)
	if err == nil {
		err = bindInterfaceToUDPConn(plane.udpClient.(*net.UDPConn), ifi)
	}
	if err == nil {
		plane.arpClient, err = arp.Dial(ifi)
	}
	if err == nil {
		go plane.watchUDP()
		go plane.watchARP()
	}
	return
}

func (p *ControlPlane) watchUDP() {
	var data [0x200]byte
	for {
		n, _, err := p.udpClient.ReadFrom(data[:])
		if err != nil {
			break
		}
		go p.handleResponse(data[:n])
	}
	close(p.discovery)
}

func (p *ControlPlane) watchARP() {
	for {
		packet, _, err := p.arpClient.Read()
		if err != nil {
			break
		}
		go p.handleARP(packet)
	}
	return
}

func (p *ControlPlane) Discovery() <-chan Module {
	return p.discovery
}

func (p *ControlPlane) SendDiscovery(product Product) (err error) {
	err = ErrUnknownModuleType
	switch product {
	case ProductCH9120:
		err = p.push(&CH9120{Kind: KindDiscoveryRequest})
	case ProductCH9121:
		err = p.push(&CH9121{Kind: KindDiscoveryRequest})
	case ProductCH9126:
		err = p.push(&CH9126{Kind: KindDiscoveryRequest})
	}
	return
}

func (p *ControlPlane) Pull(ctx context.Context, product Product, address net.HardwareAddr) (module Module, err error) {
	err = ErrUnknownModuleType
	switch product {
	case ProductCH9120:
		module, err = p.send(ctx, &CH9120{Kind: KindPullRequest, ModuleMAC: address})
	case ProductCH9121:
		module, err = p.send(ctx, &CH9121{Kind: KindPullRequest, ModuleMAC: address})
	case ProductCH9126:
		module, err = p.send(ctx, &CH9126{Kind: KindPullRequest, ModuleMAC: address})
	}
	return
}

func (p *ControlPlane) Push(ctx context.Context, module Module) (parsed Module, err error) {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	kind, address := module.identity()
	if kind != KindPushRequest {
		err = ErrModuleKindWrong
		return
	}
	parsed, err = p.send(ctx, module)
	if err != nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-p.waitFirstARP(address):
		return
	}
}

func (p *ControlPlane) Reset(ctx context.Context, product Product, address net.HardwareAddr) (module Module, err error) {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	err = ErrUnknownModuleType
	switch product {
	case ProductCH9120:
		module, err = p.send(ctx, &CH9120{Kind: KindResetRequest, ModuleMAC: address})
	case ProductCH9121:
		module, err = p.send(ctx, &CH9121{Kind: KindResetRequest, ModuleMAC: address})
	case ProductCH9126:
		module, err = p.send(ctx, &CH9126{Kind: KindResetRequest, ModuleMAC: address})
	}
	if err != nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-p.waitFirstARP(address):
		return
	}
}

func (p *ControlPlane) send(ctx context.Context, module Module) (parsed Module, err error) {
	ctx, cancel := context.WithTimeout(ctx, p.SendTimeout)
	defer cancel()
	module.setClientMAC(p.clientMAC)
	_, addr := module.identity()
	if addr == nil {
		err = ErrModuleMustMAC
		return
	} else if _, ok := p.pairs[addr.String()]; ok {
		err = ErrTaskRunning
		return
	}
	returns := make(chan Module, 1)
	p.pairs[addr.String()] = func(parsed Module) { returns <- parsed }
	defer delete(p.pairs, addr.String())
	if err = p.push(module); err != nil {
		return
	}
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case parsed = <-returns:
		err, _ = parsed.(error)
	}
	return
}

func (p *ControlPlane) push(module Module) (err error) {
	ip := module.moduleIP()
	if ip == nil {
		ip = net.IPv4bcast
	}
	var buf bytes.Buffer
	_, _ = module.WriteTo(&buf)
	_, err = p.udpClient.WriteTo(buf.Bytes(), &net.UDPAddr{IP: ip, Port: controlPort})
	return
}

func (p *ControlPlane) waitFirstARP(address net.HardwareAddr) chan struct{} {
	returns := make(chan struct{}, 1)
	p.arpTable[address.String()] = func() { returns <- struct{}{} }
	defer delete(p.arpTable, address.String())
	return returns
}

func (p *ControlPlane) handleResponse(data []byte) {
	var module Module
	switch {
	case bytes.HasPrefix(data, []byte(magicCH9120)):
		module = new(CH9120)
	case bytes.HasPrefix(data, []byte(magicCH9121)):
		module = new(CH9121)
	case bytes.HasPrefix(data, []byte(magicCH9126)):
		module = new(CH9126)
	case bytes.HasPrefix(data, []byte(magicModule)):
		return
	default:
		panic(ErrUnknownModuleType)
	}
	_, _ = module.ReadFrom(bytes.NewReader(data))
	kind, addr := module.identity()
	if kind == KindDiscoveryResponse {
		p.discovery <- module
	} else if fn, ok := p.pairs[addr.String()]; ok {
		fn(module)
	}
	return
}

func (p *ControlPlane) handleARP(packet *arp.Packet) {
	if fn, ok := p.arpTable[packet.SenderHardwareAddr.String()]; ok {
		fn()
	}
}

func (p *ControlPlane) Close() (err error) {
	err = p.udpClient.Close()
	if err == nil {
		err = p.arpClient.Close()
	}
	return
}
