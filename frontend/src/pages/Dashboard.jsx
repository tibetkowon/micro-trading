import { useState, useEffect, useRef } from 'react'
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
function StatusDot({ ok }) {
  return (
    <span className={`inline-block rounded-full w-1.5 h-1.5 ${ok ? 'bg-emerald-400' : 'bg-gray-600'}`} />
  )
}
StatusDot.propTypes = { ok: PropTypes.bool }

/* ── 트레이더 상태 색 ── */
function traderColor(state) {
  if (!state || state === 'IDLE') return 'text-gray-500'
  if (state === 'MONITORING') return 'text-red-400'
  if (state === 'SEARCHING' || state === 'SELECTING') return 'text-emerald-400'
  return 'text-amber-400'
}
function traderLabel(state) {
  const map = {
    IDLE: 'IDLE',
    SELECTING: 'SELECTING',
    SEARCHING: 'SEARCHING',
    ORDERING: 'ORDERING',
    WAITING_FILL: 'WAITING FILL',
    MONITORING: 'MONITORING',
  }
  return map[state] || (state || 'IDLE')
}

/* ── 서버 상태 행 ── */
function StatusRow({ label, children }) {
  return (
    <div>
      <p className="text-[10px] text-gray-600 uppercase tracking-widest mb-1">{label}</p>
      {children}
    </div>
  )
}
StatusRow.propTypes = { label: PropTypes.string, children: PropTypes.node }

const REFRESH_OPTIONS = [
  { value: 0, label: '끄기' },
  { value: 10, label: '10초' },
  { value: 30, label: '30초' },
  { value: 60, label: '1분' },
  { value: 300, label: '5분' },
]

export default function Dashboard() {
  const { data: status, loading: statusLoading, refetch: refetchStatus } = useApi('/api/server/status')
  const { data: balance, loading: balLoading, error: balError, refetch: refetchBal } = useApi('/api/balance')
  const { data: posData, loading: posLoading, refetch: refetchPos } = useApi('/api/positions')
  const [wsLoading, setWsLoading] = useState(false)
  const [wsMsg, setWsMsg] = useState(null)
  const [refreshInterval, setRefreshInterval] = useState(0)
  const intervalRef = useRef(null)

  function refetchAll() {
    refetchStatus()
    refetchBal()
    refetchPos()
  }

  useEffect(() => {
    if (intervalRef.current) clearInterval(intervalRef.current)
    if (refreshInterval > 0) {
      intervalRef.current = setInterval(refetchAll, refreshInterval * 1000)
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshInterval])

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
  const totalEval = balance?.total_eval

  const holdings = posData?.positions || []

  return (
    <div className="space-y-6">

      {/* ── 페이지 헤더 ── */}
      <div className="flex items-center justify-between pt-2">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">대시보드</h1>
          <p className="text-xs text-gray-500 mt-0.5 uppercase tracking-widest">실시간 트레이딩 현황</p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={refreshInterval}
            onChange={(e) => setRefreshInterval(Number(e.target.value))}
            className="text-xs px-2 py-2 bg-[#1F1F22] border border-white/10 rounded-lg text-gray-400 focus:outline-none focus:ring-1 focus:ring-orange-500/50"
          >
            {REFRESH_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
          <button
            onClick={refetchAll}
            className="flex items-center gap-1.5 text-xs px-3 py-2 bg-[#1F1F22] hover:bg-[#2A2A2D] rounded-lg transition-colors text-gray-400 hover:text-white"
          >
            <span className="material-symbols-outlined text-[16px]">refresh</span>
            새로고침
          </button>
        </div>
      </div>

      {/* ── 자산 히어로 패널 ── */}
      {!balLoading && !balError && (
        <div className="glass-panel p-6 md:p-8 bg-gradient-to-br from-[#1F1F22] to-[#161618]">
          <p className="text-[10px] text-gray-500 uppercase tracking-widest mb-3">총 평가금액</p>
          <p className="text-4xl md:text-5xl font-bold text-white tracking-tighter font-data leading-none">
            {fmtKRW(totalEval)}
          </p>
          <div className="flex flex-wrap gap-4 mt-4">
            <div>
              <p className="text-[10px] text-gray-600 uppercase tracking-widest">전일대비</p>
              <p className={`text-lg font-semibold font-data mt-0.5 ${
                changeAmt > 0 ? 'text-red-400' : changeAmt < 0 ? 'text-blue-400' : 'text-gray-500'
              }`}>
                {changeAmt !== 0 ? (changeAmt > 0 ? '+' : '') + fmtKRW(changeAmt) : '-'}
                {changeRate !== null && (
                  <span className="text-sm ml-1.5 opacity-80">
                    ({changeRate > 0 ? '+' : ''}{changeRate.toFixed(2)}%)
                  </span>
                )}
              </p>
            </div>
            <div>
              <p className="text-[10px] text-gray-600 uppercase tracking-widest">출금가능금액</p>
              <p className="text-lg font-semibold font-data text-gray-300 mt-0.5">
                {fmtKRW(balance?.withdrawable_amount)}
              </p>
            </div>
          </div>
        </div>
      )}
      {balError && (
        <div className="bg-red-500/10 rounded-xl p-4 text-sm text-red-400">{balError}</div>
      )}

      {/* ── 서버 상태 ── */}
      {!statusLoading && status && (
        <div className="bg-[#1F1F22] rounded-xl p-6">
          <p className="text-[10px] text-gray-600 uppercase tracking-widest mb-5">서버 상태</p>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-8 gap-y-5">

            <StatusRow label="국장 (KR)">
              <div className="flex items-center gap-2">
                <StatusDot ok={status.market_open} />
                <span className="text-sm font-medium text-gray-200">
                  {status.market_open ? '개장' : '폐장'}
                </span>
              </div>
            </StatusRow>

            <StatusRow label="미장 (US)">
              <div className="flex items-center gap-2">
                <StatusDot ok={status.us_market_open} />
                <span className="text-sm font-medium text-gray-200">
                  {status.us_market_open ? '개장' : '폐장'}
                </span>
              </div>
            </StatusRow>

            <StatusRow label="WebSocket">
              <div className="flex items-center gap-2 flex-wrap">
                <div className="flex items-center gap-1.5">
                  <StatusDot ok={status.ws_connected} />
                  <span className="text-sm font-medium text-gray-200">
                    {status.ws_connected ? '연결됨' : '미연결'}
                  </span>
                </div>
                {status.ws_connected ? (
                  <button
                    onClick={() => handleWs('disconnect')}
                    disabled={wsLoading}
                    className="text-xs px-2 py-0.5 rounded-full bg-[#2A2A2D] hover:text-red-400 disabled:opacity-40 transition-colors text-gray-400"
                  >
                    해제
                  </button>
                ) : (
                  <button
                    onClick={() => handleWs('connect')}
                    disabled={wsLoading}
                    className="text-xs px-2 py-0.5 rounded-full bg-[#2A2A2D] hover:text-emerald-400 disabled:opacity-40 transition-colors text-gray-400"
                  >
                    연결
                  </button>
                )}
              </div>
              {wsMsg && (
                <p className={`text-xs mt-1 ${wsMsg.ok ? 'text-emerald-400' : 'text-red-400'}`}>{wsMsg.text}</p>
              )}
            </StatusRow>

            <StatusRow label="모니터링 포지션">
              <span className={`text-sm font-semibold font-data ${status.monitored_count > 0 ? 'text-red-400' : 'text-gray-500'}`}>
                {status.monitored_count}개
              </span>
            </StatusRow>

            <StatusRow label="국장 트레이더">
              <div className="flex items-center gap-2">
                <div className={`w-1 h-4 rounded-full ${
                  !status.trader_state || status.trader_state === 'IDLE' ? 'bg-gray-700' :
                  status.trader_state === 'MONITORING' ? 'bg-red-400' :
                  'bg-emerald-400'
                }`} />
                <span className={`text-sm font-medium ${traderColor(status.trader_state)}`}>
                  {traderLabel(status.trader_state)}
                </span>
              </div>
            </StatusRow>

            <StatusRow label="미장 트레이더">
              <div className="flex items-center gap-2">
                <div className={`w-1 h-4 rounded-full ${
                  !status.trader_state_us || status.trader_state_us === 'IDLE' ? 'bg-gray-700' :
                  status.trader_state_us === 'MONITORING' ? 'bg-red-400' :
                  'bg-emerald-400'
                }`} />
                <span className={`text-sm font-medium ${traderColor(status.trader_state_us)}`}>
                  {traderLabel(status.trader_state_us)}
                </span>
              </div>
            </StatusRow>

            <StatusRow label="주문가능금액">
              <span className="text-sm font-medium font-data text-gray-200">{fmtKRW(status.available_cash)}</span>
            </StatusRow>

          </div>
        </div>
      )}

      {/* ── 보유 종목 ── */}
      <div>
        <p className="text-[10px] text-gray-600 uppercase tracking-widest mb-4">보유 종목</p>
        {posLoading ? (
          <p className="text-gray-600 text-sm">로딩 중...</p>
        ) : holdings.length === 0 ? (
          <div className="bg-[#1F1F22] rounded-xl p-10 text-center">
            <span className="material-symbols-outlined text-[40px] text-gray-700 block mb-2">inbox</span>
            <p className="text-gray-500 text-sm">보유 종목 없음</p>
          </div>
        ) : (
          <div className="bg-[#1B1B1E] rounded-xl overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-[10px] text-gray-600 uppercase tracking-widest">
                    <th className="text-left px-5 py-3.5 font-medium">종목</th>
                    <th className="text-left px-5 py-3.5 font-medium hidden sm:table-cell">시장</th>
                    <th className="text-right px-5 py-3.5 font-medium">수량</th>
                    <th className="text-right px-5 py-3.5 font-medium hidden sm:table-cell">매입평균가</th>
                    <th className="text-right px-5 py-3.5 font-medium">현재가</th>
                    <th className="text-right px-5 py-3.5 font-medium hidden sm:table-cell">평가손익</th>
                    <th className="text-right px-5 py-3.5 font-medium">수익률</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/[0.04]">
                  {holdings.map((h) => {
                    const rate = calcRate(h)
                    const pnl = parseFloat(h.evlu_pfls_amt ?? 0)
                    const isKR = !h.market || h.market !== 'US'
                    const ratePos = rate > 0
                    const rateNeg = rate < 0
                    return (
                      <tr key={h.pdno} className="hover:bg-white/[0.02] transition-colors">
                        <td className="px-5 py-4">
                          <span className="font-medium text-gray-100">{h.prdt_name}</span>
                          <span className="ml-2 text-xs text-gray-600 font-data">{h.pdno}</span>
                        </td>
                        <td className="px-5 py-4 hidden sm:table-cell">
                          {isKR ? (
                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-blue-500/10 text-blue-400">국내</span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-orange-500/10 text-orange-400">해외</span>
                          )}
                        </td>
                        <td className="px-5 py-4 text-right text-gray-400 font-data">{fmtNum(h.hldg_qty)}주</td>
                        <td className="px-5 py-4 text-right text-gray-400 font-data hidden sm:table-cell">
                          {isKR ? fmtKRW(h.pchs_avg_pric) : fmtUSD(h.pchs_avg_pric)}
                        </td>
                        <td className="px-5 py-4 text-right font-medium text-gray-100 font-data">
                          {isKR ? fmtKRW(h.prpr) : fmtUSD(h.prpr)}
                        </td>
                        <td className={`px-5 py-4 text-right font-medium font-data hidden sm:table-cell ${
                          ratePos ? 'text-red-400' : rateNeg ? 'text-blue-400' : 'text-gray-500'
                        }`}>
                          {pnl > 0 ? '+' : ''}{fmtNum(h.evlu_pfls_amt)}{isKR ? '원' : '$'}
                        </td>
                        <td className="px-5 py-4 text-right">
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-data font-medium ${
                            ratePos ? 'bg-red-500/10 text-red-400' :
                            rateNeg ? 'bg-blue-500/10 text-blue-400' :
                            'bg-white/5 text-gray-500'
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
