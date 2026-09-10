import { useEffect, useMemo, useRef, useState } from 'react'
import { Activity, Play, Send, Square } from 'lucide-react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { generateClient, getReplayTemplate, getWorkflow, runBenchmark, sendReplay } from '../../api/analyses'
import { EmptyState, MethodBadge, StatusCode, formatMs } from '../../components/workbench'
import type { BenchmarkRequest, BenchmarkResponse, ReplayRequest, ReplayResponse, ReplayTemplate, WorkflowNode } from '../../types/api'

const methods = ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS']
const safeRepeatMethods = new Set(['GET', 'HEAD', 'OPTIONS'])

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
      <div className="flex flex-col gap-3 min-w-0">
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
        <BenchmarkPanel analysisID={analysisID} requestID={requestID} replayRequest={request()} />
        <GenerateClient analysisID={analysisID} requestID={requestID} />
      </div>
      <ResponseInspector result={result} loading={replay.isPending} />
    </div>
  )
}

function BenchmarkPanel({ analysisID, requestID, replayRequest }: { analysisID: string; requestID: string; replayRequest: ReplayRequest }) {
  const [runs, setRuns] = useState(3)
  const [allowRepeat, setAllowRepeat] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<BenchmarkResponse | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const signature = JSON.stringify({ requestID, replayRequest })
  const repeatedRisk = !safeRepeatMethods.has(replayRequest.method.toUpperCase()) && runs > 1

  useEffect(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setLoading(false)
    setError('')
    setResult(null)
    setAllowRepeat(false)
  }, [signature])
  useEffect(() => () => abortRef.current?.abort(), [])

  async function start() {
    const controller = new AbortController()
    abortRef.current = controller
    setLoading(true)
    setError('')
    setResult(null)
    try {
      const payload: BenchmarkRequest = { ...replayRequest, analysisId: analysisID, requestId: requestID, runs, allowRepeatedNonIdempotent: allowRepeat }
      const next = await runBenchmark(payload, controller.signal)
      if (abortRef.current === controller) setResult(next)
    } catch (err) {
      if ((err as Error).name !== 'AbortError' && abortRef.current === controller) setError((err as Error).message)
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null
        setLoading(false)
      }
    }
  }

  return (
    <section className="replay-panel">
      <div className="mb-3 flex items-center gap-2"><Activity className="h-4 w-4 text-[var(--accent)]" /><div className="replay-panel-header !mb-0">Browser vs HTTP</div></div>
      <p className="text-xs text-[var(--muted)]">Compares observed browser request timing with bounded direct HTTP replay.</p>
      <label className="field-label">Runs</label>
      <input className="field-input mono max-w-24" type="number" min="1" max="5" value={runs} onChange={(event) => { setRuns(Number(event.target.value)); setResult(null); setAllowRepeat(false) }} disabled={loading} />
      {repeatedRisk ? (
        <label className="mt-3 flex items-start gap-2 text-xs text-[var(--text)]">
          <input className="mt-0.5" type="checkbox" checked={allowRepeat} onChange={(event) => setAllowRepeat(event.target.checked)} disabled={loading} />
          <span>This request will be sent {runs} times and may change server state.</span>
        </label>
      ) : null}
      <div className="mt-3 flex flex-wrap gap-2">
        <button className="inline-flex h-8 items-center gap-2 rounded border border-[var(--accent)] bg-[var(--accent)] px-3 text-xs font-medium text-white disabled:opacity-50" type="button" disabled={loading || !requestID || (repeatedRisk && !allowRepeat)} onClick={start}>
          <Play className="h-3.5 w-3.5" /> Run Benchmark
        </button>
        {loading ? <button className="inline-flex h-8 items-center gap-2 rounded border border-[var(--border)] px-3 text-xs" type="button" onClick={() => abortRef.current?.abort()}><Square className="h-3.5 w-3.5" /> Cancel</button> : null}
      </div>
      {loading ? <p className="mt-3 text-xs text-[var(--muted)]">Running sequential HTTP replay.</p> : null}
      {error ? <div className="mt-3 border border-[var(--danger-border)] bg-[var(--danger-bg)] p-2 text-xs text-[var(--danger)]">{error}</div> : null}
      {result ? <BenchmarkResultView result={result} /> : null}
    </section>
  )
}

function BenchmarkResultView({ result }: { result: BenchmarkResponse }) {
  return (
    <div className="mt-3 space-y-3 text-xs">
      <div className="grid gap-2 sm:grid-cols-2">
        <Metric label="Method" value={result.method} />
        <Metric label="Request" value={result.requestId} />
        <Metric label="Observed browser" value={result.browser.available ? formatMs(result.browser.durationMs ?? 0) : 'Unavailable'} />
        <Metric label="HTTP median" value={formatMs(result.http.medianMs)} />
        <Metric label="Speedup" value={result.comparison.available ? `${(result.comparison.medianSpeedup ?? 0).toFixed(2)}x` : 'Unavailable'} />
        <Metric label="Latency reduction" value={result.comparison.available ? `${(result.comparison.reductionPercent ?? 0).toFixed(1)}%` : 'Unavailable'} />
        <Metric label="Min / Mean / Max" value={`${formatMs(result.http.minMs)} / ${formatMs(result.http.meanMs)} / ${formatMs(result.http.maxMs)}`} />
        <Metric label="Status" value={result.http.consistentStatus ? `${result.http.runs[0]?.statusCode ?? 'n/a'} consistent` : 'Mixed'} />
      </div>
      <div className="table-shell">
        <table className="data-table">
          <thead><tr><th>Run</th><th>Duration</th><th>Status</th></tr></thead>
          <tbody>{result.http.runs.map((run, index) => <tr key={index}><td className="mono">#{index + 1}</td><td className="mono">{formatMs(run.durationMs)}</td><td><StatusCode value={run.statusCode} /></td></tr>)}</tbody>
        </table>
      </div>
      <p className="text-[11px] text-[var(--muted)]">Timing reflects observed HAR and current replay transport conditions.</p>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return <div className="border border-[var(--border)] bg-[var(--workspace)] p-2"><div className="text-[var(--muted)]">{label}</div><div className="mono mt-1 break-words">{value}</div></div>
}

function GenerateClient({ analysisID, requestID }: { analysisID: string; requestID: string }) {
  const [target, setTarget] = useState<'curl' | 'python' | 'go'>('curl')
  const [copied, setCopied] = useState(false)
  const generator = useQuery({
    queryKey: ['generate', analysisID, requestID, target],
    queryFn: () => generateClient(analysisID, requestID, target),
    enabled: analysisID.length > 0 && requestID.length > 0,
  })

  useEffect(() => setCopied(false), [generator.data?.code])

  if (!analysisID || !requestID) return null

  return (
    <section className="replay-panel">
      <div className="flex items-center justify-between mb-3">
        <div className="replay-panel-header !mb-0">Generate Client</div>
        <div className="flex gap-1">
          {(['curl', 'python', 'go'] as const).map((t) => (
            <button key={t} onClick={() => setTarget(t)} className={`px-2 py-1 text-xs font-medium rounded ${target === t ? 'bg-[var(--accent)] text-white' : 'bg-[var(--muted-bg)] text-[var(--muted)] hover:text-[var(--text)]'}`}>
              {t === 'curl' ? 'cURL' : t === 'python' ? 'Python' : 'Go'}
            </button>
          ))}
        </div>
      </div>
      {generator.isError ? (
        <div className="notice danger">{generator.error.message}</div>
      ) : generator.isLoading ? (
        <div className="text-xs text-[var(--muted)]">Generating...</div>
      ) : generator.data ? (
        <div>
          {generator.data.environmentVariables && generator.data.environmentVariables.length > 0 && (
            <div className="mb-2 text-xs">
              <span className="text-[var(--muted)]">Required environment variables:</span>
              <div className="flex flex-wrap gap-1 mt-1">
                {generator.data.environmentVariables.map((env) => <span key={env} className="px-1.5 py-0.5 rounded bg-[var(--muted-bg)] border border-[var(--border)] mono">{env}</span>)}
              </div>
            </div>
          )}
          <div className="relative group">
            <pre className="p-3 bg-[var(--surface)] border border-[var(--border)] rounded text-xs mono overflow-x-auto whitespace-pre">{generator.data.code}</pre>
            <button onClick={() => { navigator.clipboard.writeText(generator.data.code); setCopied(true); setTimeout(() => setCopied(false), 2000) }} className="absolute top-2 right-2 px-2 py-1 text-xs rounded bg-[var(--accent)] text-white opacity-0 group-hover:opacity-100 transition-opacity">
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>
      ) : null}
    </section>
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