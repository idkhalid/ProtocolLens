package analyzer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"protocollens/internal/domain"
)

const (
	maxJSONDepth               = 12
	maxScalarsPerBody          = 500
	maxDependenciesPerAnalysis = 2000
)

type DependencyAnalyzer struct{}

type jsonScalar struct {
	Path  string
	Value string
}

type targetValue struct {
	Index     int
	RequestID string
	Location  string
	Path      string
	Reason    string
}

func (DependencyAnalyzer) Analyze(exchanges []domain.Exchange) []domain.Dependency {
	ordered := orderedExchanges(exchanges)
	targets := map[string][]targetValue{}
	for index, exchange := range ordered {
		for _, target := range requestTargets(index, exchange) {
			targets[target.Value] = append(targets[target.Value], targetValue{
				Index:     index,
				RequestID: exchange.Request.ID,
				Location:  target.Location,
				Path:      target.Path,
				Reason:    target.Reason,
			})
		}
	}

	seen := map[string]bool{}
	var dependencies []domain.Dependency
	for index, exchange := range ordered {
		candidates := responseCandidates(exchange)
		counts := map[string]int{}
		values := map[string]bool{}
		for _, candidate := range candidates {
			counts[candidate.Value]++
			values[candidate.Value] = true
		}
		for _, candidate := range candidates {
			for _, target := range targets[candidate.Value] {
				if target.Index <= index || target.RequestID == exchange.Request.ID {
					continue
				}
				targetPath := target.Path
				if target.Location == "path" {
					targetPath = redactPathValues(targetPath, values)
				}
				key := exchange.Request.ID + "\x00" + target.RequestID + "\x00" + candidate.Path + "\x00" + target.Location + "\x00" + targetPath
				if seen[key] {
					continue
				}
				seen[key] = true
				dependencies = append(dependencies, domain.Dependency{
					SourceRequestID: exchange.Request.ID,
					TargetRequestID: target.RequestID,
					SourcePath:      candidate.Path,
					TargetLocation:  target.Location,
					TargetPath:      targetPath,
					Confidence:      confidence(candidate.Value, counts[candidate.Value]),
					Reason:          target.Reason,
				})
				if len(dependencies) >= maxDependenciesPerAnalysis {
					return dependencies
				}
			}
		}
	}
	return dependencies
}

func orderedExchanges(exchanges []domain.Exchange) []domain.Exchange {
	ordered := append([]domain.Exchange(nil), exchanges...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i].Request.Timestamp
		right := ordered[j].Request.Timestamp
		if left.IsZero() || right.IsZero() || left.Equal(right) {
			return false
		}
		return left.Before(right)
	})
	return ordered
}

type requestTarget struct {
	Value    string
	Location string
	Path     string
	Reason   string
}

func requestTargets(index int, exchange domain.Exchange) []requestTarget {
	_ = index
	var targets []requestTarget
	if u, err := url.Parse(exchange.Request.URL); err == nil {
		segments := decodedPathSegments(u.EscapedPath())
		for _, segment := range segments {
			if meaningful(segment) {
				targets = append(targets, requestTarget{Value: segment, Location: "path", Path: pathWithValue(u.EscapedPath(), segment), Reason: "response_json_to_path"})
			}
		}
		query := u.Query()
		for _, key := range sortedKeys(query) {
			for _, value := range query[key] {
				if meaningful(value) {
					targets = append(targets, requestTarget{Value: value, Location: "query", Path: key, Reason: "response_json_to_query"})
				}
			}
		}
	}

	for _, scalar := range bodyJSONScalars(exchange.Request.Body, exchange.Request.Headers) {
		targets = append(targets, requestTarget{Value: scalar.Value, Location: "body_json", Path: scalar.Path, Reason: "response_json_to_json_body"})
	}
	form := formValues(exchange.Request.Body, exchange.Request.Headers)
	for _, key := range sortedKeys(form) {
		for _, value := range form[key] {
			if meaningful(value) {
				targets = append(targets, requestTarget{Value: value, Location: "body_form", Path: key, Reason: "response_json_to_form_body"})
			}
		}
	}
	return targets
}

func responseCandidates(exchange domain.Exchange) []jsonScalar {
	return bodyJSONScalars(exchange.Response.Body, exchange.Response.Headers)
}

func bodyJSONScalars(body []byte, headers map[string][]string) []jsonScalar {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	contentType := contentType(headers)
	if contentType != "" && !isJSONContentType(contentType) {
		return nil
	}

	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil
	}
	var scalars []jsonScalar
	walkJSON(value, "$", 0, &scalars)
	return scalars
}

func walkJSON(value any, path string, depth int, scalars *[]jsonScalar) {
	if depth > maxJSONDepth || len(*scalars) >= maxScalarsPerBody {
		return
	}
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			walkJSON(v[key], path+"."+key, depth+1, scalars)
		}
	case []any:
		for i, item := range v {
			walkJSON(item, path+"["+strconv.Itoa(i)+"]", depth+1, scalars)
		}
	case string:
		if meaningful(v) {
			*scalars = append(*scalars, jsonScalar{Path: path, Value: strings.TrimSpace(v)})
		}
	case json.Number:
		if meaningful(v.String()) {
			*scalars = append(*scalars, jsonScalar{Path: path, Value: v.String()})
		}
	}
}

func formValues(body []byte, headers map[string][]string) url.Values {
	if !strings.HasPrefix(contentType(headers), "application/x-www-form-urlencoded") {
		return nil
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil
	}
	return values
}

func decodedPathSegments(rawPath string) []string {
	var segments []string
	for _, part := range strings.Split(rawPath, "/") {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		segments = append(segments, decoded)
	}
	return segments
}

func pathWithValue(rawPath, value string) string {
	parts := strings.Split(rawPath, "/")
	for i, part := range parts {
		decoded, err := url.PathUnescape(part)
		if err == nil && decoded == value {
			parts[i] = "{value}"
		}
	}
	out := strings.Join(parts, "/")
	if out == "" {
		return "/"
	}
	return out
}

func redactPathValues(path string, values map[string]bool) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		decoded, err := url.PathUnescape(part)
		if err == nil && values[decoded] {
			parts[i] = "{value}"
		}
	}
	out := strings.Join(parts, "/")
	if out == "" {
		return "/"
	}
	return out
}

func contentType(headers map[string][]string) string {
	for name, values := range headers {
		if !strings.EqualFold(name, "Content-Type") {
			continue
		}
		for _, value := range values {
			if trimmed := strings.TrimSpace(strings.ToLower(value)); trimmed != "" {
				if before, _, ok := strings.Cut(trimmed, ";"); ok {
					return strings.TrimSpace(before)
				}
				return trimmed
			}
		}
	}
	return ""
}

func isJSONContentType(value string) bool {
	return value == "application/json" || strings.HasSuffix(value, "+json")
}

func meaningful(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return false
	}
	switch strings.ToLower(value) {
	case "true", "false", "null", "get", "post", "put", "patch", "delete", "head", "options", "usd", "eur", "en", "200":
		return false
	}
	return true
}

func confidence(value string, sourceCount int) domain.DependencyConfidence {
	if sourceCount == 1 && (len(value) >= 8 || strings.ContainsAny(value, "_-")) {
		return domain.DependencyConfidenceHigh
	}
	return domain.DependencyConfidenceMedium
}

func sortedKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
