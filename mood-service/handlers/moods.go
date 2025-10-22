package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"mood-service/middlewares"
	"mood-service/models"
	"mood-service/utils"
)

// Helper types to scan nullable SQL values
type sqlNullString = sql.NullString
type sqlNullTime = sql.NullTime

// CreateMood - POST /moods
func CreateMood(cfg utils.Config, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID, _ := r.Context().Value(middlewares.CtxUserID).(string)
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Optional: verify user with auth-service
		authHeader := r.Header.Get("Authorization")
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
		if cfg.AuthServiceURL != "" && token != "" {
			if err := utils.VerifyUserWithAuthService(cfg, token); err != nil {
				log.Printf("auth verification failed: %v", err)
				http.Error(w, "user verification failed", http.StatusUnauthorized)
				return
			}
		}

		// Parse request body
		var req struct {
			Mood     string  `json:"mood"`
			Emoji    *string `json:"emoji,omitempty"`
			Note     *string `json:"note,omitempty"`
			MoodDate *string `json:"mood_date,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.Mood == "" {
			http.Error(w, "mood required", http.StatusBadRequest)
			return
		}

		// Use current date if not provided
		moodDate := time.Now().Format("2006-01-02")
		if req.MoodDate != nil && *req.MoodDate != "" {
			moodDate = *req.MoodDate
		}

		// Insert into DB
		var id string
		err := db.QueryRow(
			`INSERT INTO moods (user_id, mood, emoji, note, mood_date) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			userID, req.Mood, req.Emoji, req.Note, moodDate,
		).Scan(&id)
		if err != nil {
			log.Printf("insert mood error: %v", err)
			http.Error(w, "failed to insert mood", http.StatusInternalServerError)
			return
		}

		// Respond with created mood
		resp := map[string]interface{}{
			"id":        id,
			"user_id":   userID,
			"mood":      req.Mood,
			"emoji":     req.Emoji,
			"note":      req.Note,
			"mood_date": moodDate,
		}
		w.Header().Set("Content-Type", "application/json")
		pretty, _ := json.MarshalIndent(resp, "", "  ")
		w.Write(pretty)
	}
}

// ListMoods - GET /moods?from=YYYY-MM-DD&to=YYYY-MM-DD
func ListMoods(cfg utils.Config, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := r.Context().Value(middlewares.CtxUserID).(string)
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

		// Set defaults if missing
		if from == "" {
			from = "1970-01-01"
		}
		if to == "" {
			to = time.Now().Format("2006-01-02")
		}

		query := `
		SELECT id, user_id, mood, emoji, note, mood_date, created_at, updated_at
		FROM moods
		WHERE user_id=$1 AND mood_date >= $2 AND mood_date <= $3
		ORDER BY mood_date DESC
		LIMIT 100
		`

		rows, err := db.Query(query, userID, from, to)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var out []models.Mood
		for rows.Next() {
			var m models.Mood
			var moodDate sqlNullString
			var emoji, note sqlNullString
			var created, updated sqlNullTime

			if err := rows.Scan(&m.ID, &m.UserID, &m.Mood, &emoji, &note, &moodDate, &created, &updated); err != nil {
				http.Error(w, "scan error", http.StatusInternalServerError)
				return
			}

			if emoji.Valid {
				m.Emoji = &emoji.String
			}
			if note.Valid {
				m.Note = &note.String
			}
			if moodDate.Valid {
				md := moodDate.String
				m.MoodDate = &md
			}
			if created.Valid {
				t := created.Time
				m.CreatedAt = &t
			}
			if updated.Valid {
				t := updated.Time
				m.UpdatedAt = &t
			}

			out = append(out, m)
		}

		w.Header().Set("Content-Type", "application/json")
		pretty, _ := json.MarshalIndent(out, "", "  ")
		w.Write(pretty)
	}
}
