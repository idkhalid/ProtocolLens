package domain

import "time"

type Request struct {
	ID        string
	Method    string
	URL       string
	Headers   map[string][]string
	Query     map[string][]string
	Body      []byte
	Timestamp time.Time
}

type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	Duration   time.Duration
}

type Exchange struct {
	Request  Request
	Response Response
}
