package agent

import (
	"log/slog"
	"strings"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
)

// processManager collects stats for monitored processes.
type processManager struct {
	processNames []string
	monitorAll   bool
}

// newProcessManager creates a new processManager from the PROCESS_NAMES env var.
// Use PROCESS_NAMES=* to monitor all running processes.
// If PROCESS_NAMES is not set, defaults to monitoring top 50 processes by memory.
func newProcessManager() *processManager {
	names := getProcessNames()
	pm := &processManager{processNames: names}
	if len(names) == 0 || (len(names) == 1 && names[0] == "*") {
		pm.monitorAll = true
		slog.Info("Process monitoring top 50 by memory")
	} else {
		slog.Info("Process monitoring", "names", names)
	}
	return pm
}

// getProcessNames reads the PROCESS_NAMES env var (comma-separated).
func getProcessNames() []string {
	var names []string
	if envNames, _ := utils.GetEnv("PROCESS_NAMES"); envNames != "" {
		for name := range strings.SplitSeq(envNames, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

// getProcessStats collects stats for each monitored process.
func (pm *processManager) getProcessStats() []*system.ProcessInfo {
	if pm.monitorAll {
		return collectAllProcesses()
	}
	result := make([]*system.ProcessInfo, 0, len(pm.processNames))
	for _, name := range pm.processNames {
		info := collectProcessInfo(name)
		if info != nil {
			result = append(result, info)
		} else {
			result = append(result, &system.ProcessInfo{
				Name:   name,
				Status: "stopped",
			})
		}
	}
	return result
}
