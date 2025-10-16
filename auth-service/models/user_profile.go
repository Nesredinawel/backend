package models

type UserProfile struct {
	ID              string  `json:"id,omitempty"`
	UserID          string  `json:"user_id"`
	Bio             *string `json:"bio,omitempty"`
	CustomAvatarURL *string `json:"custom_avatar_url,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	UpdatedAt       *string `json:"updated_at,omitempty"`
}
