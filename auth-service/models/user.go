package models

type User struct {
	ID         string `json:"id,omitempty"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Password  string `json:"password,omitempty"`  // ✅ Add this
	AvatarURL  string `json:"avatar_url,omitempty"`
	Provider   string `json:"provider,omitempty"`
	ProviderID string `json:"provider_id,omitempty"`
	Role       string `json:"role,omitempty"`
}
