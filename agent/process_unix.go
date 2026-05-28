//go:build !windows

package agent

import (
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
)

// collectProcessInfo collects process information on Unix systems.
// Supports three lookup strategies:
//  1. If name is a numeric PID, look up directly via /proc
//  2. Try pgrep -x (exact comm name match)
//  3. Fallback to pgrep -f (match against full command line)
func collectProcessInfo(name string) *system.ProcessInfo {
	var pidLine string

	if isNumeric(name) {
		// Strategy 1: PID lookup — verify process exists via /proc/<pid>/comm
		if comm := utils.ReadStringFile("/proc/" + name + "/comm"); comm != "" {
			pidLine = name
		}
	}

	if pidLine == "" {
		// Strategy 2: exact comm name match
		pidLine = pgrepFirstPid("-x", name)
	}

	if pidLine == "" {
		// Strategy 3: full command line match
		pidLine = pgrepFirstPid("-f", name)
	}

	if pidLine == "" {
		return nil
	}

	return buildProcessInfo(name, pidLine)
}

// isNumeric returns true if s consists only of digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// pgrepFirstPid runs pgrep with the given flag and returns the first PID line, or empty string.
func pgrepFirstPid(flag, name string) string {
	cmd := exec.Command("pgrep", flag, name)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return ""
	}
	pidStr := strings.TrimSpace(string(output))
	pidLines := strings.Split(pidStr, "\n")
	return pidLines[0]
}

// buildProcessInfo creates a ProcessInfo from a PID string, reading /proc for memory data.
func buildProcessInfo(name, pidLine string) *system.ProcessInfo {
	pid, err := strconv.ParseUint(pidLine, 10, 32)
	if err != nil {
		return nil
	}

	info := &system.ProcessInfo{
		Name:   name,
		Pid:    uint32(pid),
		Status: "running",
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
