//go:build windows

package agent

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/henrygd/beszel/internal/entities/system"
)

// collectProcessInfo collects process information on Windows using tasklist.
func collectProcessInfo(name string) *system.ProcessInfo {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", name), "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	// Parse CSV output: "name","pid","session","session#","mem"
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, name) {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			continue
		}

		pidStr := strings.Trim(fields[1], "\"")
		pid, err := strconv.ParseUint(pidStr, 10, 32)
		if err != nil {
			continue
		}

		memStr := strings.Trim(fields[4], "\" ")
		memStr = strings.TrimSuffix(memStr, " K")
		memStr = strings.TrimSuffix(memStr, " k")
		var memMB float64
		if v, err := strconv.ParseFloat(memStr, 64); err == nil {
			memMB = v / 1024.0 // K to MB
		}

		return &system.ProcessInfo{
			Name:   name,
			Pid:    uint32(pid),
			Status: "running",
			Mem:    memMB,
		}
	}
	return nil
}
