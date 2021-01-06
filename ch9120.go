package ch912x

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strconv"
)

type CH9120 struct {
	Kind          Kind             `json:"-"`
	Version       string           `json:"version,omitempty"`
	ModuleName    string           `json:"module_name,omitempty"`
	ModuleMAC     net.HardwareAddr `json:"module_mac,omitempty"`
	ClientMAC     net.HardwareAddr `json:"client_mac,omitempty"`
	ModuleOptions *ModuleOptions   `json:"module_options,omitempty"`
	UART1         *UARTService     `json:"uart_1,omitempty"`
}

func (p *CH9120) setClientMAC(addr net.HardwareAddr) {
	p.ClientMAC = addr
}

func (p *CH9120) identity() (Kind, net.HardwareAddr) {
	return p.Kind, p.ModuleMAC
}

func (p *CH9120) moduleIP() net.IP {
	if p.ModuleOptions == nil {
		return nil
	}
	return p.ModuleOptions.IP
}

func (p *CH9120) MarshalJSON() ([]byte, error) {
	type Module CH9120
	module := new(struct {
		Product Product `json:"product"`
		*Module
	})
	module.Product = ProductCH9120
	module.Module = (*Module)(p)
	return json.Marshal(module)
}

func (p *CH9120) UnmarshalJSON(data []byte) (err error) {
	type Module CH9120
	module := new(struct {
		Product Product `json:"product"`
		*Module
	})
	module.Module = (*Module)(p)
	err = json.Unmarshal(data, module)
	if err == nil && module.Product != ProductCH9120 {
		err = ErrCH9120InvalidJSON
	}
	return
}

func (p *CH9120) ReadFrom(r io.Reader) (n int64, err error) {
	header := new(ch9121Header)
	err = binary.Read(r, binary.LittleEndian, header)
	if err != nil {
		return
	}
	p.Kind = header.Kind
	if p.Kind == KindDiscoveryResponse {
		discovery := new(ch9121Discovery)
		err = binary.Read(r, binary.LittleEndian, discovery)
		p.ModuleMAC = discovery.ModuleMAC[:]
		p.ClientMAC = discovery.ClientMAC[:]
		p.ModuleOptions = &ModuleOptions{IP: discovery.IP[:]}
		buf := bufio.NewReader(r)
		moduleName, _ := buf.ReadBytes(0)
		p.ModuleName = trimNull(moduleName)
		version, _ := buf.ReadByte()
		p.Version = strconv.Itoa(int(version))
		return
	}
	c := new(ch9120Configuration)
	err = binary.Read(r, binary.LittleEndian, c)
	if err != nil {
		return
	}
	p.ModuleName = trimNull(c.ModuleName[:])
	p.ModuleMAC = c.ModuleMAC[:]
	p.ClientMAC = c.ClientMAC[:]
	p.ModuleOptions = &ModuleOptions{
		MAC:             c.ModuleOptions.ModuleMAC[:],
		IP:              c.ModuleOptions.IP[:],
		Mask:            c.ModuleOptions.Mask[:],
		Gateway:         c.ModuleOptions.Gateway[:],
		UseDHCP:         fromVariantBool(c.ModuleOptions.DHCP),
		SerialNegotiate: fromVariantBool(c.ModuleOptions.SerialNegotiate),
	}
	p.UART1 = &UARTService{
		Mode:             UARTMode(c.UART.Mode),
		ClientIP:         c.UART.ClientIP[:],
		ClientPort:       c.UART.ClientPort,
		LocalPort:        c.UART.TargetPort,
		PacketSize:       c.UART.RXSize,
		PacketTimeout:    c.UART.RXTimeout,
		RandomClientPort: fromVariantBool(c.UART.RandomClientPort),
		CloseOnLost:      fromVariantBool(c.UART.CloseOnLost),
		ClearOnReconnect: fromVariantBool(c.UART.ClearOnTimeout),
		Baud:             c.UART.Baud,
		DataBits:         c.UART.DataBits,
		StopBit:          c.UART.StopBit,
		Parity:           fromCH9121Parity(c.UART.Parity),
		UseDomain:        fromVariantBool(c.UART.UseDomain),
		ClientDomain:     trimNull(c.UART.ClientDomain[:]),
	}
	return
}

func (p *CH9120) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	h := &ch9121Header{Kind: p.Kind}
	copy(h.Header[:], magicCH9120)
	_ = binary.Write(&buf, binary.LittleEndian, h)
	r := new(ch9120Configuration)
	copy(r.ModuleMAC[:], p.ModuleMAC)
	copy(r.ClientMAC[:], p.ClientMAC)
	copy(r.ModuleName[:], p.ModuleName)
	if opt := p.ModuleOptions; opt != nil {
		copy(r.ModuleOptions.ModuleMAC[:], opt.MAC)
		copy(r.ModuleOptions.IP[:], opt.IP)
		copy(r.ModuleOptions.Mask[:], opt.Mask)
		copy(r.ModuleOptions.Gateway[:], opt.Gateway)
		r.ModuleOptions.DHCP = toVariantBool(opt.UseDHCP)
		r.ModuleOptions.SerialNegotiate = toVariantBool(opt.SerialNegotiate)
	}
	if p.UART1 != nil {
		r.UART = ch9120UART{
			Mode:             byte(p.UART1.Mode),
			RandomClientPort: toVariantBool(p.UART1.RandomClientPort),
			ClientPort:       p.UART1.ClientPort,
			TargetPort:       p.UART1.LocalPort,
			Baud:             p.UART1.Baud,
			DataBits:         p.UART1.DataBits,
			StopBit:          p.UART1.StopBit,
			Parity:           toCH9121Parity(p.UART1.Parity),
			CloseOnLost:      toVariantBool(p.UART1.CloseOnLost),
			RXSize:           p.UART1.PacketSize,
			RXTimeout:        p.UART1.PacketTimeout,
			ClearOnTimeout:   toVariantBool(p.UART1.ClearOnReconnect),
		}
		copy(r.UART.ClientIP[:], p.UART1.ClientIP)
		r.UART.UseDomain = toVariantBool(p.UART1.UseDomain)
		copy(r.UART.ClientDomain[:], p.UART1.ClientDomain)
	}
	_ = binary.Write(&buf, binary.LittleEndian, r)
	_, _ = buf.Write(make([]byte, 285-buf.Len()))
	return buf.WriteTo(w)
}

type ch9120Configuration struct {
	ModuleMAC     [6]byte
	ClientMAC     [6]byte
	_             [6]byte
	ModuleName    [21]byte
	ModuleOptions ch9121ModuleOptions
	_             [75]byte
	UART          ch9120UART
}

type ch9120UART struct {
	Mode             byte
	RandomClientPort byte
	ClientPort       uint16
	ClientIP         [4]byte
	TargetPort       uint16
	Baud             uint32
	DataBits         byte
	StopBit          byte
	Parity           byte // None, Evan, Odd, Mark, Space
	CloseOnLost      byte
	RXSize           uint16
	_                [2]byte
	RXTimeout        uint16
	_                [3]byte
	ClearOnTimeout   byte
	UseDomain        byte
	ClientDomain     [20]byte
}
