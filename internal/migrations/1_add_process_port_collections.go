package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `[
		{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{10}",
					"hidden": false,
					"id": "text3208210256",
					"max": 10,
					"min": 6,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": true,
					"collectionId": "2hz5ncl8tizk5nx",
					"hidden": false,
					"id": "relation3377271179",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "system",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1579384326",
					"max": 0,
					"min": 0,
					"name": "name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "number3128971310",
					"max": null,
					"min": null,
					"name": "pid",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number2063623452",
					"max": 100,
					"min": 0,
					"name": "cpu",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number3933025333",
					"max": null,
					"min": 0,
					"name": "memory",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text2063623452",
					"max": 0,
					"min": 0,
					"name": "status",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "number3332085495",
					"max": null,
					"min": null,
					"name": "uptime",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_4112345678",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_proc_system` + "`" + ` ON ` + "`" + `processes` + "`" + ` (` + "`" + `system` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_proc_updated` + "`" + ` ON ` + "`" + `processes` + "`" + ` (` + "`" + `updated` + "`" + `)"
			],
			"listRule": null,
			"name": "processes",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		},
		{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{10}",
					"hidden": false,
					"id": "text3208210256",
					"max": 10,
					"min": 6,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": true,
					"collectionId": "2hz5ncl8tizk5nx",
					"hidden": false,
					"id": "relation3377271179",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "system",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "number2063623452",
					"max": 65535,
					"min": 1,
					"name": "port",
					"onlyInt": true,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text2744374011",
					"max": 0,
					"min": 0,
					"name": "protocol",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text2063623452",
					"max": 0,
					"min": 0,
					"name": "status",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1579384326",
					"max": 0,
					"min": 0,
					"name": "service",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_4223456789",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_port_system` + "`" + ` ON ` + "`" + `monitored_ports` + "`" + ` (` + "`" + `system` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_port_updated` + "`" + ` ON ` + "`" + `monitored_ports` + "`" + ` (` + "`" + `updated` + "`" + `)"
			],
			"listRule": null,
			"name": "monitored_ports",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}
	]`

		err := app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
		if err != nil {
			return err
		}

		// Update alerts collection to add new alert type names and item field
		alertsCollection, err := app.FindCachedCollectionByNameOrId("alerts")
		if err != nil {
			return err
		}

		// Find the "name" select field and add new values
		for _, field := range alertsCollection.Fields {
			if field.GetName() == "name" {
				if selectField, ok := field.(*core.SelectField); ok {
					selectField.Values = append(selectField.Values, "Process", "ProcessCpu", "ProcessMem", "Port")
				}
				break
			}
		}

		// Add "item" text field to store process name or port identifier
		itemField := &core.TextField{
			Name:     "item",
			Required: false,
		}
		alertsCollection.Fields.Add(itemField)

		// Update unique index to include item (user, system, name, item)
		alertsCollection.Indexes = []string{
			"CREATE UNIQUE INDEX `idx_alerts_user_system_name_item` ON `alerts` (`user`, `system`, `name`, `item`)",
		}

		return app.Save(alertsCollection)
	}, func(app core.App) error {
		return nil
	})
}
