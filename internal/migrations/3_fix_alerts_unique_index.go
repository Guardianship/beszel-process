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

		// Set any NULL item values to empty string so the unique index works correctly.
		// NULL values in SQL are not considered equal, so multiple alerts with NULL item
		// would bypass the unique constraint on (user, system, name, item).
		if _, err := app.DB().NewQuery("UPDATE alerts SET item = '' WHERE item IS NULL").Execute(); err != nil {
			return err
		}

		// Unconditionally set the correct unique index that includes item.
		// This fixes installations where migration 2 skipped the index update
		// because the fields already existed (changed=false).
		alertsCollection.Indexes = []string{
			"CREATE UNIQUE INDEX `idx_alerts_user_system_name_item` ON `alerts` (`user`, `system`, `name`, `item`)",
		}
		return app.Save(alertsCollection)
	}, func(app core.App) error {
		return nil
	})
}
