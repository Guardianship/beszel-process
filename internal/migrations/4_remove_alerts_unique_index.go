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

		// Drop the old unique index if it exists, then create a non-unique index.
		// This ensures the unique constraint is removed even if a previous migration
		// created it and app.Save alone didn't replace it properly.
		if _, err := app.DB().NewQuery("DROP INDEX IF EXISTS `idx_alerts_user_system_name_item`").Execute(); err != nil {
			return err
		}

		alertsCollection.Indexes = []string{
			"CREATE INDEX `idx_alerts_user_system_name_item` ON `alerts` (`user`, `system`, `name`, `item`)",
		}
		return app.Save(alertsCollection)
	}, func(app core.App) error {
		return nil
	})
}
