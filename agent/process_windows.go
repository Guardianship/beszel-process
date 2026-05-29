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
		if info := tasklistByFilter(fmt.Sprintf("PID eq %s", name), name); info != nil {
			return info
		}
	}

	if info := tasklistByFilter(fmt.Sprintf("IMAGENAME eq %s", name), name); info != nil {
		return info
	}

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

	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ",") {
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

		memMB := parseTasklistMem(fields[4])

		return &system.ProcessInfo{
			Name:   displayName,
			Pid:    uint32(pid),
			Status: "running",
			Mem:    memMB,
		}
	}
	return nil
}

// collectAllProcesses returns all running processes on Windows using tasklist.
func collectAllProcesses() []*system.ProcessInfo {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	var result []*system.ProcessInfo
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ",") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			continue
		}

		name := strings.Trim(fields[0], "\"")
		pidStr := strings.Trim(fields[1], "\"")
		pid, err := strconv.ParseUint(pidStr, 10, 32)
		if err != nil {
			continue
		}

		memMB := parseTasklistMem(fields[4])

		result = append(result, &system.ProcessInfo{
			Name:   name,
			Pid:    uint32(pid),
			Status: "running",
			Mem:    memMB,
		})
	}
	return result
}

// parseTasklistMem parses the memory column from tasklist CSV output (e.g. "134,752 K") to MB.
func parseTasklistMem(memField string) float64 {
	memStr := strings.Trim(memField, "\" ")
	memStr = strings.TrimSuffix(memStr, " K")
	memStr = strings.TrimSuffix(memStr, " k")
	if v, err := strconv.ParseFloat(memStr, 64); err == nil {
		return v / 1024.0
	}
	return 0
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
