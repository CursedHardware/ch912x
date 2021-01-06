package ch912x

type (
	Product    string
	Kind       byte
	UARTMode   byte
	UARTParity byte
	NTPMode    byte
)

const (
	magicCH9120                      = "CH9120_CFG_FLAG"
	magicCH9121                      = "CH9121_CFG_FLAG"
	magicCH9126                      = "CH9126_MODULE_V1.03"
	magicModule                      = "NET_MODULE_COMM"
	ProductCH9120         Product    = "CH9120"
	ProductCH9121         Product    = "CH9121"
	ProductCH9126         Product    = "CH9126"
	KindPushRequest       Kind       = 0x01
	KindPullRequest       Kind       = 0x02
	KindResetRequest      Kind       = 0x03
	KindDiscoveryRequest  Kind       = 0x04
	KindPushResponse      Kind       = 0x81
	KindPullResponse      Kind       = 0x82
	KindResetResponse     Kind       = 0x83
	KindDiscoveryResponse Kind       = 0x84
	TCPServer             UARTMode   = 0x00
	TCPClient             UARTMode   = 0x01
	UDPServer             UARTMode   = 0x02
	UDPClient             UARTMode   = 0x03
	ParityNone            UARTParity = 0x00
	ParityEven            UARTParity = 0x01
	ParityOdd             UARTParity = 0x02
	ParityMark            UARTParity = 0x03
	ParitySpace           UARTParity = 0x04
	NTPServer             NTPMode    = 0x00
	NTPClient             NTPMode    = 0x01
)
