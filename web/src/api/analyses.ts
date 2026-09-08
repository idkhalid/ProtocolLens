import { api } from './client'
import type { Analysis, Endpoint, SessionArtifactsResponse } from '../types/api'

export function listAnalyses() {
  return api<Analysis[]>('/api/v1/analyses')
}

export function getEndpoints(analysisID: string) {
  return api<Endpoint[]>(`/api/v1/analyses/${analysisID}/endpoints`)
}

export function getSessions(analysisID: string) {
  return api<SessionArtifactsResponse>(`/api/v1/analyses/${analysisID}/sessions`)
}

export function importHAR(file: File) {
  return api<Analysis>('/api/v1/import/har', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: file,
  })
}
