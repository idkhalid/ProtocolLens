import { useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { FileSearch, Upload } from 'lucide-react'
import { getDependencies, getEndpoints, getSessions, getWorkflow, importHAR, listAnalyses } from '../api/analyses'
import type { Dependency, Endpoint, SessionArtifact } from '../types/api'
import { WorkflowPage } from '../features/workflow/WorkflowPage'

const endpointColumns = createColumnHelper<Endpoint>()
const sessionColumns = createColumnHelper<SessionArtifact>()
const dependencyColumns = createColumnHelper<Dependency>()

type View = 'overview' | 'sessions' | 'dependencies' | 'workflow'

export function App() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<View>('overview')
  const [selectedAnalysisID, setSelectedAnalysisID] = useState('')
  const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null)
  const [selectedArtifact, setSelectedArtifact] = useState<SessionArtifact | null>(null)
  const [selectedDependency, setSelectedDependency] = useState<Dependency | null>(null)

  const analyses = useQuery({ queryKey: ['analyses'], queryFn: listAnalyses })
  const activeAnalysisID = selectedAnalysisID || analyses.data?.[0]?.id || ''
  const endpoints = useQuery({
    queryKey: ['endpoints', activeAnalysisID],
    queryFn: () => getEndpoints(activeAnalysisID),
    enabled: activeAnalysisID.length > 0,
  })
  const sessions = useQuery({
    queryKey: ['sessions', activeAnalysisID],
    queryFn: () => getSessions(activeAnalysisID),
    enabled: activeAnalysisID.length > 0,
  })
  const dependencies = useQuery({
    queryKey: ['dependencies', activeAnalysisID],
    queryFn: () => getDependencies(activeAnalysisID),
    enabled: activeAnalysisID.length > 0,
  })
  const workflow = useQuery({
    queryKey: ['workflow', activeAnalysisID],
    queryFn: () => getWorkflow(activeAnalysisID),
    enabled: activeAnalysisID.length > 0,
  })

  const upload = useMutation({
    mutationFn: importHAR,
    onSuccess: (analysis) => {
      setSelectedAnalysisID(analysis.id)
      setSelectedEndpoint(null)
      setSelectedArtifact(null)
      setSelectedDependency(null)
      queryClient.invalidateQueries({ queryKey: ['analyses'] })
      queryClient.invalidateQueries({ queryKey: ['endpoints', analysis.id] })
      queryClient.invalidateQueries({ queryKey: ['sessions', analysis.id] })
      queryClient.invalidateQueries({ queryKey: ['dependencies', analysis.id] })
      queryClient.invalidateQueries({ queryKey: ['workflow', analysis.id] })
    },
  })

  const tableColumns = useMemo(
    () => [
      endpointColumns.accessor('method', { header: 'Method' }),
      endpointColumns.accessor('host', { header: 'Host' }),
      endpointColumns.accessor('path', { header: 'Path' }),
      endpointColumns.accessor('requestCount', { header: 'Requests' }),
      endpointColumns.accessor((row) => row.statusCodes.join(', '), { id: 'statusCodes', header: 'Status' }),
      endpointColumns.accessor((row) => row.averageDurationMs.toFixed(1), {
        id: 'averageDurationMs',
        header: 'Avg ms',
      }),
    ],
    [],
  )
  const table = useReactTable({
    data: endpoints.data ?? [],
    columns: tableColumns,
    getCoreRowModel: getCoreRowModel(),
  })

  const sessionTableColumns = useMemo(
    () => [
      sessionColumns.accessor((row) => artifactType(row.type), { id: 'type', header: 'Type' }),
      sessionColumns.accessor('name', { header: 'Name' }),
      sessionColumns.accessor('source', { header: 'Source' }),
      sessionColumns.accessor('occurrences', { header: 'Occurrences' }),
      sessionColumns.accessor((row) => new Date(row.firstSeenAt).toLocaleString(), {
        id: 'firstSeenAt',
        header: 'First Seen',
      }),
    ],
    [],
  )
  const sessionTable = useReactTable({
    data: sessions.data?.artifacts ?? [],
    columns: sessionTableColumns,
    getCoreRowModel: getCoreRowModel(),
  })

  const dependencyTableColumns = useMemo(
    () => [
      dependencyColumns.accessor('sourceRequestId', { header: 'Source Request' }),
      dependencyColumns.accessor('sourcePath', { header: 'Source Field' }),
      dependencyColumns.accessor('targetRequestId', { header: 'Target Request' }),
      dependencyColumns.accessor('targetLocation', { header: 'Target Location' }),
      dependencyColumns.accessor('targetPath', { header: 'Target Field' }),
      dependencyColumns.accessor('confidence', { header: 'Confidence' }),
    ],
    [],
  )
  const dependencyTable = useReactTable({
    data: dependencies.data?.dependencies ?? [],
    columns: dependencyTableColumns,
    getCoreRowModel: getCoreRowModel(),
  })

  const activeAnalysis = analyses.data?.find((analysis) => analysis.id === activeAnalysisID)

  return (
    <main className="min-h-screen bg-slate-50">
      <div className="mx-auto flex w-full max-w-7xl gap-6 px-6 py-6">
        <aside className="w-60 shrink-0">
          <div className="mb-6 flex items-center gap-2 text-xl font-semibold text-slate-950">
            <FileSearch className="h-6 w-6 text-teal-600" />
            ProtocolLens
          </div>
          <nav className="space-y-1 text-sm">
            <NavButton active={view === 'overview'} onClick={() => setView('overview')}>Overview</NavButton>
            <NavButton active={view === 'sessions'} onClick={() => setView('sessions')}>Sessions</NavButton>
            <NavButton active={view === 'dependencies'} onClick={() => setView('dependencies')}>Dependencies</NavButton>
            <NavButton active={view === 'workflow'} onClick={() => setView('workflow')}>Workflow</NavButton>
          </nav>
        </aside>

        <section className="min-w-0 flex-1 space-y-5">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 pb-4">
            <div>
              <h1 className="text-2xl font-semibold text-slate-950">HAR endpoint analysis</h1>
              <p className="mt-1 text-sm text-slate-600">
                Upload a HAR file. The Go API parses, normalizes, analyzes, and stores the result.
              </p>
            </div>
            <label className="inline-flex cursor-pointer items-center gap-2 rounded-md bg-teal-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-teal-700">
              <Upload className="h-4 w-4" />
              Upload HAR
              <input
                className="sr-only"
                type="file"
                accept=".har,application/json"
                onChange={(event) => {
                  const file = event.target.files?.[0]
                  if (file) upload.mutate(file)
                }}
              />
            </label>
          </div>

          {upload.error ? (
            <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
              {upload.error.message}
            </div>
          ) : null}

          <div className="grid gap-4 sm:grid-cols-4">
            <Metric label="Requests" value={activeAnalysis?.requestCount ?? 0} />
            <Metric label="Endpoints" value={activeAnalysis?.endpointCount ?? 0} />
            <Metric label="Session Artifacts" value={activeAnalysis?.sessionArtifactCount ?? 0} />
            <Metric label="Dependencies" value={activeAnalysis?.dependencyCount ?? 0} />
          </div>

          {view === 'workflow' ? (
            <WorkflowPage graph={workflow.data} loading={workflow.isLoading} />
          ) : view === 'overview' ? (
            <>
              <DataTable
                table={table}
                empty={!endpoints.isLoading && table.getRowModel().rows.length === 0}
                emptyText="No endpoints yet."
                colSpan={tableColumns.length}
                onRowClick={(endpoint) => setSelectedEndpoint(endpoint)}
              />
              {selectedEndpoint ? (
                <Panel title={`${selectedEndpoint.method} ${selectedEndpoint.path}`}>
                  <Detail label="Host" value={selectedEndpoint.host} />
                  <Detail label="Duration" value={`${selectedEndpoint.minDurationMs.toFixed(1)}-${selectedEndpoint.maxDurationMs.toFixed(1)} ms`} />
                  <Detail label="Content types" value={selectedEndpoint.contentTypes.join(', ') || 'Unknown'} />
                </Panel>
              ) : null}
            </>
          ) : view === 'sessions' ? (
            <>
              <DataTable
                table={sessionTable}
                empty={!sessions.isLoading && sessionTable.getRowModel().rows.length === 0}
                emptyText="No session artifacts yet."
                colSpan={sessionTableColumns.length}
                onRowClick={(artifact) => setSelectedArtifact(artifact)}
              />
              {selectedArtifact ? (
                <Panel title={`${artifactType(selectedArtifact.type)} ${selectedArtifact.name}`}>
                  <Detail label="Source" value={selectedArtifact.source} />
                  <Detail label="First request" value={selectedArtifact.firstRequestId} />
                  <Detail label="Metadata" value={metadata(selectedArtifact.metadata)} />
                </Panel>
              ) : null}
            </>
          ) : (
            <>
              <DataTable
                table={dependencyTable}
                empty={!dependencies.isLoading && dependencyTable.getRowModel().rows.length === 0}
                emptyText="No dependencies yet."
                colSpan={dependencyTableColumns.length}
                onRowClick={(dependency) => setSelectedDependency(dependency)}
              />
              {selectedDependency ? (
                <Panel title={`${selectedDependency.sourceRequestId} to ${selectedDependency.targetRequestId}`}>
                  <Detail label="Reason" value={reason(selectedDependency.reason)} />
                  <Detail label="Confidence" value={selectedDependency.confidence} />
                  <Detail label="Capture order" value="source before target" />
                </Panel>
              ) : null}
            </>
          )}
        </section>
      </div>
    </main>
  )
}

function NavButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: string }) {
  return (
    <button
      className={`flex w-full items-center rounded-md px-3 py-2 text-left font-medium ${active ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-600'}`}
      onClick={onClick}
    >
      {children}
    </button>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-slate-200 bg-white p-4">
      <div className="text-sm text-slate-500">{label}</div>
      <div className="mt-1 text-2xl font-semibold text-slate-950">{value}</div>
    </div>
  )
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-md border border-slate-200 bg-white p-4">
      <h2 className="text-base font-semibold text-slate-950">{title}</h2>
      <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-3">{children}</dl>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-1 truncate font-medium text-slate-800">{value}</dd>
    </div>
  )
}

function DataTable<T>({
  table,
  empty,
  emptyText,
  colSpan,
  onRowClick,
}: {
  table: ReturnType<typeof useReactTable<T>>
  empty: boolean
  emptyText: string
  colSpan: number
  onRowClick: (row: T) => void
}) {
  return (
    <div className="overflow-hidden rounded-md border border-slate-200 bg-white">
      <table className="w-full table-fixed text-left text-sm">
        <thead className="bg-slate-100 text-xs uppercase text-slate-500">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id} className="px-3 py-2 font-semibold">
                  {flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-slate-100">
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id} className="cursor-pointer hover:bg-slate-50" onClick={() => onRowClick(row.original)}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id} className="truncate px-3 py-3 text-slate-700">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
          {empty ? (
            <tr>
              <td className="px-3 py-10 text-center text-slate-500" colSpan={colSpan}>
                {emptyText}
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
  )
}

function artifactType(type: SessionArtifact['type']) {
  if (type === 'api_key') return 'API Key'
  return type.charAt(0).toUpperCase() + type.slice(1)
}

function metadata(value: Record<string, string> | undefined) {
  if (!value || Object.keys(value).length === 0) return 'None'
  return Object.entries(value)
    .map(([key, item]) => `${key}: ${item}`)
    .join(', ')
}

function reason(value: string) {
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
