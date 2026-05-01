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

func TestProcessAlertDownUp(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	// Create a Process alert with item = "nginx"
	alertRecord, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Process",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "nginx",
		"min":    1,
	})
	require.NoError(t, err)
	assert.False(t, alertRecord.GetBool("triggered"), "Alert should not be triggered initially")

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	// Process is running — should not trigger
	dataRunning := &system.CombinedData{
		Processes: []*system.ProcessInfo{
			{Name: "nginx", Pid: 1234, Status: "running"},
		},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataRunning))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "No pending alert when process is running")

	// Process is down — should schedule pending alert
	dataDown := &system.CombinedData{
		Processes: []*system.ProcessInfo{},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataDown))
	assert.Equal(t, 1, am.GetPendingAlertsCount(), "Pending alert should be scheduled when process is down")

	// Expire and process the pending alert
	am.ForceExpirePendingAlerts()
	processed, err := am.ProcessPendingAlerts()
	require.NoError(t, err)
	assert.Len(t, processed, 1, "One alert should be processed")

	// Verify alert is triggered
	alertFresh, err := hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.True(t, alertFresh.GetBool("triggered"), "Alert should be triggered after process down")

	// Process comes back up — should resolve
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataRunning))
	alertFresh, err = hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Alert should be resolved after process is up")
}

func TestProcessAlertRecoveryBeforeDeadline(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	alertRecord, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Process",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "nginx",
		"min":    1,
	})
	require.NoError(t, err)

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))
	dataRunning := &system.CombinedData{
		Processes: []*system.ProcessInfo{
			{Name: "nginx", Pid: 1234, Status: "running"},
		},
	}
	dataDown := &system.CombinedData{}

	// Process goes down
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataDown))
	assert.Equal(t, 1, am.GetPendingAlertsCount(), "Pending alert should be scheduled")

	// Process recovers before alert fires
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataRunning))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "Pending alert should be canceled on recovery")

	// Alert should not be triggered
	alertFresh, err := hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Alert should not be triggered if process recovers before deadline")
}

func TestProcessCpuThresholdAlert(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	// Create a ProcessCpu alert with threshold 80%, item = "nginx"
	alertRecord, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "ProcessCpu",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "nginx",
		"value":  80,
		"min":    1,
	})
	require.NoError(t, err)
	assert.False(t, alertRecord.GetBool("triggered"), "Alert should not be triggered initially")

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	// CPU below threshold — should not trigger
	dataBelow := &system.CombinedData{
		Processes: []*system.ProcessInfo{
			{Name: "nginx", Cpu: 50},
		},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataBelow))
	alertFresh, err := hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Alert should not trigger below threshold")

	// CPU above threshold — should trigger
	dataAbove := &system.CombinedData{
		Processes: []*system.ProcessInfo{
			{Name: "nginx", Cpu: 90},
		},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataAbove))
	alertFresh, err = hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.True(t, alertFresh.GetBool("triggered"), "Alert should trigger above threshold")

	// CPU drops below threshold — should resolve
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataBelow))
	alertFresh, err = hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Alert should resolve when CPU drops below threshold")
}

func TestProcessMemThresholdAlert(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	alertRecord, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "ProcessMem",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "postgres",
		"value":  70,
		"min":    1,
	})
	require.NoError(t, err)

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	// Memory below threshold
	dataBelow := &system.CombinedData{
		Processes: []*system.ProcessInfo{
			{Name: "postgres", Mem: 40},
		},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataBelow))
	alertFresh, err := hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.False(t, alertFresh.GetBool("triggered"), "Memory alert should not trigger below threshold")

	// Memory above threshold
	dataAbove := &system.CombinedData{
		Processes: []*system.ProcessInfo{
			{Name: "postgres", Mem: 80},
		},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataAbove))
	alertFresh, err = hub.FindRecordById("alerts", alertRecord.Id)
	require.NoError(t, err)
	assert.True(t, alertFresh.GetBool("triggered"), "Memory alert should trigger above threshold")
}

func TestProcessAlertNoAlertRecord(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))

	// No alert record exists — HandleProcessAlerts should return nil
	dataDown := &system.CombinedData{
		Processes: []*system.ProcessInfo{},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataDown))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "No pending alert when no alert record exists")
}

func TestProcessAlertEmptyItem(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "up")
	require.NoError(t, err)
	systemRecord := systems[0]

	// Create alert with empty item — should be skipped
	_, err = beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "Process",
		"system": systemRecord.Id,
		"user":   user.Id,
		"item":   "",
		"min":    1,
	})
	require.NoError(t, err)

	am := alerts.NewTestAlertManagerWithoutWorker(hub)
	require.NoError(t, am.GetSystemAlertsCache().PopulateFromDB(true))
	dataDown := &system.CombinedData{
		Processes: []*system.ProcessInfo{},
	}
	require.NoError(t, am.HandleProcessAlerts(systemRecord, dataDown))
	assert.Equal(t, 0, am.GetPendingAlertsCount(), "No pending alert when item is empty")
}
