package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"blog-service/models"
)

func InsertPost(cfg Config, post models.Post) (string, error) {
	query := `
	mutation InsertBlog(
		$user_id: uuid!,
		$title: String!,
		$content: String!,
		$excerpt: String,
		$category: String,
		$tags: jsonb,
		$read_time: Int,
		$published: Boolean!,
		$images: [blog_service_blogs_images_insert_input!]!
	) {
	  insert_blog_service_blogs_one(object: {
	    user_id: $user_id,
	    title: $title,
	    content: $content,
	    excerpt: $excerpt,
	    category: $category,
	    tags: $tags,
	    read_time: $read_time,
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

	cat := post.Category
	if cat == "" {
		cat = "general"
	}
	readTime := post.ReadTime
	if readTime < 1 {
		readTime = 1
	}

	variables := map[string]interface{}{
		"user_id":   post.UserID,
		"title":     post.Title,
		"content":   post.Content,
		"excerpt":   post.Excerpt,
		"category":  cat,
		"tags":      post.Tags,
		"read_time": readTime,
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

func GetPosts(cfg Config, onlyPublished bool, category string, limit, offset int) ([]models.Post, error) {
	conditions := []string{}
	variables := map[string]interface{}{}
	varDecl := "$limit: Int!, $offset: Int!"

	if onlyPublished {
		conditions = append(conditions, "published: {_eq: true}")
	}
	if category != "" {
		conditions = append(conditions, "category: {_eq: $category}")
		variables["category"] = category
		varDecl = "$limit: Int!, $offset: Int!, $category: String"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "where: {" + joinConditions(conditions) + "}"
	}

	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	variables["limit"] = limit
	variables["offset"] = offset

	queryVar := ""
	if varDecl != "" {
		queryVar = "(" + varDecl + ")"
	}
	query := fmt.Sprintf(`
	query GetPosts%s {
	  blog_service_blogs(%s order_by: {created_at: desc}, limit: $limit, offset: $offset) {
	    id
	    user_id
	    title
	    content
	    excerpt
	    category
	    tags
	    read_time
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
	}`, queryVar, whereClause)

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

func GetPostsCount(cfg Config, onlyPublished bool, category string) (int, error) {
	conditions := []string{}
	variables := map[string]interface{}{}
	varDecl := ""

	if onlyPublished {
		conditions = append(conditions, "published: {_eq: true}")
	}
	if category != "" {
		conditions = append(conditions, "category: {_eq: $category}")
		variables["category"] = category
		varDecl = "$category: String"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "where: {" + joinConditions(conditions) + "}"
	}

	queryVar := ""
	if varDecl != "" {
		queryVar = "(" + varDecl + ")"
	}
	query := fmt.Sprintf(`
	query GetPostsCount%s {
	  blog_service_blogs_aggregate(%s) {
	    aggregate {
	      count
	    }
	  }
	}`, queryVar, whereClause)

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
		return 0, err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			Aggregate struct {
				Aggregate struct {
					Count int `json:"count"`
				} `json:"aggregate"`
			} `json:"blog_service_blogs_aggregate"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return 0, err
	}
	if len(respData.Errors) > 0 {
		return 0, fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	return respData.Data.Aggregate.Aggregate.Count, nil
}

func GetPostByID(cfg Config, id string) (models.Post, error) {
	query := `
	query GetPost($id: uuid!) {
	  blog_service_blogs_by_pk(id: $id) {
	    id
	    user_id
	    title
	    content
	    excerpt
	    category
	    tags
	    read_time
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

func UpdatePost(cfg Config, id string, post models.Post) error {
	query := `
	mutation UpdatePost($id: uuid!, $title: String, $content: String, $excerpt: String, $category: String, $tags: jsonb, $read_time: Int, $published: Boolean) {
	  update_blog_service_blogs_by_pk(pk_columns: {id: $id}, _set: {
	    title: $title,
	    content: $content,
	    excerpt: $excerpt,
	    category: $category,
	    tags: $tags,
	    read_time: $read_time,
	    published: $published
	  }) {
	    id
	  }
	}`

	variables := map[string]interface{}{
		"id":        id,
		"title":     post.Title,
		"content":   post.Content,
		"excerpt":   post.Excerpt,
		"category":  post.Category,
		"tags":      post.Tags,
		"read_time": post.ReadTime,
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

func joinConditions(conds []string) string {
	result := ""
	for i, c := range conds {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
