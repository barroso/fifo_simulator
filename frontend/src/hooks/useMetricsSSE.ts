import { useEffect, useRef } from 'react'
import type { MetricsSnapshot } from '../types'
import { useMetricsStore } from '../store/metricsStore'

const API_BASE = ''

export function useMetricsSSE(enabled: boolean) {
  const ref = useRef<EventSource | null>(null)
  const setSnapshot = useMetricsStore((s) => s.setSnapshot)

  useEffect(() => {
    if (!enabled) return
    const url = `${API_BASE}/api/events`
    const es = new EventSource(url)
    ref.current = es
    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data) as MetricsSnapshot
        setSnapshot(data)
      } catch {
        // ignore
      }
    }
    es.onerror = () => {
      es.close()
      ref.current = null
    }
    return () => {
      es.close()
      ref.current = null
    }
  }, [enabled, setSnapshot])
}
