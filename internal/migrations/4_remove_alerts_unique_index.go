package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		alertsCollection, err := app.FindCachedCollectionByNameOrId("alerts")
		if err != nil {
			return err
		}

		// Replace the unique index with a non-unique index to allow
		// multiple alerts with the same (user, system, name, item).
		alertsCollection.Indexes = []string{
			"CREATE INDEX `idx_alerts_user_system_name_item` ON `alerts` (`user`, `system`, `name`, `item`)",
		}
		return app.Save(alertsCollection)
	}, func(app core.App) error {
		return nil
	})
}
