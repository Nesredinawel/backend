package handlers

import (
	"log"
	"net/http"

	middleware "notification-service/middlewares"
	"notification-service/services"
	"notification-service/utils"
)

func GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	if userID == "" {
		utils.WriteJSONError(w, utils.NewAuthError("Unauthorized"), http.StatusUnauthorized)
		return
	}
	log.Printf("[INFO] GET /notifications called | user_id=%s", userID)

	list := services.GetNotifications(userID)
	unread := services.GetUnreadCount(userID)

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"user_id":       userID,
		"total":         len(list),
		"unread_count":  unread,
		"notifications": list,
	})
}

func UnreadCountHandler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	if userID == "" {
		utils.WriteJSONError(w, utils.NewAuthError("Unauthorized"), http.StatusUnauthorized)
		return
	}
	log.Printf("[INFO] GET /notifications/count called | user_id=%s", userID)

	cnt := services.GetUnreadCount(userID)

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"user_id":      userID,
		"unread_count": cnt,
	})
}

func MarkReadHandler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	id := r.URL.Query().Get("id")
	if userID == "" {
		utils.WriteJSONError(w, utils.NewAuthError("Unauthorized"), http.StatusUnauthorized)
		return
	}
	log.Printf("[INFO] POST /notifications/mark-read called | user_id=%s, id=%s", userID, id)

	if id == "" {
		utils.WriteJSONError(w, utils.NewBadRequestError("Missing required parameter: id"), http.StatusBadRequest)
		return
	}

	ok := services.MarkRead(userID, id)
	unread := services.GetUnreadCount(userID)

	msg := "Notification marked as read"
	if !ok {
		msg = "Notification not found or already read"
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":         ok,
		"user_id":         userID,
		"notification_id": id,
		"unread_count":    unread,
		"message":         msg,
	})
}
