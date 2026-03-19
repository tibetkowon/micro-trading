import { useState, useEffect, useRef, useCallback } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

function fmtKRW(n) {
  if (n == null || n === '') return '-'
  return Number(n).toLocaleString('ko-KR') + '원'
}
function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}
function pctBetween(a, b) {
  if (!a || !b || b === 0) return null
  return ((a - b) / b * 100).toFixed(2)
}

/* ── 새로고침 주기 옵션 ── */
const INTERVAL_OPTIONS = [
  { label: '수동', value: 0 },
  { label: '5초', value: 5 },
  { label: '10초', value: 10 },
  { label: '30초', value: 30 },
  { label: '1분', value: 60 },
]

function SectionTitle({ children }) {
  return <p className="text-xs font-semibold text-th-on-subtle uppercase tracking-widest mb-3">{children}</p>
}
SectionTitle.propTypes = { children: PropTypes.node }

export default function Monitor() {
  const { data, loading, error, refetch } = useApi('/api/monitor/positions')
  const [removingCodes, setRemovingCodes] = useState(new Set())
  const [intervalSec, setIntervalSec] = useState(0)
  const timerRef = useRef(null)

  /* 자동 새로고침 */
  const startTimer = useCallback((sec) => {
    if (timerRef.current) clearInterval(timerRef.current)
    if (sec > 0) {
      timerRef.current = setInterval(() => refetch(), sec * 1000)
    }
  }, [refetch])

  useEffect(() => {
    startTimer(intervalSec)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [intervalSec, startTimer])

  const positions = data?.positions || []

  async function handleRemove(code) {
    if (!confirm(`${code} 모니터링을 해제하시겠습니까?\n해제 시 자동 매도는 일어나지 않습니다.`)) return
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
    <div className="space-y-5">
      {/* 헤더 */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold text-th-on-surface">모니터링</h1>
          <p className="text-xs text-th-on-muted mt-0.5">
            {loading ? '로딩 중...' : `모니터링 중 ${positions.length}개`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* 새로고침 주기 선택 */}
          <div className="flex items-center gap-1 bg-th-surface border border-th-outline rounded-lg p-1">
            {INTERVAL_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                onClick={() => setIntervalSec(opt.value)}
                className={`px-2.5 py-1 text-xs rounded-md transition-colors ${
                  intervalSec === opt.value
                    ? 'bg-th-surface-high text-th-on-surface font-medium'
                    : 'text-th-on-muted hover:text-th-on-surface'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
          <button
            onClick={refetch}
            className="text-sm px-3 py-2 bg-th-surface hover:bg-th-surface-high border border-th-outline rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            새로고침
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-th-loss/10 border border-th-loss/20 text-th-loss rounded-xl p-4 text-sm">{error}</div>
      )}

      {loading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : positions.length === 0 ? (
        <div className="bg-th-surface border border-th-outline rounded-xl p-12 text-center">
          <p className="text-th-on-muted font-medium">모니터링 중인 포지션이 없습니다</p>
          <p className="text-xs text-th-on-subtle mt-2">
            주문 시 <code className="bg-th-surface-high px-1 rounded">target_pct</code>와{' '}
            <code className="bg-th-surface-high px-1 rounded">stop_pct</code> 설정 시 체결 후 자동 등록됩니다.
          </p>
        </div>
      ) : (
        <>
          {/* 데스크탑 테이블 */}
          <div className="hidden sm:block bg-th-surface border border-th-outline rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-th-outline text-xs text-th-on-subtle">
                  <th className="text-left px-5 py-3 font-medium">종목</th>
                  <th className="text-left px-5 py-3 font-medium">시장</th>
                  <th className="text-right px-5 py-3 font-medium">체결가</th>
                  <th className="text-right px-5 py-3 font-medium">목표가</th>
                  <th className="text-right px-5 py-3 font-medium">손절가</th>
                  <th className="text-center px-5 py-3 font-medium">목표 수익률</th>
                  <th className="text-center px-5 py-3 font-medium">손절 비율</th>
                  <th className="text-left px-5 py-3 font-medium">등록시각</th>
                  <th className="px-5 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-th-outline">
                {positions.map((p) => {
                  const targetPct = pctBetween(p.target_price, p.filled_price)
                  const stopPct = pctBetween(p.filled_price, p.stop_price)
                  const isRemoving = removingCodes.has(p.stock_code)
                  return (
                    <tr key={p.stock_code} className="hover:bg-th-surface-high transition-colors">
                      <td className="px-5 py-4">
                        <span className="font-medium text-th-on-surface">{p.stock_name || p.stock_code}</span>
                        {p.stock_name && (
                          <span className="ml-1.5 text-xs text-th-on-subtle font-data">{p.stock_code}</span>
                        )}
                      </td>
                      <td className="px-5 py-4">
                        {p.market === 'US' ? (
                          <span className="badge bg-[#7C3AED]/10 text-[#7C3AED] border-[#7C3AED]/20 dark:bg-[#7C3AED]/15 dark:text-[#A78BFA] dark:border-[#7C3AED]/30">미장</span>
                        ) : (
                          <span className="badge bg-th-surface-high text-th-on-muted border-th-outline">국장</span>
                        )}
                      </td>
                      <td className="px-5 py-4 text-right text-th-on-muted font-data">{fmtKRW(p.filled_price)}</td>
                      <td className="px-5 py-4 text-right text-th-loss font-medium font-data">{fmtKRW(p.target_price)}</td>
                      <td className="px-5 py-4 text-right text-[#3B82F6] font-medium font-data">{fmtKRW(p.stop_price)}</td>
                      <td className="px-5 py-4 text-center">
                        {targetPct !== null ? (
                          <span className="badge bg-th-loss/10 text-th-loss border-th-loss/20 font-data">+{targetPct}%</span>
                        ) : '-'}
                      </td>
                      <td className="px-5 py-4 text-center">
                        {stopPct !== null ? (
                          <span className="badge bg-[#3B82F6]/10 text-[#3B82F6] border-[#3B82F6]/20 font-data">-{stopPct}%</span>
                        ) : '-'}
                      </td>
                      <td className="px-5 py-4 text-th-on-subtle text-xs">{fmtDate(p.created_at)}</td>
                      <td className="px-5 py-4">
                        <button
                          onClick={() => handleRemove(p.stock_code)}
                          disabled={isRemoving}
                          className="text-xs px-3 py-1 text-th-on-subtle hover:text-th-loss hover:bg-th-loss/10 rounded-full disabled:opacity-40 transition-colors border border-transparent hover:border-th-loss/20"
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

          {/* 모바일 카드 */}
          <div className="sm:hidden space-y-3">
            {positions.map((p) => {
              const isRemoving = removingCodes.has(p.stock_code)
              return (
                <div key={p.stock_code} className="bg-th-surface border border-th-outline rounded-xl p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <span className="font-medium text-th-on-surface">{p.stock_name || p.stock_code}</span>
                      {p.stock_name && (
                        <span className="ml-1.5 text-xs text-th-on-subtle font-data">{p.stock_code}</span>
                      )}
                    </div>
                    <button
                      onClick={() => handleRemove(p.stock_code)}
                      disabled={isRemoving}
                      className="text-xs px-2.5 py-1 text-th-on-subtle hover:text-th-loss hover:bg-th-loss/10 rounded-full disabled:opacity-40 transition-colors"
                    >
                      {isRemoving ? '...' : '해제'}
                    </button>
                  </div>
                  <div className="grid grid-cols-3 gap-3 text-xs">
                    <div>
                      <p className="text-th-on-subtle mb-1">체결가</p>
                      <p className="text-th-on-muted font-data">{fmtKRW(p.filled_price)}</p>
                    </div>
                    <div>
                      <p className="text-th-on-subtle mb-1">목표가</p>
                      <p className="text-th-loss font-medium font-data">{fmtKRW(p.target_price)}</p>
                    </div>
                    <div>
                      <p className="text-th-on-subtle mb-1">손절가</p>
                      <p className="text-[#3B82F6] font-medium font-data">{fmtKRW(p.stop_price)}</p>
                    </div>
                  </div>
                  <p className="text-xs text-th-on-subtle mt-3">{fmtDate(p.created_at)}</p>
                </div>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}
