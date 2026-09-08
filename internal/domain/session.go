package domain

import "time"

type SessionArtifactType string

const (
	SessionArtifactCookie SessionArtifactType = "cookie"
	SessionArtifactBearer SessionArtifactType = "bearer"
	SessionArtifactCSRF   SessionArtifactType = "csrf"
	SessionArtifactAPIKey SessionArtifactType = "api_key"
)

type SessionArtifact struct {
	ID             string
	AnalysisID     string
	Type           SessionArtifactType
	Name           string
	Source         string
	FirstRequestID string
	FirstSeenAt    time.Time
	Occurrences    int
	Metadata       map[string]string
}
