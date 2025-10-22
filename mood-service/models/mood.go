package models

import "time"

type Mood struct {
	ID             string     `json:"id,omitempty"`
	UserID         string     `json:"user_id,omitempty"`
	Mood           string     `json:"mood"`
	Emoji          *string    `json:"emoji,omitempty"`
	Note           *string    `json:"note,omitempty"`
	MoodDate       *string    `json:"mood_date,omitempty"` // yyyy-mm-dd
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}
