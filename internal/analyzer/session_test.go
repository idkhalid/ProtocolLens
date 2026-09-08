package analyzer

import (
	"testing"
	"time"

	"protocollens/internal/domain"
)

func TestSessionAnalyzerDetectsAndAggregatesArtifacts(t *testing.T) {
	exchanges := []domain.Exchange{
		{
			Request: domain.Request{
				ID:        "r1",
				Timestamp: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
				Headers: map[string][]string{
					"Authorization": {"Bearer <REDACTED>"},
					"Cookie":        {"session_id; locale"},
					"X-Csrf-Token":  {"<REDACTED>"},
					"X-Api-Key":     {"<REDACTED>"},
				},
			},
			Response: domain.Response{
				Headers: map[string][]string{
					"Set-Cookie": {"session_id; Path=/; HttpOnly; Secure; SameSite=Lax"},
				},
			},
		},
		{
			Request: domain.Request{
				ID:        "r2",
				Timestamp: time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
				Headers: map[string][]string{
					"Cookie": {"session_id"},
				},
			},
		},
	}

	got := SessionAnalyzer{}.Analyze(exchanges)
	assertArtifact(t, got, domain.SessionArtifactBearer, "Authorization", "request_header", 1)
	assertArtifact(t, got, domain.SessionArtifactCSRF, "X-Csrf-Token", "request_header", 1)
	assertArtifact(t, got, domain.SessionArtifactAPIKey, "X-Api-Key", "request_header", 1)
	assertArtifact(t, got, domain.SessionArtifactCookie, "session_id", "request_cookie", 2)
	responseCookie := assertArtifact(t, got, domain.SessionArtifactCookie, "session_id", "response_cookie", 1)
	if responseCookie.FirstRequestID != "r1" || !responseCookie.FirstSeenAt.Equal(exchanges[0].Request.Timestamp) {
		t.Fatalf("first seen = %#v", responseCookie)
	}
	if responseCookie.Metadata["HttpOnly"] != "true" || responseCookie.Metadata["SameSite"] != "Lax" {
		t.Fatalf("metadata = %#v", responseCookie.Metadata)
	}
}

func TestSessionAnalyzerIgnoresNonBearerAuthorizationAndMalformedSetCookie(t *testing.T) {
	got := SessionAnalyzer{}.Analyze([]domain.Exchange{{
		Request: domain.Request{
			ID:      "r1",
			Headers: map[string][]string{"Authorization": {"<REDACTED>"}},
		},
		Response: domain.Response{
			Headers: map[string][]string{"Set-Cookie": {""}},
		},
	}})
	if len(got) != 0 {
		t.Fatalf("artifacts = %#v", got)
	}
}

func assertArtifact(t *testing.T, artifacts []domain.SessionArtifact, typ domain.SessionArtifactType, name, source string, occurrences int) domain.SessionArtifact {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Type == typ && artifact.Name == name && artifact.Source == source {
			if artifact.Occurrences != occurrences {
				t.Fatalf("%s/%s occurrences = %d", name, source, artifact.Occurrences)
			}
			return artifact
		}
	}
	t.Fatalf("missing artifact %s %s %s in %#v", typ, name, source, artifacts)
	return domain.SessionArtifact{}
}
