package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"protocollens/internal/analyzer"
	"protocollens/internal/capture/har"
	"protocollens/internal/domain"
	"protocollens/internal/normalize"
)

type Store interface {
	SaveAnalysis(ctx context.Context, analysis domain.Analysis, exchanges []domain.Exchange, endpoints []domain.EndpointSummary, artifacts []domain.SessionArtifact, dependencies []domain.Dependency) error
	GetAnalysis(ctx context.Context, id string) (domain.Analysis, error)
	ListAnalyses(ctx context.Context) ([]domain.Analysis, error)
	ListEndpoints(ctx context.Context, analysisID string) ([]domain.EndpointSummary, error)
	ListExchanges(ctx context.Context, analysisID string) ([]domain.Exchange, error)
	ListSessionArtifacts(ctx context.Context, analysisID string) ([]domain.SessionArtifact, error)
	ListDependencies(ctx context.Context, analysisID string) ([]domain.Dependency, error)
}

type ImportAnalysis struct {
	importer *har.Importer
	store    Store
}

func NewImportAnalysis(importer *har.Importer, store Store) *ImportAnalysis {
	return &ImportAnalysis{importer: importer, store: store}
}

func (uc *ImportAnalysis) Execute(ctx context.Context, input io.Reader) (domain.Analysis, error) {
	start := time.Now()

	entries, err := uc.importer.Import(input)
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("import HAR: %w", err)
	}
	exchanges, err := normalize.FromHAR(entries)
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("normalize HAR: %w", err)
	}
	endpoints := analyzer.EndpointAnalyzer{}.Analyze(exchanges)
	artifacts := analyzer.SessionAnalyzer{}.Analyze(exchanges)
	dependencies := analyzer.DependencyAnalyzer{}.Analyze(exchanges)

	analysis := domain.Analysis{
		ID:              newID(),
		CreatedAt:       time.Now().UTC(),
		RequestCount:    len(exchanges),
		EndpointCount:   len(endpoints),
		SessionCount:    len(artifacts),
		DependencyCount: len(dependencies),
		ImportDuration:  time.Since(start).Milliseconds(),
	}
	for n := range artifacts {
		artifacts[n].ID = newID()
		artifacts[n].AnalysisID = analysis.ID
	}
	for n := range dependencies {
		dependencies[n].ID = newID()
		dependencies[n].AnalysisID = analysis.ID
	}
	if err := uc.store.SaveAnalysis(ctx, analysis, exchanges, endpoints, artifacts, dependencies); err != nil {
		return domain.Analysis{}, fmt.Errorf("save analysis: %w", err)
	}
	return analysis, nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
