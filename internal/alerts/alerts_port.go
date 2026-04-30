package alerts

import (
	"fmt"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/pocketbase/pocketbase/core"
)

// HandlePortAlerts evaluates port-related alerts (Port open/closed).
func (am *AlertManager) HandlePortAlerts(systemRecord *core.Record, data *system.CombinedData) error {
	if len(data.Ports) == 0 {
		return nil
	}

	alerts := am.alertsCache.GetAlertsByName(systemRecord.Id, "Port")
	if len(alerts) == 0 {
		return nil
	}

	// Build a map of port identifier -> PortInfo for quick lookup
	portMap := make(map[string]*system.PortInfo, len(data.Ports))
	for _, p := range data.Ports {
		portMap[p.Service] = p
	}

	for _, alertData := range alerts {
		portItem := alertData.Item
		if portItem == "" {
			continue
		}

		port, exists := portMap[portItem]
		isOpen := exists && port.Status == "open"

		am.handlePortStatusAlert(systemRecord, alertData, portItem, isOpen)
	}

	return nil
}

// handlePortStatusAlert handles port open/closed state-change alerts.
// It follows the same pending-timer pattern as system status alerts.
func (am *AlertManager) handlePortStatusAlert(systemRecord *core.Record, alertData CachedAlertData, portItem string, isOpen bool) {
	systemName := systemRecord.GetString("name")

	if !isOpen {
		// Port is closed — schedule a pending alert with min delay
		min := max(1, int(alertData.Min))
		am.schedulePendingPortAlert(systemName, alertData, portItem, time.Duration(min)*time.Minute)
	} else {
		// Port is open — cancel pending alert and resolve if previously triggered
		if am.cancelPendingAlert(alertData.Id) {
			return
		}
		if !alertData.Triggered {
			return
		}
		if err := am.sendPortAlert("open", systemName, alertData, portItem); err != nil {
			am.hub.Logger().Error("Failed to send port alert", "err", err)
		}
	}
}

// schedulePendingPortAlert sets up a timer to send a "closed" alert after the specified delay.
func (am *AlertManager) schedulePendingPortAlert(systemName string, alertData CachedAlertData, portItem string, delay time.Duration) bool {
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

// sendPortAlert sends a port open/closed alert notification.
func (am *AlertManager) sendPortAlert(status, systemName string, alertData CachedAlertData, portItem string) error {
	triggered := status == "closed"
	if err := am.setAlertTriggered(alertData, triggered); err != nil {
		return err
	}

	var emoji string
	if status == "open" {
		emoji = "✅"
	} else {
		emoji = "\U0001F534"
	}

	title := fmt.Sprintf("Port %s is %s on %s %v", portItem, status, systemName, emoji)
	message := fmt.Sprintf("Port %s is %s on %s", portItem, status, systemName)

	return am.SendAlert(AlertMessageData{
		UserID:   alertData.UserID,
		SystemID: alertData.SystemID,
		Title:    title,
		Message:  message,
		Link:     am.hub.MakeLink("system", alertData.SystemID),
		LinkText: "View " + systemName,
	})
}
