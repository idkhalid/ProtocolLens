import { EmptyState } from '../../components/workbench'
import type { SessionArtifact } from '../../types/api'

export function SessionsPage({ artifacts, loading, selectedID, onSelect }: { artifacts: SessionArtifact[]; loading: boolean; selectedID: string; onSelect: (artifact: SessionArtifact) => void }) {
  if (loading) return <EmptyState>Loading session artifacts.</EmptyState>
  if (artifacts.length === 0) return <EmptyState>No session artifacts yet.</EmptyState>

  return (
    <div className="table-shell">
      <table className="data-table">
        <thead>
          <tr>
            <th>Type</th>
            <th>Name</th>
            <th>Source</th>
            <th className="numeric">Occurrences</th>
            <th>First Seen</th>
          </tr>
        </thead>
        <tbody>
          {artifacts.map((artifact) => (
            <tr key={artifact.id} className={selectedID === artifact.id ? 'selected' : ''} onClick={() => onSelect(artifact)}>
              <td><span className="artifact">{artifactType(artifact.type)}</span></td>
              <td className="mono strong">{artifact.name}</td>
              <td className="mono">{artifact.source}</td>
              <td className="numeric mono">{artifact.occurrences}</td>
              <td className="mono">{new Date(artifact.firstSeenAt).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function artifactType(type: SessionArtifact['type']) {
  return type === 'api_key' ? 'API Key' : type.charAt(0).toUpperCase() + type.slice(1)
}