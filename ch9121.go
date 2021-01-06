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

type CH9121 struct {
	Kind          Kind             `json:"-"`
	Version       string           `json:"version,omitempty"`
	ModuleName    string           `json:"module_name,omitempty"`
	ModuleMAC     net.HardwareAddr `json:"module_mac,omitempty"`
	ClientMAC     net.HardwareAddr `json:"client_mac,omitempty"`
	ModuleOptions *ModuleOptions   `json:"module_options,omitempty"`
	UART1         *UARTService     `json:"uart_1,omitempty"`
	UART2         *UARTService     `json:"uart_2,omitempty"`
}

func (p *CH9121) setClientMAC(addr net.HardwareAddr) {
	p.ClientMAC = addr
}

func (p *CH9121) identity() (Kind, net.HardwareAddr) {
	return p.Kind, p.ModuleMAC
}

func (p *CH9121) moduleIP() net.IP {
	if p.ModuleOptions == nil {
		return nil
	}
	return p.ModuleOptions.IP
}

func (p *CH9121) MarshalJSON() ([]byte, error) {
	type Module CH9121
	module := new(struct {
		Product Product `json:"product"`
		*Module
	})
	module.Product = ProductCH9121
	module.Module = (*Module)(p)
	return json.Marshal(module)
}

func (p *CH9121) UnmarshalJSON(data []byte) (err error) {
	type Module CH9121
	module := new(struct {
		Product Product `json:"product"`
		*Module
	})
	module.Module = (*Module)(p)
	err = json.Unmarshal(data, module)
	if err == nil && module.Product != ProductCH9121 {
		err = ErrCH9121InvalidJSON
	}
	return
}

func (p *CH9121) ReadFrom(r io.Reader) (n int64, err error) {
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
	c := new(ch9121Configuration)
	err = binary.Read(r, binary.LittleEndian, c)
	if err != nil {
		return
	}
	p.ModuleName = trimNull(c.ModuleName[:])
	p.ModuleMAC = c.ModuleMAC[:]
	p.ClientMAC = c.ClientMAC[:]
	p.ModuleOptions = &ModuleOptions{
		MAC:              c.ModuleOptions.ModuleMAC[:],
		IP:               c.ModuleOptions.IP[:],
		Mask:             c.ModuleOptions.Mask[:],
		Gateway:          c.ModuleOptions.Gateway[:],
		UseDHCP:          fromVariantBool(c.ModuleOptions.DHCP),
		SerialNegotiate:  fromVariantBool(c.ModuleOptions.SerialNegotiate),
		EnabledMinorUART: fromVariantBool(c.EnabledUART2),
	}
	setUART := func(uart ch9121UART) *UARTService {
		return &UARTService{
			Mode:             UARTMode(uart.Mode),
			ClientIP:         uart.ClientIP[:],
			ClientPort:       uart.ClientPort,
			LocalPort:        uart.TargetPort,
			PacketSize:       uart.RXSize,
			PacketTimeout:    uart.RXTimeout,
			RandomClientPort: fromVariantBool(uart.RandomClientPort),
			CloseOnLost:      fromVariantBool(uart.CloseOnLost),
			ClearOnReconnect: fromVariantBool(uart.ClearOnTimeout),
			Baud:             uart.BaudRate,
			DataBits:         uart.DataBits,
			StopBit:          uart.StopBit,
			Parity:           fromCH9121Parity(uart.Parity),
			UseDomain:        fromVariantBool(uart.UseDomain),
			ClientDomain:     trimNull(uart.ClientDomain[:]),
		}
	}
	p.UART1 = setUART(c.UART1)
	p.UART2 = setUART(c.UART2)
	return
}

func (p *CH9121) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	h := &ch9121Header{Kind: p.Kind}
	copy(h.Header[:], magicCH9121)
	_ = binary.Write(&buf, binary.LittleEndian, h)
	r := new(ch9121Configuration)
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
		r.EnabledUART2 = toVariantBool(opt.EnabledMinorUART)
	}
	setUART := func(uart *UARTService) ch9121UART {
		if uart == nil {
			return ch9121UART{}
		}
		options := ch9121UART{
			Mode:             byte(uart.Mode),
			RandomClientPort: toVariantBool(uart.RandomClientPort),
			ClientPort:       uart.ClientPort,
			TargetPort:       uart.LocalPort,
			BaudRate:         uart.Baud,
			DataBits:         uart.DataBits,
			StopBit:          uart.StopBit,
			Parity:           toCH9121Parity(uart.Parity),
			CloseOnLost:      toVariantBool(uart.CloseOnLost),
			RXSize:           uart.PacketSize,
			RXTimeout:        uart.PacketTimeout,
			ClearOnTimeout:   toVariantBool(uart.ClearOnReconnect),
			UseDomain:        toVariantBool(uart.UseDomain),
		}
		copy(options.ClientIP[:], uart.ClientIP)
		copy(options.ClientDomain[:], uart.ClientDomain)
		return options
	}
	r.UART1 = setUART(p.UART1)
	r.UART2 = setUART(p.UART2)
	_ = binary.Write(&buf, binary.LittleEndian, r)
	_, _ = buf.Write(make([]byte, 285-buf.Len()))
	return buf.WriteTo(w)
}

func fromCH9121Parity(p byte) UARTParity {
	if p == 4 {
		return ParityNone
	}
	return UARTParity(p + 1)
}

func toCH9121Parity(p UARTParity) byte {
	if p == ParityNone {
		return 0
	}
	return byte(p - 1)
}

type ch9121Header struct {
	Header [16]byte
	Kind   Kind
}

type ch9121Discovery struct {
	ModuleMAC [6]byte
	ClientMAC [6]byte
	_         byte
	IP        [4]byte
}

type ch9121Configuration struct {
	ModuleMAC     [6]byte
	ClientMAC     [6]byte
	_             [6]byte
	ModuleName    [21]byte
	ModuleOptions ch9121ModuleOptions
	_             [9]byte
	EnabledUART2  byte
	UART2, UART1  ch9121UART
}

type ch9121ModuleOptions struct {
	ModuleMAC       [6]byte
	IP              [4]byte
	Gateway         [4]byte
	Mask            [4]byte
	DHCP            byte
	_               [20]byte
	SerialNegotiate byte
}

type ch9121UART struct {
	Mode             byte // TCP Server, TCP Client, UDP Server, UDP Client
	RandomClientPort byte
	ClientPort       uint16
	ClientIP         [4]byte
	TargetPort       uint16
	BaudRate         uint32
	DataBits         byte
	StopBit          byte
	Parity           byte // None, Evan, Odd, Mark, Space
	CloseOnLost      byte
	RXSize           uint16
	RXTimeout        uint16
	_                [5]byte
	ClearOnTimeout   byte
	UseDomain        byte
	ClientDomain     [21]byte
	_                [15]byte
}
