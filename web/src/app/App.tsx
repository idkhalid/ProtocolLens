import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getDependencies, getEndpoints, getSessions, getWorkflow, importHAR, listAnalyses } from '../api/analyses'
import { AppShell, DetailPanel, InspectorSection, KeyValue, formatMs } from '../components/workbench'
import type { View } from '../components/workbench'
import { DependenciesPage, edgeLabel, reason } from '../features/dependencies/DependenciesPage'
import { EndpointsPage, endpointKey } from '../features/endpoints/EndpointsPage'
import { OverviewPage } from '../features/overview/OverviewPage'
import { SessionsPage, artifactType } from '../features/sessions/SessionsPage'
import { WorkflowPage } from '../features/workflow/WorkflowPage'
import type { Dependency, Endpoint, SessionArtifact, WorkflowEdge, WorkflowNode } from '../types/api'

type Theme = 'light' | 'dark'
type WorkflowSelection = { type: 'node'; node: WorkflowNode } | { type: 'edge'; edge: WorkflowEdge } | null

export function App() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<View>('overview')
  const [theme, setTheme] = useState<Theme>(() => initialTheme())
  const [selectedAnalysisID, setSelectedAnalysisID] = useState('')
  const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null)
  const [selectedArtifact, setSelectedArtifact] = useState<SessionArtifact | null>(null)
  const [selectedDependency, setSelectedDependency] = useState<Dependency | null>(null)
  const [workflowSelection, setWorkflowSelection] = useState<WorkflowSelection>(null)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('protocol-lens-theme', theme)
  }, [theme])

  const analyses = useQuery({ queryKey: ['analyses'], queryFn: listAnalyses })
  const activeAnalysisID = selectedAnalysisID || analyses.data?.[0]?.id || ''
  const activeAnalysis = analyses.data?.find((analysis) => analysis.id === activeAnalysisID)

  const endpoints = useQuery({ queryKey: ['endpoints', activeAnalysisID], queryFn: () => getEndpoints(activeAnalysisID), enabled: activeAnalysisID.length > 0 })
  const sessions = useQuery({ queryKey: ['sessions', activeAnalysisID], queryFn: () => getSessions(activeAnalysisID), enabled: activeAnalysisID.length > 0 })
  const dependencies = useQuery({ queryKey: ['dependencies', activeAnalysisID], queryFn: () => getDependencies(activeAnalysisID), enabled: activeAnalysisID.length > 0 })
  const workflow = useQuery({ queryKey: ['workflow', activeAnalysisID], queryFn: () => getWorkflow(activeAnalysisID), enabled: activeAnalysisID.length > 0 })

  const clearSelection = () => {
    setSelectedEndpoint(null)
    setSelectedArtifact(null)
    setSelectedDependency(null)
    setWorkflowSelection(null)
  }

  const upload = useMutation({
    mutationFn: importHAR,
    onSuccess: (analysis) => {
      setSelectedAnalysisID(analysis.id)
      clearSelection()
      queryClient.invalidateQueries({ queryKey: ['analyses'] })
      queryClient.invalidateQueries({ queryKey: ['endpoints', analysis.id] })
      queryClient.invalidateQueries({ queryKey: ['sessions', analysis.id] })
      queryClient.invalidateQueries({ queryKey: ['dependencies', analysis.id] })
      queryClient.invalidateQueries({ queryKey: ['workflow', analysis.id] })
    },
  })

  const counts = useMemo(() => ({
    requests: activeAnalysis?.requestCount ?? 0,
    endpoints: activeAnalysis?.endpointCount ?? 0,
    sessions: activeAnalysis?.sessionArtifactCount ?? 0,
    dependencies: activeAnalysis?.dependencyCount ?? 0,
  }), [activeAnalysis])

  return (
    <AppShell
      view={view}
      onViewChange={setView}
      activeAnalysis={activeAnalysis}
      analyses={analyses.data ?? []}
      onAnalysisChange={(id) => { setSelectedAnalysisID(id); clearSelection() }}
      onUpload={(file) => upload.mutate(file)}
      uploading={upload.isPending}
      uploadError={upload.error?.message}
      theme={theme}
      onToggleTheme={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))}
      counts={counts}
      detail={detailFor(view, selectedEndpoint, selectedArtifact, selectedDependency, workflowSelection)}
    >
      {view === 'overview' ? <OverviewPage analysis={activeAnalysis} /> : null}
      {view === 'endpoints' ? <EndpointsPage endpoints={endpoints.data ?? []} loading={endpoints.isLoading} selectedKey={endpointKey(selectedEndpoint)} onSelect={setSelectedEndpoint} /> : null}
      {view === 'sessions' ? <SessionsPage artifacts={sessions.data?.artifacts ?? []} loading={sessions.isLoading} selectedID={selectedArtifact?.id ?? ''} onSelect={setSelectedArtifact} /> : null}
      {view === 'dependencies' ? <DependenciesPage dependencies={dependencies.data?.dependencies ?? []} loading={dependencies.isLoading} selectedID={selectedDependency?.id ?? ''} onSelect={setSelectedDependency} /> : null}
      {view === 'workflow' ? <WorkflowPage graph={workflow.data} loading={workflow.isLoading} onSelectNode={(node) => setWorkflowSelection({ type: 'node', node })} onSelectEdge={(edge) => setWorkflowSelection({ type: 'edge', edge })} /> : null}
    </AppShell>
  )
}

function detailFor(view: View, endpoint: Endpoint | null, artifact: SessionArtifact | null, dependency: Dependency | null, workflowSelection: WorkflowSelection) {
  if (view === 'endpoints' && endpoint) {
    return (
      <DetailPanel title="Endpoint Detail" subtitle={`${endpoint.method} ${endpoint.path}`}>
        <InspectorSection title="Route"><KeyValue label="Method" value={endpoint.method} /><KeyValue label="Host" value={endpoint.host} /><KeyValue label="Path" value={endpoint.path} /></InspectorSection>
        <InspectorSection title="Timing"><KeyValue label="Average" value={formatMs(endpoint.averageDurationMs)} /><KeyValue label="Minimum" value={formatMs(endpoint.minDurationMs)} /><KeyValue label="Maximum" value={formatMs(endpoint.maxDurationMs)} /></InspectorSection>
        <InspectorSection title="Response"><KeyValue label="Status" value={endpoint.statusCodes.join(', ') || 'n/a'} /><KeyValue label="Content Types" value={endpoint.contentTypes.join(', ') || 'unknown'} /></InspectorSection>
      </DetailPanel>
    )
  }
  if (view === 'sessions' && artifact) {
    return (
      <DetailPanel title="Session Artifact" subtitle={`${artifactType(artifact.type)} ${artifact.name}`}>
        <InspectorSection title="Artifact"><KeyValue label="Type" value={artifactType(artifact.type)} /><KeyValue label="Name" value={artifact.name} /><KeyValue label="Source" value={artifact.source} /><KeyValue label="Occurrences" value={artifact.occurrences} /></InspectorSection>
        <InspectorSection title="First Seen"><KeyValue label="Request" value={artifact.firstRequestId} /><KeyValue label="Time" value={new Date(artifact.firstSeenAt).toLocaleString()} /><KeyValue label="Metadata" value={metadata(artifact.metadata)} /></InspectorSection>
      </DetailPanel>
    )
  }
  if (view === 'dependencies' && dependency) {
    return (
      <DetailPanel title="Dependency Evidence" subtitle={`${dependency.sourceRequestId} to ${dependency.targetRequestId}`}>
        <InspectorSection title="Edge"><KeyValue label="Source" value={dependency.sourceRequestId} /><KeyValue label="Target" value={dependency.targetRequestId} /><KeyValue label="Confidence" value={dependency.confidence} /></InspectorSection>
        <InspectorSection title="Mapping"><KeyValue label="Source Path" value={dependency.sourcePath} /><KeyValue label="Target Area" value={edgeLabel(dependency.targetLocation)} /><KeyValue label="Target Path" value={dependency.targetPath} /><KeyValue label="Reason" value={reason(dependency.reason)} /></InspectorSection>
      </DetailPanel>
    )
  }
  if (view === 'workflow' && workflowSelection?.type === 'node') {
    const node = workflowSelection.node
    return (
      <DetailPanel title="Workflow Node" subtitle={`${node.method} ${node.path}`}>
        <InspectorSection title="Request"><KeyValue label="Request ID" value={node.requestId} /><KeyValue label="Order" value={node.order + 1} /><KeyValue label="Host" value={node.host || 'unknown'} /></InspectorSection>
        <InspectorSection title="Response"><KeyValue label="Status" value={node.statusCode || 'n/a'} /><KeyValue label="Duration" value={formatMs(node.durationMs)} /></InspectorSection>
      </DetailPanel>
    )
  }
  if (view === 'workflow' && workflowSelection?.type === 'edge') {
    const edge = workflowSelection.edge
    return (
      <DetailPanel title="Workflow Edge" subtitle={`${edge.source} to ${edge.target}`}>
        <InspectorSection title="Requests"><KeyValue label="Source" value={edge.source} /><KeyValue label="Target" value={edge.target} /><KeyValue label="Confidence" value={edge.confidence} /></InspectorSection>
        <InspectorSection title="Data Flow"><KeyValue label="Source Path" value={edge.sourcePath} /><KeyValue label="Target Area" value={edgeLabel(edge.targetLocation)} /><KeyValue label="Target Path" value={edge.targetPath} /><KeyValue label="Reason" value={reason(edge.reason)} /></InspectorSection>
      </DetailPanel>
    )
  }
  return <DetailPanel title="Inspector" />
}

function metadata(value: Record<string, string> | undefined) {
  if (!value || Object.keys(value).length === 0) return 'none'
  return Object.entries(value).map(([key, item]) => `${key}: ${item}`).join(', ')
}

function initialTheme(): Theme {
  const stored = localStorage.getItem('protocol-lens-theme')
  if (stored === 'light' || stored === 'dark') return stored
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}