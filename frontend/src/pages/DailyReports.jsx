import { useState } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

function fmt(val, digits = 0) {
  if (val == null || val === '') return '-'
  const n = Number(val)
  if (isNaN(n)) return String(val)
  return digits > 0 ? n.toFixed(digits) : n.toLocaleString()
}

function WinRateBadge({ winning, total }) {
  if (!total) return <span className="text-th-on-muted text-xs">거래 없음</span>
  const pct = (winning / total) * 100
  const color = pct >= 60 ? 'text-emerald-400' : pct >= 40 ? 'text-yellow-400' : 'text-red-400'
  return <span className={`font-semibold text-sm ${color}`}>{pct.toFixed(0)}%</span>
}
WinRateBadge.propTypes = { winning: PropTypes.number, total: PropTypes.number }

function ProfitColor({ amount }) {
  const n = Number(amount)
  if (isNaN(n) || n === 0) return <span className="text-th-on-muted text-sm">0원</span>
  const color = n > 0 ? 'text-emerald-400' : 'text-red-400'
  return <span className={`font-semibold text-sm ${color}`}>{n > 0 ? '+' : ''}{n.toLocaleString()}원</span>
}
ProfitColor.propTypes = { amount: PropTypes.number }

function TradeSummaryTable({ json }) {
  const [open, setOpen] = useState(false)
  if (!json || json === 'null') return <span className="text-th-on-subtle text-xs">-</span>
  let items
  try { items = JSON.parse(json) } catch { return null }
  if (!Array.isArray(items) || items.length === 0) return <span className="text-th-on-subtle text-xs">-</span>

  return (
    <div>
      <button onClick={() => setOpen((o) => !o)} className="text-xs text-orange-400 hover:underline">
        {open ? '접기' : `${items.length}건 보기`}
      </button>
      {open && (
        <div className="mt-2 overflow-x-auto -mx-1">
          <table className="w-full text-xs border-collapse min-w-[320px]">
            <thead>
              <tr className="text-th-on-muted border-b border-black/10 dark:border-white/10">
                <th className="text-left py-1.5 pr-2">종목</th>
                <th className="text-right py-1.5 pr-2">매수가</th>
                <th className="text-right py-1.5 pr-2">매도가</th>
                <th className="text-right py-1.5 pr-2">수익률</th>
                <th className="text-right py-1.5">손익</th>
              </tr>
            </thead>
            <tbody>
              {items.map((t, i) => {
                const pct = Number(t.profit_pct)
                const pctColor = pct > 0 ? 'text-emerald-400' : pct < 0 ? 'text-red-400' : ''
                return (
                  <tr key={i} className="border-b border-black/5 dark:border-white/5">
                    <td className="py-1.5 pr-2 max-w-[80px] truncate">{t.stock_name || t.stock_code}</td>
                    <td className="text-right py-1.5 pr-2">{fmt(t.buy_price)}</td>
                    <td className="text-right py-1.5 pr-2">{fmt(t.sell_price)}</td>
                    <td className={`text-right py-1.5 pr-2 font-medium ${pctColor}`}>
                      {pct > 0 ? '+' : ''}{pct.toFixed(2)}%
                    </td>
                    <td className={`text-right py-1.5 font-medium ${pctColor}`}>
                      {Number(t.profit_amount) > 0 ? '+' : ''}{fmt(t.profit_amount)}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
TradeSummaryTable.propTypes = { json: PropTypes.string }

function BestWorstCard({ label, json, color }) {
  if (!json || json === 'null') return null
  let t
  try { t = JSON.parse(json) } catch { return null }
  if (!t) return null
  const pct = Number(t.profit_pct)
  return (
    <div className={`p-3 rounded-lg bg-black/5 dark:bg-white/5 border-l-4 ${color}`}>
      <p className="text-[10px] uppercase tracking-widest text-th-on-muted mb-1">{label}</p>
      <p className="font-semibold text-sm truncate">{t.stock_name || t.stock_code}</p>
      <p className={`text-xs font-medium ${pct >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
        {pct >= 0 ? '+' : ''}{pct.toFixed(2)}% ({pct >= 0 ? '+' : ''}{fmt(t.profit_amount)}원)
      </p>
      {t.sell_reason && <p className="text-xs text-th-on-muted mt-1 truncate">{t.sell_reason}</p>}
    </div>
  )
}
BestWorstCard.propTypes = { label: PropTypes.string, json: PropTypes.string, color: PropTypes.string }

export default function DailyReports() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [generating, setGenerating] = useState(false)
  const [expandedIds, setExpandedIds] = useState(new Set())

  const params = new URLSearchParams()
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  params.set('limit', '30')

  const { data, loading, error, refetch } = useApi(`/api/reports/daily?${params}`)
  const reports = data?.reports || []

  function toggleExpand(id) {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function handleGenerate() {
    setGenerating(true)
    try {
      const res = await fetch('/api/reports/daily/generate', { method: 'POST' })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        alert('생성 실패: ' + (body.error || res.status))
      } else {
        refetch()
      }
    } finally {
      setGenerating(false)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6 gap-2 flex-wrap">
        <h1 className="text-xl font-bold">일별 거래 리포트</h1>
        <div className="flex gap-2">
          <button
            onClick={handleGenerate}
            disabled={generating}
            className="text-sm px-3 py-1.5 bg-orange-500 hover:bg-orange-600 disabled:opacity-50 text-white rounded font-medium"
          >
            {generating ? '생성 중...' : '오늘 리포트 생성'}
          </button>
          <button onClick={refetch} className="text-sm px-3 py-1.5 bg-th-surface hover:bg-th-surface-high rounded text-th-on-muted hover:text-th-on-surface transition-colors">
            새로고침
          </button>
        </div>
      </div>

      {/* 날짜 필터 */}
      <div className="flex flex-wrap gap-2 mb-6">
        <div className="flex items-center gap-2 text-sm text-th-on-muted">
          <span>기간</span>
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)}
            className="bg-th-surface border border-black/10 dark:border-white/10 rounded px-2 py-1 text-th-on-surface text-sm" />
          <span>~</span>
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)}
            className="bg-th-surface border border-black/10 dark:border-white/10 rounded px-2 py-1 text-th-on-surface text-sm" />
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-700 text-red-300 rounded p-4 mb-4 text-sm">{error}</div>
      )}

      {loading ? (
        <p className="text-th-on-muted">로딩 중...</p>
      ) : reports.length === 0 ? (
        <p className="text-th-on-muted">일별 리포트가 없습니다. 거래 후 &ldquo;오늘 리포트 생성&rdquo; 버튼을 눌러보세요.</p>
      ) : (
        <div className="space-y-2">
          {reports.map((r) => {
            const isOpen = expandedIds.has(r.id)
            const profitN = Number(r.total_profit_amount)
            const profitColor = profitN > 0 ? 'text-emerald-400' : profitN < 0 ? 'text-red-400' : 'text-th-on-muted'
            return (
              <div key={r.id} className="bg-th-surface border border-black/10 dark:border-white/10 rounded-xl overflow-hidden">
                {/* 접을 수 있는 헤더 */}
                <button
                  onClick={() => toggleExpand(r.id)}
                  className="w-full flex items-center justify-between px-4 py-3 hover:bg-black/5 dark:hover:bg-white/5 transition-colors text-left"
                >
                  <div className="flex items-center gap-3 flex-wrap">
                    <span className="font-bold text-sm">{r.date}</span>
                    <WinRateBadge winning={r.winning_trades} total={r.total_trades} />
                    <span className="text-xs text-th-on-muted">{r.total_trades}건</span>
                    <span className={`text-sm font-semibold ${profitColor}`}>
                      {profitN > 0 ? '+' : ''}{profitN.toLocaleString()}원
                    </span>
                  </div>
                  <span className="text-th-on-muted text-sm shrink-0 ml-2">{isOpen ? '▲' : '▼'}</span>
                </button>

                {/* 펼쳐지는 내용 */}
                {isOpen && (
                  <div className="px-4 pb-4 border-t border-black/5 dark:border-white/5 pt-4 space-y-4">
                    {/* 핵심 지표 */}
                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                      <div className="text-center p-2 rounded-lg bg-black/5 dark:bg-white/5">
                        <p className="text-[10px] uppercase tracking-widest text-th-on-muted mb-1">총 거래</p>
                        <p className="font-bold text-base">{r.total_trades}건</p>
                      </div>
                      <div className="text-center p-2 rounded-lg bg-black/5 dark:bg-white/5">
                        <p className="text-[10px] uppercase tracking-widest text-th-on-muted mb-1">승/패</p>
                        <p className="font-bold text-base">
                          <span className="text-emerald-400">{r.winning_trades}</span>
                          <span className="text-th-on-muted"> / </span>
                          <span className="text-red-400">{r.losing_trades}</span>
                        </p>
                      </div>
                      <div className="text-center p-2 rounded-lg bg-black/5 dark:bg-white/5">
                        <p className="text-[10px] uppercase tracking-widest text-th-on-muted mb-1">총 손익</p>
                        <ProfitColor amount={r.total_profit_amount} />
                      </div>
                      <div className="text-center p-2 rounded-lg bg-black/5 dark:bg-white/5">
                        <p className="text-[10px] uppercase tracking-widest text-th-on-muted mb-1">평균 수익률</p>
                        <p className={`font-semibold text-sm ${Number(r.avg_profit_pct) >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
                          {Number(r.avg_profit_pct) >= 0 ? '+' : ''}{fmt(r.avg_profit_pct, 2)}%
                        </p>
                      </div>
                    </div>

                    {/* 최고/최하 거래 */}
                    {(r.best_trade !== 'null' || r.worst_trade !== 'null') && (
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <BestWorstCard label="최고 수익" json={r.best_trade} color="border-emerald-500" />
                        <BestWorstCard label="최대 손실" json={r.worst_trade} color="border-red-500" />
                      </div>
                    )}

                    {/* 거래 목록 */}
                    <div>
                      <p className="text-[10px] uppercase tracking-widest text-th-on-muted mb-2">거래 내역</p>
                      <TradeSummaryTable json={r.trade_summary} />
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
