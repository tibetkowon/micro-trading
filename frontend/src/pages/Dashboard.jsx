import { useState } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

/* ── 숫자 포맷 유틸 ── */
function fmtKRW(n) {
  if (n == null || n === '') return '-'
  return Number(n).toLocaleString('ko-KR') + '원'
}
function fmtUSD(n) {
  if (n == null || n === '') return '-'
  return '$' + Number(n).toLocaleString('en-US', { minimumFractionDigits: 2 })
}
function fmtNum(n) {
  if (n == null || n === '') return '-'
  const v = parseFloat(n)
  return isNaN(v) ? '-' : v.toLocaleString('ko-KR')
}
function fmtRate(r) {
  if (r == null || r === '' || r === '-') return null
  const n = parseFloat(r)
  if (isNaN(n)) return null
  return n
}
// 보유종목 수익률: evlu_erng_rt 가 0이면 pchs_avg_pric / prpr 로 직접 계산
function calcRate(h) {
  const rt = fmtRate(h.evlu_erng_rt)
  if (rt !== null && rt !== 0) return rt
  const avg = parseFloat(h.pchs_avg_pric)
  const cur = parseFloat(h.prpr)
  if (!isNaN(avg) && avg > 0 && !isNaN(cur)) {
    return (cur - avg) / avg * 100
  }
  return rt ?? 0
}

/* ── 상태 점 ── */
function StatusDot({ ok, size = 'sm' }) {
  const dim = size === 'lg' ? 'w-2.5 h-2.5' : 'w-1.5 h-1.5'
  return (
    <span className={`inline-block rounded-full ${dim} ${ok ? 'bg-th-growth' : 'bg-th-on-subtle'}`} />
  )
}
StatusDot.propTypes = { ok: PropTypes.bool, size: PropTypes.string }

/* ── 트레이더 상태 색 ── */
function traderColor(state) {
  if (!state || state === 'IDLE') return 'text-th-on-subtle'
  if (state === 'MONITORING') return 'text-th-loss'
  if (state === 'SEARCHING' || state === 'SELECTING') return 'text-th-growth'
  return 'text-th-warn'
}
function traderLabel(state) {
  const map = {
    IDLE: 'IDLE',
    SELECTING: 'SELECTING (종목탐색)',
    SEARCHING: 'SEARCHING (종목탐색)',
    ORDERING: 'ORDERING (주문중)',
    WAITING_FILL: 'WAITING_FILL (체결대기)',
    MONITORING: 'MONITORING (모니터링)',
  }
  return map[state] || (state || 'IDLE')
}

/* ── 섹션 헤더 ── */
function SectionTitle({ children }) {
  return <p className="text-xs font-semibold text-th-on-subtle uppercase tracking-widest mb-3">{children}</p>
}
SectionTitle.propTypes = { children: PropTypes.node }

/* ── 통계 카드 ── */
function StatCard({ title, value, sub, valueClass = '' }) {
  return (
    <div className="bg-th-surface border border-th-outline rounded-xl p-4">
      <p className="text-xs text-th-on-muted mb-1.5">{title}</p>
      <p className={`text-xl font-bold font-data tracking-tight ${valueClass || 'text-th-on-surface'}`}>{value}</p>
      {sub && <p className="text-xs text-th-on-subtle mt-1">{sub}</p>}
    </div>
  )
}
StatCard.propTypes = { title: PropTypes.string, value: PropTypes.node, sub: PropTypes.string, valueClass: PropTypes.string }

export default function Dashboard() {
  const { data: status, loading: statusLoading, refetch: refetchStatus } = useApi('/api/server/status')
  const { data: balance, loading: balLoading, error: balError, refetch: refetchBal } = useApi('/api/balance')
  const { data: posData, loading: posLoading, refetch: refetchPos } = useApi('/api/positions')
  const [wsLoading, setWsLoading] = useState(false)
  const [wsMsg, setWsMsg] = useState(null)

  function refetchAll() {
    refetchStatus()
    refetchBal()
    refetchPos()
  }

  async function handleWs(action) {
    setWsLoading(true)
    setWsMsg(null)
    try {
      const res = await fetch(`/api/ws/${action}`, { method: 'POST' })
      const text = await res.text()
      setWsMsg({ ok: res.ok, text })
      setTimeout(() => { refetchStatus(); setWsMsg(null) }, 1200)
    } catch (e) {
      setWsMsg({ ok: false, text: e.message })
    } finally {
      setWsLoading(false)
    }
  }

  const changeRate = fmtRate(balance?.asset_change_rate)
  const changeAmt = parseFloat(balance?.asset_change_amt ?? 0)
  const rateColor = changeRate === null ? '' : changeRate > 0 ? 'text-th-loss' : changeRate < 0 ? 'text-[#3B82F6]' : 'text-th-on-muted'

  const holdings = posData?.positions || []

  return (
    <div className="space-y-6">
      {/* 헤더 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-th-on-surface">대시보드</h1>
          <p className="text-xs text-th-on-muted mt-0.5">실시간 트레이딩 현황</p>
        </div>
        <button
          onClick={refetchAll}
          className="text-sm px-4 py-2 bg-th-surface hover:bg-th-surface-high border border-th-outline rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface"
        >
          새로고침
        </button>
      </div>

      {/* 서버 상태 */}
      {!statusLoading && status && (
        <div className="bg-th-surface border border-th-outline rounded-xl p-5">
          <SectionTitle>서버 상태</SectionTitle>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-4">

            {/* 국장 */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">국장 (KR)</p>
              <div className="flex items-center gap-2">
                <StatusDot ok={status.market_open} />
                <span className="text-sm font-medium text-th-on-surface">
                  {status.market_open ? '개장' : '폐장'}
                </span>
              </div>
            </div>

            {/* 미장 */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">미장 (US)</p>
              <div className="flex items-center gap-2">
                <StatusDot ok={status.us_market_open} />
                <span className="text-sm font-medium text-th-on-surface">
                  {status.us_market_open ? '개장' : '폐장'}
                </span>
              </div>
            </div>

            {/* WebSocket */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">WebSocket</p>
              <div className="flex items-center gap-2 flex-wrap">
                <div className="flex items-center gap-1.5">
                  <StatusDot ok={status.ws_connected} />
                  <span className="text-sm font-medium text-th-on-surface">
                    {status.ws_connected ? '연결됨' : '미연결'}
                  </span>
                </div>
                {status.ws_connected ? (
                  <button
                    onClick={() => handleWs('disconnect')}
                    disabled={wsLoading}
                    className="text-xs px-2.5 py-0.5 rounded-full bg-th-surface-high border border-th-outline hover:border-th-loss hover:text-th-loss disabled:opacity-40 transition-colors"
                  >
                    해제
                  </button>
                ) : (
                  <button
                    onClick={() => handleWs('connect')}
                    disabled={wsLoading}
                    className="text-xs px-2.5 py-0.5 rounded-full bg-th-surface-high border border-th-outline hover:border-th-growth hover:text-th-growth disabled:opacity-40 transition-colors"
                  >
                    연결
                  </button>
                )}
              </div>
              {wsMsg && (
                <p className={`text-xs mt-1 ${wsMsg.ok ? 'text-th-growth' : 'text-th-loss'}`}>{wsMsg.text}</p>
              )}
            </div>

            {/* 모니터링 */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">모니터링 포지션</p>
              <span className={`text-sm font-medium font-data ${status.monitored_count > 0 ? 'text-th-loss' : 'text-th-on-muted'}`}>
                {status.monitored_count}개
              </span>
            </div>

            {/* 국장 트레이더 */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">국장 트레이더</p>
              <span className={`text-sm font-medium ${traderColor(status.trader_state)}`}>
                {traderLabel(status.trader_state)}
              </span>
            </div>

            {/* 미장 트레이더 */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">미장 트레이더</p>
              <span className={`text-sm font-medium ${traderColor(status.trader_state_us)}`}>
                {traderLabel(status.trader_state_us)}
              </span>
            </div>

            {/* 주문가능금액 */}
            <div>
              <p className="text-xs text-th-on-subtle mb-1">주문가능금액</p>
              <span className="text-sm font-medium font-data text-th-on-surface">{fmtKRW(status.available_cash)}</span>
            </div>

          </div>
        </div>
      )}

      {/* 자산 */}
      {balError && (
        <div className="bg-th-loss/10 border border-th-loss/20 text-th-loss rounded-xl p-4 text-sm">{balError}</div>
      )}
      {!balLoading && (
        <div>
          <SectionTitle>자산 현황</SectionTitle>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
            <StatCard title="총 평가금액" value={fmtKRW(balance?.total_eval)} />
            <StatCard title="출금가능금액" value={fmtKRW(balance?.withdrawable_amount)} sub="출금가능" />
            <StatCard
              title="자산증감액 (전일대비)"
              value={changeAmt !== 0 ? (changeAmt > 0 ? '+' : '') + fmtKRW(changeAmt) : '-'}
              valueClass={changeAmt > 0 ? 'text-th-loss' : changeAmt < 0 ? 'text-[#3B82F6]' : ''}
            />
            <StatCard
              title="자산증감률 (전일대비)"
              value={changeRate !== null ? (changeRate > 0 ? '+' : '') + changeRate.toFixed(2) + '%' : '-'}
              valueClass={rateColor}
            />
          </div>
        </div>
      )}

      {/* 보유 종목 */}
      <div>
        <SectionTitle>보유 종목</SectionTitle>
        {posLoading ? (
          <p className="text-th-on-subtle text-sm">로딩 중...</p>
        ) : holdings.length === 0 ? (
          <div className="bg-th-surface border border-th-outline rounded-xl p-8 text-center text-th-on-subtle text-sm">
            보유 종목 없음
          </div>
        ) : (
          <div className="bg-th-surface border border-th-outline rounded-xl overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-th-outline text-xs text-th-on-subtle">
                    <th className="text-left px-5 py-3 font-medium">종목</th>
                    <th className="text-left px-5 py-3 font-medium hidden sm:table-cell">시장</th>
                    <th className="text-right px-5 py-3 font-medium">수량</th>
                    <th className="text-right px-5 py-3 font-medium hidden sm:table-cell">매입평균가</th>
                    <th className="text-right px-5 py-3 font-medium">현재가</th>
                    <th className="text-right px-5 py-3 font-medium hidden sm:table-cell">평가손익</th>
                    <th className="text-right px-5 py-3 font-medium">수익률</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-th-outline">
                  {holdings.map((h) => {
                    const rate = calcRate(h)
                    const pnl = parseFloat(h.evlu_pfls_amt ?? 0)
                    const isKR = !h.market || h.market !== 'US'
                    const ratePos = rate > 0
                    const rateNeg = rate < 0
                    const rateColor = ratePos ? 'text-th-loss' : rateNeg ? 'text-[#3B82F6]' : 'text-th-on-muted'
                    return (
                      <tr key={h.pdno} className="hover:bg-th-surface-high transition-colors">
                        <td className="px-5 py-3.5">
                          <span className="font-medium text-th-on-surface">{h.prdt_name}</span>
                          <span className="ml-1.5 text-xs text-th-on-subtle font-data">{h.pdno}</span>
                        </td>
                        <td className="px-5 py-3.5 hidden sm:table-cell">
                          {isKR ? (
                            <span className="badge bg-th-surface-high text-th-on-muted border-th-outline">국장</span>
                          ) : (
                            <span className="badge bg-[#7C3AED]/10 text-[#7C3AED] border-[#7C3AED]/20 dark:bg-[#7C3AED]/15 dark:text-[#A78BFA] dark:border-[#7C3AED]/30">미장</span>
                          )}
                        </td>
                        <td className="px-5 py-3.5 text-right text-th-on-muted font-data">{fmtNum(h.hldg_qty)}주</td>
                        <td className="px-5 py-3.5 text-right text-th-on-muted font-data hidden sm:table-cell">
                          {isKR ? fmtKRW(h.pchs_avg_pric) : fmtUSD(h.pchs_avg_pric)}
                        </td>
                        <td className="px-5 py-3.5 text-right font-medium text-th-on-surface font-data">
                          {isKR ? fmtKRW(h.prpr) : fmtUSD(h.prpr)}
                        </td>
                        <td className={`px-5 py-3.5 text-right font-medium font-data hidden sm:table-cell ${rateColor}`}>
                          {pnl > 0 ? '+' : ''}{fmtNum(h.evlu_pfls_amt)}{isKR ? '원' : '$'}
                        </td>
                        <td className="px-5 py-3.5 text-right">
                          <span className={`badge font-data ${
                            ratePos ? 'bg-th-loss/10 text-th-loss border-th-loss/20' :
                            rateNeg ? 'bg-[#3B82F6]/10 text-[#3B82F6] border-[#3B82F6]/20' :
                            'bg-th-surface-high text-th-on-muted border-th-outline'
                          }`}>
                            {ratePos ? '+' : ''}{rate.toFixed(2)}%
                          </span>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
