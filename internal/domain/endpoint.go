package domain

type Endpoint struct {
	Method string
	Host   string
	Path   string
}

type EndpointSummary struct {
	Method            string
	Host              string
	Path              string
	RequestCount      int
	StatusCodes       []int
	AverageDurationMs float64
	MinDurationMs     float64
	MaxDurationMs     float64
	ContentTypes      []string
}
