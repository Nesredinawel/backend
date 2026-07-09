package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"blog-service/middlewares"
	"blog-service/models"
	"blog-service/utils"

	"github.com/go-chi/chi/v5"
)

func CreatePost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDVal := r.Context().Value(middlewares.CtxUserID)
		if userIDVal == nil {
			utils.WriteJSONError(w, utils.NewAuthError("Unauthorized. Please log in again."), http.StatusUnauthorized)
			return
		}
		userID := userIDVal.(string)

		var req models.Post
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to decode request body: %v", err)
			utils.WriteJSONError(w, utils.NewBadRequestError("Invalid request body. Please provide valid JSON."), http.StatusBadRequest)
			return
		}
		req.UserID = userID

		if req.Images == nil {
			req.Images = []models.Image{}
		}

		log.Printf("Creating post: %+v", req)

		id, err := utils.InsertPost(cfg, req)
		if err != nil {
			log.Printf("InsertPost failed: %v", err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to create post. Please try again."), http.StatusInternalServerError)
			return
		}

		utils.PublishNotification(utils.Rdb, "blog_events", utils.NotificationEvent{
			UserID:        userID,
			Title:         "New Post Created",
			Message:       "Your blog post has been published successfully.",
			SourceService: "blog-service",
			Action:        "POST_CREATE",
			Meta: map[string]interface{}{
				"post_id": id,
				"title":   req.Title,
			},
		})

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"post_id": id,
		})
	}
}

func GetPosts(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit := 20
		offset := 0

		if limitStr != "" {
			if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
				limit = v
			}
		}
		if offsetStr != "" {
			if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
				offset = v
			}
		}

		posts, err := utils.GetPosts(cfg, true, category, limit, offset)
		if err != nil {
			log.Printf("GetPosts failed: %v", err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to fetch posts. Please try again."), http.StatusInternalServerError)
			return
		}

		total, err := utils.GetPostsCount(cfg, true, category)
		if err != nil {
			log.Printf("GetPostsCount failed: %v", err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to fetch post count."), http.StatusInternalServerError)
			return
		}

		if posts == nil {
			posts = []models.Post{}
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"posts":   posts,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
		})
	}
}

func GetPost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			utils.WriteJSONError(w, utils.NewBadRequestError("Missing post ID parameter"), http.StatusBadRequest)
			return
		}

		post, err := utils.GetPostByID(cfg, id)
		if err != nil {
			log.Printf("GetPostByID failed for id=%s: %v", id, err)
			utils.WriteJSONError(w, utils.NewNotFoundError("Post not found."), http.StatusNotFound)
			return
		}

		if !post.Published {
			utils.WriteJSONError(w, utils.NewNotFoundError("Post not found."), http.StatusNotFound)
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"post":    post,
		})
	}
}

func UpdatePost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			utils.WriteJSONError(w, utils.NewBadRequestError("Missing post ID parameter"), http.StatusBadRequest)
			return
		}

		var req models.Post
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to decode request body for update id=%s: %v", id, err)
			utils.WriteJSONError(w, utils.NewBadRequestError("Invalid request body. Please provide valid JSON."), http.StatusBadRequest)
			return
		}

		log.Printf("Updating post id=%s: %+v", id, req)

		if err := utils.UpdatePost(cfg, id, req); err != nil {
			log.Printf("UpdatePost failed id=%s: %v", id, err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to update post. Please try again."), http.StatusInternalServerError)
			return
		}

		userIDVal := r.Context().Value(middlewares.CtxUserID)
		if userIDVal != nil {
			userID := userIDVal.(string)
			utils.PublishNotification(utils.Rdb, "blog_events", utils.NotificationEvent{
				UserID:        userID,
				Title:         "Post Updated",
				Message:       "A blog post was updated successfully.",
				SourceService: "blog-service",
				Action:        "POST_UPDATE",
				Meta: map[string]interface{}{
					"post_id": id,
					"title":   req.Title,
				},
			})
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}

func DeletePost(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			utils.WriteJSONError(w, utils.NewBadRequestError("Missing post ID parameter"), http.StatusBadRequest)
			return
		}

		log.Printf("Deleting post id=%s", id)

		if err := utils.DeletePost(cfg, id); err != nil {
			log.Printf("DeletePost failed id=%s: %v", id, err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to delete post. Please try again."), http.StatusInternalServerError)
			return
		}

		userIDVal := r.Context().Value(middlewares.CtxUserID)
		if userIDVal != nil {
			userID := userIDVal.(string)
			utils.PublishNotification(utils.Rdb, "blog_events", utils.NotificationEvent{
				UserID:        userID,
				Title:         "Post Deleted",
				Message:       "A blog post was deleted.",
				SourceService: "blog-service",
				Action:        "POST_DELETE",
				Meta: map[string]interface{}{
					"post_id": id,
				},
			})
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}

type devtoArticle struct {
	ID                 int    `json:"id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	URL                string `json:"url"`
	CoverImage         string `json:"cover_image"`
	Tags               string `json:"tags"`
	ReadingTimeMinutes int    `json:"reading_time_minutes"`
	PublishedAt        string `json:"published_at"`
	User               struct {
		Name        string `json:"name"`
		Username    string `json:"username"`
		ProfileImage string `json:"profile_image"`
	} `json:"user"`
}

func GetExternalPosts(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := r.URL.Query().Get("tag")
		perPage := r.URL.Query().Get("per_page")

		if tag == "" {
			tag = "habits"
		}

		url := "https://dev.to/api/articles?tag=" + tag
		if perPage != "" {
			url += "&per_page=" + perPage
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Dev.to API request failed: %v", err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to fetch external posts."), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Dev.to API returned status %d", resp.StatusCode)
			utils.WriteJSONError(w, utils.NewServerError("External API returned an error."), http.StatusInternalServerError)
			return
		}

		var articles []devtoArticle
		if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
			log.Printf("Failed to decode Dev.to response: %v", err)
			utils.WriteJSONError(w, utils.NewServerError("Failed to parse external posts."), http.StatusInternalServerError)
			return
		}

		if articles == nil {
			articles = []devtoArticle{}
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"articles": articles,
			"tag":      tag,
		})
	}
}
