//go:build !windows

package agent

import (
	"os"
	"os/exec"
	"slices"
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
		if comm := utils.ReadStringFile("/proc/" + name + "/comm"); comm != "" {
			pidLine = name
		}
	}

	if pidLine == "" {
		pidLine = pgrepFirstPid("-x", name)
	}

	if pidLine == "" {
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

	if statusStr := utils.ReadStringFile("/proc/" + pidLine + "/status"); statusStr != "" {
		for line := range strings.SplitSeq(statusStr, "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if rssKB, err := strconv.ParseFloat(fields[1], 64); err == nil {
						info.Mem = rssKB / 1024.0
					}
				}
				break
			}
		}
	}

	return info
}

// collectAllProcesses returns all running processes on Unix by scanning /proc.
func collectAllProcesses() []*system.ProcessInfo {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var result []*system.ProcessInfo
	for _, entry := range entries {
		if !entry.IsDir() || !isNumeric(entry.Name()) {
			continue
		}
		pidLine := entry.Name()
		comm := utils.ReadStringFile("/proc/" + pidLine + "/comm")
		if comm == "" {
			continue
		}
		comm = strings.TrimSpace(comm)
		info := buildProcessInfo(comm, pidLine)
		if info != nil {
			result = append(result, info)
		}
	}

	// Sort by memory descending and limit to top 50
	slices.SortFunc(result, func(a, b *system.ProcessInfo) int {
		if b.Mem > a.Mem {
			return 1
		}
		if b.Mem < a.Mem {
			return -1
		}
		return 0
	})
	if len(result) > 50 {
		result = result[:50]
	}

	return result
}
