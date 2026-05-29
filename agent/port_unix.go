//go:build !windows

package agent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
)

// collectAllPorts returns all listening ports on Linux by reading /proc/net/tcp and /proc/net/udp.
func collectAllPorts() []*system.PortInfo {
	var result []*system.PortInfo
	result = append(result, parseProcNetPorts("/proc/net/tcp", "tcp")...)
	result = append(result, parseProcNetPorts("/proc/net/tcp6", "tcp")...)
	result = append(result, parseProcNetPorts("/proc/net/udp", "udp")...)
	result = append(result, parseProcNetPorts("/proc/net/udp6", "udp")...)
	return result
}

// parseProcNetPorts parses /proc/net/tcp or /proc/net/udp to find listening ports.
// Format: sl local_address rem_address st tx_queue:rx_queue ...
// State 0A = LISTEN
func parseProcNetPorts(path string, protocol string) []*system.PortInfo {
	data := utils.ReadStringFile(path)
	if data == "" {
		return nil
	}

	seen := make(map[uint16]bool)
	var result []*system.PortInfo

	lines := strings.SplitSeq(data, "\n")
	for line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// State 0A = LISTEN (10 decimal)
		if fields[3] != "0A" {
			continue
		}

		// local_address is in hex format: AABBCCCC where AA=byte3 BB=byte2 CC=byte1 DD=byte0 (little endian IP)
		// port is the last 4 hex chars after the colon
		localAddr := fields[1]
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			continue
		}
		portHex := parts[1]
		port, err := strconv.ParseUint(portHex, 16, 16)
		if err != nil || port == 0 {
			continue
		}

		if seen[uint16(port)] {
			continue
		}
		seen[uint16(port)] = true

		service := fmt.Sprintf("%d/%s", port, protocol)
		result = append(result, &system.PortInfo{
			Port:     uint16(port),
			Protocol: protocol,
			Status:   "open",
			Service:  service,
		})
	}
	return result
}
