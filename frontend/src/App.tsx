import { Routes, Route, NavLink } from 'react-router-dom'
import { Config } from './pages/Config'
import { Dashboard } from './pages/Dashboard'
import { DLQ } from './pages/DLQ'
import { Analysis } from './pages/Analysis'
import styles from './App.module.css'

function Nav() {
  return (
    <nav className={styles.nav}>
      <NavLink to="/" className={({ isActive }) => (isActive ? styles.active : '')} end>
        Configuração
      </NavLink>
      <NavLink to="/dashboard" className={({ isActive }) => (isActive ? styles.active : '')}>
        Dashboard
      </NavLink>
      <NavLink to="/dlq" className={({ isActive }) => (isActive ? styles.active : '')}>
        DLQ
      </NavLink>
      <NavLink to="/analysis" className={({ isActive }) => (isActive ? styles.active : '')}>
        Análise
      </NavLink>
    </nav>
  )
}

export default function App() {
  return (
    <div className={styles.app}>
      <header className={styles.header}>
        <h1 className={styles.title}>Simulador de Fila FIFO</h1>
        <Nav />
      </header>
      <main className={styles.main}>
        <Routes>
          <Route path="/" element={<Config />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/dlq" element={<DLQ />} />
          <Route path="/analysis" element={<Analysis />} />
        </Routes>
      </main>
    </div>
  )
}
