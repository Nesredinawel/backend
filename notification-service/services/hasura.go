package services

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"notification-service/config"
	"notification-service/models"
	"strings"
)

func PersistNotificationToHasura(n models.Notification) {
	cfg := config.LoadConfig()

	// If no Hasura credentials, skip silently
	if cfg.HasuraEndpoint == "" || cfg.HasuraAdminSecret == "" {
		return
	}

	// Build mutation name based on schema + table
	// Example: "notification_service" → "insert_notification_service_notifications_one"
	schema := cfg.HasuraSchema
	safeSchema := strings.ReplaceAll(schema, "-", "_")
	mutationName := "insert_" + safeSchema + "_notifications_one"

	query := `
	mutation InsertNotification($object: ` + safeSchema + `_notifications_insert_input!) {
		` + mutationName + `(object: $object) {
			id
		}
	}`

	object := map[string]interface{}{
		"id":             n.ID,
		"user_id":        n.UserID,
		"title":          n.Title,
		"message":        n.Message,
		"source_service": n.SourceService,
	}

	// Optional fields
	if n.Action != "" {
		object["action"] = n.Action
	}
	if n.Meta != nil {
		object["meta"] = n.Meta
	}

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"object": object,
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ Hasura persist failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		log.Printf("⚠️ Hasura error %d: %s", resp.StatusCode, string(raw))
	}
}
