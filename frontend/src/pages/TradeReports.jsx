import { useState, useEffect } from 'react'
import { collection, query, orderBy, limit, getDocs } from 'firebase/firestore'
import { fmt, fmtPct, fmtSigned, Badge } from '../components/shared'
import { db } from '../lib/firebase'

const SELL_REASON_COLOR = {
  '목표가 도달': 'green',
  '손절': 'red',
  'RSI과매수': 'yellow',
  'MACD데드크로스': 'orange',
  '스태그네이션': 'purple',
  '수동': 'gray',
}

function SellReasonBadge({ reason }) {
  return <Badge color={SELL_REASON_COLOR[reason] || 'gray'}>{reason || '—'}</Badge>
}

function fmtTime(ts) {
  if (!ts) return '—'
  const d = ts.toDate ? ts.toDate() : new Date(ts)
  return d.toLocaleString('ko-KR', { timeZone: 'Asia/Seoul', hour: '2-digit', minute: '2-digit', hour12: false })
}

function calcHoldPeriod(start, end) {
  if (!start || !end) return '—'
  const s = start.toDate ? start.toDate() : new Date(start)
  const e = end.toDate ? end.toDate() : new Date(end)
  const mins = Math.round((e - s) / 60000)
  if (mins < 60) return `${mins}m`
  return `${Math.floor(mins / 60)}h ${mins % 60}m`
}

function groupByDate(reports) {
  const groups = {}
  for (const r of reports) {
    const date = r.date || 'unknown'
    if (!groups[date]) groups[date] = { date, day_pnl: 0, trades: [] }
    groups[date].day_pnl += r.profit_amount || 0
    groups[date].trades.push({
      id: r._docId,
      stock_code: r.stock_code,
      stock_name: r.stock_name,
      buy_price: r.buy_price,
      sell_price: r.sell_price,
      pnl_amount: r.profit_amount,
      pnl_pct: r.profit_pct,
      sell_reason: r.sell_reason,
      buy_time: fmtTime(r.created_at),
      sell_time: fmtTime(r.sold_at),
      hold_period: calcHoldPeriod(r.created_at, r.sold_at),
      indicators: (() => { try { return JSON.parse(r.buy_indicators || '{}') } catch { return {} } })(),
    })
  }
  return Object.values(groups).sort((a, b) => b.date.localeCompare(a.date))
}

const PAGE_SIZE = 20

export default function TradeReports() {
  const [dateFilter, setDateFilter] = useState('')
  const [codeFilter, setCodeFilter] = useState('')
  const [expandedId, setExpandedId] = useState(null)
  const [page, setPage] = useState(1)
  const [allReports, setAllReports] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const q = query(collection(db, 'trade_reports'), orderBy('created_at', 'desc'), limit(500))
    getDocs(q)
      .then(snap => setAllReports(snap.docs.map(d => ({ _docId: d.id, ...d.data() }))))
      .finally(() => setLoading(false))
  }, [])

  const filtered = allReports.filter(r => {
    if (dateFilter && r.date !== dateFilter) return false
    if (codeFilter && !r.stock_code?.includes(codeFilter.toUpperCase())) return false
    return true
  })

  const groups = groupByDate(filtered)
  const totalPages = Math.max(1, Math.ceil(groups.length / PAGE_SIZE))
  const pagedGroups = groups.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  return (
    <div>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
        <input className="form-input" style={{ width: 160 }} type="date"
          value={dateFilter} onChange={e => { setDateFilter(e.target.value); setPage(1) }} />
        <input className="form-input" style={{ width: 160 }} placeholder="종목코드 검색..."
          value={codeFilter} onChange={e => { setCodeFilter(e.target.value); setPage(1) }} />
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 32, color: 'var(--text-muted)' }}>로딩 중...</div>
      ) : pagedGroups.length === 0 ? (
        <div className="card">
          <div style={{ textAlign: 'center', padding: 48, color: 'var(--text-muted)' }}>거래 리포트가 없습니다</div>
        </div>
      ) : (
        pagedGroups.map(group => (
          <div key={group.date} style={{ marginBottom: 24 }}>
            <div className="date-group-header">
              <span className="date-label">{group.date}</span>
              <div style={{ flex: 1, height: 1, background: 'var(--border)' }}></div>
              <span className="mono" style={{
                fontWeight: 700, fontSize: 13,
                color: (group.day_pnl ?? 0) >= 0 ? 'var(--up)' : 'var(--down)',
              }}>
                당일 손익 {fmtSigned(group.day_pnl ?? 0)}
              </span>
            </div>

            {(group.trades || []).map(t => (
              <div className="trade-card" key={t.id}>
                <div className="trade-card-main">
                  <div>
                    <div style={{ fontWeight: 700, fontSize: 15 }}>{t.stock_name}</div>
                    <div className="mono muted" style={{ fontSize: 11, marginBottom: 8 }}>{t.stock_code}</div>
                    <div className="muted" style={{ fontSize: 12 }}>
                      {t.buy_time} <span style={{ color: 'var(--text-dim)' }}>→</span> {t.sell_time || '—'}
                    </div>
                    <div className="muted" style={{ fontSize: 12 }}>보유: {t.hold_period || '—'}</div>
                  </div>
                  <div>
                    <div className="muted" style={{ fontSize: 11, marginBottom: 4 }}>매수가 → 매도가</div>
                    <div className="mono" style={{ fontSize: 13 }}>
                      {fmt(t.buy_price)} <span style={{ color: 'var(--text-dim)' }}>→</span> {t.sell_price ? fmt(t.sell_price) : <span className="muted">—</span>}
                    </div>
                    <div className="mono" style={{
                      fontSize: 20, fontWeight: 700, marginTop: 6,
                      color: (t.pnl_amount ?? 0) >= 0 ? 'var(--up)' : 'var(--down)',
                    }}>
                      {fmtSigned(t.pnl_amount ?? 0)}
                    </div>
                    <div className="mono" style={{ fontSize: 14, color: (t.pnl_pct ?? 0) >= 0 ? 'var(--up)' : 'var(--down)' }}>
                      {fmtPct(t.pnl_pct ?? 0)}
                    </div>
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-end', justifyContent: 'space-between' }}>
                    <SellReasonBadge reason={t.sell_reason} />
                    <button className="btn btn-outline btn-xs"
                      onClick={() => setExpandedId(expandedId === t.id ? null : t.id)}>
                      {expandedId === t.id ? '▲ 접기' : '▼ 지표 보기'}
                    </button>
                  </div>
                </div>

                {expandedId === t.id && t.indicators && (
                  <div className="trade-card-expand">
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 12 }}>
                      {[
                        ['RSI', t.indicators.rsi?.toFixed(1), '#F59E0B'],
                        ['MACD', t.indicators.macd_bullish ? 'BULL' : 'BEAR', '#3B82F6'],
                        ['VWAP 이격', t.indicators.vwap_disparity != null ? t.indicators.vwap_disparity.toFixed(2) + '%' : '—', '#8B5CF6'],
                        ['체결강도', t.indicators.strength?.toFixed(1), 'var(--up)'],
                        ['종합점수', t.indicators.total_score?.toFixed(1), '#EA6C10'],
                      ].map(([label, value, color]) => (
                        <div key={label} style={{ background: 'var(--surface)', borderRadius: 6, padding: '10px 12px' }}>
                          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>{label}</div>
                          <div className="mono" style={{ fontSize: 16, fontWeight: 700, color }}>{value ?? '—'}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        ))
      )}

      <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 16 }}>
        <button className="btn btn-outline btn-xs" disabled={page === 1} onClick={() => setPage(p => p - 1)}>← 이전</button>
        <span className="muted" style={{ padding: '2px 8px', fontSize: 12 }}>페이지 {page}</span>
        <button className="btn btn-outline btn-xs" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>다음 →</button>
      </div>
    </div>
  )
}
