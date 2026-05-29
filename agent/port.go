package agent

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
)

// portManager monitors the status of network ports.
type portManager struct {
	portChecks []portCheck
	monitorAll bool
}

type portCheck struct {
	Port     uint16
	Protocol string
	Service  string
}

// newPortManager creates a new portManager from the PORT_CHECKS env var.
// Format: "80,443/tcp,8080/tcp:myapp" (port[/protocol][:label])
// Use PORT_CHECKS=* to monitor all listening ports.
func newPortManager() *portManager {
	checks := getPortChecks()
	if len(checks) == 0 {
		return nil
	}
	pm := &portManager{portChecks: checks}
	if len(checks) == 1 && checks[0].Service == "*" {
		pm.monitorAll = true
		slog.Info("Port monitoring all listening ports")
	} else {
		slog.Info("Port monitoring", "count", len(checks))
	}
	return pm
}

// getPortChecks parses the PORT_CHECKS env var.
func getPortChecks() []portCheck {
	envVal, _ := utils.GetEnv("PORT_CHECKS")
	if envVal == "" {
		return nil
	}

	// Handle wildcard: PORT_CHECKS=*
	trimmed := strings.TrimSpace(envVal)
	if trimmed == "*" {
		return []portCheck{{Service: "*"}}
	}

	var checks []portCheck
	for entry := range strings.SplitSeq(envVal, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		pc := portCheck{Protocol: "tcp"}

		// Check for label: "8080/tcp:myapp"
		label := ""
		if idx := strings.LastIndex(entry, ":"); idx > 0 {
			candidate := entry[idx+1:]
			if !strings.Contains(candidate, "/") && !strings.Contains(candidate, "]") {
				label = candidate
				entry = entry[:idx]
			}
		}

		// Check for protocol: "443/udp"
		if idx := strings.LastIndex(entry, "/"); idx > 0 {
			pc.Protocol = entry[idx+1:]
			entry = entry[:idx]
		}

		port, err := parseUint16(entry)
		if err != nil {
			slog.Warn("Invalid port number", "entry", entry, "err", err)
			continue
		}
		pc.Port = port
		if label != "" {
			pc.Service = label
		} else {
			pc.Service = fmt.Sprintf("%d/%s", port, pc.Protocol)
		}

		checks = append(checks, pc)
	}
	return checks
}

// getPortStats checks the status of each monitored port.
func (pm *portManager) getPortStats() []*system.PortInfo {
	if pm.monitorAll {
		return collectAllPorts()
	}
	result := make([]*system.PortInfo, 0, len(pm.portChecks))
	for _, pc := range pm.portChecks {
		status := checkPortStatus(pc.Port, pc.Protocol)
		result = append(result, &system.PortInfo{
			Port:     pc.Port,
			Protocol: pc.Protocol,
			Status:   status,
			Service:  pc.Service,
		})
	}
	return result
}

// checkPortStatus attempts to connect to a port to determine if it's open.
func checkPortStatus(port uint16, protocol string) string {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	timeout := 3 * time.Second

	switch protocol {
	case "udp":
		conn, err := net.DialTimeout("udp", address, timeout)
		if err != nil {
			return "closed"
		}
		conn.Close()
		return "open"
	default: // tcp
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return "closed"
		}
		conn.Close()
		return "open"
	}
}

func parseUint16(s string) (uint16, error) {
	v, err := parseUint(s)
	return uint16(v), err
}

func parseUint(s string) (uint64, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid port: %q", s)
		}
		n = n*10 + uint64(c-'0')
		if n > 65535 {
			return 0, fmt.Errorf("port out of range: %q", s)
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("port cannot be 0")
	}
	return n, nil
}
