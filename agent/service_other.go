//go:build !windows

package agent

import (
	"errors"

	"github.com/henrygd/beszel/internal/entities/systemd"
)

// serviceManager is a stub for non-Windows systems.
// On Linux, the existing systemdManager handles service monitoring.
type serviceManager struct {
	hasFreshStats bool
}

func newServiceManager() (*serviceManager, error) {
	return &serviceManager{}, nil
}

func (sm *serviceManager) getServiceStats() []*systemd.Service {
	return nil
}

func (sm *serviceManager) getServiceStatsCount() int {
	return 0
}

func (sm *serviceManager) getFailedServiceCount() uint16 {
	return 0
}

func (sm *serviceManager) getServiceDetails(string) (systemd.ServiceDetails, error) {
	return nil, errors.New("service manager unavailable")
}
