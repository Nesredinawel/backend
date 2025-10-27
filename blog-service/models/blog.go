package models

import "time"

type Image struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	URL       string    `json:"url"`
	Caption   string    `json:"caption"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      any       `json:"tags"`
	Published bool      `json:"published"`
	Images    []Image   `json:"images"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
