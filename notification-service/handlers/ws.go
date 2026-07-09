package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"notification-service/config"
	middleware "notification-service/middlewares"
	"notification-service/services"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" || origin == "http://localhost:8081" || origin == "http://localhost:5173" || origin == "http://localhost:3000" {
			return true
		}
		allowedOrigin := config.LoadConfig().CORsOrigin
		return allowedOrigin == "*" || allowedOrigin == origin
	},
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	log.Printf("[INFO] WebSocket connection attempt | user_id=%s", userID)
	if userID == "" {
		log.Printf("[WARN] Unauthorized WebSocket connection attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket upgrade failed | user_id=%s, error=%v", userID, err)
		return
	}
	defer conn.Close()
	log.Printf("[INFO] WebSocket connection established | user_id=%s", userID)

	subID, ch := services.Subscribe(userID)
	defer func() {
		services.Unsubscribe(userID, subID)
		log.Printf("[INFO] WebSocket unsubscribed | user_id=%s, subID=%s", userID, subID)
	}()

	// Send existing notifications
	existing := services.GetNotifications(userID)
	if len(existing) > 0 {
		if bytes, err := json.Marshal(existing); err == nil {
			conn.WriteMessage(websocket.TextMessage, bytes)
			log.Printf("[INFO] Sent %d existing notifications | user_id=%s", len(existing), userID)
		} else {
			log.Printf("[ERROR] Failed to marshal existing notifications | user_id=%s, error=%v", userID, err)
		}
	}

	for {
		select {
		case n, ok := <-ch:
			if !ok {
				log.Printf("[INFO] Notification channel closed | user_id=%s", userID)
				return
			}
			n.CreatedAt = n.CreatedAt.UTC()
			payload, _ := json.Marshal(n)
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf("[ERROR] WebSocket write failed | user_id=%s, error=%v", userID, err)
				return
			}
			log.Printf("[INFO] Sent notification via WebSocket | user_id=%s, title=%s", userID, n.Title)
		case <-time.After(60 * time.Second):
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("[WARN] WebSocket ping failed | user_id=%s, error=%v", userID, err)
				return
			}
			log.Printf("[DEBUG] WebSocket ping sent | user_id=%s", userID)
		}
	}
}
