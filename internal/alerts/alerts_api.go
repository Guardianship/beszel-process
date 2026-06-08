package alerts

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// UpsertUserAlerts handles API request to create or update alerts for a user
// across multiple systems (POST /api/beszel/user-alerts)
func UpsertUserAlerts(e *core.RequestEvent) error {
	userID := e.Auth.Id

	reqData := struct {
		Id        string   `json:"id"`
		Min       uint8    `json:"min"`
		Value     float64  `json:"value"`
		Name      string   `json:"name"`
		Item      string   `json:"item"`
		Systems   []string `json:"systems"`
		Overwrite bool     `json:"overwrite"`
	}{}
	err := e.BindBody(&reqData)
	if err != nil || userID == "" || reqData.Name == "" || len(reqData.Systems) == 0 {
		return e.BadRequestError("Bad data", err)
	}

	alertsCollection, err := e.App.FindCachedCollectionByNameOrId("alerts")
	if err != nil {
		return err
	}

	err = e.App.RunInTransaction(func(txApp core.App) error {
		for _, systemId := range reqData.Systems {
			if reqData.Id != "" {
				// Update existing alert by ID
				alertRecord, err := txApp.FindRecordById(alertsCollection, reqData.Id)
				if err != nil {
					return err
				}
				if alertRecord.GetString("user") != userID {
					return e.ForbiddenError("Not your alert", nil)
				}
				alertRecord.Set("value", reqData.Value)
				alertRecord.Set("min", reqData.Min)
				if err := txApp.SaveNoValidate(alertRecord); err != nil {
					return err
				}
			} else {
				// Create a new alert
				alertRecord := core.NewRecord(alertsCollection)
				alertRecord.Set("user", userID)
				alertRecord.Set("system", systemId)
				alertRecord.Set("name", reqData.Name)
				alertRecord.Set("item", reqData.Item)
				alertRecord.Set("value", reqData.Value)
				alertRecord.Set("min", reqData.Min)
				if err := txApp.SaveNoValidate(alertRecord); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Check for duplicates (same user, system, name, item) for warning
	var duplicateSystems []string
	for _, systemId := range reqData.Systems {
		count, _ := e.App.CountRecords("alerts", dbx.HashExp{
			"user": userID, "system": systemId, "name": reqData.Name, "item": reqData.Item,
		})
		if count > 1 {
			duplicateSystems = append(duplicateSystems, systemId)
		}
	}

	return e.JSON(http.StatusOK, map[string]any{
		"success":    true,
		"duplicates": duplicateSystems,
	})
}

// DeleteUserAlerts handles API request to delete alerts for a user across multiple systems
// (DELETE /api/beszel/user-alerts)
func DeleteUserAlerts(e *core.RequestEvent) error {
	userID := e.Auth.Id

	reqData := struct {
		Id        string   `json:"id"`
		AlertName string   `json:"name"`
		Item      string   `json:"item"`
		Systems   []string `json:"systems"`
	}{}
	err := e.BindBody(&reqData)
	if err != nil || userID == "" || reqData.AlertName == "" || len(reqData.Systems) == 0 {
		return e.BadRequestError("Bad data", err)
	}

	var numDeleted uint16

	err = e.App.RunInTransaction(func(txApp core.App) error {
		for _, systemId := range reqData.Systems {
			if reqData.Id != "" {
				// Delete specific alert by ID
				alertRecord, err := txApp.FindRecordById("alerts", reqData.Id)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						continue
					}
					return err
				}
				if alertRecord.GetString("user") != userID {
					return e.ForbiddenError("Not your alert", nil)
				}
				if err := txApp.Delete(alertRecord); err != nil {
					return err
				}
				numDeleted++
			} else {
				// Delete ALL matching alerts for this system
				hashExp := dbx.HashExp{
					"system": systemId,
					"name":   reqData.AlertName,
					"user":   userID,
				}
				if reqData.Item != "" {
					hashExp["item"] = reqData.Item
				}
				records, err := txApp.FindAllRecords("alerts", hashExp)
				if err != nil {
					return err
				}
				for _, record := range records {
					if err := txApp.Delete(record); err != nil {
						return err
					}
					numDeleted++
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{"success": true, "count": numDeleted})
}

// SendTestNotification handles API request to send a test notification to a specified Shoutrrr URL
func (am *AlertManager) SendTestNotification(e *core.RequestEvent) error {
	var data struct {
		URL string `json:"url"`
	}
	err := e.BindBody(&data)
	if err != nil || data.URL == "" {
		return e.BadRequestError("URL is required", err)
	}
	// Only allow admins to send test notifications to internal URLs
	isAdmin := e.Auth.IsSuperuser() || e.Auth.GetString("role") == "admin"
	if !isAdmin {
		internalURL, err := isInternalURL(data.URL)
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}
		if internalURL {
			return e.ForbiddenError("Only admins can send to internal destinations", nil)
		}
	}
	err = am.sendTestNotificationSafe(data.URL, isAdmin)
	if err != nil {
		return e.JSON(200, map[string]string{"err": err.Error()})
	}
	return e.JSON(200, map[string]bool{"err": false})
}

// sendTestNotificationSafe sends a test notification with SSRF protection for non-admin users
func (am *AlertManager) sendTestNotificationSafe(notificationURL string, isAdmin bool) error {
	if isAdmin {
		// Admins can send to any URL without SSRF protection
		return am.SendShoutrrrAlert(notificationURL, "Test Alert", "This is a notification from Beszel.", am.hub.Settings().Meta.AppURL, "View Beszel")
	}

	// For non-admins, validate the URL resolves to a non-internal IP at connection time
	// This prevents DNS rebinding attacks where DNS resolves to different IPs between check and use
	parsedURL, err := url.Parse(notificationURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Hostname()
	if host != "" && !strings.EqualFold(host, "localhost") {
		if ip := net.ParseIP(host); ip != nil {
			if isInternalIP(ip) {
				return fmt.Errorf("blocked connection to internal IP: %s", ip)
			}
		} else if strings.Contains(host, ".") {
			// Resolve and validate at send time (not check time) to prevent DNS rebinding
			ips, err := net.LookupIP(host)
			if err == nil {
				for _, ip := range ips {
					if isInternalIP(ip) {
						return fmt.Errorf("blocked connection to internal IP: %s (DNS rebinding detected)", ip)
					}
				}
			}
		}
	}

	return am.SendShoutrrrAlert(notificationURL, "Test Alert", "This is a notification from Beszel.", am.hub.Settings().Meta.AppURL, "View Beszel")
}

// isInternalURL checks if the given shoutrrr URL points to an internal destination (localhost or private IP)
func isInternalURL(rawURL string) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}

	host := parsedURL.Hostname()
	if host == "" {
		return false, nil
	}

	if strings.EqualFold(host, "localhost") {
		return true, nil
	}

	if ip := net.ParseIP(host); ip != nil {
		return isInternalIP(ip), nil
	}

	// Some Shoutrrr URLs use the host position for service identifiers rather than a
	// network hostname (for example, discord://token@webhookid). Restrict DNS lookups
	// to names that look like actual hostnames so valid service URLs keep working.
	if !strings.Contains(host, ".") {
		return false, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return false, nil
	}

	if slices.ContainsFunc(ips, isInternalIP) {
		return true, nil
	}

	return false, nil
}

func isInternalIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified()
}
