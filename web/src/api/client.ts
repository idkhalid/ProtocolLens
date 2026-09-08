const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiURL}${path}`, init)
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    throw new Error(body?.error?.message ?? `Request failed: ${response.status}`)
  }
  return response.json() as Promise<T>
}
