import { useEffect, useRef, useState } from 'react'
import { ExternalLink, Globe, Play, Square } from 'lucide-react'
import { captureURL } from '../../api/analyses'
import { EmptyState } from '../../components/workbench'
import type { Analysis, Capabilities } from '../../types/api'

export function CapturePage({ capabilities, onCaptured, onOpen }: { capabilities?: Capabilities; onCaptured: (analysis: Analysis) => void; onOpen: (analysisID: string) => void }) {
  const [url, setURL] = useState('https://example.com')
  const [duration, setDuration] = useState(5)
  const [headed, setHeaded] = useState(false)
  const [running, setRunning] = useState(false)
  const [elapsed, setElapsed] = useState(0)
  const [error, setError] = useState('')
  const [result, setResult] = useState<Analysis | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => () => abortRef.current?.abort(), [])
  useEffect(() => {
    if (!running) return
    const started = Date.now()
    const id = window.setInterval(() => setElapsed(Math.floor((Date.now() - started) / 1000)), 500)
    return () => window.clearInterval(id)
  }, [running])

  const localCapture = capabilities?.local_capture
  const enabled = !!localCapture?.enabled
  const ready = enabled && localCapture.adapter_built && localCapture.node_available

  async function start() {
    setError('')
    setResult(null)
    if (!/^https?:\/\//i.test(url.trim())) {
      setError('Target URL must start with http:// or https://')
      return
    }
    const controller = new AbortController()
    abortRef.current = controller
    setRunning(true)
    setElapsed(0)
    try {
      const analysis = await captureURL({ url: url.trim(), duration_seconds: duration, headed }, controller.signal)
      setResult(analysis)
      onCaptured(analysis)
    } catch (err) {
      if ((err as Error).name !== 'AbortError') setError((err as Error).message)
    } finally {
      abortRef.current = null
      setRunning(false)
    }
  }

  if (!enabled) {
    return <EmptyState>Local browser capture is disabled on this server.</EmptyState>
  }

  return (
    <div className="grid min-h-full gap-3 xl:grid-cols-[minmax(0,1fr)_320px]">
      <section className="min-w-0 border border-[var(--border)] bg-[var(--panel)]">
        <div className="flex items-center gap-2 border-b border-[var(--border)] px-3 py-2">
          <Globe className="h-4 w-4 text-[var(--accent)]" />
          <h1 className="text-sm font-medium">Capture</h1>
        </div>
        <div className="space-y-3 p-3">
          <label className="block text-xs">
            <span className="mb-1 block text-[var(--muted)]">Target URL</span>
            <input className="h-8 w-full border border-[var(--border)] bg-[var(--workspace)] px-2 font-mono text-xs text-[var(--text)] outline-none focus:border-[var(--accent)]" value={url} onChange={(event) => setURL(event.target.value)} disabled={running} />
          </label>
          <div className="grid gap-3 sm:grid-cols-[160px_minmax(0,1fr)]">
            <label className="block text-xs">
              <span className="mb-1 block text-[var(--muted)]">Capture duration</span>
              <input className="h-8 w-full border border-[var(--border)] bg-[var(--workspace)] px-2 font-mono text-xs text-[var(--text)] outline-none focus:border-[var(--accent)]" type="number" min="1" max="60" value={duration} onChange={(event) => setDuration(Number(event.target.value))} disabled={running} />
            </label>
            <label className="flex h-8 items-end gap-2 text-xs sm:mt-5">
              <input type="checkbox" checked={headed} onChange={(event) => setHeaded(event.target.checked)} disabled={running} />
              <span>Headed Chromium</span>
            </label>
          </div>
          <div className="flex flex-wrap gap-2">
            <button className="inline-flex h-8 items-center gap-1.5 border border-[var(--accent)] bg-[var(--accent)] px-3 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-60" type="button" onClick={start} disabled={running || !ready}>
              <Play className="h-3.5 w-3.5" /> Capture & Analyze
            </button>
            {running ? (
              <button className="inline-flex h-8 items-center gap-1.5 border border-[var(--border)] bg-[var(--workspace)] px-3 text-xs" type="button" onClick={() => abortRef.current?.abort()}>
                <Square className="h-3.5 w-3.5" /> Cancel
              </button>
            ) : null}
          </div>
          {running ? <p className="text-xs text-[var(--muted)]">Capturing browser traffic... {elapsed}s</p> : null}
          {!ready ? <p className="text-xs text-[var(--danger)]">Capture dependencies are unavailable.</p> : null}
          {error ? <p className="border border-[var(--danger-border)] bg-[var(--danger-bg)] px-3 py-2 text-xs text-[var(--danger)]">{error}</p> : null}
          {result ? (
            <div className="border border-[var(--border)] bg-[var(--workspace)] p-3 text-xs">
              <div className="mb-2 font-medium">Complete</div>
              <dl className="grid gap-1 font-mono text-[11px] text-[var(--muted)] sm:grid-cols-2">
                <div>Analysis ID: {result.id}</div>
                <div>Requests: {result.requestCount}</div>
                <div>Endpoints: {result.endpointCount}</div>
                <div>Session artifacts: {result.sessionArtifactCount}</div>
                <div>Dependencies: {result.dependencyCount}</div>
              </dl>
              <button className="mt-3 inline-flex h-8 items-center gap-1.5 border border-[var(--border)] bg-[var(--panel)] px-3 text-xs" type="button" onClick={() => onOpen(result.id)}>
                <ExternalLink className="h-3.5 w-3.5" /> Open Analysis
              </button>
            </div>
          ) : null}
        </div>
      </section>
      <aside className="border border-[var(--border)] bg-[var(--panel)] p-3 text-xs">
        <h2 className="mb-2 text-[11px] font-medium uppercase text-[var(--muted)]">Local Capture</h2>
        <dl className="divide-y divide-[var(--border)] border-y border-[var(--border)]">
          <Row label="Mode" value="Local only" />
          <Row label="Adapter" value={localCapture?.adapter_built ? 'Built' : 'Missing'} />
          <Row label="Node" value={localCapture?.node_available ? 'Available' : 'Unavailable'} />
          <Row label="Duration" value="1-60s" />
        </dl>
        {!localCapture?.adapter_built ? <p className="mt-3 font-mono text-[11px] text-[var(--muted)]">cd browser; npm install; npm run build; npx playwright install chromium</p> : null}
      </aside>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-2 py-2"><dt className="text-[var(--muted)]">{label}</dt><dd className="font-mono">{value}</dd></div>
}