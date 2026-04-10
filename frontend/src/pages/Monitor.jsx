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

const INTERVAL_OPTIONS = [
  { label: '수동', value: 0 },
  { label: '5초', value: 5 },
  { label: '10초', value: 10 },
  { label: '30초', value: 30 },
  { label: '1분', value: 60 },
]

function SectionLabel({ children }) {
  return <p className="text-[10px] text-th-on-subtle uppercase tracking-widest mb-1">{children}</p>
}
SectionLabel.propTypes = { children: PropTypes.node }

export default function Monitor() {
  const { data, loading, error, refetch } = useApi('/api/monitor/positions')
  const [removingCodes, setRemovingCodes] = useState(new Set())
  const [sellingCodes, setSellingCodes] = useState(new Set())
  const [intervalSec, setIntervalSec] = useState(0)
  const timerRef = useRef(null)

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

  async function handlePanicSell() {
    if (!window.confirm('전체 보유 종목을 즉시 시장가 매도합니다. 계속하시겠습니까?')) return
    if (!window.confirm('정말로 전량 매도하시겠습니까? 되돌릴 수 없습니다.')) return
    try {
      await fetch('/api/monitor/liquidate-all', { method: 'POST' })
      setTimeout(() => refetch(), 1500) // 주문 완료 후 짧은 딜레이
    } catch (err) {
      alert(`전체 매도 오류: ${err.message}`)
    }
  }

  async function handleForceSell(code, name) {
    if (!confirm(`[강제매도] ${name || code}\n\n현재 보유 수량 전량을 시장가로 즉시 매도하고 모니터링에서 해제합니다.\n계속하시겠습니까?`)) return
    setSellingCodes((prev) => new Set(prev).add(code))
    try {
      const res = await fetch(`/api/monitor/positions/${code}/sell`, { method: 'POST' })
      const json = await res.json()
      if (!res.ok) {
        alert(`강제매도 실패: ${json.error || '알 수 없는 오류'}`)
      }
      refetch()
    } catch (err) {
      alert(`강제매도 오류: ${err.message}`)
    } finally {
      setSellingCodes((prev) => {
        const next = new Set(prev)
        next.delete(code)
        return next
      })
    }
  }

  return (
    <div className="space-y-6">
      {/* 헤더 */}
      <div className="flex items-center justify-between flex-wrap gap-3 pt-2">
        <div>
          <h1 className="text-2xl font-bold text-th-on-surface tracking-tight">모니터링</h1>
          <p className="text-xs text-th-on-muted mt-0.5 uppercase tracking-widest">
            {loading ? '로딩 중...' : `모니터링 중 ${positions.length}개`}
          </p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          {/* 새로고침 주기 */}
          <div className="flex items-center gap-0.5 bg-th-surface rounded-lg p-1">
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
            className="flex items-center gap-1.5 text-xs px-3 py-2 bg-th-surface hover:bg-th-surface-high rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            <span className="material-symbols-outlined text-[16px]">refresh</span>
            새로고침
          </button>
          {positions.length > 0 && (
            <button
              onClick={handlePanicSell}
              className="flex items-center gap-1.5 text-xs px-3 py-2 bg-red-500 hover:bg-red-600 text-white font-semibold rounded-lg transition-colors"
            >
              <span className="material-symbols-outlined text-[16px]">warning</span>
              전체 매도
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="bg-red-500/10 rounded-xl p-4 text-sm text-red-400">{error}</div>
      )}

      {loading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : positions.length === 0 ? (
        <div className="bg-th-surface rounded-xl p-12 text-center">
          <span className="material-symbols-outlined text-[40px] text-th-on-subtle block mb-2">monitor_heart</span>
          <p className="text-th-on-muted font-medium">모니터링 중인 포지션이 없습니다</p>
          <p className="text-xs text-th-on-subtle mt-2">
            주문 체결 후 <code className="bg-th-surface-high px-1 rounded text-th-on-muted">target_pct</code>와{' '}
            <code className="bg-th-surface-high px-1 rounded text-th-on-muted">stop_pct</code> 설정 시 자동 등록됩니다.
          </p>
        </div>
      ) : (
        <>
          {/* 데스크탑 테이블 */}
          <div className="hidden sm:block bg-th-surface-low rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-[10px] text-th-on-subtle uppercase tracking-widest">
                  <th className="text-left px-5 py-3.5 font-medium">종목</th>
                  <th className="text-left px-5 py-3.5 font-medium">시장</th>
                  <th className="text-right px-5 py-3.5 font-medium">체결가</th>
                  <th className="text-right px-5 py-3.5 font-medium">목표가</th>
                  <th className="text-right px-5 py-3.5 font-medium">손절가</th>
                  <th className="text-center px-5 py-3.5 font-medium">목표 수익률</th>
                  <th className="text-center px-5 py-3.5 font-medium">손절 비율</th>
                  <th className="text-left px-5 py-3.5 font-medium">등록시각</th>
                  <th className="px-5 py-3.5"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {positions.map((p) => {
                  const targetPct = pctBetween(p.target_price, p.filled_price)
                  const stopPct = pctBetween(p.filled_price, p.stop_price)
                  const isRemoving = removingCodes.has(p.stock_code)
                  return (
                    <tr key={p.stock_code} className="hover:bg-white/[0.02] transition-colors">
                      <td className="px-5 py-4">
                        <span className="font-medium text-th-on-surface">{p.stock_name || p.stock_code}</span>
                        {p.stock_name && (
                          <span className="ml-2 text-xs text-th-on-subtle font-data">{p.stock_code}</span>
                        )}
                      </td>
                      <td className="px-5 py-4">
                        {p.market === 'US' ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-orange-500/10 text-orange-400">해외</span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-blue-500/10 text-blue-400">국내</span>
                        )}
                      </td>
                      <td className="px-5 py-4 text-right text-th-on-muted font-data">{fmtKRW(p.filled_price)}</td>
                      <td className="px-5 py-4 text-right text-red-400 font-medium font-data">{fmtKRW(p.target_price)}</td>
                      <td className="px-5 py-4 text-right text-blue-400 font-medium font-data">{fmtKRW(p.stop_price)}</td>
                      <td className="px-5 py-4 text-center">
                        {targetPct !== null ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-red-500/10 text-red-400 font-data">+{targetPct}%</span>
                        ) : '-'}
                      </td>
                      <td className="px-5 py-4 text-center">
                        {stopPct !== null ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-blue-500/10 text-blue-400 font-data">-{stopPct}%</span>
                        ) : '-'}
                      </td>
                      <td className="px-5 py-4 text-th-on-subtle text-xs">{fmtDate(p.created_at)}</td>
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-1.5">
                          <button
                            onClick={() => handleForceSell(p.stock_code, p.stock_name)}
                            disabled={sellingCodes.has(p.stock_code) || isRemoving}
                            className="text-xs px-3 py-1 text-red-400 hover:text-white hover:bg-red-500 border border-red-500/30 rounded-full disabled:opacity-40 transition-colors"
                          >
                            {sellingCodes.has(p.stock_code) ? '...' : '강제매도'}
                          </button>
                          <button
                            onClick={() => handleRemove(p.stock_code)}
                            disabled={isRemoving || sellingCodes.has(p.stock_code)}
                            className="text-xs px-3 py-1 text-th-on-muted hover:text-th-on-surface hover:bg-th-surface-high rounded-full disabled:opacity-40 transition-colors"
                          >
                            {isRemoving ? '...' : '해제'}
                          </button>
                        </div>
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
              const isSelling = sellingCodes.has(p.stock_code)
              return (
                <div key={p.stock_code} className="bg-th-surface rounded-xl p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <span className="font-medium text-th-on-surface">{p.stock_name || p.stock_code}</span>
                      {p.stock_name && (
                        <span className="ml-2 text-xs text-th-on-subtle font-data">{p.stock_code}</span>
                      )}
                    </div>
                    <div className="flex items-center gap-1.5">
                      <button
                        onClick={() => handleForceSell(p.stock_code, p.stock_name)}
                        disabled={isSelling || isRemoving}
                        className="text-xs px-2.5 py-1 text-red-400 hover:text-white hover:bg-red-500 border border-red-500/30 rounded-full disabled:opacity-40 transition-colors"
                      >
                        {isSelling ? '...' : '강제매도'}
                      </button>
                      <button
                        onClick={() => handleRemove(p.stock_code)}
                        disabled={isRemoving || isSelling}
                        className="text-xs px-2.5 py-1 text-th-on-muted hover:text-th-on-surface hover:bg-th-surface-high rounded-full disabled:opacity-40 transition-colors"
                      >
                        {isRemoving ? '...' : '해제'}
                      </button>
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-3 text-xs">
                    <div>
                      <SectionLabel>체결가</SectionLabel>
                      <p className="text-th-on-muted font-data">{fmtKRW(p.filled_price)}</p>
                    </div>
                    <div>
                      <SectionLabel>목표가</SectionLabel>
                      <p className="text-red-400 font-medium font-data">{fmtKRW(p.target_price)}</p>
                    </div>
                    <div>
                      <SectionLabel>손절가</SectionLabel>
                      <p className="text-blue-400 font-medium font-data">{fmtKRW(p.stop_price)}</p>
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
