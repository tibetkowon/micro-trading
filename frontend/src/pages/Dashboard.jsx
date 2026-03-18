import { useState } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'
import Card from '../components/Card'

function fmt(n) {
  if (!n && n !== 0) return '-'
  return Number(n).toLocaleString('ko-KR') + '원'
}

function fmtRate(r) {
  if (r === undefined || r === null || r === '' || r === '-') return '-'
  const n = parseFloat(r)
  if (isNaN(n)) return '-'
  return (n > 0 ? '+' : '') + n.toFixed(2) + '%'
}

function fmtNum(s) {
  if (!s && s !== 0) return '-'
  const n = parseFloat(s)
  return isNaN(n) ? '-' : n.toLocaleString('ko-KR')
}

function StatusDot({ ok }) {
  return (
    <span className={`inline-block w-2 h-2 rounded-full mr-1.5 ${ok ? 'bg-emerald-400' : 'bg-zinc-600'}`} />
  )
}
StatusDot.propTypes = { ok: PropTypes.bool }

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
      setTimeout(() => { refetchStatus(); setWsMsg(null) }, 1000)
    } catch (e) {
      setWsMsg({ ok: false, text: e.message })
    } finally {
      setWsLoading(false)
    }
  }

  const changeRate = balance?.asset_change_rate
  const changeAmt = balance?.asset_change_amt ?? 0
  const changeColor =
    changeRate && changeRate !== '-'
      ? parseFloat(changeRate) > 0
        ? 'text-red-400'
        : parseFloat(changeRate) < 0
        ? 'text-blue-400'
        : ''
      : ''

  const holdings = posData?.positions || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">대시보드</h1>
          <p className="text-sm text-zinc-500 mt-0.5">실시간 트레이딩 현황</p>
        </div>
        <button
          onClick={refetchAll}
          className="text-sm px-4 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
        >
          새로고침
        </button>
      </div>

      {/* Server Status */}
      {!statusLoading && status && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5">
          <p className="text-sm font-medium text-white mb-4">서버 상태</p>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-5">
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">국장(KR)</p>
              <p className="flex items-center text-sm font-medium">
                <StatusDot ok={status.market_open} />
                {status.market_open ? '개장' : '폐장'}
              </p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">미장(US)</p>
              <p className="flex items-center text-sm font-medium">
                <StatusDot ok={status.us_market_open} />
                {status.us_market_open ? '개장' : '폐장'}
              </p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">WebSocket</p>
              <div className="flex items-center gap-2">
                <p className="flex items-center text-sm font-medium">
                  <StatusDot ok={status.ws_connected} />
                  {status.ws_connected ? '연결됨' : '미연결'}
                </p>
                {status.ws_connected ? (
                  <button
                    onClick={() => handleWs('disconnect')}
                    disabled={wsLoading}
                    className="text-xs px-2.5 py-0.5 rounded-full bg-zinc-800 hover:bg-red-900/50 hover:text-red-400 border border-zinc-700 disabled:opacity-40 transition-colors"
                  >
                    해제
                  </button>
                ) : (
                  <button
                    onClick={() => handleWs('connect')}
                    disabled={wsLoading}
                    className="text-xs px-2.5 py-0.5 rounded-full bg-zinc-800 hover:bg-emerald-900/50 hover:text-emerald-400 border border-zinc-700 disabled:opacity-40 transition-colors"
                  >
                    연결
                  </button>
                )}
              </div>
              {wsMsg && (
                <p className={`text-xs mt-1 ${wsMsg.ok ? 'text-emerald-400' : 'text-red-400'}`}>{wsMsg.text}</p>
              )}
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">모니터링 포지션</p>
              <p className="text-sm font-medium">
                <span className={status.monitored_count > 0 ? 'text-red-400' : 'text-zinc-400'}>
                  {status.monitored_count}개
                </span>
              </p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">주문가능금액</p>
              <p className="text-sm font-medium">{fmt(status.available_cash)}</p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">국장 트레이더</p>
              <p className="text-sm font-medium">
                <span className={
                  status.trader_state === 'IDLE' ? 'text-zinc-400' :
                  status.trader_state === 'MONITORING' ? 'text-red-400' :
                  status.trader_state === 'SEARCHING' ? 'text-emerald-400' :
                  'text-yellow-400'
                }>
                  {status.trader_state === 'SEARCHING' ? 'SEARCHING (종목탐색)' : (status.trader_state || 'IDLE')}
                </span>
              </p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1.5">미장 트레이더</p>
              <p className="text-sm font-medium">
                <span className={
                  status.trader_state_us === 'IDLE' ? 'text-zinc-400' :
                  status.trader_state_us === 'MONITORING' ? 'text-red-400' :
                  status.trader_state_us === 'SEARCHING' ? 'text-emerald-400' :
                  'text-yellow-400'
                }>
                  {status.trader_state_us === 'SEARCHING' ? 'SEARCHING (종목탐색)' : (status.trader_state_us || 'IDLE')}
                </span>
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Balance */}
      {balError && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl p-4 text-sm">
          {balError}
        </div>
      )}
      {!balLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card title="총 평가금액" value={fmt(balance?.total_eval)} />
          <Card title="출금가능금액" value={fmt(balance?.withdrawable_amount)} sub="출금가능" />
          <Card
            title="자산증감액"
            value={fmt(changeAmt)}
            sub="전일 대비"
            className={changeAmt > 0 ? 'border-red-500/30' : changeAmt < 0 ? 'border-blue-500/30' : ''}
          />
          <Card
            title="자산증감수익률"
            value={fmtRate(changeRate)}
            sub="전일 대비"
            className={changeColor ? changeColor.replace('text-', 'border-').replace('400', '500/30') : ''}
          />
        </div>
      )}

      {/* Holdings */}
      <div>
        <p className="text-sm font-medium text-zinc-300 mb-3">보유 종목</p>
        {posLoading ? (
          <p className="text-zinc-500 text-sm">로딩 중...</p>
        ) : holdings.length === 0 ? (
          <p className="text-zinc-500 text-sm">보유 종목 없음</p>
        ) : (
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-zinc-800 text-xs text-zinc-500">
                    <th className="text-left px-5 py-3 font-medium">종목</th>
                    <th className="text-right px-5 py-3 font-medium">보유수량</th>
                    <th className="text-right px-5 py-3 font-medium">매입평균가</th>
                    <th className="text-right px-5 py-3 font-medium">현재가</th>
                    <th className="text-right px-5 py-3 font-medium">평가손익</th>
                    <th className="text-right px-5 py-3 font-medium">수익률</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/60">
                  {holdings.map((h) => {
                    const rate = parseFloat(h.evlu_erng_rt ?? 0)
                    const pnl = parseFloat(h.evlu_pfls_amt ?? 0)
                    const rateColor = rate > 0 ? 'text-red-400' : rate < 0 ? 'text-blue-400' : 'text-zinc-400'
                    return (
                      <tr key={h.pdno} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-3.5">
                          <span className="font-medium text-white">{h.prdt_name}</span>
                          <span className="ml-1.5 text-xs text-zinc-500 font-mono">{h.pdno}</span>
                        </td>
                        <td className="px-5 py-3.5 text-right text-zinc-300">{fmtNum(h.hldg_qty)}주</td>
                        <td className="px-5 py-3.5 text-right text-zinc-400">{fmt(h.pchs_avg_pric)}</td>
                        <td className="px-5 py-3.5 text-right font-medium text-white">{fmt(h.prpr)}</td>
                        <td className={`px-5 py-3.5 text-right font-medium ${rateColor}`}>
                          {pnl > 0 ? '+' : ''}{fmtNum(h.evlu_pfls_amt)}원
                        </td>
                        <td className={`px-5 py-3.5 text-right font-medium ${rateColor}`}>
                          <span className={`inline-block px-2.5 py-0.5 rounded-full text-xs border ${
                            rate > 0 ? 'bg-red-500/15 text-red-400 border-red-500/20' :
                            rate < 0 ? 'bg-blue-500/15 text-blue-400 border-blue-500/20' :
                            'bg-zinc-700/50 text-zinc-400 border-zinc-700'
                          }`}>
                            {rate > 0 ? '+' : ''}{rate.toFixed(2)}%
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
