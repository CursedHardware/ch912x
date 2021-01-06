package ch912x

import (
	"net"
	"strings"
	"syscall"
)

func trimNull(values []byte) string {
	return strings.TrimRight(string(values), "\x00")
}

func fromVariantBool(x byte) bool {
	return x != 0
}

func toVariantBool(x bool) byte {
	if x {
		return 1
	}
	return 0
}

func bindInterfaceToUDPConn(conn *net.UDPConn, ifi *net.Interface) (err error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return
	}
	return raw.Control(func(fd uintptr) {
		// see https://stackoverflow.com/a/57013928
		_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.IP_BOUND_IF, ifi.Index)
	})
}
