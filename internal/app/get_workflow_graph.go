package app

import (
	"context"
	"fmt"

	"protocollens/internal/graph"
)

type GetWorkflowGraph struct {
	store Store
}

func NewGetWorkflowGraph(store Store) *GetWorkflowGraph {
	return &GetWorkflowGraph{store: store}
}

func (uc *GetWorkflowGraph) Execute(ctx context.Context, analysisID string) (graph.Graph, error) {
	if _, err := uc.store.GetAnalysis(ctx, analysisID); err != nil {
		return graph.Graph{}, err
	}
	exchanges, err := uc.store.ListExchanges(ctx, analysisID)
	if err != nil {
		return graph.Graph{}, fmt.Errorf("list exchanges: %w", err)
	}
	dependencies, err := uc.store.ListDependencies(ctx, analysisID)
	if err != nil {
		return graph.Graph{}, fmt.Errorf("list dependencies: %w", err)
	}
	workflow, err := graph.Builder{}.Build(exchanges, dependencies)
	if err != nil {
		return graph.Graph{}, err
	}
	return workflow, nil
}
