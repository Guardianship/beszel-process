package system

// PortInfo represents the status of a monitored port on a system.
type PortInfo struct {
	Port     uint16 `json:"p" cbor:"0,keyasint"`
	Protocol string `json:"t" cbor:"1,keyasint"` // "tcp" or "udp"
	Status   string `json:"s" cbor:"2,keyasint"` // "open" or "closed"
	Service  string `json:"n" cbor:"3,keyasint"` // service name or label
	Process  string `json:"pr,omitempty" cbor:"4,keyasint,omitempty"` // process name
}
