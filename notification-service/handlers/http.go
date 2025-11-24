package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"notification-service/services"
)

// GET /notifications?user_id=...
func GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	log.Printf("[INFO] GET /notifications called | user_id=%s", userID)

	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	list := services.GetNotifications(userID)
	unread := services.GetUnreadCount(userID)

	response := map[string]interface{}{
		"success":       true,
		"user_id":       userID,
		"total":         len(list),
		"unread_count":  unread,
		"notifications": list,
		"message":       "Notifications fetched successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON response: %v", err)
	}
}

// GET /notifications/count?user_id=...
func UnreadCountHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	log.Printf("[INFO] GET /notifications/count called | user_id=%s", userID)

	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	cnt := services.GetUnreadCount(userID)

	response := map[string]interface{}{
		"success":      true,
		"user_id":      userID,
		"unread_count": cnt,
		"message":      "Unread notification count fetched successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON response: %v", err)
	}
}

// POST /notifications/mark-read?user_id=...&id=...
func MarkReadHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	id := r.URL.Query().Get("id")
	log.Printf("[INFO] POST /notifications/mark-read called | user_id=%s, id=%s", userID, id)

	if userID == "" || id == "" {
		http.Error(w, "user_id and id required", http.StatusBadRequest)
		return
	}

	ok := services.MarkRead(userID, id)
	unread := services.GetUnreadCount(userID)

	msg := "Notification marked as read"
	if !ok {
		msg = "Notification not found or already read"
	}

	response := map[string]interface{}{
		"success":         ok,
		"user_id":         userID,
		"notification_id": id,
		"unread_count":    unread,
		"message":         msg,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON response: %v", err)
	}
}
