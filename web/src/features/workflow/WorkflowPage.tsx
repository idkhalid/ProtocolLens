import { useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { Edge, Node, NodeProps } from '@xyflow/react'
import { Background, Controls, Handle, MarkerType, Position, ReactFlow } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { WorkflowEdge, WorkflowGraph, WorkflowNode as WorkflowNodeData } from '../../types/api'

const maxRenderedNodes = 500
const nodeTypes = { workflow: WorkflowNode }

type WorkflowPageProps = {
  graph?: WorkflowGraph
  loading: boolean
}

type NodeData = WorkflowNodeData & Record<string, unknown>
type EdgeData = WorkflowEdge & Record<string, unknown>

export function WorkflowPage({ graph, loading }: WorkflowPageProps) {
  const [selectedNode, setSelectedNode] = useState<WorkflowNodeData | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<WorkflowEdge | null>(null)
  const allNodes = graph?.nodes ?? []
  const allEdges = graph?.edges ?? []
  const renderedIDs = useMemo(() => new Set(allNodes.slice(0, maxRenderedNodes).map((node) => node.id)), [allNodes])

  const nodes = useMemo<Node<NodeData>[]>(
    () =>
      allNodes.slice(0, maxRenderedNodes).map((node) => ({
        id: node.id,
        type: 'workflow',
        position: { x: node.order * 260, y: 0 },
        data: node as NodeData,
      })),
    [allNodes],
  )
  const edges = useMemo<Edge<EdgeData>[]>(
    () =>
      allEdges
        .filter((edge) => renderedIDs.has(edge.source) && renderedIDs.has(edge.target))
        .map((edge) => ({
          id: edge.id,
          source: edge.source,
          target: edge.target,
          label: edgeLabel(edge.targetLocation),
          markerEnd: { type: MarkerType.ArrowClosed },
          data: edge as EdgeData,
          className: edge.confidence === 'high' ? 'stroke-teal-600' : 'stroke-slate-500',
        })),
    [allEdges, renderedIDs],
  )

  if (loading) {
    return <div className="rounded-md border border-slate-200 bg-white px-4 py-10 text-center text-slate-500">Loading workflow.</div>
  }
  if (allNodes.length === 0) {
    return <div className="rounded-md border border-slate-200 bg-white px-4 py-10 text-center text-slate-500">No workflow data available.</div>
  }

  return (
    <div className="space-y-4">
      {allNodes.length > maxRenderedNodes ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          Showing first {maxRenderedNodes} of {allNodes.length} workflow nodes. Edges connected to hidden nodes are omitted.
        </div>
      ) : null}
      {allEdges.length === 0 ? (
        <div className="rounded-md border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600">
          No inferred request dependencies were found.
        </div>
      ) : null}

      <div className="h-[520px] overflow-hidden rounded-md border border-slate-200 bg-white">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          minZoom={0.2}
          maxZoom={1.4}
          onNodeClick={(_, node) => {
            setSelectedNode(node.data as WorkflowNodeData)
            setSelectedEdge(null)
          }}
          onEdgeClick={(_, edge) => {
            setSelectedEdge(edge.data as WorkflowEdge)
            setSelectedNode(null)
          }}
        >
          <Background />
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>

      {selectedEdge ? <EdgeDetails edge={selectedEdge} /> : selectedNode ? <NodeDetails node={selectedNode} /> : null}
    </div>
  )
}

function WorkflowNode({ data }: NodeProps<Node<NodeData>>) {
  return (
    <div className="w-52 rounded-md border border-slate-300 bg-white px-3 py-2 text-left shadow-sm">
      <Handle type="target" position={Position.Left} />
      <div className="flex items-center justify-between gap-2">
        <span className="rounded bg-teal-50 px-2 py-0.5 text-xs font-semibold text-teal-700">{data.method}</span>
        <span className="text-xs text-slate-500">#{data.order + 1}</span>
      </div>
      <div className="mt-2 truncate text-sm font-semibold text-slate-950">{data.path}</div>
      <div className="mt-1 flex items-center justify-between text-xs text-slate-500">
        <span>{data.statusCode || 'n/a'}</span>
        <span>{data.durationMs} ms</span>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  )
}

function NodeDetails({ node }: { node: WorkflowNodeData }) {
  return (
    <Panel title={`${node.method} ${node.path}`}>
      <Detail label="Request ID" value={node.requestId} />
      <Detail label="Capture Order" value={String(node.order)} />
      <Detail label="Host" value={node.host || 'Unknown'} />
      <Detail label="Status" value={String(node.statusCode || 'n/a')} />
      <Detail label="Duration" value={`${node.durationMs} ms`} />
    </Panel>
  )
}

function EdgeDetails({ edge }: { edge: WorkflowEdge }) {
  return (
    <Panel title={`${edge.source} to ${edge.target}`}>
      <Detail label="Source Request" value={edge.source} />
      <Detail label="Target Request" value={edge.target} />
      <Detail label="Source JSON Path" value={edge.sourcePath} />
      <Detail label="Target Location" value={edgeLabel(edge.targetLocation)} />
      <Detail label="Target Path" value={edge.targetPath} />
      <Detail label="Confidence" value={edge.confidence} />
      <Detail label="Reason" value={reason(edge.reason)} />
    </Panel>
  )
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-md border border-slate-200 bg-white p-4">
      <h2 className="text-base font-semibold text-slate-950">{title}</h2>
      <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-4">{children}</dl>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-1 truncate font-medium text-slate-800">{value}</dd>
    </div>
  )
}

function edgeLabel(value: string) {
  switch (value) {
    case 'query':
      return 'query'
    case 'path':
      return 'path'
    case 'body_json':
      return 'JSON body'
    case 'body_form':
      return 'form body'
    default:
      return value
  }
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