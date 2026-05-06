//go:build testing

package alerts_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/henrygd/beszel/internal/alerts"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	pbTests "github.com/pocketbase/pocketbase/tests"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/stretchr/testify/assert"
)

// marshal to json and return an io.Reader (for use in ApiScenario.Body)
func jsonReader(v any) io.Reader {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(data)
}

func TestIsInternalURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		internal bool
	}{
		{name: "loopback ipv4", url: "generic://127.0.0.1", internal: true},
		{name: "localhost hostname", url: "generic://localhost", internal: true},
		{name: "localhost hostname", url: "generic+http://localhost/api/v1/postStuff", internal: true},
		{name: "localhost hostname", url: "generic+http://127.0.0.1:8080/api/v1/postStuff", internal: true},
		{name: "localhost hostname", url: "generic+https://beszel.dev/api/v1/postStuff", internal: false},
		{name: "public ipv4", url: "generic://8.8.8.8", internal: false},
		{name: "token style service url", url: "discord://abc123@123456789", internal: false},
		{name: "single label service url", url: "slack://token@team/channel", internal: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			internal, err := alerts.IsInternalURL(testCase.url)
			assert.NoError(t, err)
			assert.Equal(t, testCase.internal, internal)
		})
	}
}

func TestUserAlertsApi(t *testing.T) {
	hub, _ := beszelTests.NewTestHub(t.TempDir())
	defer hub.Cleanup()

	hub.StartHub()

	user1, _ := beszelTests.CreateUser(hub, "alertstest@example.com", "password")
	user1Token, _ := user1.NewAuthToken()

	user2, _ := beszelTests.CreateUser(hub, "alertstest2@example.com", "password")
	user2Token, _ := user2.NewAuthToken()

	system1, _ := beszelTests.CreateRecord(hub, "systems", map[string]any{
		"name":  "system1",
		"users": []string{user1.Id},
		"host":  "127.0.0.1",
	})

	system2, _ := beszelTests.CreateRecord(hub, "systems", map[string]any{
		"name":  "system2",
		"users": []string{user1.Id, user2.Id},
		"host":  "127.0.0.2",
	})

	userRecords, _ := hub.CountRecords("users")
	assert.EqualValues(t, 2, userRecords, "all users should be created")

	systemRecords, _ := hub.CountRecords("systems")
	assert.EqualValues(t, 2, systemRecords, "all systems should be created")

	testAppFactory := func(t testing.TB) *pbTests.TestApp {
		return hub.TestApp
	}

	scenarios := []beszelTests.ApiScenario{
		{
			Name:            "POST no auth",
			Method:          http.MethodPost,
			URL:             "/api/beszel/user-alerts",
			ExpectedStatus:  401,
			ExpectedContent: []string{"requires valid"},
			TestAppFactory:  testAppFactory,
		},
		{
			Name:   "POST no body",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  400,
			ExpectedContent: []string{"Bad data"},
			TestAppFactory:  testAppFactory,
		},
		{
			Name:   "POST bad data",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  400,
			ExpectedContent: []string{"Bad data"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"invalidField": "this should cause validation error",
				"threshold":    "not a number",
			}),
		},
		{
			Name:   "POST malformed JSON",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  400,
			ExpectedContent: []string{"Bad data"},
			TestAppFactory:  testAppFactory,
			Body:            strings.NewReader(`{"alertType": "cpu", "threshold": 80, "enabled": true,}`),
		},
		{
			Name:   "POST valid alert data multiple systems",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"success\":true"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "CPU",
				"value":   69,
				"min":     9,
				"systems": []string{system1.Id, system2.Id},
			}),
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 2, alerts, "should have 2 alerts")
				matchingAlerts, _ := app.CountRecords("alerts", dbx.HashExp{"name": "CPU", "user": user1.Id, "system": system1.Id, "value": 69, "min": 9})
				assert.EqualValues(t, 1, matchingAlerts, "should have 1 alert")
			},
		},
		{
			Name:   "POST valid alert data single system",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"success\":true"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "Memory",
				"systems": []string{system1.Id},
				"value":   90,
				"min":     10,
			}),
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				user1Alerts, _ := app.CountRecords("alerts", dbx.HashExp{"user": user1.Id})
				assert.EqualValues(t, 3, user1Alerts, "should have 3 alerts")
			},
		},
		{
			Name:   "POST without id always creates new alert (allows duplicates)",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"success\":true", "\"duplicates\""},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "CPU",
				"value":   45,
				"min":     5,
				"systems": []string{system1.Id},
			}),
			BeforeTestFunc: func(t testing.TB, app *pbTests.TestApp, e *core.ServeEvent) {
				beszelTests.ClearCollection(t, app, "alerts")
				beszelTests.CreateRecord(app, "alerts", map[string]any{
					"name":   "CPU",
					"item":   "",
					"system": system1.Id,
					"user":   user1.Id,
					"value":  80,
					"min":    10,
				})
			},
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 2, alerts, "should have 2 alerts (original + new duplicate)")
			},
		},
		{
			Name:            "DELETE no auth",
			Method:          http.MethodDelete,
			URL:             "/api/beszel/user-alerts",
			ExpectedStatus:  401,
			ExpectedContent: []string{"requires valid"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "CPU",
				"systems": []string{system1.Id},
			}),
			BeforeTestFunc: func(t testing.TB, app *pbTests.TestApp, e *core.ServeEvent) {
				beszelTests.ClearCollection(t, app, "alerts")
				beszelTests.CreateRecord(app, "alerts", map[string]any{
					"name":   "CPU",
					"item":   "",
					"system": system1.Id,
					"user":   user1.Id,
					"value":  80,
					"min":    10,
				})
			},
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 1, alerts, "should have 1 alert")
			},
		},
		{
			Name:   "DELETE alert by name (deletes all matching)",
			Method: http.MethodDelete,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"count\":1", "\"success\":true"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "CPU",
				"item":    "",
				"systems": []string{system1.Id},
			}),
			BeforeTestFunc: func(t testing.TB, app *pbTests.TestApp, e *core.ServeEvent) {
				beszelTests.ClearCollection(t, app, "alerts")
				beszelTests.CreateRecord(app, "alerts", map[string]any{
					"name":   "CPU",
					"item":   "",
					"system": system1.Id,
					"user":   user1.Id,
					"value":  80,
					"min":    10,
				})
			},
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.Zero(t, alerts, "should have 0 alerts")
			},
		},
		{
			Name:   "DELETE alert multiple systems",
			Method: http.MethodDelete,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"count\":2", "\"success\":true"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "Memory",
				"item":    "",
				"systems": []string{system1.Id, system2.Id},
			}),
			BeforeTestFunc: func(t testing.TB, app *pbTests.TestApp, e *core.ServeEvent) {
				beszelTests.ClearCollection(t, app, "alerts")
				for _, systemId := range []string{system1.Id, system2.Id} {
					_, err := beszelTests.CreateRecord(app, "alerts", map[string]any{
						"name":   "Memory",
						"item":   "",
						"system": systemId,
						"user":   user1.Id,
						"value":  90,
						"min":    10,
					})
					assert.NoError(t, err, "should create alert")
				}
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 2, alerts, "should have 2 alerts")
			},
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.Zero(t, alerts, "should have 0 alerts")
			},
		},
		{
			Name:   "User 2 should not be able to delete alert of user 1",
			Method: http.MethodDelete,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user2Token,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"count\":1", "\"success\":true"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"name":    "CPU",
				"item":    "",
				"systems": []string{system2.Id},
			}),
			BeforeTestFunc: func(t testing.TB, app *pbTests.TestApp, e *core.ServeEvent) {
				beszelTests.ClearCollection(t, app, "alerts")
				for _, user := range []string{user1.Id, user2.Id} {
					beszelTests.CreateRecord(app, "alerts", map[string]any{
						"name":   "CPU",
						"item":   "",
						"system": system2.Id,
						"user":   user,
						"value":  80,
						"min":    10,
					})
				}
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 2, alerts, "should have 2 alerts")
			},
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				user1AlertCount, _ := app.CountRecords("alerts", dbx.HashExp{"user": user1.Id})
				assert.EqualValues(t, 1, user1AlertCount, "should have 1 alert")
				user2AlertCount, _ := app.CountRecords("alerts", dbx.HashExp{"user": user2.Id})
				assert.Zero(t, user2AlertCount, "should have 0 alerts")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestUserAlertsByIdApi(t *testing.T) {
	hub, _ := beszelTests.NewTestHub(t.TempDir())
	defer hub.Cleanup()

	hub.StartHub()

	user1, _ := beszelTests.CreateUser(hub, "alertbyid@example.com", "password")
	user1Token, _ := user1.NewAuthToken()

	system1, _ := beszelTests.CreateRecord(hub, "systems", map[string]any{
		"name":  "system1",
		"users": []string{user1.Id},
		"host":  "127.0.0.1",
	})

	// Create alert records upfront so their IDs are available in scenario bodies
	alertToUpdate, _ := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "CPU",
		"item":   "",
		"system": system1.Id,
		"user":   user1.Id,
		"value":  80,
		"min":    10,
	})

	alertToDelete, _ := beszelTests.CreateRecord(hub, "alerts", map[string]any{
		"name":   "CPU",
		"item":   "",
		"system": system1.Id,
		"user":   user1.Id,
		"value":  90,
		"min":    5,
	})

	testAppFactory := func(t testing.TB) *pbTests.TestApp {
		return hub.TestApp
	}

	scenarios := []beszelTests.ApiScenario{
		{
			Name:   "POST with id updates existing alert",
			Method: http.MethodPost,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			Body: jsonReader(map[string]any{
				"id":      alertToUpdate.Id,
				"name":    "CPU",
				"value":   45,
				"min":     5,
				"systems": []string{system1.Id},
			}),
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"success\":true"},
			TestAppFactory:  testAppFactory,
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 2, alerts, "should still have 2 alerts total (updated in place, not duplicated)")
				alert, _ := app.FindRecordById("alerts", alertToUpdate.Id)
				assert.EqualValues(t, 45, alert.Get("value"), "should have updated value to 45")
			},
		},
		{
			Name:   "DELETE by id deletes only the specific alert",
			Method: http.MethodDelete,
			URL:    "/api/beszel/user-alerts",
			Headers: map[string]string{
				"Authorization": user1Token,
			},
			Body: jsonReader(map[string]any{
				"id":      alertToDelete.Id,
				"name":    "CPU",
				"item":    "",
				"systems": []string{system1.Id},
			}),
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"count\":1", "\"success\":true"},
			TestAppFactory:  testAppFactory,
			AfterTestFunc: func(t testing.TB, app *pbTests.TestApp, res *http.Response) {
				alerts, _ := app.CountRecords("alerts")
				assert.EqualValues(t, 1, alerts, "should have 1 alert remaining (the other one)")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSendTestNotification(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()

	userToken, err := user.NewAuthToken()

	adminUser, err := beszelTests.CreateUserWithRole(hub, "admin@example.com", "password123", "admin")
	assert.NoError(t, err, "Failed to create admin user")
	adminUserToken, err := adminUser.NewAuthToken()

	superuser, err := beszelTests.CreateSuperuser(hub, "superuser@example.com", "password123")
	assert.NoError(t, err, "Failed to create superuser")
	superuserToken, err := superuser.NewAuthToken()
	assert.NoError(t, err, "Failed to create superuser auth token")

	testAppFactory := func(t testing.TB) *pbTests.TestApp {
		return hub.TestApp
	}

	scenarios := []beszelTests.ApiScenario{
		{
			Name:            "POST /test-notification - no auth should fail",
			Method:          http.MethodPost,
			URL:             "/api/beszel/test-notification",
			ExpectedStatus:  401,
			ExpectedContent: []string{"requires valid"},
			TestAppFactory:  testAppFactory,
			Body: jsonReader(map[string]any{
				"url": "generic://127.0.0.1",
			}),
		},
		{
			Name:           "POST /test-notification - with external auth should succeed",
			Method:         http.MethodPost,
			URL:            "/api/beszel/test-notification",
			TestAppFactory: testAppFactory,
			Headers: map[string]string{
				"Authorization": userToken,
			},
			Body: jsonReader(map[string]any{
				"url": "generic://8.8.8.8",
			}),
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"err\":"},
		},
		{
			Name:           "POST /test-notification - local url with user auth should fail",
			Method:         http.MethodPost,
			URL:            "/api/beszel/test-notification",
			TestAppFactory: testAppFactory,
			Headers: map[string]string{
				"Authorization": userToken,
			},
			Body: jsonReader(map[string]any{
				"url": "generic://localhost:8010",
			}),
			ExpectedStatus:  403,
			ExpectedContent: []string{"Only admins"},
		},
		{
			Name:           "POST /test-notification - internal url with user auth should fail",
			Method:         http.MethodPost,
			URL:            "/api/beszel/test-notification",
			TestAppFactory: testAppFactory,
			Headers: map[string]string{
				"Authorization": userToken,
			},
			Body: jsonReader(map[string]any{
				"url": "generic+http://192.168.0.5",
			}),
			ExpectedStatus:  403,
			ExpectedContent: []string{"Only admins"},
		},
		{
			Name:           "POST /test-notification - internal url with admin auth should succeed",
			Method:         http.MethodPost,
			URL:            "/api/beszel/test-notification",
			TestAppFactory: testAppFactory,
			Headers: map[string]string{
				"Authorization": adminUserToken,
			},
			Body: jsonReader(map[string]any{
				"url": "generic://127.0.0.1",
			}),
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"err\":"},
		},
		{
			Name:           "POST /test-notification - internal url with superuser auth should succeed",
			Method:         http.MethodPost,
			URL:            "/api/beszel/test-notification",
			TestAppFactory: testAppFactory,
			Headers: map[string]string{
				"Authorization": superuserToken,
			},
			Body: jsonReader(map[string]any{
				"url": "generic://127.0.0.1",
			}),
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"err\":"},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
