package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"blog-service/middlewares"
	"blog-service/models"
	"blog-service/utils"

	"github.com/go-chi/chi/v5"
)

// CreatePost creates a new blog post (admin only)
func CreatePost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userIDVal := r.Context().Value(middlewares.CtxUserID)
		if userIDVal == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := userIDVal.(string)

		// Decode request body
		var req models.Post
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("❌ Failed to decode request body: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		req.UserID = userID

		// Ensure Images slice is not nil
		if req.Images == nil {
			req.Images = []models.Image{}
		}

		// Log the payload for debugging
		log.Printf("Creating post: %+v", req)

		// Insert post using utils
		id, err := utils.InsertPost(cfg, req)
		if err != nil {
			log.Printf("❌ InsertPost failed: %v", err)
			http.Error(w, "failed to insert post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"post_id": id,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// GetPosts returns posts; non-admins only get published posts
func GetPosts(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleVal := r.Context().Value(middlewares.CtxRole)
		role := ""
		if roleVal != nil {
			role = roleVal.(string)
		}

		onlyPublished := role != "admin"

		posts, err := utils.GetPosts(cfg, onlyPublished)
		if err != nil {
			log.Printf("❌ GetPosts failed: %v", err)
			http.Error(w, "failed to fetch posts: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"posts":   posts,
		})
	}
}

// GetPost returns a single post by id (published unless admin)
func GetPost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing id param", http.StatusBadRequest)
			return
		}

		post, err := utils.GetPostByID(cfg, id)
		if err != nil {
			log.Printf("❌ GetPostByID failed for id=%s: %v", id, err)
			http.Error(w, "failed to fetch post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		roleVal := r.Context().Value(middlewares.CtxRole)
		role := ""
		if roleVal != nil {
			role = roleVal.(string)
		}
		if !post.Published && role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"post":    post,
		})
	}
}

// UpdatePost updates a post (admin only)
func UpdatePost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing id param", http.StatusBadRequest)
			return
		}

		var req models.Post
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("❌ Failed to decode request body for update id=%s: %v", id, err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Updating post id=%s: %+v", id, req)

		if err := utils.UpdatePost(cfg, id, req); err != nil {
			log.Printf("❌ UpdatePost failed id=%s: %v", id, err)
			http.Error(w, "failed to update post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	}
}

// DeletePost deletes a post (admin only)
func DeletePost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing id param", http.StatusBadRequest)
			return
		}

		log.Printf("Deleting post id=%s", id)

		if err := utils.DeletePost(cfg, id); err != nil {
			log.Printf("❌ DeletePost failed id=%s: %v", id, err)
			http.Error(w, "failed to delete post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	}
}
