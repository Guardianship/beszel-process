//go:build windows

package agent

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/henrygd/beszel/internal/entities/system"
)

// collectAllPorts returns all listening ports on Windows using netstat.
func collectAllPorts() []*system.PortInfo {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []*system.PortInfo

	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Only interested in LISTENING state
		state := fields[3]
		if state != "LISTENING" {
			continue
		}

		localAddr := fields[1]
		protocol := "tcp"
		if strings.HasPrefix(strings.ToLower(fields[0]), "udp") {
			protocol = "udp"
			// UDP has no state column, fields shift
			if len(fields) < 3 {
				continue
			}
			localAddr = fields[1]
		}

		// Extract port from address (e.g. "0.0.0.0:80" or "[::]:443")
		portStr := extractPort(localAddr)
		if portStr == "" {
			continue
		}
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil || port == 0 {
			continue
		}

		key := fmt.Sprintf("%d/%s", port, protocol)
		if seen[key] {
			continue
		}
		seen[key] = true

		result = append(result, &system.PortInfo{
			Port:     uint16(port),
			Protocol: protocol,
			Status:   "open",
			Service:  key,
		})
	}
	return result
}

// extractPort extracts the port number from an address string like "0.0.0.0:80" or "[::]:443".
func extractPort(addr string) string {
	// Handle IPv6: [::]:port
	if idx := strings.LastIndex(addr, "]:"); idx >= 0 {
		return addr[idx+2:]
	}
	// Handle IPv4: host:port
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[idx+1:]
	}
	return ""
}
