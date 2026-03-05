import { useState } from 'react'
import { postJobs } from '../api/client'
import type { JobType } from '../types'
import styles from './Config.module.css'

export function Config() {
  const [count, setCount] = useState(100)
  const [jobType, setJobType] = useState<JobType>('email')
  const [delayMs, setDelayMs] = useState(0)
  const [intervalMs, setIntervalMs] = useState(0)
  const [failPercent, setFailPercent] = useState(10)
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<{ type: 'ok' | 'err'; text: string } | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setMessage(null)
    setLoading(true)
    try {
      const res = await postJobs({
        count,
        job_type: jobType,
        delay_ms: delayMs,
        interval_ms: intervalMs,
        fail_percent: failPercent,
      })
      setMessage({ type: 'ok', text: res.message })
    } catch (err) {
      setMessage({ type: 'err', text: err instanceof Error ? err.message : 'Erro ao enviar' })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <h1>Configuração de Testes</h1>
      <p className={styles.subtitle}>
        Defina a quantidade de registros, tipo de tarefa e comportamento da fila. Ao clicar em &quot;Iniciar teste&quot;, as mensagens serão enviadas ao Kafka.
      </p>
      <p className={styles.tip}>
        Dica: deixe o <strong>Dashboard</strong> aberto em outra aba para ver fila, processados, falhas e DLQ em tempo real enquanto o teste roda.
      </p>
      <form onSubmit={handleSubmit} className={styles.form}>
        <label>
          Quantidade de registros (1–10000)
          <input
            type="number"
            min={1}
            max={10000}
            value={count}
            onChange={(e) => setCount(Number(e.target.value))}
          />
        </label>
        <label>
          Tipo de tarefa
          <select value={jobType} onChange={(e) => setJobType(e.target.value as JobType)}>
            <option value="email">Envio de e-mail (leve, ~20–50 ms)</option>
            <option value="image">Processamento de imagem (pesado, ~200–500 ms)</option>
          </select>
        </label>
        <label>
          Delay base (ms) — 0 = usar padrão pelo tipo
          <input
            type="number"
            min={0}
            value={delayMs}
            onChange={(e) => setDelayMs(Number(e.target.value))}
          />
        </label>
        <label>
          Intervalo entre envios (ms) — 0 = envio em rajada
          <input
            type="number"
            min={0}
            value={intervalMs}
            onChange={(e) => setIntervalMs(Number(e.target.value))}
          />
        </label>
        <label>
          Simular falhas (% de mensagens que falharão e irão para retry/DLQ)
          <input
            type="number"
            min={0}
            max={100}
            value={failPercent}
            onChange={(e) => setFailPercent(Number(e.target.value))}
          />
        </label>
        <button type="submit" disabled={loading}>
          {loading ? 'Enviando…' : 'Iniciar teste'}
        </button>
      </form>
      {message && (
        <p className={message.type === 'ok' ? styles.msgOk : styles.msgErr}>{message.text}</p>
      )}
    </div>
  )
}
