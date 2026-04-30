package alerts

import (
	"fmt"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/pocketbase/pocketbase/core"
)

// HandleProcessAlerts evaluates process-related alerts (Process, ProcessCpu, ProcessMem).
func (am *AlertManager) HandleProcessAlerts(systemRecord *core.Record, data *system.CombinedData) error {
	if len(data.Processes) == 0 {
		return nil
	}

	alerts := am.alertsCache.GetAlertsByNames(systemRecord.Id, "Process", "ProcessCpu", "ProcessMem")
	if len(alerts) == 0 {
		return nil
	}

	// Build a map of process name -> ProcessInfo for quick lookup
	processMap := make(map[string]*system.ProcessInfo, len(data.Processes))
	for _, p := range data.Processes {
		processMap[p.Name] = p
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
				am.handleProcessThresholdAlert(systemRecord, alertData, processName, "Memory", proc.Mem)
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
	alert := &alertInfo{
		systemName: systemName,
		alertData:  alertData,
		expireTime: time.Now().Add(delay),
	}

	storedAlert, loaded := am.pendingAlerts.LoadOrStore(alertData.Id, alert)
	if loaded {
		return false
	}

	stored := storedAlert.(*alertInfo)
	stored.timer = time.AfterFunc(time.Until(stored.expireTime), func() {
		am.processPendingAlert(alertData.Id)
	})
	return true
}

// handleProcessThresholdAlert handles CPU/memory threshold alerts for a process.
func (am *AlertManager) handleProcessThresholdAlert(systemRecord *core.Record, alertData CachedAlertData, processName, metricName string, value float64) {
	threshold := alertData.Value
	triggered := alertData.Triggered

	// Skip if no state change
	if (!triggered && value <= threshold) || (triggered && value > threshold) {
		return
	}

	systemName := systemRecord.GetString("name")
	newTriggered := value > threshold

	if err := am.setAlertTriggered(alertData, newTriggered); err != nil {
		return
	}

	var subject string
	unit := "%"
	if newTriggered {
		subject = fmt.Sprintf("%s process %s %s above threshold", systemName, processName, metricName)
	} else {
		subject = fmt.Sprintf("%s process %s %s below threshold", systemName, processName, metricName)
	}
	body := fmt.Sprintf("Process %s %s averaged %.2f%s (threshold: %.0f%s).", processName, metricName, value, unit, threshold, unit)

	am.SendAlert(AlertMessageData{
		UserID:   alertData.UserID,
		SystemID: alertData.SystemID,
		Title:    subject,
		Message:  body,
		Link:     am.hub.MakeLink("system", alertData.SystemID),
		LinkText: "View " + systemName,
	})
}

// sendProcessAlert sends a process up/down alert notification.
func (am *AlertManager) sendProcessAlert(status, systemName string, alertData CachedAlertData, processName string) error {
	triggered := status == "down"
	if err := am.setAlertTriggered(alertData, triggered); err != nil {
		return err
	}

	var emoji string
	if status == "up" {
		emoji = "✅"
	} else {
		emoji = "\U0001F534"
	}

	title := fmt.Sprintf("Process %s is %s on %s %v", processName, status, systemName, emoji)
	message := fmt.Sprintf("Process %s is %s on %s", processName, status, systemName)

	return am.SendAlert(AlertMessageData{
		UserID:   alertData.UserID,
		SystemID: alertData.SystemID,
		Title:    title,
		Message:  message,
		Link:     am.hub.MakeLink("system", alertData.SystemID),
		LinkText: "View " + systemName,
	})
}
