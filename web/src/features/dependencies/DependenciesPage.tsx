import { EmptyState } from '../../components/workbench'
import type { Dependency } from '../../types/api'

export function DependenciesPage({ dependencies, loading, selectedID, onSelect }: { dependencies: Dependency[]; loading: boolean; selectedID: string; onSelect: (dependency: Dependency) => void }) {
  if (loading) return <EmptyState>Loading dependencies.</EmptyState>
  if (dependencies.length === 0) return <EmptyState>No dependencies yet.</EmptyState>

  return (
    <div className="table-shell">
      <table className="data-table">
        <thead>
          <tr>
            <th>Source</th>
            <th>Source Field</th>
            <th>Target</th>
            <th>Target Location</th>
            <th>Target Field</th>
            <th>Confidence</th>
          </tr>
        </thead>
        <tbody>
          {dependencies.map((dependency) => (
            <tr key={dependency.id} className={selectedID === dependency.id ? 'selected' : ''} onClick={() => onSelect(dependency)}>
              <td className="mono">{dependency.sourceRequestId}</td>
              <td className="mono strong">{dependency.sourcePath}</td>
              <td className="mono">{dependency.targetRequestId}</td>
              <td><span className="artifact">{edgeLabel(dependency.targetLocation)}</span></td>
              <td className="mono strong">{dependency.targetPath}</td>
              <td><span className={`confidence ${dependency.confidence}`}>{dependency.confidence}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function edgeLabel(value: string) {
  switch (value) {
    case 'body_json':
      return 'JSON body'
    case 'body_form':
      return 'form body'
    default:
      return value
  }
}

export function reason(value: string) {
  switch (value) {
    case 'response_json_to_path':
      return 'response JSON to path'
    case 'response_json_to_query':
      return 'response JSON to query'
    case 'response_json_to_json_body':
      return 'response JSON to JSON body'
    case 'response_json_to_form_body':
      return 'response JSON to form body'
    default:
      return value
  }
}