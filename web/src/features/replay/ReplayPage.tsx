import { useEffect, useMemo, useState } from 'react'
import { Send } from 'lucide-react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { getReplayTemplate, getWorkflow, sendReplay } from '../../api/analyses'
import { EmptyState, MethodBadge, StatusCode, formatMs } from '../../components/workbench'
import type { ReplayRequest, ReplayResponse, ReplayTemplate, WorkflowNode } from '../../types/api'

const methods = ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS']

export function ReplayPage({ analysisID }: { analysisID: string }) {
  const [requestID, setRequestID] = useState('')
  const [method, setMethod] = useState('GET')
  const [url, setURL] = useState('')
  const [query, setQuery] = useState('')
  const [headers, setHeaders] = useState('')
  const [body, setBody] = useState('')
  const [followRedirects, setFollowRedirects] = useState(false)
  const [result, setResult] = useState<ReplayResponse | null>(null)

  const workflow = useQuery({ queryKey: ['workflow', analysisID], queryFn: () => getWorkflow(analysisID), enabled: analysisID.length > 0 })
  const template = useQuery({ queryKey: ['replay-template', analysisID, requestID], queryFn: () => getReplayTemplate(analysisID, requestID), enabled: analysisID.length > 0 && requestID.length > 0 })
  const replay = useMutation({ mutationFn: sendReplay, onSuccess: setResult })

  useEffect(() => {
    if (!requestID && workflow.data?.nodes[0]) setRequestID(workflow.data.nodes[0].requestId)
  }, [requestID, workflow.data])

  useEffect(() => {
    if (!template.data) return
    setMethod(template.data.method)
    setURL(template.data.url)
    setQuery(queryText(template.data))
    setHeaders(JSON.stringify(template.data.headers, null, 2))
    setBody(template.data.bodyAvailable ? template.data.body : '')
    setResult(null)
  }, [template.data])

  const nodes = workflow.data?.nodes ?? []
  const selectedNode = useMemo(() => nodes.find((node) => node.requestId === requestID), [nodes, requestID])
  if (!analysisID) return <EmptyState>Import a HAR file before preparing replay requests.</EmptyState>

  const request = (): ReplayRequest => ({ analysisId: analysisID, requestId: requestID, method, url: urlWithQuery(url, query), headers: parseHeaders(headers), body, followRedirects })

  return (
    <div className="grid min-w-0 gap-3 xl:grid-cols-[minmax(0,1fr)_420px]">
      <section className="replay-panel">
        <div className="replay-panel-header">Request Editor</div>
        <label className="field-label">Captured request</label>
        <select className="field-input" value={requestID} onChange={(event) => setRequestID(event.target.value)}>
          {nodes.map((node) => <option key={node.id} value={node.requestId}>{requestLabel(node)}</option>)}
        </select>
        {selectedNode ? <div className="mt-2 flex items-center gap-2 text-xs text-[var(--muted)]"><MethodBadge method={selectedNode.method} /><span className="mono truncate">{selectedNode.host}{selectedNode.path}</span></div> : null}
        {template.data?.requiresReview ? <div className="notice mt-3">Template contains redacted placeholders. Replace required values before sending.</div> : null}
        {template.data && !template.data.bodyAvailable ? <div className="notice muted mt-3">Captured body was omitted: {template.data.bodyReason}. Provide a new body if needed.</div> : null}

        <div className="mt-3 grid gap-2 sm:grid-cols-[110px_minmax(0,1fr)]">
          <select className="field-input" value={method} onChange={(event) => setMethod(event.target.value)}>{methods.map((item) => <option key={item}>{item}</option>)}</select>
          <input className="field-input mono" value={url} onChange={(event) => setURL(event.target.value)} placeholder="https://example.com/api" />
        </div>
        <label className="field-label">Query</label>
        <textarea className="field-textarea mono h-20" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="name=value" />
        <label className="field-label">Headers JSON</label>
        <textarea className="field-textarea mono h-32" value={headers} onChange={(event) => setHeaders(event.target.value)} />
        <label className="field-label">Body</label>
        <textarea className="field-textarea mono h-40" value={body} onChange={(event) => setBody(event.target.value)} />
        <label className="mt-3 flex items-center gap-2 text-xs text-[var(--text)]"><input type="checkbox" checked={followRedirects} onChange={(event) => setFollowRedirects(event.target.checked)} /> Follow redirects</label>
        <button className="mt-3 inline-flex h-8 items-center gap-2 rounded border border-[var(--accent)] bg-[var(--accent)] px-3 text-xs font-medium text-white disabled:opacity-50" type="button" disabled={replay.isPending || template.isLoading} onClick={() => replay.mutate(request())}>
          <Send className="h-3.5 w-3.5" /> Send Request
        </button>
        {replay.error ? <div className="mt-3 border border-[var(--danger-border)] bg-[var(--danger-bg)] p-2 text-xs text-[var(--danger)]">{replay.error.message}</div> : null}
      </section>
      <ResponseInspector result={result} loading={replay.isPending} />
    </div>
  )
}

function ResponseInspector({ result, loading }: { result: ReplayResponse | null; loading: boolean }) {
  if (loading) return <section className="replay-panel"><div className="replay-panel-header">Response Inspector</div><EmptyState>Sending request.</EmptyState></section>
  if (!result) return <section className="replay-panel"><div className="replay-panel-header">Response Inspector</div><EmptyState>No replay response yet.</EmptyState></section>
  const body = prettyBody(result)
  return (
    <section className="replay-panel">
      <div className="replay-panel-header">Response Inspector</div>
      <div className="grid grid-cols-3 gap-2 text-xs"><div><span className="text-[var(--muted)]">Status</span><div><StatusCode value={result.statusCode} /></div></div><div><span className="text-[var(--muted)]">Duration</span><div className="mono">{formatMs(result.durationMs)}</div></div><div><span className="text-[var(--muted)]">Truncated</span><div className="mono">{String(result.truncated)}</div></div></div>
      <div className="mt-3 text-xs"><div className="text-[var(--muted)]">Final URL</div><div className="mono break-words">{result.finalUrl}</div></div>
      <div className="mt-3 table-shell"><table className="data-table replay-headers"><tbody>{Object.entries(result.headers).map(([name, values]) => <tr key={name}><td className="mono strong">{name}</td><td className="mono">{values.join(', ')}</td></tr>)}</tbody></table></div>
      <pre className="response-body">{result.bodyAvailable ? body : `Body unavailable (${result.contentType || 'unknown content type'})`}</pre>
    </section>
  )
}

function requestLabel(node: WorkflowNode) {
  return `${node.requestId} ${node.method} ${node.path}`
}

function queryText(template: ReplayTemplate) {
  return Object.entries(template.query).flatMap(([name, values]) => values.map((value) => `${name}=${value}`)).join('\n')
}

function parseHeaders(value: string) {
  try { return JSON.parse(value) as Record<string, string[]> } catch { return {} }
}

function urlWithQuery(rawURL: string, query: string) {
  try {
    const parsed = new URL(rawURL)
    parsed.search = ''
    for (const line of query.split('\n')) {
      const trimmed = line.trim()
      if (!trimmed) continue
      const [name, ...rest] = trimmed.split('=')
      parsed.searchParams.append(name, rest.join('='))
    }
    return parsed.toString()
  } catch {
    return rawURL
  }
}

function prettyBody(result: ReplayResponse) {
  if (!result.contentType.toLowerCase().includes('json')) return result.body
  try { return JSON.stringify(JSON.parse(result.body), null, 2) } catch { return result.body }
}