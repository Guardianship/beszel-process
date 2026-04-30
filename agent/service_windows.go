//go:build windows

package agent

import (
	"log/slog"
	"os/exec"
	"strings"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/systemd"
)

// serviceManager collects stats for Windows services.
type serviceManager struct {
	hasFreshStats bool
	serviceNames  []string
}

// newServiceManager creates a new serviceManager from the SERVICE_NAMES env var.
func newServiceManager() (*serviceManager, error) {
	names := getServiceNames()
	if len(names) == 0 {
		return &serviceManager{}, nil
	}
	slog.Info("Windows service monitoring", "names", names)
	return &serviceManager{serviceNames: names}, nil
}

// getServiceNames reads the SERVICE_NAMES env var (comma-separated).
func getServiceNames() []string {
	var names []string
	if envNames, _ := utils.GetEnv("SERVICE_NAMES"); envNames != "" {
		for name := range strings.SplitSeq(envNames, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

// getServiceStats returns stats for all monitored Windows services.
func (sm *serviceManager) getServiceStats() []*systemd.Service {
	if len(sm.serviceNames) == 0 {
		return nil
	}

	var services []*systemd.Service
	for _, name := range sm.serviceNames {
		service := sm.queryService(name)
		services = append(services, service)
	}
	sm.hasFreshStats = true
	return services
}

// getServiceStatsCount returns the number of monitored services.
func (sm *serviceManager) getServiceStatsCount() int {
	return len(sm.serviceNames)
}

// getFailedServiceCount returns the number of services in a failed/stopped state.
func (sm *serviceManager) getFailedServiceCount() uint16 {
	services := sm.getServiceStats()
	var count uint16
	for _, s := range services {
		if s.State == systemd.StatusInactive || s.State == systemd.StatusFailed {
			count++
		}
	}
	return count
}

// queryService queries a single Windows service using sc.exe.
func (sm *serviceManager) queryService(name string) *systemd.Service {
	cmd := exec.Command("sc", "query", name)
	output, err := cmd.Output()
	if err != nil {
		return &systemd.Service{Name: name, State: systemd.StatusInactive}
	}

	state := parseWindowsServiceState(string(output))
	return &systemd.Service{Name: name, State: state}
}

// parseWindowsServiceState extracts the state from sc query output.
func parseWindowsServiceState(output string) systemd.ServiceState {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STATE") {
			if strings.Contains(line, "RUNNING") {
				return systemd.StatusActive
			}
			if strings.Contains(line, "STOPPED") {
				return systemd.StatusInactive
			}
			if strings.Contains(line, "FAILED") {
				return systemd.StatusFailed
			}
			if strings.Contains(line, "START_PENDING") || strings.Contains(line, "STOP_PENDING") {
				return systemd.StatusActivating
			}
		}
	}
	return systemd.StatusInactive
}
