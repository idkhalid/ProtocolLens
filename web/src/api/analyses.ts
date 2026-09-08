import { api } from './client'
import type { Analysis, Endpoint } from '../types/api'

export function listAnalyses() {
  return api<Analysis[]>('/api/v1/analyses')
}

export function getEndpoints(analysisID: string) {
  return api<Endpoint[]>(`/api/v1/analyses/${analysisID}/endpoints`)
}

export function importHAR(file: File) {
  return api<Analysis>('/api/v1/import/har', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: file,
  })
}
