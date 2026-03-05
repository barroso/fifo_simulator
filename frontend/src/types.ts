export interface LogEntry {
  ts: string
  node: string
  job_id?: string
  message: string
}

export interface MetricsSnapshot {
  published: number
  processed: number
  failed: number
  dlq_count: number
  queue_size: number
  in_flight: number
  success_rate: number
  throughput_rps: number
  latency_avg_ms: number
  latency_p95_ms: number
  last_event_at: string
  recent_logs: LogEntry[]
}

export interface DLQItem {
  id: string
  type: string
  created_at: string
  retry_count: number
  failed_at: string
  payload?: string
}

export interface HistoryResponse {
  queue_size: { ts: number; value: number }[]
  latency: { ts: number; value: number }[]
  processed: { ts: number; value: number }[]
}

export type JobType = 'image' | 'email'

export interface JobConfig {
  count: number
  job_type: JobType
  delay_ms: number
  interval_ms: number
  fail_percent: number
}
