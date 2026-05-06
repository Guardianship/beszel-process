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

		// Check if "item" field already exists
		hasItem := false
		hasProcessPortNames := false
		for _, field := range alertsCollection.Fields {
			if field.GetName() == "item" {
				hasItem = true
			}
			if field.GetName() == "name" {
				if selectField, ok := field.(*core.SelectField); ok {
					for _, v := range selectField.Values {
						if v == "Process" || v == "Port" {
							hasProcessPortNames = true
							break
						}
					}
				}
			}
		}

		// Add "item" text field if missing
		if !hasItem {
			itemField := &core.TextField{
				Name:     "item",
				Required: false,
			}
			alertsCollection.Fields.Add(itemField)
		}

		// Add new alert type names to "name" select field if missing
		if !hasProcessPortNames {
			for _, field := range alertsCollection.Fields {
				if field.GetName() == "name" {
					if selectField, ok := field.(*core.SelectField); ok {
						selectField.Values = append(selectField.Values, "Process", "ProcessCpu", "ProcessMem", "Port")
					}
					break
				}
			}
		}

		// Always update the index to include item, even if fields were already present
		alertsCollection.Indexes = []string{
			"CREATE UNIQUE INDEX `idx_alerts_user_system_name_item` ON `alerts` (`user`, `system`, `name`, `item`)",
		}
		return app.Save(alertsCollection)
	}, func(app core.App) error {
		return nil
	})
}
