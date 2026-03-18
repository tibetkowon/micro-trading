import { useState } from 'react'
import { useApi } from '../hooks/useApi'

function fmt(n) {
  if (!n && n !== 0) return '-'
  return Number(n).toLocaleString('ko-KR') + '원'
}

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

function pct(a, b) {
  if (!a || !b || b === 0) return null
  return ((a - b) / b * 100).toFixed(2)
}

export default function Monitor() {
  const { data, loading, error, refetch } = useApi('/api/monitor/positions')
  const [removingCodes, setRemovingCodes] = useState(new Set())

  const positions = data?.positions || []

  async function handleRemove(code) {
    setRemovingCodes((prev) => new Set(prev).add(code))
    try {
      await fetch(`/api/monitor/positions/${code}`, { method: 'DELETE' })
      refetch()
    } finally {
      setRemovingCodes((prev) => {
        const next = new Set(prev)
        next.delete(code)
        return next
      })
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-white">실시간 모니터</h1>
          {!loading && (
            <p className="text-sm text-zinc-500 mt-0.5">모니터링 중 {positions.length}개</p>
          )}
        </div>
        <button
          onClick={refetch}
          className="text-sm px-4 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
        >
          새로고침
        </button>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl p-4 mb-4 text-sm">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-zinc-500">로딩 중...</p>
      ) : positions.length === 0 ? (
        <div className="text-center py-16 text-zinc-500 border border-zinc-800 rounded-xl">
          <p className="font-medium">모니터링 중인 포지션이 없습니다</p>
          <p className="text-sm mt-2 text-zinc-600">
            주문 시 <code className="bg-zinc-800 px-1 rounded">target_pct</code>와{' '}
            <code className="bg-zinc-800 px-1 rounded">stop_pct</code>를 포함하면 체결 후 자동 등록됩니다.
          </p>
        </div>
      ) : (
        <>
        {/* Desktop table */}
        <div className="hidden sm:block bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-800 text-xs text-zinc-500">
                <th className="text-left px-5 py-3 font-medium">종목</th>
                <th className="text-right px-5 py-3 font-medium">체결가</th>
                <th className="text-right px-5 py-3 font-medium">목표가</th>
                <th className="text-right px-5 py-3 font-medium">손절가</th>
                <th className="text-center px-5 py-3 font-medium">목표 수익률</th>
                <th className="text-center px-5 py-3 font-medium">손절 비율</th>
                <th className="text-left px-5 py-3 font-medium">등록시각</th>
                <th className="px-5 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/60">
              {positions.map((p) => {
                const targetPct = pct(p.target_price, p.filled_price)
                const stopPct = pct(p.filled_price, p.stop_price)
                const isRemoving = removingCodes.has(p.stock_code)
                return (
                  <tr key={p.stock_code} className="hover:bg-zinc-800/40 transition-colors">
                    <td className="px-5 py-4">
                      <span className="font-medium text-white">{p.stock_name || p.stock_code}</span>
                      {p.stock_name && (
                        <span className="ml-1.5 text-xs text-zinc-500 font-mono">{p.stock_code}</span>
                      )}
                    </td>
                    <td className="px-5 py-4 text-right text-zinc-400">{fmt(p.filled_price)}</td>
                    {/* 목표가: 상승 = 빨강 */}
                    <td className="px-5 py-4 text-right text-red-400 font-medium">
                      {fmt(p.target_price)}
                    </td>
                    {/* 손절가: 하락 = 파랑 */}
                    <td className="px-5 py-4 text-right text-blue-400 font-medium">
                      {fmt(p.stop_price)}
                    </td>
                    <td className="px-5 py-4 text-center">
                      {targetPct !== null ? (
                        <span className="inline-block px-2.5 py-0.5 rounded-full text-xs border bg-red-500/15 text-red-400 border-red-500/20">
                          +{targetPct}%
                        </span>
                      ) : '-'}
                    </td>
                    <td className="px-5 py-4 text-center">
                      {stopPct !== null ? (
                        <span className="inline-block px-2.5 py-0.5 rounded-full text-xs border bg-blue-500/15 text-blue-400 border-blue-500/20">
                          -{stopPct}%
                        </span>
                      ) : '-'}
                    </td>
                    <td className="px-5 py-4 text-zinc-500 text-xs">{fmtDate(p.created_at)}</td>
                    <td className="px-5 py-4">
                      <button
                        onClick={() => handleRemove(p.stock_code)}
                        disabled={isRemoving}
                        className="text-xs px-2.5 py-1 text-zinc-500 hover:text-red-400 hover:bg-red-500/10 rounded-full disabled:opacity-40 transition-colors"
                      >
                        {isRemoving ? '...' : '해제'}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>

        {/* Mobile cards */}
        <div className="sm:hidden grid grid-cols-1 gap-3">
          {positions.map((p) => {
            const isRemoving = removingCodes.has(p.stock_code)
            return (
              <div key={p.stock_code} className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <span className="font-medium text-white">{p.stock_name || p.stock_code}</span>
                    {p.stock_name && (
                      <span className="ml-1.5 text-xs text-zinc-500 font-mono">{p.stock_code}</span>
                    )}
                  </div>
                  <button
                    onClick={() => handleRemove(p.stock_code)}
                    disabled={isRemoving}
                    className="text-xs px-2.5 py-1 text-zinc-500 hover:text-red-400 hover:bg-red-500/10 rounded-full disabled:opacity-40 transition-colors"
                  >
                    {isRemoving ? '...' : '해제'}
                  </button>
                </div>
                <div className="grid grid-cols-3 gap-3 text-xs">
                  <div>
                    <p className="text-zinc-500 mb-1">체결가</p>
                    <p className="text-zinc-300">{fmt(p.filled_price)}</p>
                  </div>
                  <div>
                    <p className="text-zinc-500 mb-1">목표가</p>
                    <p className="text-red-400 font-medium">{fmt(p.target_price)}</p>
                  </div>
                  <div>
                    <p className="text-zinc-500 mb-1">손절가</p>
                    <p className="text-blue-400 font-medium">{fmt(p.stop_price)}</p>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
        </>
      )}
    </div>
  )
}
