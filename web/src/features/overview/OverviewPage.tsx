import { EmptyState, InspectorSection, KeyValue, formatMs } from '../../components/workbench'
import type { Analysis } from '../../types/api'

export function OverviewPage({ analysis }: { analysis?: Analysis }) {
  if (!analysis) return <EmptyState>Import a HAR file to inspect requests, endpoints, sessions, dependencies, and workflow.</EmptyState>

  return (
    <div className="space-y-3">
      <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="Requests" value={analysis.requestCount} />
        <Metric label="Endpoints" value={analysis.endpointCount} />
        <Metric label="Session Artifacts" value={analysis.sessionArtifactCount} />
        <Metric label="Dependencies" value={analysis.dependencyCount} />
      </div>
      <InspectorSection title="Active Analysis">
        <KeyValue label="Analysis ID" value={analysis.id} />
        <KeyValue label="Created" value={new Date(analysis.createdAt).toLocaleString()} />
        <KeyValue label="Import Time" value={formatMs(analysis.importDurationMs)} />
      </InspectorSection>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="metric">
      <div className="text-[11px] uppercase text-[var(--muted)]">{label}</div>
      <div className="mt-1 font-mono text-lg text-[var(--text)]">{value}</div>
    </div>
  )
}