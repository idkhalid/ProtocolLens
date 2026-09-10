package api

import (
	"time"

	"protocollens/internal/app"
	"protocollens/internal/domain"
	"protocollens/internal/graph"
	"protocollens/internal/replay"
)

type analysisResponse struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	RequestCount    int       `json:"requestCount"`
	EndpointCount   int       `json:"endpointCount"`
	SessionCount    int       `json:"sessionArtifactCount"`
	DependencyCount int       `json:"dependencyCount"`
	ImportDuration  int64     `json:"importDurationMs"`
}

type endpointResponse struct {
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

type sessionArtifactsResponse struct {
	Artifacts []sessionArtifactResponse `json:"artifacts"`
}

type sessionArtifactResponse struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Name           string            `json:"name"`
	Source         string            `json:"source"`
	FirstRequestID string            `json:"firstRequestId"`
	FirstSeenAt    time.Time         `json:"firstSeenAt"`
	Occurrences    int               `json:"occurrences"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type dependenciesResponse struct {
	Dependencies []dependencyResponse `json:"dependencies"`
}

type dependencyResponse struct {
	ID              string `json:"id"`
	SourceRequestID string `json:"sourceRequestId"`
	TargetRequestID string `json:"targetRequestId"`
	SourcePath      string `json:"sourcePath"`
	TargetLocation  string `json:"targetLocation"`
	TargetPath      string `json:"targetPath"`
	Confidence      string `json:"confidence"`
	Reason          string `json:"reason"`
}

type workflowResponse struct {
	Nodes []workflowNodeResponse `json:"nodes"`
	Edges []workflowEdgeResponse `json:"edges"`
}

type workflowNodeResponse struct {
	ID         string `json:"id"`
	RequestID  string `json:"requestId"`
	Order      int    `json:"order"`
	Method     string `json:"method"`
	Host       string `json:"host"`
	Path       string `json:"path"`
	StatusCode int    `json:"statusCode"`
	DurationMs int64  `json:"durationMs"`
}

type workflowEdgeResponse struct {
	ID             string `json:"id"`
	Source         string `json:"source"`
	Target         string `json:"target"`
	Type           string `json:"type"`
	Confidence     string `json:"confidence"`
	Reason         string `json:"reason"`
	SourcePath     string `json:"sourcePath"`
	TargetLocation string `json:"targetLocation"`
	TargetPath     string `json:"targetPath"`
}

type replayRequest struct {
	AnalysisID      string              `json:"analysisId,omitempty"`
	RequestID       string              `json:"requestId,omitempty"`
	Method          string              `json:"method"`
	URL             string              `json:"url"`
	Headers         map[string][]string `json:"headers"`
	Body            string              `json:"body"`
	FollowRedirects bool                `json:"followRedirects"`
}

type replayResponse struct {
	StatusCode    int                 `json:"statusCode"`
	DurationMs    int64               `json:"durationMs"`
	FinalURL      string              `json:"finalUrl"`
	Headers       map[string][]string `json:"headers"`
	Body          string              `json:"body"`
	BodyAvailable bool                `json:"bodyAvailable"`
	ContentLength int64               `json:"contentLength"`
	ContentType   string              `json:"contentType"`
	Truncated     bool                `json:"truncated"`
}

type benchmarkRequest struct {
	AnalysisID                 string              `json:"analysisId"`
	RequestID                  string              `json:"requestId"`
	Method                     string              `json:"method"`
	URL                        string              `json:"url"`
	Headers                    map[string][]string `json:"headers"`
	Body                       string              `json:"body"`
	FollowRedirects            bool                `json:"followRedirects"`
	Runs                       int                 `json:"runs"`
	AllowRepeatedNonIdempotent bool                `json:"allowRepeatedNonIdempotent"`
}

type benchmarkResponse struct {
	Method     string                      `json:"method"`
	RequestID  string                      `json:"requestId"`
	Browser    benchmarkBrowserResponse    `json:"browser"`
	HTTP       benchmarkHTTPResponse       `json:"http"`
	Comparison benchmarkComparisonResponse `json:"comparison"`
}

type benchmarkBrowserResponse struct {
	Available  bool    `json:"available"`
	DurationMs float64 `json:"durationMs,omitempty"`
}

type benchmarkHTTPResponse struct {
	Runs             []benchmarkRunResponse `json:"runs"`
	MinMs            float64                `json:"minMs"`
	MedianMs         float64                `json:"medianMs"`
	MeanMs           float64                `json:"meanMs"`
	MaxMs            float64                `json:"maxMs"`
	ConsistentStatus bool                   `json:"consistentStatus"`
}

type benchmarkRunResponse struct {
	DurationMs float64 `json:"durationMs"`
	StatusCode int     `json:"statusCode"`
	FinalURL   string  `json:"finalUrl"`
	Truncated  bool    `json:"truncated"`
}

type benchmarkComparisonResponse struct {
	Available        bool    `json:"available"`
	MedianSpeedup    float64 `json:"medianSpeedup,omitempty"`
	ReductionPercent float64 `json:"reductionPercent,omitempty"`
}

type generateRequest struct {
	AnalysisID string `json:"analysisId"`
	RequestID  string `json:"requestId"`
	Target     string `json:"target"`
}

func toAnalysisResponse(analysis domain.Analysis) analysisResponse {
	return analysisResponse{ID: analysis.ID, CreatedAt: analysis.CreatedAt, RequestCount: analysis.RequestCount, EndpointCount: analysis.EndpointCount, SessionCount: analysis.SessionCount, DependencyCount: analysis.DependencyCount, ImportDuration: analysis.ImportDuration}
}

func toAnalysisResponses(analyses []domain.Analysis) []analysisResponse {
	out := make([]analysisResponse, len(analyses))
	for n, analysis := range analyses {
		out[n] = toAnalysisResponse(analysis)
	}
	return out
}

func toSessionArtifactsResponse(artifacts []domain.SessionArtifact) sessionArtifactsResponse {
	out := make([]sessionArtifactResponse, len(artifacts))
	for n, artifact := range artifacts {
		out[n] = sessionArtifactResponse{ID: artifact.ID, Type: string(artifact.Type), Name: artifact.Name, Source: artifact.Source, FirstRequestID: artifact.FirstRequestID, FirstSeenAt: artifact.FirstSeenAt, Occurrences: artifact.Occurrences, Metadata: artifact.Metadata}
	}
	return sessionArtifactsResponse{Artifacts: out}
}

func toDependenciesResponse(dependencies []domain.Dependency) dependenciesResponse {
	out := make([]dependencyResponse, len(dependencies))
	for n, dependency := range dependencies {
		out[n] = dependencyResponse{ID: dependency.ID, SourceRequestID: dependency.SourceRequestID, TargetRequestID: dependency.TargetRequestID, SourcePath: dependency.SourcePath, TargetLocation: dependency.TargetLocation, TargetPath: dependency.TargetPath, Confidence: string(dependency.Confidence), Reason: dependency.Reason}
	}
	return dependenciesResponse{Dependencies: out}
}

func toWorkflowResponse(workflow graph.Graph) workflowResponse {
	nodes := make([]workflowNodeResponse, len(workflow.Nodes))
	for n, node := range workflow.Nodes {
		nodes[n] = workflowNodeResponse{ID: node.ID, RequestID: node.RequestID, Order: node.Order, Method: node.Method, Host: node.Host, Path: node.Path, StatusCode: node.StatusCode, DurationMs: node.Duration.Milliseconds()}
	}
	edges := make([]workflowEdgeResponse, len(workflow.Edges))
	for n, edge := range workflow.Edges {
		edges[n] = workflowEdgeResponse{ID: edge.ID, Source: edge.Source, Target: edge.Target, Type: string(edge.Type), Confidence: string(edge.Confidence), Reason: edge.Reason, SourcePath: edge.SourcePath, TargetLocation: edge.TargetLocation, TargetPath: edge.TargetPath}
	}
	return workflowResponse{Nodes: nodes, Edges: edges}
}

func toEndpointResponses(endpoints []domain.EndpointSummary) []endpointResponse {
	out := make([]endpointResponse, len(endpoints))
	for n, endpoint := range endpoints {
		out[n] = endpointResponse{Method: endpoint.Method, Host: endpoint.Host, Path: endpoint.Path, RequestCount: endpoint.RequestCount, StatusCodes: endpoint.StatusCodes, AverageDurationMs: endpoint.AverageDurationMs, MinDurationMs: endpoint.MinDurationMs, MaxDurationMs: endpoint.MaxDurationMs, ContentTypes: endpoint.ContentTypes}
	}
	return out
}

func toReplayResponse(result replay.Result) replayResponse {
	return replayResponse{StatusCode: result.StatusCode, DurationMs: result.Duration.Milliseconds(), FinalURL: result.FinalURL, Headers: result.Headers, Body: string(result.Body), BodyAvailable: result.BodyAvailable, ContentLength: result.ContentLength, ContentType: result.ContentType, Truncated: result.Truncated}
}

func toBenchmarkResponse(result app.BenchmarkResult) benchmarkResponse {
	runs := make([]benchmarkRunResponse, len(result.HTTP.Runs))
	for n, run := range result.HTTP.Runs {
		runs[n] = benchmarkRunResponse{DurationMs: durationMs(run.Duration), StatusCode: run.Status, FinalURL: run.FinalURL, Truncated: run.Truncated}
	}
	return benchmarkResponse{
		Method:    result.Method,
		RequestID: result.RequestID,
		Browser:   benchmarkBrowserResponse{Available: result.Browser.Available, DurationMs: durationMs(result.Browser.Duration)},
		HTTP: benchmarkHTTPResponse{
			Runs:             runs,
			MinMs:            durationMs(result.HTTP.Min),
			MedianMs:         durationMs(result.HTTP.Median),
			MeanMs:           durationMs(result.HTTP.Mean),
			MaxMs:            durationMs(result.HTTP.Max),
			ConsistentStatus: result.HTTP.ConsistentStatus,
		},
		Comparison: benchmarkComparisonResponse{Available: result.Comparison.Available, MedianSpeedup: result.Comparison.MedianSpeedup, ReductionPercent: result.Comparison.ReductionPercent},
	}
}

func durationMs(value time.Duration) float64 {
	return float64(value) / float64(time.Millisecond)
}
