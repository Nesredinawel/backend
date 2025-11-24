package utils

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Full event structure expected by notification-service
type NotificationEvent struct {
	UserID        string                 `json:"user_id"`
	Title         string                 `json:"title"`
	Message       string                 `json:"message"`
	SourceService string                 `json:"source_service"`
	Action        string                 `json:"action"`
	Meta          map[string]interface{} `json:"meta"`
	Timestamp     time.Time              `json:"timestamp"`
}

// PublishNotification publishes structured event to Redis
func PublishNotification(rdb *redis.Client, channel string, event NotificationEvent) {
	ctx := context.Background()

	// Fill timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("❌ Failed to marshal notification event: %v", err)
		return
	}

	if err := rdb.Publish(ctx, channel, payload).Err(); err != nil {
		log.Printf("❌ Failed to publish notification: %v", err)
	} else {
		log.Printf("📢 Published event → channel=%s | action=%s | user=%s",
			channel, event.Action, event.UserID)
	}
}
