import { useEffect, useState } from 'react'
import { getDLQ } from '../api/client'
import type { DLQItem } from '../types'
import styles from './DLQ.module.css'

export function DLQ() {
  const [items, setItems] = useState<DLQItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const data = await getDLQ()
        if (!cancelled) setItems(data)
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Erro ao carregar DLQ')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    const t = setInterval(load, 3000)
    return () => {
      cancelled = true
      clearInterval(t)
    }
  }, [])

  if (loading) return <div className={styles.page}><p>Carregando…</p></div>
  if (error) return <div className={styles.page}><p className={styles.err}>{error}</p></div>

  return (
    <div className={styles.page}>
      <h1>Dead Letter Queue (DLQ)</h1>
      <p className={styles.subtitle}>
        Tarefas que falharam após o número máximo de tentativas são enviadas aqui. É fundamental para o aprendizado de sistemas resilientes.
      </p>
      {items.length === 0 ? (
        <p className={styles.empty}>Nenhum item na DLQ no momento.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>ID</th>
                <th>Tipo</th>
                <th>Criado em</th>
                <th>Tentativas</th>
                <th>Falhou em</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.id}>
                  <td className={styles.id}>{it.id}</td>
                  <td>{it.type}</td>
                  <td>{new Date(it.created_at).toLocaleString()}</td>
                  <td>{it.retry_count}</td>
                  <td>{new Date(it.failed_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
