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
    <div className="bg-red-900/30 border border-red-700 text-red-300 rounded p-4 text-sm">{error}</div>
  )
  if (loading) return <p className="text-gray-500">로딩 중...</p>
  if (logs.length === 0) return <p className="text-gray-500">기록된 KIS API 에러가 없습니다.</p>

  return (
    <div className="space-y-2">
      {logs.map((log) => (
        <div key={log.id} className="bg-gray-900 border border-red-900/40 rounded-lg px-3 py-2.5">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="flex items-center gap-2 min-w-0">
              <span className="text-xs bg-red-900/60 text-red-300 px-1.5 py-0.5 rounded font-mono shrink-0">
                {log.error_code || 'ERR'}
              </span>
              <span className="text-xs text-gray-500 font-mono truncate">{log.endpoint}</span>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <span className="text-xs text-gray-600">{fmtDate(log.timestamp)}</span>
              <button
                onClick={() => handleDelete(log.id)}
                disabled={deletingIds.has(log.id)}
                className="text-xs px-1.5 py-0.5 text-gray-500 hover:text-red-400 hover:bg-red-900/20 rounded disabled:opacity-40 transition-colors"
              >
                {deletingIds.has(log.id) ? '...' : '삭제'}
              </button>
            </div>
          </div>
          {log.error_message && (
            <p className="text-xs text-gray-400 mt-1">{log.error_message}</p>
          )}
          {log.raw_response && (
            <details className="mt-1">
              <summary className="text-xs text-gray-600 cursor-pointer hover:text-gray-400">
                Raw Response
              </summary>
              <pre className="mt-1 text-xs text-gray-500 bg-gray-950 rounded p-2 overflow-x-auto whitespace-pre-wrap">
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
  const { data, loading, error, refetch } = useApi(`/api/logs/service?limit=200&source=${source}`)

  const logs = data?.logs || []

  if (error) return (
    <div className="bg-red-900/30 border border-red-700 text-red-300 rounded p-4 text-sm">{error}</div>
  )

  return (
    <div>
      <div className="flex items-center gap-2 mb-4">
        {SOURCE_OPTIONS.map((s) => (
          <button
            key={s}
            onClick={() => setSource(s)}
            className={`text-xs px-2.5 py-1 rounded transition-colors ${
              source === s
                ? 'bg-gray-700 text-white'
                : 'text-gray-400 hover:text-white hover:bg-gray-800'
            }`}
          >
            {s}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="text-gray-500">로딩 중...</p>
      ) : logs.length === 0 ? (
        <p className="text-gray-500">기록된 서비스 로그가 없습니다.</p>
      ) : (
        <div className="space-y-2">
          {logs.map((log) => (
            <div
              key={log.id}
              className={`bg-gray-900 border rounded-lg px-3 py-2.5 ${
                log.level === 'ERROR' ? 'border-red-900/40' : 'border-yellow-900/40'
              }`}
            >
              <div className="flex items-center gap-2 flex-wrap">
                <span
                  className={`text-xs px-1.5 py-0.5 rounded font-mono shrink-0 ${
                    log.level === 'ERROR'
                      ? 'bg-red-900/60 text-red-300'
                      : 'bg-yellow-900/60 text-yellow-300'
                  }`}
                >
                  {log.level}
                </span>
                <span className="text-xs text-gray-500 shrink-0">{log.source}</span>
                <span className="text-xs text-gray-300 flex-1 min-w-0">{log.message}</span>
                <span className="text-xs text-gray-600 shrink-0">{fmtDate(log.timestamp)}</span>
              </div>
              {log.detail && (
                <details className="mt-1">
                  <summary className="text-xs text-gray-600 cursor-pointer hover:text-gray-400">
                    상세
                  </summary>
                  <pre className="mt-1 text-xs text-gray-500 bg-gray-950 rounded p-2 overflow-x-auto whitespace-pre-wrap">
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
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">에러 로그</h1>
      </div>

      <div className="flex gap-1 mb-6 border-b border-gray-800">
        {[
          { key: 'service', label: '서비스 로그' },
          { key: 'kis', label: 'KIS API 에러' },
        ].map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
              tab === key
                ? 'border-blue-500 text-white'
                : 'border-transparent text-gray-400 hover:text-white'
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
