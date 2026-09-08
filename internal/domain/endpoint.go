package domain

type Endpoint struct {
	Method string `json:"method"`
	Host   string `json:"host"`
	Path   string `json:"path"`
}

type EndpointSummary struct {
	Method            string   `json:"method"`
	Host              string   `json:"host"`
	Path              string   `json:"path"`
	RequestCount      int      `json:"requestCount"`
	StatusCodes       []int    `json:"statusCodes"`
	AverageDurationMs float64  `json:"averageDurationMs"`
	MinDurationMs     float64  `json:"minDurationMs"`
	MaxDurationMs     float64  `json:"maxDurationMs"`
	ContentTypes      []string `json:"contentTypes"`
}
