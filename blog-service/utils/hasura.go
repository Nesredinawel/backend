package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"blog-service/models"
)

// =========================
// Blog Post Operations
// =========================

// InsertPost inserts a new blog post with related images
func InsertPost(cfg Config, post models.Post) (string, error) {
	query := `
	mutation InsertBlog($user_id: uuid!, $title: String!, $content: String!, $tags: jsonb, $published: Boolean!, $images: [blog_service_blogs_images_insert_input!]!) {
	  insert_blog_service_blogs_one(object: {
	    user_id: $user_id,
	    title: $title,
	    content: $content,
	    tags: $tags,
	    published: $published,
	    blogs_images: { data: $images }
	  }) {
	    id
	  }
	}`

	imageData := []map[string]interface{}{}
	for _, img := range post.Images {
		imageData = append(imageData, map[string]interface{}{
			"url":     img.URL,
			"caption": img.Caption,
		})
	}

	variables := map[string]interface{}{
		"user_id":   post.UserID,
		"title":     post.Title,
		"content":   post.Content,
		"tags":      post.Tags,
		"published": post.Published,
		"images":    imageData,
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("hasura returned non-200: %d, %s", resp.StatusCode, string(b))
	}

	var respData struct {
		Data struct {
			InsertBlogServiceBlogsOne struct {
				ID string `json:"id"`
			} `json:"insert_blog_service_blogs_one"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}
	if len(respData.Errors) > 0 {
		return "", fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	return respData.Data.InsertBlogServiceBlogsOne.ID, nil
}

// GetPosts fetches all blog posts with images (optionally only published)
func GetPosts(cfg Config, onlyPublished bool) ([]models.Post, error) {
	query := `
	query GetPosts($published: Boolean) {
	  blog_service_blogs(where: {published: {_eq: $published}}, order_by: {created_at: desc}) {
	    id
	    user_id
	    title
	    content
	    tags
	    published
	    created_at
	    updated_at
	    blogs_images {
	      id
	      url
	      caption
	      created_at
	    }
	  }
	}`

	variables := map[string]interface{}{"published": nil}
	if onlyPublished {
		variables["published"] = true
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			Posts []models.Post `json:"blog_service_blogs"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}
	if len(respData.Errors) > 0 {
		return nil, fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	return respData.Data.Posts, nil
}

// GetPostByID fetches a single blog post with images by ID
func GetPostByID(cfg Config, id string) (models.Post, error) {
	query := `
	query GetPost($id: uuid!) {
	  blog_service_blogs_by_pk(id: $id) {
	    id
	    user_id
	    title
	    content
	    tags
	    published
	    created_at
	    updated_at
	    blogs_images {
	      id
	      url
	      caption
	      created_at
	    }
	  }
	}`

	payload := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"id": id},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Post{}, err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			Post models.Post `json:"blog_service_blogs_by_pk"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return models.Post{}, err
	}
	if len(respData.Errors) > 0 {
		return models.Post{}, fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	return respData.Data.Post, nil
}

// UpdatePost updates a blog post
func UpdatePost(cfg Config, id string, post models.Post) error {
	query := `
	mutation UpdatePost($id: uuid!, $title: String, $content: String, $tags: jsonb, $published: Boolean) {
	  update_blog_service_blogs_by_pk(pk_columns: {id: $id}, _set: {
	    title: $title,
	    content: $content,
	    tags: $tags,
	    published: $published
	  }) {
	    id
	  }
	}`

	variables := map[string]interface{}{
		"id":        id,
		"title":     post.Title,
		"content":   post.Content,
		"tags":      post.Tags,
		"published": post.Published,
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			UpdateBlogServiceBlogsByPk struct {
				ID string `json:"id"`
			} `json:"update_blog_service_blogs_by_pk"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return err
	}
	if len(respData.Errors) > 0 {
		return fmt.Errorf("hasura errors: %v", respData.Errors)
	}
	return nil
}

// DeletePost deletes a blog post
func DeletePost(cfg Config, id string) error {
	query := `
	mutation DeletePost($id: uuid!) {
	  delete_blog_service_blogs_by_pk(id: $id) {
	    id
	  }
	}`

	payload := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"id": id},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			DeleteBlogServiceBlogsByPk struct {
				ID string `json:"id"`
			} `json:"delete_blog_service_blogs_by_pk"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return err
	}
	if len(respData.Errors) > 0 {
		return fmt.Errorf("hasura errors: %v", respData.Errors)
	}
	return nil
}

// =========================
// Image Management
// =========================

// AddImage attaches a new image to an existing blog post
func AddImage(cfg Config, postID, url, caption string) (string, error) {
	query := `
	mutation InsertImage($post_id: uuid!, $url: String!, $caption: String) {
	  insert_blog_service_blogs_images_one(object: {
	    post_id: $post_id,
	    url: $url,
	    caption: $caption
	  }) {
	    id
	  }
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"post_id": postID,
			"url":     url,
			"caption": caption,
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			InsertImage struct {
				ID string `json:"id"`
			} `json:"insert_blog_service_blogs_images_one"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}
	if len(respData.Errors) > 0 {
		return "", fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	return respData.Data.InsertImage.ID, nil
}
