package domain

import "time"

type Analysis struct {
	ID             string
	CreatedAt      time.Time
	RequestCount   int
	EndpointCount  int
	ImportDuration int64
}
