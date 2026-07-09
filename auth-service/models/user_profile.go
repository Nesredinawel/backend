package models

type UserProfile struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	AvatarURL       *string `json:"avatar_url"`
	Bio             *string `json:"bio"`
	CustomAvatarURL *string `json:"custom_avatar_url"`
	CreatedAt       *string `json:"created_at"`
	UpdatedAt       *string `json:"updated_at"`
}

type UserProfileInput struct {
	Bio             *string `json:"bio"`
	CustomAvatarURL *string `json:"custom_avatar_url"`
}
