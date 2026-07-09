package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

		result := make([]map[string]interface{}, 0, len(posts)+100)
		for _, p := range posts {
			item := map[string]interface{}{
				"id":        p.ID,
				"title":     p.Title,
				"content":   p.Content,
				"excerpt":   p.Excerpt,
				"category":  p.Category,
				"tags":      tagsToString(p.Tags),
				"read_time": p.ReadTime,
				"source":    "local",
			}
			if !p.CreatedAt.IsZero() {
				item["published_at"] = p.CreatedAt.Format(time.RFC3339)
			}
			result = append(result, item)
		}

		external := utils.GetCachedExternalArticles(r.Context())
		for i := range external {
			item := map[string]interface{}{
				"title":         external[i].Title,
				"excerpt":       external[i].Excerpt,
				"tags":          external[i].Tags,
				"read_time":     external[i].ReadTime,
				"url":           external[i].URL,
				"cover_image":   external[i].CoverImage,
				"author_name":   external[i].AuthorName,
				"author_avatar": external[i].AuthorAvatar,
				"published_at":  external[i].PublishedAt,
				"source":        external[i].Source,
			}
			result = append(result, item)
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"posts":       result,
			"total_local": total,
			"total":       len(result),
			"limit":       limit,
			"offset":      offset,
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

func GetExternalPosts(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		articles := utils.GetCachedExternalArticles(r.Context())
		if articles == nil {
			articles = []utils.ExternalArticle{}
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"articles": articles,
			"tag":      "habits",
		})
	}
}

func StreamExternalPosts(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			utils.WriteJSONError(w, utils.NewServerError("streaming not supported"), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
		ch := make(chan utils.ExternalArticle, 10)
		utils.RegisterSSEClient(clientID, ch)
		defer utils.UnregisterSSEClient(clientID)

		ctx := r.Context()

		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		flusher.Flush()

		for {
			select {
			case article := <-ch:
				data, _ := json.Marshal(article)
				fmt.Fprintf(w, "event: new_article\ndata: %s\n\n", data)
				flusher.Flush()
			case <-ctx.Done():
				return
			}
		}
	}
}

func tagsToString(tags any) string {
	if tags == nil {
		return ""
	}
	switch v := tags.(type) {
	case string:
		return v
	case []interface{}:
		s := ""
		for i, t := range v {
			if i > 0 {
				s += ", "
			}
			s += fmt.Sprintf("%v", t)
		}
		return s
	}
	return ""
}
