package ch912x

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
)

type CH9126 struct {
	Kind          Kind             `json:"-"`
	Version       string           `json:"version,omitempty"`
	ModuleName    string           `json:"module_name,omitempty"`
	ModuleMAC     net.HardwareAddr `json:"module_mac,omitempty"`
	ClientMAC     net.HardwareAddr `json:"client_mac,omitempty"`
	ModuleOptions *ModuleOptions   `json:"module_options,omitempty"`
	UART1         *UARTService     `json:"uart_1,omitempty"`
	NTP           *NTPService      `json:"ntp,omitempty"`
}

func (p *CH9126) setClientMAC(addr net.HardwareAddr) {
	p.ClientMAC = addr
}

func (p *CH9126) identity() (Kind, net.HardwareAddr) {
	return p.Kind, p.ModuleMAC
}

func (p *CH9126) moduleIP() net.IP {
	if p.ModuleOptions == nil {
		return nil
	}
	return p.ModuleOptions.IP
}

func (p *CH9126) MarshalJSON() ([]byte, error) {
	type Module CH9126
	module := new(struct {
		Product Product `json:"product"`
		*Module
	})
	module.Product = ProductCH9126
	module.Module = (*Module)(p)
	return json.Marshal(module)
}

func (p *CH9126) UnmarshalJSON(data []byte) (err error) {
	type Module CH9126
	module := new(struct {
		Product Product `json:"product"`
		*Module
	})
	module.Module = (*Module)(p)
	err = json.Unmarshal(data, module)
	if err == nil && module.Product != ProductCH9126 {
		err = ErrCH9126InvalidJSON
	}
	return
}

func (p *CH9126) ReadFrom(r io.Reader) (n int64, err error) {
	c := new(ch9126Configuration)
	err = binary.Read(r, binary.LittleEndian, c)
	if err != nil {
		return
	}
	p.Version = string(c.Version[:])
	p.Kind = c.Kind
	p.ModuleName = trimNull(c.ModuleName[:])
	p.ModuleMAC = c.ModuleMAC[:]
	p.ClientMAC = c.ClientMAC[:]
	p.ModuleOptions = &ModuleOptions{
		MAC:              c.ModuleOptions.Address[:],
		IP:               c.ModuleOptions.IP[:],
		Gateway:          c.ModuleOptions.Gateway[:],
		EnabledMinorUART: fromVariantBool(c.UARTService.Enabled),
	}
	p.NTP = &NTPService{
		Enabled:     fromVariantBool(c.NTPService.Enabled),
		Mode:        NTPMode(c.NTPService.Mode - 5),
		ClientIP:    c.NTPService.ClientIP[:],
		Polling:     c.NTPService.Polling,
		PulseOutput: fromVariantBool(c.PulseOutput),
		KeepAlive:   fromVariantBool(c.KeepAlive.Enabled),
	}
	p.UART1 = &UARTService{
		Mode:          UARTMode(c.UARTService.Mode - 1),
		ClientIP:      c.UARTService.ClientIP[:],
		ClientPort:    c.UARTService.ClientPort,
		PacketSize:    c.UARTOptions.PacketSize,
		PacketTimeout: c.UARTOptions.PacketTimeout,
		LocalPort:     c.UARTService.LocalPort,
		Baud:          c.UARTOptions.Baud,
		DataBits:      c.UARTOptions.DataBits,
		StopBit:       c.UARTOptions.StopBit,
		Parity:        UARTParity(c.UARTOptions.Parity),
	}
	return
}

func (p *CH9126) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	r := &ch9126Configuration{Kind: p.Kind}
	copy(r.Header[:], magicCH9126)
	copy(r.ModuleName[:], p.ModuleName)
	copy(r.ModuleMAC[:], p.ModuleMAC)
	copy(r.ClientMAC[:], p.ClientMAC)
	if opt := p.ModuleOptions; opt != nil {
		copy(r.ModuleOptions.Address[:], opt.MAC)
		copy(r.ModuleOptions.IP[:], opt.IP)
		copy(r.ModuleOptions.Mask[:], opt.Mask)
		copy(r.ModuleOptions.Gateway[:], opt.Gateway)
		r.UARTService.Enabled = toVariantBool(opt.EnabledMinorUART)
	}
	if ntp := p.NTP; ntp != nil {
		copy(r.NTPService.ClientIP[:], ntp.ClientIP)
		r.NTPService.Enabled = toVariantBool(ntp.Enabled)
		r.NTPService.Mode = byte(ntp.Mode + 0x05)
		r.NTPService.Polling = ntp.Polling
		r.PulseOutput = toVariantBool(ntp.PulseOutput)
		r.KeepAlive.Enabled = toVariantBool(ntp.KeepAlive)
	}
	if uart := p.UART1; uart != nil {
		r.UARTOptions.Baud = uart.Baud
		r.UARTOptions.DataBits = uart.DataBits
		r.UARTOptions.StopBit = uart.StopBit
		r.UARTOptions.Parity = byte(uart.Parity)
		r.UARTOptions.PacketSize = uart.PacketSize
		r.UARTOptions.PacketTimeout = uart.PacketTimeout
		r.UARTService.Mode = byte(uart.Mode)
		r.UARTService.ClientPort = uart.ClientPort
		r.UARTService.LocalPort = uart.LocalPort
		copy(r.UARTService.ClientIP[:], uart.ClientIP)
	}
	_ = binary.Write(&buf, binary.LittleEndian, r)
	_, _ = buf.Write(make([]byte, 367-buf.Len()))
	return buf.WriteTo(w)
}

type ch9126Configuration struct {
	Header      [0x40]byte // CH9126_MODULE_V1.03
	Version     [4]byte    // V110
	ModuleName  [0x40]byte
	_           [8]byte
	Kind        Kind
	ModuleMAC   [6]byte
	ClientMAC   [6]byte
	UARTOptions struct {
		Baud          uint32
		DataBits      byte
		StopBit       byte
		Parity        byte // Evan, Odd, Mark, Space, None
		PacketSize    uint16
		PacketTimeout uint16
		_             byte
	}
	ModuleOptions struct {
		Address [6]byte
		IP      [4]byte
		Mask    [4]byte
		Gateway [4]byte
		_       [13]byte
	}
	PulseOutput byte
	_           [3]byte
	KeepAlive   struct {
		Enabled  byte
		Time     uint32
		Interval uint32
		Probes   uint32
	}
	NTPService struct {
		Enabled  byte
		Mode     byte // 5: NTP Server, 6: NTP Client
		_        [2]byte
		ClientIP [4]byte
		Polling  uint16
		_        [3]byte
	}
	UARTService struct {
		Enabled    byte
		Mode       byte // _, TCP Server, TCP Client, UDP Server, UDP Client
		_          byte
		_          byte
		ClientIP   [4]byte
		ClientPort uint16
		_          byte
		LocalPort  uint16
	}
}
