import { useState } from 'react'
import { useApi } from '../hooks/useApi'

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

const SOURCE_OPTIONS = [
  { key: 'ALL', label: '전체' },
  { key: 'TRADER', label: '트레이더' },
  { key: 'MONITOR', label: '모니터' },
  { key: 'SYSTEM', label: '시스템' },
]

function ServiceLogsTab() {
  const [source, setSource] = useState('ALL')
  const { data, loading, error } = useApi(
    `/api/logs/service?limit=200&source=${source}`
  )
  const logs = data?.logs || []

  return (
    <div>
      <div className="flex items-center gap-0.5 bg-th-surface rounded-lg p-1 w-fit mb-5">
        {SOURCE_OPTIONS.map((s) => (
          <button
            key={s.key}
            onClick={() => setSource(s.key)}
            className={`px-3 py-1 text-xs rounded-md transition-colors ${
              source === s.key
                ? 'bg-th-surface-high text-th-on-surface font-medium'
                : 'text-th-on-muted hover:text-th-on-surface'
            }`}
          >
            {s.label}
          </button>
        ))}
      </div>

      {error && <div className="bg-red-500/10 text-red-400 rounded-xl p-4 text-sm mb-3">{error}</div>}
      {loading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : logs.length === 0 ? (
        <div className="bg-th-surface rounded-xl p-8 text-center">
          <span className="material-symbols-outlined text-[36px] text-th-on-subtle block mb-2">check_circle</span>
          <p className="text-th-on-muted text-sm">기록된 서비스 로그가 없습니다.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {logs.map((log) => (
            <div
              key={log.id}
              className="rounded-xl px-4 py-3 border-l-2 bg-th-surface border-red-500/50"
            >
              <div className="flex items-center gap-2 flex-wrap">
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium shrink-0 bg-red-500/10 text-red-400">
                  ERROR
                </span>
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-white/5 text-gray-500 shrink-0">
                  {log.source}
                </span>
                <span className="text-sm text-th-on-surface flex-1 min-w-0">{log.message}</span>
                <span className="text-xs text-th-on-subtle shrink-0">{fmtDate(log.timestamp)}</span>
              </div>
              {log.detail && (
                <details className="mt-2">
                  <summary className="text-xs text-th-on-subtle cursor-pointer hover:text-th-on-muted select-none">
                    상세 보기
                  </summary>
                  <pre className="mt-1.5 text-xs text-th-on-muted bg-th-bg rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
                    {log.detail}
                  </pre>
                </details>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function KISLogsTab() {
  const { data, loading, error, refetch } = useApi('/api/logs/kis?limit=100')
  const [deletingIds, setDeletingIds] = useState(new Set())
  const logs = data?.logs || []

  async function handleDelete(id) {
    setDeletingIds((prev) => new Set(prev).add(id))
    try {
      await fetch(`/api/logs/kis/${id}`, { method: 'DELETE' })
      refetch()
    } finally {
      setDeletingIds((prev) => { const n = new Set(prev); n.delete(id); return n })
    }
  }

  if (error) return <div className="bg-red-500/10 text-red-400 rounded-xl p-4 text-sm">{error}</div>
  if (loading) return <p className="text-th-on-subtle text-sm">로딩 중...</p>
  if (logs.length === 0) return (
    <div className="bg-th-surface rounded-xl p-8 text-center">
      <span className="material-symbols-outlined text-[36px] text-th-on-subtle block mb-2">check_circle</span>
      <p className="text-th-on-muted text-sm">기록된 KIS API 에러가 없습니다.</p>
    </div>
  )

  return (
    <div className="space-y-2">
      {logs.map((log) => (
        <div key={log.id} className="bg-th-surface border-l-2 border-red-500/40 rounded-xl px-4 py-3">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="flex items-center gap-2 min-w-0">
              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium bg-red-500/10 text-red-400 shrink-0 font-data">
                {log.error_code || 'ERR'}
              </span>
              <span className="text-xs text-th-on-muted font-data truncate">{log.endpoint}</span>
            </div>
            <div className="flex items-center gap-3 shrink-0">
              <span className="text-xs text-th-on-subtle">{fmtDate(log.timestamp)}</span>
              <button
                onClick={() => handleDelete(log.id)}
                disabled={deletingIds.has(log.id)}
                className="text-xs px-2.5 py-0.5 text-th-on-muted hover:text-red-400 hover:bg-red-500/10 rounded-full disabled:opacity-40 transition-colors"
              >
                {deletingIds.has(log.id) ? '...' : '삭제'}
              </button>
            </div>
          </div>
          {log.error_message && (
            <p className="text-xs text-th-on-muted mt-1.5">{log.error_message}</p>
          )}
          {log.raw_response && (
            <details className="mt-1.5">
              <summary className="text-xs text-th-on-subtle cursor-pointer hover:text-th-on-muted select-none">
                Raw Response
              </summary>
              <pre className="mt-1 text-xs text-th-on-muted bg-th-bg rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
                {log.raw_response}
              </pre>
            </details>
          )}
        </div>
      ))}
    </div>
  )
}

const TABS = [
  { key: 'service', label: '서비스 로그' },
  { key: 'kis', label: 'KIS API 에러' },
]

export default function ErrorLogs() {
  const [tab, setTab] = useState('service')

  return (
    <div className="space-y-6">
      <div className="pt-2">
        <h1 className="text-2xl font-bold text-th-on-surface tracking-tight">에러 로그</h1>
        <p className="text-xs text-th-on-muted mt-0.5 uppercase tracking-widest">서비스 운영 이벤트 및 KIS API 오류</p>
      </div>

      <div className="flex gap-0 border-b border-black/5 dark:border-white/5">
        {TABS.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
              tab === key
                ? 'border-orange-500 text-th-on-surface'
                : 'border-transparent text-th-on-muted hover:text-th-on-surface'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === 'service' ? <ServiceLogsTab /> : <KISLogsTab />}
    </div>
  )
}
