export type Analysis = {
  id: string
  createdAt: string
  requestCount: number
  endpointCount: number
  sessionArtifactCount: number
  dependencyCount: number
  importDurationMs: number
}

export type Endpoint = {
  method: string
  host: string
  path: string
  requestCount: number
  statusCodes: number[]
  averageDurationMs: number
  minDurationMs: number
  maxDurationMs: number
  contentTypes: string[]
}

export type SessionArtifact = {
  id: string
  type: 'cookie' | 'bearer' | 'csrf' | 'api_key'
  name: string
  source: string
  firstRequestId: string
  firstSeenAt: string
  occurrences: number
  metadata?: Record<string, string>
}

export type SessionArtifactsResponse = {
  artifacts: SessionArtifact[]
}

export type Dependency = {
  id: string
  sourceRequestId: string
  targetRequestId: string
  sourcePath: string
  targetLocation: string
  targetPath: string
  confidence: 'high' | 'medium'
  reason: string
}

export type DependenciesResponse = {
  dependencies: Dependency[]
}
export type WorkflowNode = {
  id: string
  requestId: string
  order: number
  method: string
  host: string
  path: string
  statusCode: number
  durationMs: number
}

export type WorkflowEdge = {
  id: string
  source: string
  target: string
  type: 'data_dependency'
  confidence: 'high' | 'medium'
  reason: string
  sourcePath: string
  targetLocation: string
  targetPath: string
}

export type WorkflowGraph = {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
}

export type ReplayTemplate = {
  analysisId: string
  requestId: string
  method: string
  url: string
  scheme: string
  host: string
  port: string
  path: string
  query: Record<string, string[]>
  headers: Record<string, string[]>
  body: string
  bodyAvailable: boolean
  bodyReason?: string
  contentType?: string
  requiresReview: boolean
}

export type ReplayRequest = {
  analysisId?: string
  requestId?: string
  method: string
  url: string
  headers: Record<string, string[]>
  body: string
  followRedirects: boolean
}

export type ReplayResponse = {
  statusCode: number
  durationMs: number
  finalUrl: string
  headers: Record<string, string[]>
  body: string
  bodyAvailable: boolean
  contentLength: number
  contentType: string
  truncated: boolean
}

export type GeneratorTarget = 'curl' | 'python' | 'go'

export type GeneratorOutput = {
	target: GeneratorTarget
	language: string
	code: string
	environmentVariables?: string[]
}
