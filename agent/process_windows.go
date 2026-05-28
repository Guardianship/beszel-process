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
// Supports three lookup strategies:
//  1. If name is a numeric PID, look up via tasklist /FI "PID eq <pid>"
//  2. Try tasklist /FI "IMAGENAME eq <name>" (exact image name match)
//  3. If name doesn't end with .exe, retry with .exe appended
func collectProcessInfo(name string) *system.ProcessInfo {
	if isNumeric(name) {
		// Strategy 1: PID lookup
		if info := tasklistByFilter(fmt.Sprintf("PID eq %s", name), name); info != nil {
			return info
		}
	}

	// Strategy 2: exact image name match
	if info := tasklistByFilter(fmt.Sprintf("IMAGENAME eq %s", name), name); info != nil {
		return info
	}

	// Strategy 3: auto-append .exe if not present
	if !strings.HasSuffix(strings.ToLower(name), ".exe") {
		if info := tasklistByFilter(fmt.Sprintf("IMAGENAME eq %s.exe", name), name); info != nil {
			return info
		}
	}

	return nil
}

// tasklistByFilter runs tasklist with the given filter and parses the first matching result.
func tasklistByFilter(filter, displayName string) *system.ProcessInfo {
	cmd := exec.Command("tasklist", "/FI", filter, "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// tasklist returns an info message (not CSV) when no results match
		if !strings.Contains(line, ",") {
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
			Name:   displayName,
			Pid:    uint32(pid),
			Status: "running",
			Mem:    memMB,
		}
	}
	return nil
}

// isNumeric returns true if s consists only of digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
