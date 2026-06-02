//go:build windows

package agent

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/henrygd/beszel/internal/entities/system"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// processFriendlyNames maps executable names to human-readable display names.
var processFriendlyNames = map[string]string{
	"Code.exe":              "Visual Studio Code",
	"Weixin.exe":            "微信",
	"WeChatAppEx.exe":       "微信小程序",
	"claude.exe":            "Claude",
	"msedge.exe":            "Microsoft Edge",
	"chrome.exe":            "Google Chrome",
	"firefox.exe":           "Mozilla Firefox",
	"explorer.exe":          "Windows Explorer",
	"SearchUI.exe":          "Search",
	"ShellExperienceHost.exe": "Shell Experience",
	"Taskmgr.exe":           "Task Manager",
	"Lingma.exe":            "通义灵码",
	"wps.exe":               "WPS Office",
	"WINWORD.EXE":           "Microsoft Word",
	"EXCEL.EXE":             "Microsoft Excel",
	"POWERPNT.EXE":          "Microsoft PowerPoint",
	"OUTLOOK.EXE":           "Microsoft Outlook",
	"Teams.exe":             "Microsoft Teams",
	"Slack.exe":             "Slack",
	"Discord.exe":           "Discord",
	"Spotify.exe":           "Spotify",
	"notepad.exe":           "Notepad",
	"cmd.exe":               "Command Prompt",
	"powershell.exe":        "PowerShell",
	"WindowsTerminal.exe":   "Windows Terminal",
	"docker.exe":            "Docker",
	"node.exe":              "Node.js",
	"python.exe":            "Python",
	"java.exe":              "Java",
}

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

	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil || len(records) == 0 {
		return nil
	}

	for _, fields := range records {
		if len(fields) < 5 {
			continue
		}

		pid, err := strconv.ParseUint(fields[1], 10, 32)
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

// gbkToUTF8 converts GBK encoded bytes to UTF-8 string.
func gbkToUTF8(b []byte) string {
	reader := transform.NewReader(bytes.NewReader(b), simplifiedchinese.GBK.NewDecoder())
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return string(b)
	}
	return string(utf8Bytes)
}

// collectAllProcesses returns all running processes on Windows using tasklist.
func collectAllProcesses() []*system.ProcessInfo {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	// Convert from system encoding (GBK) to UTF-8
	utf8Output := gbkToUTF8(output)
	reader := csv.NewReader(strings.NewReader(utf8Output))
	records, err := reader.ReadAll()
	if err != nil || len(records) == 0 {
		return nil
	}

	var result []*system.ProcessInfo
	for _, fields := range records {
		if len(fields) < 5 {
			continue
		}

		name := fields[0]
		// Use friendly name if available
		if friendly, ok := processFriendlyNames[name]; ok {
			name = friendly
		}

		pid, err := strconv.ParseUint(fields[1], 10, 32)
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

// parseTasklistMem parses the memory column from tasklist CSV output (e.g. "134,752 K") to MB.
func parseTasklistMem(memField string) float64 {
	memStr := strings.TrimSpace(memField)
	memStr = strings.TrimSuffix(memStr, " K")
	memStr = strings.TrimSuffix(memStr, " k")
	memStr = strings.ReplaceAll(memStr, ",", "")
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
