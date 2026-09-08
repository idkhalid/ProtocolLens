package domain

import "time"

type Request struct {
	ID        string              `json:"id"`
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Headers   map[string]string   `json:"headers"`
	Query     map[string][]string `json:"query"`
	Body      []byte              `json:"body,omitempty"`
	Timestamp time.Time           `json:"timestamp"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body,omitempty"`
	Duration   time.Duration     `json:"duration"`
}

type Exchange struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}
