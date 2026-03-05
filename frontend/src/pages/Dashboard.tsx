import { useEffect, useState } from 'react'
import { useMetricsSSE } from '../hooks/useMetricsSSE'
import { useMetricsStore } from '../store/metricsStore'
import { postReset } from '../api/client'
import {
  AreaChart, Area,
  LineChart, Line,
  XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts'
import styles from './Dashboard.module.css'

/** Formats milliseconds into human-readable string: <1s → ms, <60s → s, ≥60s → min */
function fmtLatency(ms: number): string {
  if (ms <= 0) return '—'
  if (ms < 1000) return `${ms} ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(2)} s`
  return `${(ms / 60_000).toFixed(1)} min`
}

function LiveIndicator() {
  const snapshot = useMetricsStore((s) => s.snapshot)
  const [isLive, setIsLive] = useState(false)
  useEffect(() => {
    if (!snapshot?.last_event_at) return
    setIsLive(true)
    const t = setTimeout(() => setIsLive(false), 3000)
    return () => clearTimeout(t)
  }, [snapshot?.processed, snapshot?.queue_size, snapshot?.failed, snapshot?.dlq_count, snapshot?.in_flight])
  return (
    <div className={styles.liveBox}>
      <span className={`${styles.liveDot} ${isLive ? styles.liveDotOn : ''}`} />
      <span className={styles.liveText}>
        {snapshot?.last_event_at
          ? `Atualizado: ${new Date(snapshot.last_event_at).toLocaleTimeString('pt-BR')}`
          : 'Aguardando dados…'}
      </span>
    </div>
  )
}

export function Dashboard() {
  useMetricsSSE(true)
  const snapshot = useMetricsStore((s) => s.snapshot)
  const history = useMetricsStore((s) => s.history)
  const setHistory = useMetricsStore((s) => s.setHistory)
  const [resetting, setResetting] = useState(false)

  async function handleReset() {
    setResetting(true)
    await postReset()
    setHistory({ queueSize: [], latency: [], processed: [] })
    setResetting(false)
  }

  const queueData = history.queueSize.slice(-80).map((p) => ({
    ts: new Date(p.ts).toLocaleTimeString('pt-BR', { hour12: false }),
    fila: p.value,
  }))
  const processedData = history.processed.slice(-80).map((p) => ({
    ts: new Date(p.ts).toLocaleTimeString('pt-BR', { hour12: false }),
    processados: p.value,
  }))

  const snap = snapshot

  return (
    <div className={styles.page}>
      <div className={styles.headerRow}>
        <h1>Dashboard em tempo real</h1>
        <LiveIndicator />
        <button className={styles.resetBtn} onClick={handleReset} disabled={resetting}>
          {resetting ? 'Limpando…' : 'Resetar métricas'}
        </button>
      </div>
      <p className={styles.subtitle}>
        Ao clicar em "Iniciar teste" na tela Configuração, as métricas são zeradas automaticamente.
      </p>

      {/* Cards principais: estado da fila */}
      <section className={styles.metricsGrid}>
        <div className={`${styles.metricCard} ${styles.metricQueue}`}>
          <span className={styles.metricLabel}>Aguardando na fila</span>
          <span className={styles.metricValue}>{snap?.queue_size ?? 0}</span>
          <span className={styles.metricHint}>publicados – (processados + DLQ)</span>
        </div>
        <div className={`${styles.metricCard} ${styles.metricFlight}`}>
          <span className={styles.metricLabel}>Em processamento agora</span>
          <span className={styles.metricValue}>{snap?.in_flight ?? 0}</span>
          <span className={styles.metricHint}>mensagens sendo trabalhadas</span>
        </div>
        <div className={`${styles.metricCard} ${styles.metricProcessed}`}>
          <span className={styles.metricLabel}>Processados com sucesso</span>
          <span className={styles.metricValue}>{snap?.processed ?? 0}</span>
          <span className={styles.metricHint}>de {snap?.published ?? 0} publicados</span>
        </div>
        <div className={`${styles.metricCard} ${styles.metricFailed}`}>
          <span className={styles.metricLabel}>Tentativas com falha</span>
          <span className={styles.metricValue}>{snap?.failed ?? 0}</span>
          <span className={styles.metricHint}>retentadas antes do DLQ</span>
        </div>
        <div className={`${styles.metricCard} ${styles.metricDLQ}`}>
          <span className={styles.metricLabel}>Dead Letter Queue</span>
          <span className={styles.metricValue}>{snap?.dlq_count ?? 0}</span>
          <span className={styles.metricHint}>esgotaram retries (não processados)</span>
        </div>
      </section>

      {/* Linha secundária: desempenho */}
      <section className={styles.secondaryRow}>
        <div className={styles.smallCard}>
          <span className={styles.label}>Taxa de sucesso</span>
          <span className={styles.value}>
            {snap?.success_rate != null && snap.success_rate > 0
              ? `${snap.success_rate.toFixed(1)}%`
              : '—'}
          </span>
        </div>
        <div className={styles.smallCard}>
          <span className={styles.label}>Throughput</span>
          <span className={styles.value}>
            {snap?.throughput_rps != null && snap.throughput_rps > 0
              ? `${snap.throughput_rps.toFixed(1)} /s`
              : '—'}
          </span>
        </div>
        <div className={styles.smallCard}>
          <span className={styles.label}>Latência média</span>
          <span className={styles.value}>{fmtLatency(snap?.latency_avg_ms ?? 0)}</span>
        </div>
        <div className={styles.smallCard}>
          <span className={styles.label}>Latência P95</span>
          <span className={styles.value}>{fmtLatency(snap?.latency_p95_ms ?? 0)}</span>
        </div>
      </section>

      {/* Legenda */}
      <div className={styles.legend}>
        <span className={`${styles.legendItem} ${styles.legendQueue}`}>■ Fila: aguardando</span>
        <span className={`${styles.legendItem} ${styles.legendFlight}`}>■ Em processamento agora</span>
        <span className={`${styles.legendItem} ${styles.legendOk}`}>■ Sucesso</span>
        <span className={`${styles.legendItem} ${styles.legendWarn}`}>■ Falha/retry</span>
        <span className={`${styles.legendItem} ${styles.legendDlq}`}>■ DLQ: falhou {'>'}3x</span>
      </div>

      {/* Gráficos */}
      <div className={styles.chartsRow}>
        <div className={styles.chartSection}>
          <h2>Fila ao longo do tempo</h2>
          <p className={styles.chartHint}>Sobe com novos jobs, cai conforme são processados ou vão para DLQ.</p>
          <div className={styles.chart}>
            <ResponsiveContainer width="100%" height={240}>
              <AreaChart data={queueData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="ts" tick={{ fontSize: 11 }} interval="preserveStartEnd" />
                <YAxis allowDecimals={false} width={40} />
                <Tooltip formatter={(v: number) => [v, 'Na fila']} />
                <Area type="monotone" dataKey="fila" stroke="#1a1a2e" fill="#1a1a2e" fillOpacity={0.2} dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
        <div className={styles.chartSection}>
          <h2>Processados com sucesso (acumulado)</h2>
          <p className={styles.chartHint}>Curva crescente = workers consumindo. Inclinação = throughput.</p>
          <div className={styles.chart}>
            <ResponsiveContainer width="100%" height={240}>
              <LineChart data={processedData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="ts" tick={{ fontSize: 11 }} interval="preserveStartEnd" />
                <YAxis allowDecimals={false} width={40} />
                <Tooltip formatter={(v: number) => [v, 'Processados']} />
                <Line type="monotone" dataKey="processados" stroke="#0a7" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  )
}
