package alerts

import (
	"fmt"
	"strconv"
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

	// Build maps for flexible port lookup by service name, port number, or port/protocol
	portByService := make(map[string]*system.PortInfo, len(data.Ports))
	portByNumber := make(map[uint16]*system.PortInfo, len(data.Ports))
	portByNumberProto := make(map[string]*system.PortInfo, len(data.Ports))
	for _, p := range data.Ports {
		portByService[p.Service] = p
		portByNumber[p.Port] = p
		portByNumberProto[fmt.Sprintf("%d/%s", p.Port, p.Protocol)] = p
	}

	for _, alertData := range alerts {
		portItem := alertData.Item
		if portItem == "" {
			continue
		}

		// Try matching by service name, then port/protocol (e.g. "80/tcp"), then port number
		var port *system.PortInfo
		var exists bool
		if port, exists = portByService[portItem]; !exists {
			if port, exists = portByNumberProto[portItem]; !exists {
				// Try parsing as port number
				if portNum, err := strconv.ParseUint(portItem, 10, 16); err == nil {
					port, exists = portByNumber[uint16(portNum)]
				}
			}
		}
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
	// Check if this alert is already triggered to prevent re-scheduling after the
	// pending entry was consumed by processPendingAlert.
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

// sendPortAlert sends a port open/closed alert notification.
func (am *AlertManager) sendPortAlert(status, systemName string, alertData CachedAlertData, portItem string) error {
	triggered := status == "closed"
	if err := am.setAlertTriggered(alertData, triggered); err != nil {
		return err
	}

	var emoji string
	var statusCn string
	if status == "open" {
		emoji = "✅"
		statusCn = "开放"
	} else {
		emoji = "\U0001F534"
		statusCn = "关闭"
	}

	title := fmt.Sprintf("端口 %s 状态: %s（系统: %s）%v", portItem, statusCn, systemName, emoji)
	message := fmt.Sprintf("端口 %s 状态: %s（系统: %s）", portItem, statusCn, systemName)

	return am.SendAlert(AlertMessageData{
		UserID:   alertData.UserID,
		SystemID: alertData.SystemID,
		Title:    title,
		Message:  message,
		Link:     am.hub.MakeLink("system", alertData.SystemID),
		LinkText: "View " + systemName,
	})
}
