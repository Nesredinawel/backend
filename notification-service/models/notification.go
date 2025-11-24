package models

import "time"

// Notification is the object delivered to clients
type Notification struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	SourceService string    `json:"source_service,omitempty"` // e.g. "auth-service"
	Action        string    `json:"action,omitempty"`         // e.g. "created", "updated"
	Meta          any       `json:"meta,omitempty"`           // optional extra payload
	Read          bool      `json:"read"`
	CreatedAt     time.Time `json:"created_at"`
}

// IncomingEvent is the expected JSON shape published by other services
// At minimum, other services must publish user_id, title, message
type IncomingEvent struct {
	UserID        string      `json:"user_id"`
	Title         string      `json:"title"`
	Message       string      `json:"message"`
	SourceService string      `json:"source_service,omitempty"`
	Action        string      `json:"action,omitempty"`
	Meta          interface{} `json:"meta,omitempty"`
	Timestamp     *time.Time  `json:"timestamp,omitempty"` // optional
}
