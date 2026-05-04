import { useState } from 'react'
import useApi from '../hooks/useApi'
import { fmt, fmtPct, fmtSigned, Badge } from '../components/shared'

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

export default function TradeReports() {
  const [dateFilter, setDateFilter] = useState('')
  const [codeFilter, setCodeFilter] = useState('')
  const [expandedId, setExpandedId] = useState(null)
  const [page, setPage] = useState(1)

  const query = new URLSearchParams({ page, limit: 20 })
  if (dateFilter) query.set('date', dateFilter)
  if (codeFilter) query.set('stock_code', codeFilter)

  const { data, loading } = useApi(`/api/reports/trades?${query}`)
  const groups = data?.data || []

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
      ) : groups.length === 0 ? (
        <div className="card">
          <div style={{ textAlign: 'center', padding: 48, color: 'var(--text-muted)' }}>거래 리포트가 없습니다</div>
        </div>
      ) : (
        groups.map(group => (
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
        <button className="btn btn-outline btn-xs" disabled={groups.length < 20} onClick={() => setPage(p => p + 1)}>다음 →</button>
      </div>
    </div>
  )
}
