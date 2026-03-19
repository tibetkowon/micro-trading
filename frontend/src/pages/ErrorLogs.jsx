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
      {/* 출처 필터 */}
      <div className="flex items-center gap-1 bg-th-surface border border-th-outline rounded-lg p-1 w-fit mb-4">
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

      {error && <div className="bg-th-loss/10 border border-th-loss/20 text-th-loss rounded-xl p-4 text-sm mb-3">{error}</div>}
      {loading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : logs.length === 0 ? (
        <p className="text-th-on-subtle text-sm">기록된 서비스 로그가 없습니다.</p>
      ) : (
        <div className="space-y-2">
          {logs.map((log) => (
            <div
              key={log.id}
              className={`bg-th-surface border rounded-xl px-4 py-3 ${
                log.level === 'ERROR' ? 'border-th-loss/20' : 'border-th-warn/20'
              }`}
            >
              <div className="flex items-center gap-2 flex-wrap">
                <span className={`badge shrink-0 ${
                  log.level === 'ERROR'
                    ? 'bg-th-loss/10 text-th-loss border-th-loss/20'
                    : 'bg-th-warn/10 text-th-warn border-th-warn/20'
                }`}>
                  {log.level}
                </span>
                <span className="badge bg-th-surface-high text-th-on-muted border-th-outline shrink-0">
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
                  <pre className="mt-1.5 text-xs text-th-on-muted bg-th-surface-low rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
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

  if (error) return <div className="bg-th-loss/10 border border-th-loss/20 text-th-loss rounded-xl p-4 text-sm">{error}</div>
  if (loading) return <p className="text-th-on-subtle text-sm">로딩 중...</p>
  if (logs.length === 0) return <p className="text-th-on-subtle text-sm">기록된 KIS API 에러가 없습니다.</p>

  return (
    <div className="space-y-2">
      {logs.map((log) => (
        <div key={log.id} className="bg-th-surface border border-th-loss/20 rounded-xl px-4 py-3">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="flex items-center gap-2 min-w-0">
              <span className="badge bg-th-loss/10 text-th-loss border-th-loss/20 shrink-0 font-data">
                {log.error_code || 'ERR'}
              </span>
              <span className="text-xs text-th-on-subtle font-data truncate">{log.endpoint}</span>
            </div>
            <div className="flex items-center gap-3 shrink-0">
              <span className="text-xs text-th-on-subtle">{fmtDate(log.timestamp)}</span>
              <button
                onClick={() => handleDelete(log.id)}
                disabled={deletingIds.has(log.id)}
                className="text-xs px-2.5 py-0.5 text-th-on-subtle hover:text-th-loss hover:bg-th-loss/10 rounded-full disabled:opacity-40 transition-colors"
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
              <pre className="mt-1 text-xs text-th-on-muted bg-th-surface-low rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
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
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold text-th-on-surface">에러 로그</h1>
        <p className="text-xs text-th-on-muted mt-0.5">서비스 운영 이벤트 및 KIS API 오류</p>
      </div>

      <div className="flex gap-0 border-b border-th-outline">
        {TABS.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
              tab === key
                ? 'border-th-primary text-th-on-surface'
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
