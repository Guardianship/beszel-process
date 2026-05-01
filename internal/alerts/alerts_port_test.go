//go:build testing

package alerts_test

import (
	"testing"

	"github.com/henrygd/beszel/internal/alerts"
	"github.com/henrygd/beszel/internal/entities/system"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortAlertClosedOpen(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	// Create a Port alert with item = "80/tcp"
	alertRecord, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Port",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "80/tcp",
		"min":    1,
	})
	require.NoError(t, err)
	assert.False(t, alertRecord.GetBool("triggered"), "Alert should not be triggered initially")

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	// Port is open — should not trigger
	dataOpen := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 80, Protocol: "tcp", Status: "open", Service: "80/tcp"},
		},
	}
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataOpen))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "No pending alert when port is open")

	// Port is closed — should schedule pending alert
	dataClosed := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 80, Protocol: "tcp", Status: "closed", Service: "80/tcp"},
		},
	}
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataClosed))
	assert.Equal(t, 1, am.GetPendingAlertsCount(), "Pending alert should be scheduled when port is closed")

	// Expire and process the pending alert
	am.ForceExpirePendingAlerts()
	processed, err := am.ProcessPendingAlerts()
	require.NoError(t, err)
	assert.Len(t, processed, 1, "One alert should be processed")

	// Verify alert is triggered
	alertFresh, err := hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.True(t, alertFresh.GetBool("triggered"), "Alert should be triggered after port closed")

	// Port reopens — should resolve
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataOpen))
	alertFresh, err = hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Alert should be resolved after port reopens")
}

func TestPortAlertRecoveryBeforeDeadline(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	alertRecord, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Port",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "443/tcp",
		"min":    1,
	})
	require.NoError(t, err)

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))
	dataOpen := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 443, Protocol: "tcp", Status: "open", Service: "443/tcp"},
		},
	}
	dataClosed := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 443, Protocol: "tcp", Status: "closed", Service: "443/tcp"},
		},
	}

	// Port closes
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataClosed))
	assert.Equal(t, 1, am.GetPendingAlertsCount(), "Pending alert should be scheduled")

	// Port reopens before alert fires
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataOpen))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "Pending alert should be canceled on recovery")

	alertFresh, err := hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Alert should not be triggered if port reopens before deadline")
}

func TestPortAlertNoAlertRecord(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	dataClosed := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 80, Protocol: "tcp", Status: "closed", Service: "80/tcp"},
		},
	}
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataClosed))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "No pending alert when no alert record exists")
}

func TestPortAlertEmptyItem(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	_, err = beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Port",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "",
		"min":    1,
	})
	require.NoError(t, err)

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))
	dataClosed := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 80, Protocol: "tcp", Status: "closed", Service: "80/tcp"},
		},
	}
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataClosed))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "No pending alert when item is empty")
}

func TestPortAlertPortNotInData(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	_, err = beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Port",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "9999/tcp",
		"min":    1,
	})
	require.NoError(t, err)

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	// Data has a different port — the monitored port doesn't exist in data
	dataDifferentPort := &system.CombinedData{
		Ports: []*system.PortInfo{
			{Port: 80, Protocol: "tcp", Status: "open", Service: "80/tcp"},
		},
	}
	require.NoError(t, am.HandlePortAlerts(systemRecord, dataDifferentPort))
	assert.Equal(t, 1, am.GetPendingAlertsCount(), "Pending alert should be scheduled when port is not found in data (treated as closed)")
}
