import { EmptyState, MethodBadge, StatusCode, formatMs } from '../../components/workbench'
import type { Endpoint } from '../../types/api'

export function EndpointsPage({ endpoints, loading, selectedKey, onSelect }: { endpoints: Endpoint[]; loading: boolean; selectedKey: string; onSelect: (endpoint: Endpoint) => void }) {
  if (loading) return <EmptyState>Loading endpoints.</EmptyState>
  if (endpoints.length === 0) return <EmptyState>No endpoints yet.</EmptyState>

  return (
    <div className="table-shell">
      <table className="data-table">
        <thead>
          <tr>
            <th>Method</th>
            <th>Host</th>
            <th>Path</th>
            <th>Status</th>
            <th className="numeric">Calls</th>
            <th className="numeric">Avg</th>
          </tr>
        </thead>
        <tbody>
          {endpoints.map((endpoint) => {
            const key = endpointKey(endpoint)
            return (
              <tr key={key} className={selectedKey === key ? 'selected' : ''} onClick={() => onSelect(endpoint)}>
                <td><MethodBadge method={endpoint.method} /></td>
                <td className="mono">{endpoint.host}</td>
                <td className="mono strong">{endpoint.path}</td>
                <td>{endpoint.statusCodes.map((code) => <StatusCode key={code} value={code} />)}</td>
                <td className="numeric mono">{endpoint.requestCount}</td>
                <td className="numeric mono">{formatMs(endpoint.averageDurationMs)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export function endpointKey(endpoint?: Endpoint | null) {
  return endpoint ? `${endpoint.method} ${endpoint.host} ${endpoint.path}` : ''
}