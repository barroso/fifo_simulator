import { create } from 'zustand'
import type { MetricsSnapshot } from '../types'

interface MetricsState {
  snapshot: MetricsSnapshot | null
  history: {
    queueSize: { ts: number; value: number }[]
    latency: { ts: number; value: number }[]
    processed: { ts: number; value: number }[]
  }
  setSnapshot: (s: MetricsSnapshot) => void
  setHistory: (h: MetricsState['history']) => void
  appendQueueSize: (ts: number, value: number) => void
  appendLatency: (ts: number, value: number) => void
  appendProcessed: (ts: number, value: number) => void
}

const defaultSnapshot: MetricsSnapshot = {
  published: 0,
  processed: 0,
  failed: 0,
  dlq_count: 0,
  queue_size: 0,
  in_flight: 0,
  success_rate: 0,
  throughput_rps: 0,
  latency_avg_ms: 0,
  latency_p95_ms: 0,
  last_event_at: '',
}

export const useMetricsStore = create<MetricsState>((set) => ({
  snapshot: null,
  history: { queueSize: [], latency: [], processed: [] },
  setSnapshot: (snapshot) => set((s) => {
    const next = { ...defaultSnapshot, ...snapshot }
    const ts = Date.now()
    const max = 500
    const qs = [...s.history.queueSize.slice(-(max - 1)), { ts, value: next.queue_size }]
    const lat = [...s.history.latency.slice(-(max - 1)), { ts, value: next.latency_avg_ms }]
    const proc = [...s.history.processed.slice(-(max - 1)), { ts, value: next.processed }]
    return {
      snapshot: next,
      history: { queueSize: qs, latency: lat, processed: proc },
    }
  }),
  setHistory: (history) => set({ history }),
  appendQueueSize: (ts, value) =>
    set((s) => ({
      history: {
        ...s.history,
        queueSize: [...s.history.queueSize.slice(-499), { ts, value }],
      },
    })),
  appendLatency: (ts, value) =>
    set((s) => ({
      history: {
        ...s.history,
        latency: [...s.history.latency.slice(-499), { ts, value }],
      },
    })),
  appendProcessed: (ts, value) =>
    set((s) => ({
      history: {
        ...s.history,
        processed: [...s.history.processed.slice(-499), { ts, value }],
      },
    })),
}))
