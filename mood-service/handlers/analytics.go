package handlers

import (
	"log"
	"net/http"
	"time"

	"mood-service/middlewares"
	"mood-service/models"
	"mood-service/utils"
)

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func MoodKPI(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDVal := r.Context().Value(middlewares.CtxUserID)
		if userIDVal == nil {
			utils.WriteJSONError(w, utils.NewBadRequestError("Unauthorized. Please log in again."), http.StatusUnauthorized)
			return
		}
		userID := userIDVal.(string)

		q := r.URL.Query()
		from := q.Get("from")
		to := q.Get("to")
		rng := q.Get("range")

		var start, end time.Time
		now := time.Now().UTC()

		if from != "" && to != "" {
			var err error
			start, err = time.Parse("2006-01-02", from)
			if err != nil {
				utils.WriteJSONError(w, utils.NewValidationError("Invalid 'from' date format", "Use YYYY-MM-DD format (e.g. 2024-01-15)."), http.StatusBadRequest)
				return
			}
			end, err = time.Parse("2006-01-02", to)
			if err != nil {
				utils.WriteJSONError(w, utils.NewValidationError("Invalid 'to' date format", "Use YYYY-MM-DD format (e.g. 2024-01-15)."), http.StatusBadRequest)
				return
			}
		} else {
			if rng == "month" {
				start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				end = start.AddDate(0, 1, -1)
			} else {
				weekday := int(now.Weekday())
				offset := -(weekday - 1)
				if weekday == 0 {
					offset = -6
				}
				start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset)
				end = start.AddDate(0, 0, 6)
			}
		}

		fromStr := formatDate(start)
		toStr := formatDate(end)

		moods, err := utils.GetMoodsForPeriod(cfg, userID, fromStr, toStr)
		if err != nil {
			log.Printf("KPI GetMoodsForPeriod error: %v", err)
			switch err.Code {
			case utils.ErrCodeHasura:
				utils.WriteJSONError(w, err, http.StatusInternalServerError)
			default:
				utils.WriteJSONError(w, err, http.StatusInternalServerError)
			}
			return
		}

		total := 0
		sumScores := 0.0
		counts := map[string]int{}
		dayMap := map[string]struct {
			sum float64
			cnt int
		}{}

		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dayMap[formatDate(d)] = struct {
				sum float64
				cnt int
			}{sum: 0, cnt: 0}
		}

		for _, m := range moods {
			d := ""
			if m.MoodDate != nil {
				d = *m.MoodDate
			} else if m.CreatedAt != nil {
				d = m.CreatedAt.Format("2006-01-02")
			} else {
				continue
			}

			counts[m.Mood]++
			total++

			if m.MoodScore != nil {
				sumScores += float64(*m.MoodScore)
				ds := dayMap[d]
				ds.sum += float64(*m.MoodScore)
				ds.cnt++
				dayMap[d] = ds
			}
		}

		avgScore := 0.0
		if total > 0 {
			scoredCount := 0
			for _, ds := range dayMap {
				scoredCount += ds.cnt
			}
			if scoredCount > 0 {
				avgScore = sumScores / float64(scoredCount)
			}
		}

		countsArr := []models.MoodCount{}
		for mood, c := range counts {
			countsArr = append(countsArr, models.MoodCount{Mood: mood, Count: c})
		}

		ts := []models.TimePoint{}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			ds := dayMap[formatDate(d)]
			avg := 0.0
			if ds.cnt > 0 {
				avg = ds.sum / float64(ds.cnt)
			}
			ts = append(ts, models.TimePoint{
				Date:     formatDate(d),
				AvgScore: avg,
				Count:    ds.cnt,
			})
		}

		writeJSON(w, http.StatusOK, models.KPIResponse{
			Success:      true,
			From:         fromStr,
			To:           toStr,
			AvgScore:     avgScore,
			TotalEntries: total,
			CountsByMood: countsArr,
			TimeSeries:   ts,
		})
	}
}
