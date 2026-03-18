import { useState } from 'react'
import { useApi } from '../hooks/useApi'

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
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
      setDeletingIds((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  if (error) return (
    <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl p-4 text-sm">{error}</div>
  )
  if (loading) return <p className="text-zinc-500">로딩 중...</p>
  if (logs.length === 0) return <p className="text-zinc-500">기록된 KIS API 에러가 없습니다.</p>

  return (
    <div className="space-y-2">
      {logs.map((log) => (
        <div key={log.id} className="bg-zinc-900 border border-red-500/20 rounded-xl px-4 py-3">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="flex items-center gap-2 min-w-0">
              <span className="inline-block px-2.5 py-0.5 rounded-full text-xs border bg-red-500/15 text-red-400 border-red-500/20 shrink-0 font-mono">
                {log.error_code || 'ERR'}
              </span>
              <span className="text-xs text-zinc-500 font-mono truncate">{log.endpoint}</span>
            </div>
            <div className="flex items-center gap-3 shrink-0">
              <span className="text-xs text-zinc-600">{fmtDate(log.timestamp)}</span>
              <button
                onClick={() => handleDelete(log.id)}
                disabled={deletingIds.has(log.id)}
                className="text-xs px-2.5 py-0.5 text-zinc-500 hover:text-red-400 hover:bg-red-500/10 rounded-full disabled:opacity-40 transition-colors"
              >
                {deletingIds.has(log.id) ? '...' : '삭제'}
              </button>
            </div>
          </div>
          {log.error_message && (
            <p className="text-xs text-zinc-400 mt-1.5">{log.error_message}</p>
          )}
          {log.raw_response && (
            <details className="mt-1.5">
              <summary className="text-xs text-zinc-600 cursor-pointer hover:text-zinc-400">
                Raw Response
              </summary>
              <pre className="mt-1 text-xs text-zinc-500 bg-zinc-950 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
                {log.raw_response}
              </pre>
            </details>
          )}
        </div>
      ))}
    </div>
  )
}

const SOURCE_OPTIONS = ['ALL', 'TRADER', 'MONITOR', 'SYSTEM']

function ServiceLogsTab() {
  const [source, setSource] = useState('ALL')
  const { data, loading, error } = useApi(`/api/logs/service?limit=200&source=${source}`)

  const logs = data?.logs || []

  if (error) return (
    <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl p-4 text-sm">{error}</div>
  )

  return (
    <div>
      <div className="flex items-center gap-2 mb-4">
        {SOURCE_OPTIONS.map((s) => (
          <button
            key={s}
            onClick={() => setSource(s)}
            className={`text-xs px-3 py-1.5 rounded-full border transition-colors ${
              source === s
                ? 'bg-zinc-800 text-white border-zinc-700'
                : 'text-zinc-500 border-zinc-800 hover:text-white hover:border-zinc-700'
            }`}
          >
            {s}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="text-zinc-500">로딩 중...</p>
      ) : logs.length === 0 ? (
        <p className="text-zinc-500">기록된 서비스 로그가 없습니다.</p>
      ) : (
        <div className="space-y-2">
          {logs.map((log) => (
            <div
              key={log.id}
              className={`bg-zinc-900 border rounded-xl px-4 py-3 ${
                log.level === 'ERROR' ? 'border-red-500/20' : 'border-yellow-500/20'
              }`}
            >
              <div className="flex items-center gap-2 flex-wrap">
                <span
                  className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium border shrink-0 ${
                    log.level === 'ERROR'
                      ? 'bg-red-500/15 text-red-400 border-red-500/20'
                      : 'bg-yellow-500/15 text-yellow-400 border-yellow-500/20'
                  }`}
                >
                  {log.level}
                </span>
                <span className="inline-block px-2.5 py-0.5 rounded-full text-xs border bg-zinc-700/50 text-zinc-400 border-zinc-700 shrink-0">
                  {log.source}
                </span>
                <span className="text-xs text-zinc-200 flex-1 min-w-0">{log.message}</span>
                <span className="text-xs text-zinc-600 shrink-0">{fmtDate(log.timestamp)}</span>
              </div>
              {log.detail && (
                <details className="mt-1.5">
                  <summary className="text-xs text-zinc-600 cursor-pointer hover:text-zinc-400">
                    상세
                  </summary>
                  <pre className="mt-1 text-xs text-zinc-500 bg-zinc-950 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
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

export default function ErrorLogs() {
  const [tab, setTab] = useState('service')

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-white">에러 로그</h1>
        <p className="text-sm text-zinc-500 mt-0.5">서비스 운영 이벤트 및 KIS API 오류</p>
      </div>

      <div className="flex gap-1 mb-6 border-b border-zinc-800">
        {[
          { key: 'service', label: '서비스 로그' },
          { key: 'kis', label: 'KIS API 에러' },
        ].map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
              tab === key
                ? 'border-white text-white'
                : 'border-transparent text-zinc-500 hover:text-zinc-300'
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
