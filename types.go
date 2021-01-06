package ch912x

import (
	"io"
	"net"
)

type Module interface {
	identity() (Kind, net.HardwareAddr)
	moduleIP() net.IP
	setClientMAC(net.HardwareAddr)
	io.ReaderFrom
	io.WriterTo
}

type ModuleOptions struct {
	MAC              net.HardwareAddr `json:"mac,omitempty"`
	IP               net.IP           `json:"ip,omitempty"`
	Mask             net.IP           `json:"mask,omitempty"`
	Gateway          net.IP           `json:"gateway,omitempty"`
	UseDHCP          bool             `json:"use_dhcp,omitempty"`
	SerialNegotiate  bool             `json:"serial_negotiate,omitempty"`
	EnabledMinorUART bool             `json:"enabled_minor_uart,omitempty"`
}

type UARTService struct {
	Mode             UARTMode   `json:"mode"`
	ClientIP         net.IP     `json:"client_ip"`
	ClientPort       uint16     `json:"client_port"`
	ClientDomain     string     `json:"client_domain"`
	UseDomain        bool       `json:"use_domain"`
	LocalPort        uint16     `json:"local_port"`
	PacketSize       uint16     `json:"packet_size"`
	PacketTimeout    uint16     `json:"packet_timeout"`
	RandomClientPort bool       `json:"random_client_port,omitempty"`
	CloseOnLost      bool       `json:"close_on_lost"`
	ClearOnReconnect bool       `json:"clear_on_reconnect"`
	Baud             uint32     `json:"baud"`
	DataBits         uint8      `json:"data_bits"`
	StopBit          uint8      `json:"stop_bit"`
	Parity           UARTParity `json:"parity"`
}

type NTPService struct {
	Enabled     bool    `json:"enabled"`
	Mode        NTPMode `json:"mode"`
	ClientIP    net.IP  `json:"client_ip"`
	Polling     uint16  `json:"polling"`
	PulseOutput bool    `json:"pulse_output"`
	KeepAlive   bool    `json:"keep_alive"`
}
