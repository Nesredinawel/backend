package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"notification-service/models"
	"os"
	"time"

	"golang.org/x/oauth2/google"
)

const fcmEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

// PushMessage is the payload sent to FCM HTTP v1
type PushMessage struct {
	Message struct {
		Token        string                 `json:"token,omitempty"`
		Topic        string                 `json:"topic,omitempty"`
		Notification map[string]string      `json:"notification,omitempty"`
		Data         map[string]interface{} `json:"data,omitempty"`
	} `json:"message"`
}

// maxRetries controls how many times we retry sending push notifications
const maxRetries = 3
const retryDelay = 3 * time.Second

func SendPushNotification(n models.Notification) {
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		log.Println("⚠️ FCM disabled: GOOGLE_APPLICATION_CREDENTIALS not set")
		return
	}

	content, err := os.ReadFile(credPath)
	if err != nil {
		log.Printf("❌ Error reading credentials file: %v", err)
		return
	}

	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, content, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		log.Printf("❌ FCM credentials error: %v", err)
		return
	}

	var sa struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(content, &sa); err != nil || sa.ProjectID == "" {
		log.Printf("❌ FCM invalid service account JSON: %v", err)
		return
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		log.Printf("❌ FCM token generation failed: %v", err)
		return
	}

	msg := PushMessage{}
	msg.Message.Topic = n.UserID
	msg.Message.Notification = map[string]string{
		"title": n.Title,
		"body":  n.Message,
	}
	msg.Message.Data = map[string]interface{}{
		"source": n.SourceService,
		"action": n.Action,
		"id":     n.ID,
		"time":   n.CreatedAt.Format(time.RFC3339),
	}

	payload, _ := json.Marshal(msg)
	url := fmt.Sprintf(fcmEndpoint, sa.ProjectID)

	// Use custom HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("❌ FCM attempt %d/%d failed: %v", attempt, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}

		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			log.Printf("📲 Push notification sent to user %s (%s)", n.UserID, n.Title)
			return
		} else {
			log.Printf("⚠️ FCM attempt %d/%d failed (%d): %s", attempt, maxRetries, resp.StatusCode, resp.Status)
			time.Sleep(retryDelay)
		}
	}

	log.Printf("❌ FCM failed after %d attempts for user %s (%s)", maxRetries, n.UserID, n.Title)
}
