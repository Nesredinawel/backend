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

	// Auto-fill title if missing
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

	// Validate required fields
	if ev.UserID == "" || ev.Message == "" {
		log.Printf("[Redis][%s] ⚠️ Missing user_id or message in event: %+v", correlationID, ev)
		return
	}

	// Timestamp logic
	ts := time.Now().UTC()
	if ev.Timestamp != nil {
		ts = *ev.Timestamp
	}

	// Create Notification object
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

	// Save in memory
	services.AddNotification(n)
	log.Printf("[Redis][%s] 🔔 Added to in-memory store", correlationID)

	// Persist to Hasura (async)
	go services.PersistNotificationToHasura(n)
	log.Printf("[Redis][%s] 💻 Persisting to Hasura (async)", correlationID)

	// Push notification (async)
	go services.SendPushNotification(n)
	log.Printf("[Redis][%s] 📲 Sending push notification (async)", correlationID)

	// -----------------------------
	//  Extract EMAIL from Meta
	// -----------------------------
	var email string
	if ev.Meta != nil {
		if metaMap, ok := ev.Meta.(map[string]interface{}); ok {
			if value, found := metaMap["email"]; found {
				if emailStr, ok := value.(string); ok {
					email = emailStr
				}
			}
		}
	}

	if email == "" {
		log.Printf("[Redis][%s] ⚠️ No email found in Meta. Email notification skipped.", correlationID)
	} else {
		go services.SendEmailNotification(n, email)
		log.Printf("[Redis][%s] ✉️ Sending email notification to: %s (async)", correlationID, email)
	}

	log.Printf("[Redis][%s] 📨 Notification processing completed for user=%s | title=%s",
		correlationID, n.UserID, n.Title)
}
