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