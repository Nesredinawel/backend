package models

type User struct {
	ID        string `json:"id,omitempty"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}
