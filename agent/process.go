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
}

// newProcessManager creates a new processManager from the PROCESS_NAMES env var.
func newProcessManager() *processManager {
	names := getProcessNames()
	if len(names) == 0 {
		return nil
	}
	slog.Info("Process monitoring", "names", names)
	return &processManager{processNames: names}
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
