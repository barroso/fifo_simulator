const API_BASE = ''

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...options?.headers },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  return res.json() as Promise<T>
}

export async function postJobs(config: {
  count: number
  job_type: string
  delay_ms: number
  interval_ms: number
  fail_percent: number
}) {
  return request<{ ok: boolean; count: number; message: string }>('/api/jobs', {
    method: 'POST',
    body: JSON.stringify(config),
  })
}

export async function getMetrics() {
  return request<Record<string, unknown>>('/api/metrics')
}

export async function postReset() {
  return request<{ ok: boolean }>('/api/reset', { method: 'POST' })
}

export async function getDLQ() {
  return request<Array<{
    id: string
    type: string
    created_at: string
    retry_count: number
    failed_at: string
    payload?: string
  }>>('/api/dlq')
}

export async function getHistory() {
  return request<{
    queue_size: { ts: number; value: number }[]
    latency: { ts: number; value: number }[]
    processed: { ts: number; value: number }[]
  }>('/api/history')
}
