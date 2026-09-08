package domain

type DependencyConfidence string

const (
	DependencyConfidenceHigh   DependencyConfidence = "high"
	DependencyConfidenceMedium DependencyConfidence = "medium"
)

type Dependency struct {
	ID              string
	AnalysisID      string
	SourceRequestID string
	TargetRequestID string
	SourcePath      string
	TargetLocation  string
	TargetPath      string
	Confidence      DependencyConfidence
	Reason          string
}
