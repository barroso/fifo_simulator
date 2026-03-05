import { useEffect, useState } from 'react'
import { getHistory } from '../api/client'
import { useMetricsStore } from '../store/metricsStore'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts'
import styles from './Analysis.module.css'

export function Analysis() {
  const [loaded, setLoaded] = useState(false)
  const history = useMetricsStore((s) => s.history)
  const snapshot = useMetricsStore((s) => s.snapshot)

  useEffect(() => {
    getHistory()
      .then((h) => {
        useMetricsStore.getState().setHistory({
          queueSize: h.queue_size || [],
          latency: h.latency || [],
          processed: h.processed || [],
        })
      })
      .catch(() => {})
      .finally(() => setLoaded(true))
  }, [])

  const queueSizeData = history.queueSize.map((p) => ({
    ts: new Date(p.ts).toLocaleTimeString('pt-BR', { hour12: false }),
    fila: p.value,
  }))
  const latencyData = history.latency.map((p) => ({
    ts: new Date(p.ts).toLocaleTimeString('pt-BR', { hour12: false }),
    latencia_ms: Math.round(p.value),
  }))
  const processedData = history.processed.map((p) => ({
    ts: new Date(p.ts).toLocaleTimeString('pt-BR', { hour12: false }),
    processados: p.value,
  }))

  const rate =
    snapshot && snapshot.processed > 0 && snapshot.last_event_at
      ? (snapshot.processed / (new Date(snapshot.last_event_at).getTime() / 1000)).toFixed(1)
      : '0'

  return (
    <div className={styles.page}>
      <h1>Análise e Gráficos</h1>
      <p className={styles.subtitle}>
        Histórico da sessão: tamanho da fila, latência e throughput.
      </p>

      {snapshot && (
        <div className={styles.summary}>
          <span>Throughput (processados/s) aproximado: <strong>{rate}</strong></span>
          <span>Total processados: <strong>{snapshot.processed}</strong></span>
          <span>Total na DLQ: <strong>{snapshot.dlq_count}</strong></span>
        </div>
      )}

      <div className={styles.chartBlock}>
        <h2>Tamanho da fila ao longo do tempo</h2>
        <div className={styles.chart}>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={queueSizeData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="ts" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="fila" stroke="#1a1a2e" name="Fila" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className={styles.chartBlock}>
        <h2>Latência (média) ao longo do tempo</h2>
        <div className={styles.chart}>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={latencyData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="ts" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="latencia_ms" stroke="#0a7" name="Latência (ms)" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className={styles.chartBlock}>
        <h2>Processados acumulados</h2>
        <div className={styles.chart}>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={processedData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="ts" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="processados" stroke="#07c" name="Processados" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {!loaded && history.queueSize.length === 0 && (
        <p className={styles.loading}>Carregando histórico…</p>
      )}
    </div>
  )
}
