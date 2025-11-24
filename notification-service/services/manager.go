package services

import (
	"log"
	"sync"

	"notification-service/models"

	"github.com/google/uuid"
)

// Manager holds notifications and websocket subscribers
type Manager struct {
	mu          sync.RWMutex
	store       map[string][]models.Notification               // user_id -> slice of notifications (most recent first)
	subscribers map[string]map[string]chan models.Notification // user_id -> (subID -> channel)
	maxPerUser  int
}

var mgr *Manager

// InitManager initializes the global manager
func InitManager(maxPerUser int) {
	mgr = &Manager{
		store:       make(map[string][]models.Notification),
		subscribers: make(map[string]map[string]chan models.Notification),
		maxPerUser:  maxPerUser,
	}
}

// AddNotification stores and broadcasts a notification for a user
func AddNotification(n models.Notification) {
	if mgr == nil {
		log.Println("⚠️ manager not initialized")
		return
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// prepend to slice so newest is first
	list := mgr.store[n.UserID]
	list = append([]models.Notification{n}, list...)
	if len(list) > mgr.maxPerUser {
		list = list[:mgr.maxPerUser]
	}
	mgr.store[n.UserID] = list

	// broadcast to subscribers
	subs := mgr.subscribers[n.UserID]
	for id, ch := range subs {
		select {
		case ch <- n:
			// delivered
		default:
			// subscriber might be slow; drop oldest or skip
			log.Printf("⚠️ subscriber %s for user %s not ready; skipping", id, n.UserID)
		}
	}
}

// GetNotifications returns notifications for a user (copy)
func GetNotifications(userID string) []models.Notification {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	list := mgr.store[userID]
	out := make([]models.Notification, len(list))
	copy(out, list)
	return out
}

// GetUnreadCount returns unread count
func GetUnreadCount(userID string) int {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	cnt := 0
	for _, n := range mgr.store[userID] {
		if !n.Read {
			cnt++
		}
	}
	return cnt
}

// MarkRead marks a notification as read by id
func MarkRead(userID, id string) bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	list := mgr.store[userID]
	changed := false
	for i := range list {
		if list[i].ID == id {
			if !list[i].Read {
				list[i].Read = true
				changed = true
			}
			break
		}
	}
	mgr.store[userID] = list
	return changed
}

// Subscribe returns subID and channel to receive future notifications
func Subscribe(userID string) (subID string, ch <-chan models.Notification) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	id := uuid.NewString()
	c := make(chan models.Notification, 8)

	if mgr.subscribers[userID] == nil {
		mgr.subscribers[userID] = make(map[string]chan models.Notification)
	}
	mgr.subscribers[userID][id] = c
	return id, c
}

// Unsubscribe removes subscription
func Unsubscribe(userID, subID string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if subs := mgr.subscribers[userID]; subs != nil {
		if ch, ok := subs[subID]; ok {
			close(ch)
			delete(subs, subID)
		}
		if len(subs) == 0 {
			delete(mgr.subscribers, userID)
		}
	}
}
