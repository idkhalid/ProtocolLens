import type { ReactNode } from 'react'
import { FileSearch, GitBranch, KeyRound, LayoutDashboard, Moon, Network, RotateCw, Server, Sun, Upload } from 'lucide-react'
import type { Analysis } from '../types/api'

export type View = 'overview' | 'endpoints' | 'sessions' | 'dependencies' | 'workflow' | 'replay'

type Counts = {
  requests: number
  endpoints: number
  sessions: number
  dependencies: number
}

type NavItem = {
  id: View
  label: string
  count?: number
}

const navIcons = {
  overview: LayoutDashboard,
  endpoints: Server,
  sessions: KeyRound,
  dependencies: GitBranch,
  workflow: Network,
  replay: RotateCw,
}

export function AppShell({
  view,
  onViewChange,
  activeAnalysis,
  analyses,
  onAnalysisChange,
  onUpload,
  uploading,
  uploadError,
  theme,
  onToggleTheme,
  counts,
  detail,
  children,
}: {
  view: View
  onViewChange: (view: View) => void
  activeAnalysis?: Analysis
  analyses: Analysis[]
  onAnalysisChange: (id: string) => void
  onUpload: (file: File) => void
  uploading: boolean
  uploadError?: string
  theme: 'light' | 'dark'
  onToggleTheme: () => void
  counts: Counts
  detail: ReactNode
  children: ReactNode
}) {
  const items: NavItem[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'endpoints', label: 'Endpoints', count: counts.endpoints },
    { id: 'sessions', label: 'Sessions', count: counts.sessions },
    { id: 'dependencies', label: 'Dependencies', count: counts.dependencies },
    { id: 'workflow', label: 'Workflow' },
    { id: 'replay', label: 'Replay' },
  ]

  return (
    <main className="flex h-dvh min-w-0 flex-col bg-[var(--app-bg)] text-[var(--text)]">
      <header className="flex h-11 shrink-0 items-center justify-between gap-2 overflow-hidden border-b border-[var(--border)] bg-[var(--bar-bg)] px-2 sm:px-3">
        <div className="flex min-w-0 items-center gap-2">
          <FileSearch className="h-4 w-4 text-[var(--accent)]" />
          <div className="min-w-0">
            <div className="text-sm font-medium leading-4">ProtocolLens</div>
            <div className="hidden text-[11px] leading-3 text-[var(--muted)] sm:block">See what your browser is actually doing.</div>
          </div>
        </div>
        <div className="flex min-w-0 items-center gap-2">
          <select
            className="h-7 min-w-0 max-w-[42vw] rounded border border-[var(--border)] bg-[var(--panel)] px-2 text-xs text-[var(--text)] outline-none focus:border-[var(--accent)] sm:max-w-56"
            value={activeAnalysis?.id ?? ''}
            onChange={(event) => onAnalysisChange(event.target.value)}
            aria-label="Analysis"
          >
            {analyses.length === 0 ? <option value="">No analysis</option> : null}
            {analyses.map((analysis) => (
              <option key={analysis.id} value={analysis.id}>Analysis: {analysis.id}</option>
            ))}
          </select>
          <button className="icon-button" type="button" onClick={onToggleTheme} aria-label="Toggle theme" title="Toggle theme">
            {theme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>
          <label className="upload-button inline-flex h-7 cursor-pointer items-center gap-1.5 rounded border border-[var(--accent)] bg-[var(--accent)] px-2 text-xs font-medium text-white hover:bg-[var(--accent-strong)]" title="Import HAR">
            <Upload className="h-3.5 w-3.5" />
            <span className="upload-label">{uploading ? 'Importing' : 'Import HAR'}</span>
            <input
              className="sr-only"
              type="file"
              accept=".har,application/json"
              onChange={(event) => {
                const file = event.target.files?.[0]
                if (file) onUpload(file)
                event.currentTarget.value = ''
              }}
            />
          </label>
        </div>
      </header>

      <div className="flex min-h-0 min-w-0 flex-1">
        <aside className="hidden w-48 shrink-0 border-r border-[var(--border)] bg-[var(--sidebar-bg)] p-2 lg:block">
          <Nav items={items} view={view} onViewChange={onViewChange} />
        </aside>

        <section className="flex min-w-0 flex-1 flex-col bg-[var(--workspace)]">
          {uploadError ? <div className="border-b border-[var(--danger-border)] bg-[var(--danger-bg)] px-3 py-2 text-xs text-[var(--danger)]">{uploadError}</div> : null}
          <nav className="flex shrink-0 gap-1 overflow-x-auto border-b border-[var(--border)] bg-[var(--sidebar-bg)] p-2 lg:hidden">
            <Nav items={items} view={view} onViewChange={onViewChange} compact />
          </nav>
          <div className="flex min-h-0 min-w-0 flex-1 flex-col lg:flex-row">
            <div className="min-w-0 flex-1 overflow-auto p-2 sm:p-3">{children}</div>
            <aside className="min-h-72 shrink-0 border-t border-[var(--border)] bg-[var(--panel)] lg:h-auto lg:w-[360px] lg:border-l lg:border-t-0">
              {detail}
            </aside>
          </div>
        </section>
      </div>

      <footer className="status-bar flex shrink-0 items-center gap-3 overflow-x-auto border-t border-[var(--border)] bg-[var(--bar-bg)] px-2 py-1 font-mono text-[11px] text-[var(--muted)] sm:h-7 sm:gap-4 sm:px-3 sm:py-0">
        <span>{counts.requests} requests</span>
        <span>{counts.endpoints} endpoints</span>
        <span>{counts.sessions} session artifacts</span>
        <span>{counts.dependencies} dependencies</span>
        <span className="ml-auto hidden sm:inline">Local analysis</span>
      </footer>
    </main>
  )
}

function Nav({ items, view, onViewChange, compact = false }: { items: NavItem[]; view: View; onViewChange: (view: View) => void; compact?: boolean }) {
  return (
    <div className={compact ? 'flex gap-1' : 'space-y-1'}>
      {items.map((item) => {
        const Icon = navIcons[item.id]
        return (
          <button key={item.id} className={`nav-item ${compact ? 'min-w-max' : ''} ${view === item.id ? 'active' : ''}`} type="button" onClick={() => onViewChange(item.id)}>
            <Icon className="h-3.5 w-3.5" />
            <span className="min-w-0 flex-1 truncate text-left">{item.label}</span>
            {!compact && typeof item.count === 'number' ? <span className="font-mono text-[11px] text-[var(--muted)]">{item.count}</span> : null}
          </button>
        )
      })}
    </div>
  )
}

export function DetailPanel({ title, subtitle, children }: { title: string; subtitle?: string; children?: ReactNode }) {
  return (
    <div className="h-full min-w-0 overflow-auto p-3">
      <div className="border-b border-[var(--border)] pb-3">
        <h2 className="truncate text-sm font-medium">{title}</h2>
        {subtitle ? <p className="mt-1 truncate font-mono text-xs text-[var(--muted)]">{subtitle}</p> : null}
      </div>
      <div className="space-y-4 py-3">{children ?? <p className="text-xs text-[var(--muted)]">Select a row or graph element.</p>}</div>
    </div>
  )
}

export function InspectorSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h3 className="mb-2 text-[11px] font-medium uppercase text-[var(--muted)]">{title}</h3>
      <dl className="divide-y divide-[var(--border)] border-y border-[var(--border)]">{children}</dl>
    </section>
  )
}

export function KeyValue({ label, value, mono = true }: { label: string; value: ReactNode; mono?: boolean }) {
  return (
    <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-2 py-2 text-xs sm:grid-cols-[110px_minmax(0,1fr)] sm:gap-3">
      <dt className="text-[var(--muted)]">{label}</dt>
      <dd className={`min-w-0 break-words text-[var(--text)] ${mono ? 'font-mono' : ''}`}>{value}</dd>
    </div>
  )
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <div className="border border-[var(--border)] bg-[var(--panel)] px-3 py-10 text-center text-xs text-[var(--muted)]">{children}</div>
}

export function MethodBadge({ method }: { method: string }) {
  return <span className={`method method-${method.toLowerCase()}`}>{method}</span>
}

export function StatusCode({ value }: { value: number }) {
  const family = value >= 500 ? 's5' : value >= 400 ? 's4' : value >= 300 ? 's3' : value >= 200 ? 's2' : 'sx'
  return <span className={`status ${family}`}>{value || 'n/a'}</span>
}

export function formatMs(value: number) {
  return `${value.toFixed(value % 1 === 0 ? 0 : 1)} ms`
}