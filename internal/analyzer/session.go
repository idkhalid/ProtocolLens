package analyzer

import (
	"sort"
	"strings"

	"protocollens/internal/domain"
)

type SessionAnalyzer struct{}

func (SessionAnalyzer) Analyze(exchanges []domain.Exchange) []domain.SessionArtifact {
	byArtifact := map[string]*domain.SessionArtifact{}
	for _, exchange := range exchanges {
		observeRequestHeaders(byArtifact, exchange)
		observeSetCookies(byArtifact, exchange)
	}

	out := make([]domain.SessionArtifact, 0, len(byArtifact))
	for _, artifact := range byArtifact {
		out = append(out, *artifact)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func observeRequestHeaders(artifacts map[string]*domain.SessionArtifact, exchange domain.Exchange) {
	for _, value := range exchange.Request.Headers["Authorization"] {
		if strings.EqualFold(value, "Bearer <REDACTED>") {
			observe(artifacts, domain.SessionArtifactBearer, "Authorization", "request_header", exchange, nil)
		}
	}

	for name := range exchange.Request.Headers {
		switch strings.ToLower(name) {
		case "x-csrf-token", "x-csrftoken", "x-xsrf-token", "x-xsrftoken", "csrf-token":
			observe(artifacts, domain.SessionArtifactCSRF, name, "request_header", exchange, nil)
		case "x-api-key", "api-key":
			observe(artifacts, domain.SessionArtifactAPIKey, name, "request_header", exchange, nil)
		}
	}

	for _, value := range exchange.Request.Headers["Cookie"] {
		for _, name := range safeNames(value) {
			observe(artifacts, domain.SessionArtifactCookie, name, "request_cookie", exchange, nil)
		}
	}
}

func observeSetCookies(artifacts map[string]*domain.SessionArtifact, exchange domain.Exchange) {
	for _, value := range exchange.Response.Headers["Set-Cookie"] {
		name, metadata := setCookieParts(value)
		if name != "" {
			observe(artifacts, domain.SessionArtifactCookie, name, "response_cookie", exchange, metadata)
		}
	}
}

func observe(artifacts map[string]*domain.SessionArtifact, typ domain.SessionArtifactType, name, source string, exchange domain.Exchange, metadata map[string]string) {
	key := string(typ) + "\x00" + name + "\x00" + source
	artifact := artifacts[key]
	if artifact == nil {
		artifact = &domain.SessionArtifact{
			Type:           typ,
			Name:           name,
			Source:         source,
			FirstRequestID: exchange.Request.ID,
			FirstSeenAt:    exchange.Request.Timestamp,
			Metadata:       metadata,
		}
		artifacts[key] = artifact
	}
	artifact.Occurrences++
}

func safeNames(value string) []string {
	var names []string
	for _, part := range strings.Split(value, ";") {
		name := strings.TrimSpace(part)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func setCookieParts(value string) (string, map[string]string) {
	parts := strings.Split(value, ";")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", nil
	}

	metadata := map[string]string{}
	for _, part := range parts[1:] {
		key, val, hasValue := strings.Cut(strings.TrimSpace(part), "=")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if hasValue {
			metadata[key] = strings.TrimSpace(val)
		} else {
			metadata[key] = "true"
		}
	}
	return name, metadata
}
