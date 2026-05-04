import { useEffect } from 'react'

export function fmt(v) {
  if (v == null) return '—'
  return '₩' + Math.round(v).toLocaleString('ko-KR')
}

export function fmtPct(v) {
  if (v == null) return '—'
  return (v >= 0 ? '+' : '') + v.toFixed(2) + '%'
}

export function fmtSigned(v) {
  if (v == null) return '—'
  const sign = v >= 0 ? '+' : '-'
  return sign + '₩' + Math.round(Math.abs(v)).toLocaleString('ko-KR')
}

export function Modal({ title, children, actions, onClose }) {
  useEffect(() => {
    const h = e => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onClose])
  return (
    <div className="modal-overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="modal">
        <div className="modal-title">{title}</div>
        <div className="modal-body">{children}</div>
        <div className="modal-actions">{actions}</div>
      </div>
    </div>
  )
}

export function Toggle({ checked, onChange }) {
  return (
    <label className="toggle">
      <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)} />
      <div className="toggle-track"></div>
      <div className="toggle-thumb"></div>
    </label>
  )
}

const BADGE_COLORS = {
  green:  { bg: 'rgba(34,197,94,0.15)',   text: '#22C55E' },
  red:    { bg: 'rgba(239,68,68,0.15)',   text: '#EF4444' },
  yellow: { bg: 'rgba(245,158,11,0.15)', text: '#F59E0B' },
  orange: { bg: 'rgba(234,108,16,0.18)', text: '#EA6C10' },
  gray:   { bg: 'rgba(122,143,166,0.15)', text: '#7A8FA6' },
  blue:   { bg: 'rgba(59,130,246,0.15)',  text: '#3B82F6' },
  purple: { bg: 'rgba(139,92,246,0.15)', text: '#8B5CF6' },
}

export function Badge({ children, color }) {
  const c = BADGE_COLORS[color] || BADGE_COLORS.gray
  return (
    <span className="badge" style={{ background: c.bg, color: c.text }}>{children}</span>
  )
}

const BOT_STATE_CONFIG = {
  IDLE:          { label: 'IDLE',         bg: 'rgba(122,143,166,0.15)', dot: '#7A8FA6', text: '#7A8FA6' },
  SELECTING:     { label: 'SELECTING',    bg: 'rgba(59,130,246,0.15)',  dot: '#3B82F6', text: '#3B82F6' },
  ORDERING:      { label: 'ORDERING',     bg: 'rgba(245,158,11,0.15)', dot: '#F59E0B', text: '#F59E0B' },
  WAITING_FILL:  { label: 'WAITING FILL', bg: 'rgba(234,108,16,0.15)', dot: '#EA6C10', text: '#EA6C10' },
  MONITORING:    { label: 'MONITORING',   bg: 'rgba(34,197,94,0.15)',   dot: '#22C55E', text: '#22C55E' },
}

export function BotStateBadge({ state }) {
  const c = BOT_STATE_CONFIG[state] || BOT_STATE_CONFIG.IDLE
  return (
    <div className="bot-state-badge" style={{ background: c.bg }}>
      <div className="dot" style={{ background: c.dot }}></div>
      <span style={{ color: c.text }}>{c.label}</span>
    </div>
  )
}

export function ProgressBar({ value, color = '#EA6C10', height = 4 }) {
  return (
    <div className="progress-track" style={{ height }}>
      <div className="progress-fill" style={{ width: `${Math.min(100, Math.max(0, value))}%`, background: color, height }}></div>
    </div>
  )
}

export function PriceProgressBar({ stop, avg: _avg, target, current }) {
  const range = target - stop
  const curPct = range > 0 ? ((current - stop) / range) * 100 : 50
  const nearStop = curPct < 15
  return (
    <div style={{ position: 'relative', height: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
      <span className="mono" style={{ fontSize: 10, color: 'var(--down)', whiteSpace: 'nowrap' }}>{fmt(stop)}</span>
      <div style={{ flex: 1, position: 'relative' }}>
        <div className="progress-track" style={{ height: 6 }}>
          <div className="progress-fill" style={{ width: '100%', background: 'linear-gradient(to right, var(--down), var(--up))', height: 6, borderRadius: 3 }}></div>
        </div>
        <div style={{
          position: 'absolute',
          left: `${Math.min(98, Math.max(2, curPct))}%`,
          top: '50%',
          transform: 'translate(-50%,-50%)',
          width: 10, height: 10,
          borderRadius: '50%',
          background: nearStop ? 'var(--down)' : 'white',
          border: '2px solid var(--bg)',
          zIndex: 2,
        }}></div>
      </div>
      <span className="mono" style={{ fontSize: 10, color: 'var(--up)', whiteSpace: 'nowrap' }}>{fmt(target)}</span>
    </div>
  )
}

export function EmptyState({ icon = '📭', message }) {
  return (
    <div className="empty-state">
      <div className="empty-icon">{icon}</div>
      <p>{message}</p>
    </div>
  )
}

export function Spinner() {
  return <div className="spinner"></div>
}
