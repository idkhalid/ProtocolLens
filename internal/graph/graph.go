package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"protocollens/internal/domain"
	"protocollens/internal/normalize"
)

type Graph struct {
	Nodes []Node
	Edges []Edge
}

type Node struct {
	ID         string
	RequestID  string
	Order      int
	Method     string
	Host       string
	Path       string
	StatusCode int
	Duration   time.Duration
}

type EdgeType string

const EdgeTypeDataDependency EdgeType = "data_dependency"

type Edge struct {
	ID             string
	Source         string
	Target         string
	Type           EdgeType
	Confidence     domain.DependencyConfidence
	Reason         string
	SourcePath     string
	TargetLocation string
	TargetPath     string
}

type MissingReference struct {
	DependencyID string
	RequestID    string
	Role         string
}

type ReferenceError struct {
	Missing []MissingReference
}

func (e ReferenceError) Error() string {
	return fmt.Sprintf("workflow graph skipped %d dependency edge(s) with missing request references", len(e.Missing))
}

type Builder struct{}

func (Builder) Build(exchanges []domain.Exchange, dependencies []domain.Dependency) (Graph, error) {
	ordered := orderedExchanges(exchanges)
	pathByRequest := redactedPathByRequest(dependencies)
	nodeByRequest := make(map[string]Node, len(ordered))
	nodes := make([]Node, 0, len(ordered))
	for index, exchange := range ordered {
		node := node(exchange, index, pathByRequest[exchange.Request.ID])
		nodes = append(nodes, node)
		nodeByRequest[node.RequestID] = node
	}

	seen := map[string]bool{}
	edges := make([]Edge, 0, len(dependencies))
	var missing []MissingReference
	for _, dependency := range dependencies {
		if _, ok := nodeByRequest[dependency.SourceRequestID]; !ok {
			missing = append(missing, MissingReference{DependencyID: dependency.ID, RequestID: dependency.SourceRequestID, Role: "source"})
			continue
		}
		if _, ok := nodeByRequest[dependency.TargetRequestID]; !ok {
			missing = append(missing, MissingReference{DependencyID: dependency.ID, RequestID: dependency.TargetRequestID, Role: "target"})
			continue
		}

		targetPath := dependency.TargetPath
		if dependency.TargetLocation == "path" && pathByRequest[dependency.TargetRequestID] != "" {
			targetPath = pathByRequest[dependency.TargetRequestID]
		}
		key := edgeKey(dependency, targetPath)
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, Edge{
			ID:             edgeID(dependency, key),
			Source:         dependency.SourceRequestID,
			Target:         dependency.TargetRequestID,
			Type:           EdgeTypeDataDependency,
			Confidence:     dependency.Confidence,
			Reason:         dependency.Reason,
			SourcePath:     dependency.SourcePath,
			TargetLocation: dependency.TargetLocation,
			TargetPath:     targetPath,
		})
	}

	graph := Graph{Nodes: nodes, Edges: edges}
	if len(missing) > 0 {
		return graph, ReferenceError{Missing: missing}
	}
	return graph, nil
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

func node(exchange domain.Exchange, order int, redactedPath string) Node {
	endpoint, err := normalize.EndpointOf(exchange.Request)
	if err != nil {
		endpoint.Method = exchange.Request.Method
		endpoint.Path = "/"
	}
	if redactedPath != "" {
		endpoint.Path = redactedPath
	}
	return Node{
		ID:         exchange.Request.ID,
		RequestID:  exchange.Request.ID,
		Order:      order,
		Method:     endpoint.Method,
		Host:       endpoint.Host,
		Path:       endpoint.Path,
		StatusCode: exchange.Response.StatusCode,
		Duration:   exchange.Response.Duration,
	}
}

func redactedPathByRequest(dependencies []domain.Dependency) map[string]string {
	paths := map[string]string{}
	for _, dependency := range dependencies {
		if dependency.TargetLocation != "path" || dependency.TargetPath == "" {
			continue
		}
		paths[dependency.TargetRequestID] = mergeRedactedPath(paths[dependency.TargetRequestID], dependency.TargetPath)
	}
	return paths
}

func mergeRedactedPath(left, right string) string {
	if left == "" {
		return right
	}
	leftParts := strings.Split(left, "/")
	rightParts := strings.Split(right, "/")
	if len(leftParts) != len(rightParts) {
		return left
	}
	for n := range leftParts {
		if leftParts[n] == "{value}" || rightParts[n] == "{value}" {
			leftParts[n] = "{value}"
		}
	}
	return strings.Join(leftParts, "/")
}

func edgeKey(dependency domain.Dependency, targetPath string) string {
	return strings.Join([]string{
		dependency.SourceRequestID,
		dependency.TargetRequestID,
		dependency.SourcePath,
		dependency.TargetLocation,
		targetPath,
	}, "\x00")
}

func edgeID(dependency domain.Dependency, key string) string {
	if dependency.ID != "" {
		return dependency.ID
	}
	sum := sha256.Sum256([]byte(key))
	return "dep-" + hex.EncodeToString(sum[:8])
}
