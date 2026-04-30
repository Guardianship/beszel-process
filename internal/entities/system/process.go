package system

// ProcessInfo represents the status of a monitored process on a system.
type ProcessInfo struct {
	Name   string  `json:"n" cbor:"0,keyasint"`
	Pid    uint32  `json:"p" cbor:"1,keyasint"`
	Cpu    float64 `json:"c" cbor:"2,keyasint"`
	Mem    float64 `json:"m" cbor:"3,keyasint"`
	Status string  `json:"s" cbor:"4,keyasint"` // "running", "stopped", "zombie", etc.
	Uptime uint64  `json:"u" cbor:"5,keyasint"` // uptime in seconds
}
