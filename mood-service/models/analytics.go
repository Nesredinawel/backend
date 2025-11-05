package models

type MoodCount struct {
	Mood  string `json:"mood"`
	Count int    `json:"count"`
}

type TimePoint struct {
	Date     string  `json:"date"`      // yyyy-mm-dd
	AvgScore float64 `json:"avg_score"` // average score for that day
	Count    int     `json:"count"`     // number of entries that day
}

type KPIResponse struct {
	Success      bool        `json:"success"`
	From         string      `json:"from"` // yyyy-mm-dd
	To           string      `json:"to"`
	AvgScore     float64     `json:"avg_score"`
	TotalEntries int         `json:"total_entries"`
	CountsByMood []MoodCount `json:"counts_by_mood"`
	TimeSeries   []TimePoint `json:"time_series"` // daily points
}
