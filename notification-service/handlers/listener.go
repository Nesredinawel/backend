package handlers

import (
	"encoding/json"
	"log"
	"notification-service/models"
	"notification-service/services"
	"notification-service/utils"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// StartRedisListener subscribes to Redis channels
func StartRedisListener(channels ...string) {
	pubsub := utils.Rdb.Subscribe(utils.Ctx, channels...)
	ch := pubsub.Channel()

	log.Printf("[Redis] 📡 Listener started on channels: %v", channels)

	for msg := range ch {
		go handleIncomingEvent(msg)
	}
}

func handleIncomingEvent(msg *redis.Message) {
	correlationID := uuid.New().String()
	log.Printf("[Redis][%s] 🟢 Received message from channel [%s]: %s", correlationID, msg.Channel, msg.Payload)

	var ev models.IncomingEvent
	if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
		log.Printf("[Redis][%s] ❌ Invalid payload: %v | payload=%s", correlationID, err, msg.Payload)
		return
	}

	if ev.Title == "" {
		switch ev.SourceService {
		case "auth-service":
			ev.Title = "Login Successful"
		case "mood-service":
			ev.Title = "Mood Event"
		case "blog-service":
			ev.Title = "Blog Update"
		default:
			ev.Title = "Notification"
		}
		log.Printf("[Redis][%s] 📝 Auto-filled title: %s", correlationID, ev.Title)
	}

	if ev.UserID == "" || ev.Message == "" {
		log.Printf("[Redis][%s] ⚠️ Missing user_id or message in event: %+v", correlationID, ev)
		return
	}

	ts := time.Now().UTC()
	if ev.Timestamp != nil {
		ts = *ev.Timestamp
	}

	n := models.Notification{
		ID:            uuid.New().String(),
		UserID:        ev.UserID,
		Title:         ev.Title,
		Message:       ev.Message,
		SourceService: ev.SourceService,
		Action:        ev.Action,
		Meta:          ev.Meta,
		Read:          false,
		CreatedAt:     ts,
	}

	log.Printf("[Redis][%s] 💾 Notification object created: %+v", correlationID, n)

	services.AddNotification(n)
	log.Printf("[Redis][%s] 🔔 Added to in-memory store", correlationID)

	go services.PersistNotificationToHasura(n)
	log.Printf("[Redis][%s] 💻 Persisting to Hasura (async)", correlationID)

	go services.SendPushNotification(n)
	log.Printf("[Redis][%s] 📲 Sending push notification (async)", correlationID)

	go services.SendEmailNotification(n, "user@example.com")
	log.Printf("[Redis][%s] ✉️ Sending email notification (async)", correlationID)

	log.Printf("[Redis][%s] 📨 Notification processing completed for user=%s | title=%s", correlationID, n.UserID, n.Title)
}
