import { useMemo } from 'react'
import type { Edge, Node, NodeProps } from '@xyflow/react'
import { Background, Controls, Handle, MarkerType, Position, ReactFlow } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { EmptyState, MethodBadge, StatusCode, formatMs } from '../../components/workbench'
import { edgeLabel } from '../dependencies/DependenciesPage'
import type { WorkflowEdge, WorkflowGraph, WorkflowNode as WorkflowNodeData } from '../../types/api'

const maxRenderedNodes = 500
const nodeTypes = { workflow: WorkflowNode }

type WorkflowPageProps = {
  graph?: WorkflowGraph
  loading: boolean
  onSelectNode: (node: WorkflowNodeData) => void
  onSelectEdge: (edge: WorkflowEdge) => void
}

type NodeData = WorkflowNodeData & Record<string, unknown>
type EdgeData = WorkflowEdge & Record<string, unknown>

export function WorkflowPage({ graph, loading, onSelectNode, onSelectEdge }: WorkflowPageProps) {
  const allNodes = graph?.nodes ?? []
  const allEdges = graph?.edges ?? []
  const renderedIDs = useMemo(() => new Set(allNodes.slice(0, maxRenderedNodes).map((node) => node.id)), [allNodes])

  const nodes = useMemo<Node<NodeData>[]>(() => allNodes.slice(0, maxRenderedNodes).map((node) => ({
    id: node.id,
    type: 'workflow',
    position: { x: (node.order % 10) * 250, y: Math.floor(node.order / 10) * 140 },
    data: node as NodeData,
  })), [allNodes])

  const edges = useMemo<Edge<EdgeData>[]>(() => allEdges
    .filter((edge) => renderedIDs.has(edge.source) && renderedIDs.has(edge.target))
    .map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edgeLabel(edge.targetLocation),
      markerEnd: { type: MarkerType.ArrowClosed },
      data: edge as EdgeData,
      className: edge.confidence === 'high' ? 'workflow-edge-high' : 'workflow-edge-medium',
    })), [allEdges, renderedIDs])

  if (loading) return <EmptyState>Loading workflow.</EmptyState>
  if (allNodes.length === 0) return <EmptyState>No workflow data available.</EmptyState>

  return (
    <div className="space-y-2">
      {allNodes.length > maxRenderedNodes ? (
        <div className="notice">Showing first {maxRenderedNodes} of {allNodes.length} workflow nodes. Edges connected to hidden nodes are omitted.</div>
      ) : null}
      {allEdges.length === 0 ? <div className="notice muted">No inferred request dependencies were found.</div> : null}
      <div className="workflow-canvas">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          minZoom={0.2}
          maxZoom={1.4}
          onNodeClick={(_, node) => onSelectNode(node.data as WorkflowNodeData)}
          onEdgeClick={(_, edge) => onSelectEdge(edge.data as WorkflowEdge)}
        >
          <Background color="var(--flow-grid)" />
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>
    </div>
  )
}

function WorkflowNode({ data }: NodeProps<Node<NodeData>>) {
  return (
    <div className="workflow-node">
      <Handle type="target" position={Position.Left} />
      <div className="flex items-center justify-between gap-2">
        <MethodBadge method={data.method} />
        <span className="font-mono text-[11px] text-[var(--muted)]">#{data.order + 1}</span>
      </div>
      <div className="mt-2 truncate font-mono text-xs font-medium text-[var(--text)]">{data.path}</div>
      <div className="mt-1 flex items-center justify-between gap-2 text-[11px] text-[var(--muted)]">
        <StatusCode value={data.statusCode} />
        <span className="font-mono">{formatMs(data.durationMs)}</span>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  )
}