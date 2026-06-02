//go:build windows

package agent

import (
	"encoding/csv"
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

	// Build PID -> process name map
	pidMap := buildPidProcessMap()

	seen := make(map[string]bool)
	var result []*system.PortInfo

	utf8Output := gbkToUTF8(output)
	for line := range strings.SplitSeq(utf8Output, "\n") {
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
		pidFieldIdx := 4
		if strings.HasPrefix(strings.ToLower(fields[0]), "udp") {
			protocol = "udp"
			// UDP has no state column, fields shift
			if len(fields) < 3 {
				continue
			}
			localAddr = fields[1]
			pidFieldIdx = 3
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

		// Get process name from PID
		processName := ""
		if pidFieldIdx < len(fields) {
			pid := strings.TrimSpace(fields[pidFieldIdx])
			if name, ok := pidMap[pid]; ok {
				processName = name
			}
		}

		// Use process name as service label, fallback to port/protocol
		service := key
		if processName != "" {
			service = processName
		}

		result = append(result, &system.PortInfo{
			Port:     uint16(port),
			Protocol: protocol,
			Status:   "open",
			Service:  service,
			Process:  processName,
		})
	}
	return result
}

// buildPidProcessMap builds a map of PID -> process name using tasklist.
func buildPidProcessMap() map[string]string {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	utf8Output := gbkToUTF8(output)
	reader := csv.NewReader(strings.NewReader(utf8Output))
	records, err := reader.ReadAll()
	if err != nil {
		return nil
	}

	pidMap := make(map[string]string, len(records))
	for _, fields := range records {
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		// Use friendly name if available
		if friendly, ok := processFriendlyNames[name]; ok {
			name = friendly
		}
		pidMap[fields[1]] = name
	}
	return pidMap
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
