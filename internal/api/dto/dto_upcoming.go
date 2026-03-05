package dto

type UpcomingAnalysisRunRequest struct {
	Date string `json:"date"`
	FFID int64  `json:"ff_id"`
}

type UpcomingDay struct {
	Date   string                   `json:"date"`
	Events []map[string]interface{} `json:"events"`
}
