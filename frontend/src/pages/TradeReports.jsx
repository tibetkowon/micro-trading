import { useState } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

function fmt(val, digits = 0) {
  if (val == null || val === '') return '-'
  const n = Number(val)
  if (isNaN(n)) return String(val)
  return digits > 0 ? n.toFixed(digits) : n.toLocaleString()
}

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

function ProfitBadge({ pct }) {
  if (pct == null) return <span className="text-th-on-muted text-xs">-</span>
  const n = Number(pct)
  const color = n > 0 ? 'text-emerald-400' : n < 0 ? 'text-red-400' : 'text-th-on-muted'
  return <span className={`font-semibold text-xs ${color}`}>{n > 0 ? '+' : ''}{n.toFixed(2)}%</span>
}
ProfitBadge.propTypes = { pct: PropTypes.number }

function IndicatorPopover({ json }) {
  const [open, setOpen] = useState(false)
  if (!json) return <span className="text-th-on-subtle text-xs">-</span>
  let obj
  try { obj = JSON.parse(json) } catch { return <span className="text-th-on-subtle text-xs">오류</span> }
  return (
    <div className="relative inline-block">
      <button
        onClick={() => setOpen((o) => !o)}
        className="text-xs text-orange-400 hover:underline"
      >
        지표 보기
      </button>
      {open && (
        <>
          {/* 배경 닫기 레이어 */}
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          {/* 팝업: 모바일에서 화면 밖 나가지 않도록 right-0 기준 */}
          <div
            className="absolute z-50 right-0 top-7 w-64 max-w-[calc(100vw-2rem)] bg-th-surface border border-black/10 dark:border-white/10 rounded-lg shadow-xl p-3 text-xs text-th-on-surface space-y-1"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-2">
              <p className="font-semibold truncate pr-2">{obj.stock_name || obj.stock_code}</p>
              <button className="text-th-on-muted hover:text-th-on-surface shrink-0" onClick={() => setOpen(false)}>✕</button>
            </div>
            {[
              ['현재가', obj.current_price],
              ['MA5', fmt(obj.ma5, 2)],
              ['MA20', fmt(obj.ma20, 2)],
              ['RSI14', fmt(obj.rsi14, 2)],
              ['MACD', fmt(obj.macd_line, 4)],
              ['MACD Signal', fmt(obj.macd_signal, 4)],
              ['VWAP', fmt(obj.vwap, 0)],
              ['VWAP Diff', obj.vwap_diff != null ? fmt(obj.vwap_diff, 2) + '%' : '-'],
              ['5분봉 이격도', obj.disparity_m5 != null ? fmt(obj.disparity_m5, 2) + '%' : '-'],
              ['고가대비', obj.high_price_diff != null ? fmt(obj.high_price_diff, 2) + '%' : '-'],
              ['체결강도', obj.strength],
            ].map(([label, value]) => (
              <div key={label} className="flex justify-between gap-2">
                <span className="text-th-on-muted shrink-0">{label}</span>
                <span className="text-right">{value ?? '-'}</span>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
IndicatorPopover.propTypes = { json: PropTypes.string }

// 개별 거래 카드
function TradeCard({ r }) {
  const [open, setOpen] = useState(false)
  const sold = !!r.sold_at
  const profitN = Number(r.profit_amount)
  const profitColor = profitN > 0 ? 'text-emerald-400' : profitN < 0 ? 'text-red-400' : 'text-th-on-muted'

  return (
    <div className="border border-black/5 dark:border-white/5 rounded-lg overflow-hidden">
      {/* 미니 헤더 - 항상 노출 */}
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-black/5 dark:hover:bg-white/5 transition-colors text-left"
      >
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-semibold text-sm truncate">{r.stock_name || r.stock_code}</span>
          <span className="text-th-on-muted text-xs shrink-0">{r.stock_code}</span>
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-2">
          {sold
            ? <>
                <ProfitBadge pct={r.profit_pct} />
                <span className={`text-xs font-medium ${profitColor}`}>
                  {profitN > 0 ? '+' : ''}{profitN.toLocaleString()}원
                </span>
              </>
            : <span className="text-xs text-yellow-400 font-medium">보유 중</span>
          }
          <span className="text-th-on-muted text-xs">{open ? '▲' : '▼'}</span>
        </div>
      </button>

      {/* 상세 */}
      {open && (
        <div className="px-3 pb-3 pt-1 border-t border-black/5 dark:border-white/5">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
            {/* 매수 정보 */}
            <div className="space-y-1.5">
              <p className="text-th-on-muted font-semibold uppercase tracking-widest text-[10px]">매수</p>
              <div className="flex justify-between">
                <span className="text-th-on-muted">체결가</span>
                <span>{fmt(r.buy_price)}원 × {fmt(r.buy_qty)}주</span>
              </div>
              <div className="flex justify-between">
                <span className="text-th-on-muted">매수금액</span>
                <span>{fmt(r.buy_amount)}원</span>
              </div>
              <div className="flex justify-between">
                <span className="text-th-on-muted">매수 지표</span>
                <IndicatorPopover json={r.buy_indicators} />
              </div>
              <div className="flex justify-between">
                <span className="text-th-on-muted">시각</span>
                <span>{fmtDate(r.created_at)}</span>
              </div>
              {r.buy_reason && (
                <div className="mt-1 p-2 bg-black/5 dark:bg-white/5 rounded text-th-on-muted leading-relaxed">
                  {r.buy_reason}
                </div>
              )}
            </div>

            {/* 매도 정보 */}
            <div className="space-y-1.5">
              <p className="text-th-on-muted font-semibold uppercase tracking-widest text-[10px]">매도</p>
              {sold ? (
                <>
                  <div className="flex justify-between">
                    <span className="text-th-on-muted">체결가</span>
                    <span>{fmt(r.sell_price)}원 × {fmt(r.sell_qty)}주</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-th-on-muted">매도금액</span>
                    <span>{fmt(r.sell_amount)}원</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-th-on-muted">손익</span>
                    <span className={profitColor}>
                      {profitN >= 0 ? '+' : ''}{fmt(r.profit_amount)}원
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-th-on-muted">매도 지표</span>
                    <IndicatorPopover json={r.sell_indicators} />
                  </div>
                  <div className="flex justify-between">
                    <span className="text-th-on-muted">시각</span>
                    <span>{fmtDate(r.sold_at)}</span>
                  </div>
                  {r.sell_reason && (
                    <div className="mt-1 p-2 bg-black/5 dark:bg-white/5 rounded text-th-on-muted">
                      사유: {r.sell_reason}
                    </div>
                  )}
                </>
              ) : (
                <p className="text-th-on-subtle">아직 매도되지 않았습니다.</p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
TradeCard.propTypes = { r: PropTypes.object }

// 날짜 그룹 (접기/펼치기)
function DateGroup({ date, items }) {
  const [open, setOpen] = useState(true)
  const totalProfit = items.reduce((s, r) => s + (r.sold_at ? Number(r.profit_amount) : 0), 0)
  const profitColor = totalProfit > 0 ? 'text-emerald-400' : totalProfit < 0 ? 'text-red-400' : 'text-th-on-muted'

  return (
    <div className="bg-th-surface border border-black/10 dark:border-white/10 rounded-xl overflow-hidden">
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-4 py-3 hover:bg-black/5 dark:hover:bg-white/5 transition-colors text-left"
      >
        <div className="flex items-center gap-3">
          <span className="font-bold text-sm">{date}</span>
          <span className="text-xs text-th-on-muted">{items.length}건</span>
          {totalProfit !== 0 && (
            <span className={`text-sm font-semibold ${profitColor}`}>
              {totalProfit > 0 ? '+' : ''}{totalProfit.toLocaleString()}원
            </span>
          )}
        </div>
        <span className="text-th-on-muted text-sm">{open ? '▲' : '▼'}</span>
      </button>
      {open && (
        <div className="px-3 pb-3 space-y-2 border-t border-black/5 dark:border-white/5 pt-2">
          {items.map((r) => <TradeCard key={r.id} r={r} />)}
        </div>
      )}
    </div>
  )
}
DateGroup.propTypes = { date: PropTypes.string, items: PropTypes.array }

export default function TradeReports() {
  const [date, setDate] = useState('')
  const [stockCode, setStockCode] = useState('')
  const [page, setPage] = useState(1)

  const params = new URLSearchParams()
  if (date) params.set('date', date)
  if (stockCode) params.set('stock_code', stockCode)
  params.set('page', page)
  params.set('limit', '20')

  const { data, loading, error, refetch } = useApi(`/api/reports/trades?${params}`)
  const reports = data?.reports || []

  // 날짜별 그룹화
  const grouped = []
  const seen = new Map()
  for (const r of reports) {
    const d = r.date || '-'
    if (!seen.has(d)) {
      seen.set(d, [])
      grouped.push(d)
    }
    seen.get(d).push(r)
  }

  function handleSearch(e) {
    e.preventDefault()
    setPage(1)
    refetch()
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6 gap-2 flex-wrap">
        <h1 className="text-xl font-bold">거래 리포트</h1>
        <button onClick={refetch} className="text-sm px-3 py-1.5 bg-th-surface hover:bg-th-surface-high rounded text-th-on-muted hover:text-th-on-surface transition-colors">
          새로고침
        </button>
      </div>

      {/* 필터 */}
      <form onSubmit={handleSearch} className="flex flex-wrap gap-2 mb-6">
        <input
          type="date" value={date} onChange={(e) => setDate(e.target.value)}
          className="bg-th-surface border border-black/10 dark:border-white/10 rounded px-3 py-1.5 text-sm text-th-on-surface"
        />
        <input
          type="text" placeholder="종목코드" value={stockCode}
          onChange={(e) => setStockCode(e.target.value)}
          maxLength={6}
          className="bg-th-surface border border-black/10 dark:border-white/10 rounded px-3 py-1.5 text-sm text-th-on-surface w-28"
        />
        <button type="submit" className="text-sm px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white rounded font-medium">
          검색
        </button>
        {(date || stockCode) && (
          <button
            type="button"
            onClick={() => { setDate(''); setStockCode(''); setPage(1) }}
            className="text-sm px-3 py-1.5 bg-th-surface hover:bg-th-surface-high rounded text-th-on-muted hover:text-th-on-surface transition-colors"
          >
            초기화
          </button>
        )}
      </form>

      {error && (
        <div className="bg-red-900/30 border border-red-700 text-red-300 rounded p-4 mb-4 text-sm">{error}</div>
      )}

      {loading ? (
        <p className="text-th-on-muted">로딩 중...</p>
      ) : reports.length === 0 ? (
        <p className="text-th-on-muted">거래 리포트가 없습니다.</p>
      ) : (
        <div className="space-y-3">
          {grouped.map((d) => (
            <DateGroup key={d} date={d} items={seen.get(d)} />
          ))}
        </div>
      )}

      {/* 페이지네이션 */}
      {reports.length > 0 && (
        <div className="flex gap-2 mt-6 justify-center">
          <button
            disabled={page <= 1} onClick={() => setPage((p) => p - 1)}
            className="px-3 py-1.5 text-sm rounded bg-th-surface border border-black/10 dark:border-white/10 disabled:opacity-40"
          >
            이전
          </button>
          <span className="px-3 py-1.5 text-sm text-th-on-muted">{page} 페이지</span>
          <button
            disabled={reports.length < 20} onClick={() => setPage((p) => p + 1)}
            className="px-3 py-1.5 text-sm rounded bg-th-surface border border-black/10 dark:border-white/10 disabled:opacity-40"
          >
            다음
          </button>
        </div>
      )}
    </div>
  )
}
