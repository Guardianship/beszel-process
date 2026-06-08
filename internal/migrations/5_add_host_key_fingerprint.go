package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		systemsCollection, err := app.FindCachedCollectionByNameOrId("systems")
		if err != nil {
			return err
		}

		// Check if "host_key_fingerprint" field already exists
		hasField := false
		for _, field := range systemsCollection.Fields {
			if field.GetName() == "host_key_fingerprint" {
				hasField = true
				break
			}
		}

		// Add "host_key_fingerprint" text field if missing
		if !hasField {
			fingerprintField := &core.TextField{
				Name:     "host_key_fingerprint",
				Required: false,
			}
			systemsCollection.Fields.Add(fingerprintField)
			return app.Save(systemsCollection)
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove the field
		systemsCollection, err := app.FindCachedCollectionByNameOrId("systems")
		if err != nil {
			return err
		}
		systemsCollection.Fields.RemoveByName("host_key_fingerprint")
		return app.Save(systemsCollection)
	})
}
