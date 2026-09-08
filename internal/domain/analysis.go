package domain

import "time"

type Analysis struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	RequestCount   int       `json:"requestCount"`
	EndpointCount  int       `json:"endpointCount"`
	ImportDuration int64     `json:"importDurationMs"`
}
