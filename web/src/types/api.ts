export type Analysis = {
  id: string
  createdAt: string
  requestCount: number
  endpointCount: number
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
