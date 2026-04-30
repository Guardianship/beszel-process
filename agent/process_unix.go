//go:build !windows

package agent

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
)

// collectProcessInfo collects process information on Unix systems using pgrep and /proc.
func collectProcessInfo(name string) *system.ProcessInfo {
	cmd := exec.Command("pgrep", "-x", name)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	pidStr := strings.TrimSpace(string(output))
	pidLines := strings.Split(pidStr, "\n")
	pidLine := pidLines[0]
	pid, err := strconv.ParseUint(pidLine, 10, 32)
	if err != nil {
		return nil
	}

	info := &system.ProcessInfo{
		Name:   name,
		Pid:    uint32(pid),
		Status: "running",
	}

	// Try to read from /proc on Linux
	if statStr := utils.ReadStringFile("/proc/" + pidLine + "/stat"); statStr != "" {
		lastParen := strings.LastIndex(statStr, ")")
		if lastParen >= 0 {
			fields := strings.Split(statStr[lastParen+2:], " ")
			if len(fields) > 19 {
				if starttime, err := strconv.ParseUint(fields[19], 10, 64); err == nil {
					_ = starttime
				}
			}
		}
	}

	// Read memory from /proc/pid/status (VmRSS in kB)
	if statusStr := utils.ReadStringFile("/proc/" + pidLine + "/status"); statusStr != "" {
		for line := range strings.SplitSeq(statusStr, "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if rssKB, err := strconv.ParseFloat(fields[1], 64); err == nil {
						info.Mem = rssKB / 1024.0 // KB to MB
					}
				}
				break
			}
		}
	}

	return info
}
