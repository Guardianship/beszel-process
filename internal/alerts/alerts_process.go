package alerts

import (
	"fmt"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/pocketbase/pocketbase/core"
)

// HandleProcessAlerts evaluates process-related alerts (Process, ProcessCpu, ProcessMem).
func (am *AlertManager) HandleProcessAlerts(systemRecord *core.Record, data *system.CombinedData) error {
	alerts := am.alertsCache.GetAlertsByNames(systemRecord.Id, "Process", "ProcessCpu", "ProcessMem")
	if len(alerts) == 0 {
		return nil
	}

	// Build a map of process name -> ProcessInfo for quick lookup
	processMap := make(map[string]*system.ProcessInfo, len(data.Processes))
	for _, p := range data.Processes {
		processMap[p.Name] = p
	}

	// Get system total memory for percentage calculation
	var totalMemoryBytes uint64
	systemDetails, _ := am.hub.FindFirstRecordByFilter("system_details", "system={:system}", map[string]any{"system": systemRecord.Id})
	if systemDetails != nil {
		totalMemoryBytes = uint64(systemDetails.GetInt("memory"))
	}

	for _, alertData := range alerts {
		processName := alertData.Item
		if processName == "" {
			continue
		}

		proc, exists := processMap[processName]

		switch alertData.Name {
		case "Process":
			am.handleProcessStatusAlert(systemRecord, alertData, processName, exists, proc)
		case "ProcessCpu":
			if exists && proc != nil {
				am.handleProcessThresholdAlert(systemRecord, alertData, processName, "CPU", proc.Cpu)
			}
		case "ProcessMem":
			if exists && proc != nil {
				// Convert memory from MB to percentage of total system memory
				memPercent := proc.Mem
				if totalMemoryBytes > 0 {
					memBytes := proc.Mem * 1024 * 1024 // MB to bytes
					memPercent = (memBytes / float64(totalMemoryBytes)) * 100
				}
				am.handleProcessThresholdAlert(systemRecord, alertData, processName, "Memory", memPercent)
			}
		}
	}

	return nil
}

// handleProcessStatusAlert handles process down/up state-change alerts.
// It follows the same pending-timer pattern as system status alerts.
func (am *AlertManager) handleProcessStatusAlert(systemRecord *core.Record, alertData CachedAlertData, processName string, running bool, proc *system.ProcessInfo) {
	systemName := systemRecord.GetString("name")

	if !running {
		// Process is down — schedule a pending alert with min delay
		min := max(1, int(alertData.Min))
		am.schedulePendingProcessAlert(systemName, alertData, processName, time.Duration(min)*time.Minute)
	} else {
		// Process is up — cancel pending alert and resolve if previously triggered
		if am.cancelPendingAlert(alertData.Id) {
			return
		}
		if !alertData.Triggered {
			return
		}
		if err := am.sendProcessAlert("up", systemName, alertData, processName); err != nil {
			am.hub.Logger().Error("Failed to send process alert", "err", err)
		}
	}
}

// schedulePendingProcessAlert sets up a timer to send a "down" alert after the specified delay.
func (am *AlertManager) schedulePendingProcessAlert(systemName string, alertData CachedAlertData, processName string, delay time.Duration) bool {
	// Check if this alert is already triggered to prevent re-scheduling after the
	// pending entry was consumed by processPendingAlert. This breaks the cycle:
	// processPendingAlert deletes entry → next cycle creates new entry → timer fires →
	// Triggered=true → return early → delete → repeat.
	if refreshed, ok := am.alertsCache.Refresh(alertData); ok && refreshed.Triggered {
		return false
	}

	alert := &alertInfo{
		systemName: systemName,
		alertData:  alertData,
		expireTime: time.Now().Add(delay),
	}

	storedAlert, loaded := am.pendingAlerts.LoadOrStore(alertData.Id, alert)
	if loaded {
		// Update existing entry's alertData so it uses fresh data when the timer fires.
		stored := storedAlert.(*alertInfo)
		stored.alertData = alertData
		return false
	}

	stored := storedAlert.(*alertInfo)
	stored.timer = time.AfterFunc(time.Until(stored.expireTime), func() {
		am.processPendingAlert(alertData.Id)
	})
	return true
}

// handleProcessThresholdAlert handles CPU/memory threshold alerts for a process.
// Uses a pending timer: when the value exceeds the threshold, a timer is started
// for min minutes. If the value drops below the threshold before the timer fires,
// the timer is cancelled. The alert is only sent when the timer fires.
func (am *AlertManager) handleProcessThresholdAlert(systemRecord *core.Record, alertData CachedAlertData, processName, metricName string, value float64) {
	threshold := alertData.Value
	triggered := alertData.Triggered
	aboveThreshold := value > threshold

	// Skip if no state change
	if (!triggered && !aboveThreshold) || (triggered && aboveThreshold) {
		return
	}

	systemName := systemRecord.GetString("name")

	if aboveThreshold {
		// Value just crossed above threshold — schedule pending alert
		min := max(1, int(alertData.Min))
		am.schedulePendingThresholdAlert(systemName, alertData, processName, metricName, value, threshold, time.Duration(min)*time.Minute)
	} else {
		// Value dropped below threshold — cancel pending alert if any
		if am.cancelPendingAlert(alertData.Id) {
			return
		}
		// If previously triggered, send recovery alert immediately
		if !triggered {
			return
		}
		if err := am.setAlertTriggered(alertData, false); err != nil {
			return
		}
		unit := "%"
		subject := fmt.Sprintf("%s 进程 %s %s 低于阈值", systemName, processName, metricName)
		body := fmt.Sprintf("进程 %s %s 当前值 %.2f%s（阈值: %.0f%s）。", processName, metricName, value, unit, threshold, unit)
		am.SendAlert(AlertMessageData{
			UserID:   alertData.UserID,
			SystemID: alertData.SystemID,
			Title:    subject,
			Message:  body,
			Link:     am.hub.MakeLink("system", alertData.SystemID),
			LinkText: "View " + systemName,
		})
	}
}

// schedulePendingThresholdAlert sets up a timer to send a threshold alert after the specified delay.
func (am *AlertManager) schedulePendingThresholdAlert(systemName string, alertData CachedAlertData, processName, metricName string, value, threshold float64, delay time.Duration) {
	// Check if already triggered to prevent re-scheduling
	if refreshed, ok := am.alertsCache.Refresh(alertData); ok && refreshed.Triggered {
		return
	}

	alert := &alertInfo{
		systemName: systemName,
		alertData:  alertData,
		expireTime: time.Now().Add(delay),
	}

	storedAlert, loaded := am.pendingAlerts.LoadOrStore(alertData.Id, alert)
	if loaded {
		stored := storedAlert.(*alertInfo)
		stored.alertData = alertData
		return
	}

	stored := storedAlert.(*alertInfo)
	stored.timer = time.AfterFunc(time.Until(stored.expireTime), func() {
		am.processPendingThresholdAlert(alertData.Id, processName, metricName, value, threshold)
	})
}

// processPendingThresholdAlert sends the threshold alert after the timer fires.
func (am *AlertManager) processPendingThresholdAlert(alertId string, processName, metricName string, value, threshold float64) {
	raw, ok := am.pendingAlerts.LoadAndDelete(alertId)
	if !ok {
		return
	}
	info := raw.(*alertInfo)
	alertData := info.alertData

	// Re-check: only send if still triggered
	if refreshed, ok := am.alertsCache.Refresh(alertData); ok {
		alertData = refreshed
	}
	if alertData.Triggered {
		return
	}

	if err := am.setAlertTriggered(alertData, true); err != nil {
		return
	}

	unit := "%"
	subject := fmt.Sprintf("%s 进程 %s %s 超过阈值", info.systemName, processName, metricName)
	body := fmt.Sprintf("进程 %s %s 平均值 %.2f%s（阈值: %.0f%s）。", processName, metricName, value, unit, threshold, unit)

	am.SendAlert(AlertMessageData{
		UserID:   alertData.UserID,
		SystemID: alertData.SystemID,
		Title:    subject,
		Message:  body,
		Link:     am.hub.MakeLink("system", alertData.SystemID),
		LinkText: "View " + info.systemName,
	})
}

// sendProcessAlert sends a process up/down alert notification.
func (am *AlertManager) sendProcessAlert(status, systemName string, alertData CachedAlertData, processName string) error {
	triggered := status == "down"
	if err := am.setAlertTriggered(alertData, triggered); err != nil {
		return err
	}

	var emoji string
	var statusCn string
	if status == "up" {
		emoji = "✅"
		statusCn = "在线"
	} else {
		emoji = "\U0001F534"
		statusCn = "离线"
	}

	title := fmt.Sprintf("进程 %s 状态: %s（系统: %s）%v", processName, statusCn, systemName, emoji)
	message := fmt.Sprintf("进程 %s 状态: %s（系统: %s）", processName, statusCn, systemName)

	return am.SendAlert(AlertMessageData{
		UserID:   alertData.UserID,
		SystemID: alertData.SystemID,
		Title:    title,
		Message:  message,
		Link:     am.hub.MakeLink("system", alertData.SystemID),
		LinkText: "View " + systemName,
	})
}
