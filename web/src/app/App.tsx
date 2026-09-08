import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { FileSearch, Upload } from 'lucide-react'
import { getEndpoints, importHAR, listAnalyses } from '../api/analyses'
import type { Endpoint } from '../types/api'

const columns = createColumnHelper<Endpoint>()

export function App() {
  const queryClient = useQueryClient()
  const [selectedAnalysisID, setSelectedAnalysisID] = useState('')
  const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null)

  const analyses = useQuery({ queryKey: ['analyses'], queryFn: listAnalyses })
  const activeAnalysisID = selectedAnalysisID || analyses.data?.[0]?.id || ''
  const endpoints = useQuery({
    queryKey: ['endpoints', activeAnalysisID],
    queryFn: () => getEndpoints(activeAnalysisID),
    enabled: activeAnalysisID.length > 0,
  })

  const upload = useMutation({
    mutationFn: importHAR,
    onSuccess: (analysis) => {
      setSelectedAnalysisID(analysis.id)
      setSelectedEndpoint(null)
      queryClient.invalidateQueries({ queryKey: ['analyses'] })
      queryClient.invalidateQueries({ queryKey: ['endpoints', analysis.id] })
    },
  })

  const tableColumns = useMemo(
    () => [
      columns.accessor('method', { header: 'Method' }),
      columns.accessor('host', { header: 'Host' }),
      columns.accessor('path', { header: 'Path' }),
      columns.accessor('requestCount', { header: 'Requests' }),
      columns.accessor((row) => row.statusCodes.join(', '), { id: 'statusCodes', header: 'Status' }),
      columns.accessor((row) => row.averageDurationMs.toFixed(1), {
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
            <button className="flex w-full items-center rounded-md bg-white px-3 py-2 text-left font-medium text-slate-950 shadow-sm">
              Overview
            </button>
            <button className="flex w-full items-center rounded-md px-3 py-2 text-left text-slate-600">
              Endpoints
            </button>
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

          <div className="grid gap-4 sm:grid-cols-3">
            <Metric label="Requests" value={activeAnalysis?.requestCount ?? 0} />
            <Metric label="Endpoints" value={activeAnalysis?.endpointCount ?? 0} />
            <Metric label="Import ms" value={activeAnalysis?.importDurationMs ?? 0} />
          </div>

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
                  <tr
                    key={row.id}
                    className="cursor-pointer hover:bg-slate-50"
                    onClick={() => setSelectedEndpoint(row.original)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="truncate px-3 py-3 text-slate-700">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                ))}
                {!endpoints.isLoading && table.getRowModel().rows.length === 0 ? (
                  <tr>
                    <td className="px-3 py-10 text-center text-slate-500" colSpan={tableColumns.length}>
                      No endpoints yet.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>

          {selectedEndpoint ? (
            <div className="rounded-md border border-slate-200 bg-white p-4">
              <h2 className="text-base font-semibold text-slate-950">
                {selectedEndpoint.method} {selectedEndpoint.path}
              </h2>
              <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-3">
                <Detail label="Host" value={selectedEndpoint.host} />
                <Detail label="Duration" value={`${selectedEndpoint.minDurationMs.toFixed(1)}-${selectedEndpoint.maxDurationMs.toFixed(1)} ms`} />
                <Detail label="Content types" value={selectedEndpoint.contentTypes.join(', ') || 'Unknown'} />
              </dl>
            </div>
          ) : null}
        </section>
      </div>
    </main>
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

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-1 truncate font-medium text-slate-800">{value}</dd>
    </div>
  )
}
