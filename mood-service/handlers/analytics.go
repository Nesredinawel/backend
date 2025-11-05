package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"mood-service/middlewares"
	"mood-service/models"
	"mood-service/utils"
)

// helper: format date
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// KPI handler - supports ?range=week | month OR explicit ?from=YYYY-MM-DD&to=YYYY-MM-DD
func MoodKPI(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDVal := r.Context().Value(middlewares.CtxUserID)
		if userIDVal == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := userIDVal.(string)

		// Parse params
		q := r.URL.Query()
		from := q.Get("from")
		to := q.Get("to")
		rng := q.Get("range") // "week" or "month"

		var start, end time.Time
		now := time.Now().UTC()

		if from != "" && to != "" {
			var err error
			start, err = time.Parse("2006-01-02", from)
			if err != nil {
				http.Error(w, "invalid from date", http.StatusBadRequest)
				return
			}
			end, err = time.Parse("2006-01-02", to)
			if err != nil {
				http.Error(w, "invalid to date", http.StatusBadRequest)
				return
			}
		} else {
			// default ranges
			if rng == "month" {
				// start = first day of current month
				start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				// end = last day of current month
				end = start.AddDate(0, 1, -1)
			} else { // default "week"
				// start = beginning of week (Monday)
				weekday := int(now.Weekday())
				// In Go, Sunday==0; treat Monday as start: if Sunday, go back 6 days
				offset := -(weekday - 1)
				if weekday == 0 {
					offset = -6
				}
				start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset)
				end = start.AddDate(0, 0, 6) // 7 days
			}
		}

		fromStr := formatDate(start)
		toStr := formatDate(end)

		moods, err := utils.GetMoodsForPeriod(cfg, userID, fromStr, toStr)
		if err != nil {
			log.Printf("KPI GetMoodsForPeriod error: %v", err)
			http.Error(w, "failed to fetch moods: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Compute aggregates
		total := 0
		sumScores := 0.0
		counts := map[string]int{}
		// time series map by date
		dayMap := map[string]struct {
			sum float64
			cnt int
		}{}

		// initialize dates in timeseries to ensure zero-days present
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
			// avg only from entries that have score
			// we counted sumScores only for those that had score
			// but total could include ones without score; we may want to compute avg from scored entries only
			// compute count of scored entries:
			scoredCount := 0
			for _, ds := range dayMap {
				scoredCount += ds.cnt
			}
			if scoredCount > 0 {
				avgScore = sumScores / float64(scoredCount)
			}
		}

		// prepare counts array
		countsArr := []models.MoodCount{}
		for mood, c := range counts {
			countsArr = append(countsArr, models.MoodCount{Mood: mood, Count: c})
		}

		// prepare timeseries
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

		resp := models.KPIResponse{
			Success:      true,
			From:         fromStr,
			To:           toStr,
			AvgScore:     avgScore,
			TotalEntries: total,
			CountsByMood: countsArr,
			TimeSeries:   ts,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
