package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	devtoCacheKey = "devto:habits:articles"
	devtoKnownKey = "devto:habits:known_ids"
	devtoTag      = "habits"
	devtoPerPage  = 100
	pollInterval  = 3 * time.Minute
	cacheTTL      = 5 * time.Minute
)

type ExternalArticle struct {
	Title        string `json:"title"`
	Excerpt      string `json:"excerpt"`
	Tags         string `json:"tags"`
	ReadTime     int    `json:"read_time"`
	URL          string `json:"url"`
	CoverImage   string `json:"cover_image"`
	AuthorName   string `json:"author_name"`
	AuthorAvatar string `json:"author_avatar"`
	PublishedAt  string `json:"published_at"`
	Source       string `json:"source"`
}

var (
	sseClients = make(map[string]chan ExternalArticle)
	sseMu      sync.RWMutex
)

func RegisterSSEClient(id string, ch chan ExternalArticle) {
	sseMu.Lock()
	sseClients[id] = ch
	sseMu.Unlock()
}

func UnregisterSSEClient(id string) {
	sseMu.Lock()
	delete(sseClients, id)
	sseMu.Unlock()
}

func broadcastNewArticle(article ExternalArticle) {
	sseMu.RLock()
	defer sseMu.RUnlock()
	for id, ch := range sseClients {
		select {
		case ch <- article:
		default:
			log.Printf("[sse] client %s buffer full, skipping", id)
		}
	}
}

func getKnownIDs(ctx context.Context) (map[int]bool, error) {
	raw, err := Rdb.SMembers(ctx, devtoKnownKey).Result()
	if err != nil {
		return nil, err
	}
	ids := make(map[int]bool, len(raw))
	for _, s := range raw {
		id, err := strconv.Atoi(s)
		if err == nil {
			ids[id] = true
		}
	}
	return ids, nil
}

func StartExternalFetcher(ctx context.Context) {
	log.Printf("[fetcher] starting Dev.to poller every %v (tag=%s, per_page=%d)", pollInterval, devtoTag, devtoPerPage)

	fetchAndCache(ctx)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fetchAndCache(ctx)
		case <-ctx.Done():
			log.Printf("[fetcher] stopped")
			return
		}
	}
}

type devtoRawItem struct {
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

func fetchDevtoArticlesRaw(tag string, perPage int) ([]devtoRawItem, error) {
	url := fmt.Sprintf("https://dev.to/api/articles?tag=%s&per_page=%d", tag, perPage)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dev.to returned status %d", resp.StatusCode)
	}

	var articles []devtoRawItem
	if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
		return nil, err
	}
	return articles, nil
}

func fetchAndCache(ctx context.Context) {
	articles, err := fetchDevtoArticlesRaw(devtoTag, devtoPerPage)
	if err != nil {
		log.Printf("[fetcher] fetch failed: %v", err)
		return
	}

	knownIDs, err := getKnownIDs(ctx)
	if err != nil {
		log.Printf("[fetcher] getKnownIDs failed: %v", err)
		knownIDs = make(map[int]bool)
	}

	external := make([]ExternalArticle, 0, len(articles))
	var newArticles []ExternalArticle

	for _, a := range articles {
		art := ExternalArticle{
			Title:        a.Title,
			Excerpt:      a.Description,
			Tags:         a.Tags,
			ReadTime:     a.ReadingTimeMinutes,
			URL:          a.URL,
			CoverImage:   a.CoverImage,
			AuthorName:   a.User.Name,
			AuthorAvatar: a.User.ProfileImage,
			PublishedAt:  a.PublishedAt,
			Source:       "external",
		}
		external = append(external, art)

		if !knownIDs[a.ID] {
			newArticles = append(newArticles, art)
		}
	}

	pipe := Rdb.Pipeline()
	pipe.Del(ctx, devtoKnownKey)
	for _, a := range articles {
		pipe.SAdd(ctx, devtoKnownKey, a.ID)
	}
	pipe.Persist(ctx, devtoKnownKey)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[fetcher] failed to update known IDs: %v", err)
	}

	data, _ := json.Marshal(external)
	if err := Rdb.Set(ctx, devtoCacheKey, data, cacheTTL).Err(); err != nil {
		log.Printf("[fetcher] failed to cache articles: %v", err)
	}

	log.Printf("[fetcher] cached %d articles (%d new)", len(external), len(newArticles))

	for _, art := range newArticles {
		broadcastNewArticle(art)
	}
}

func GetCachedExternalArticles(ctx context.Context) []ExternalArticle {
	data, err := Rdb.Get(ctx, devtoCacheKey).Bytes()
	if err != nil {
		return nil
	}
	var articles []ExternalArticle
	if err := json.Unmarshal(data, &articles); err != nil {
		return nil
	}
	return articles
}
