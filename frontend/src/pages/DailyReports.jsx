import { useState, useEffect } from 'react'
import { collection, query, orderBy, limit, getDocs } from 'firebase/firestore'
import { fmtPct, fmtSigned, fmt } from '../components/shared'
import { db } from '../lib/firebase'

function transformReport(d) {
  const r = { _docId: d.id, ...d.data() }
  const totalTrades = r.total_trades || 0
  const winningTrades = r.winning_trades || 0
  const parseSafe = (s) => { try { return JSON.parse(s || 'null') } catch { return null } }
  return {
    _docId: r._docId,
    date: r.date,
    trade_count: totalTrades,
    wins: winningTrades,
    losses: r.losing_trades || 0,
    pnl: r.total_profit_amount || 0,
    pnl_pct: r.avg_profit_pct || 0,
    win_rate: totalTrades > 0 ? (winningTrades / totalTrades) * 100 : 0,
    trades: parseSafe(r.trade_summary) || [],
    best_trade: parseSafe(r.best_trade),
    worst_trade: parseSafe(r.worst_trade),
  }
}

function TradeHighlight({ trade, label, color }) {
  if (!trade) return null
  const isProfit = (trade.profit_amount ?? 0) >= 0
  return (
    <div style={{
      background: color,
      border: `1px solid ${isProfit ? 'rgba(34,197,94,0.3)' : 'rgba(239,68,68,0.3)'}`,
      borderRadius: 6,
      padding: '10px 12px',
    }}>
      <div style={{ fontSize: 10, fontWeight: 600, color: 'var(--text-muted)', marginBottom: 4, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        {label}
      </div>
      <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 2 }}>{trade.stock_name || trade.stock_code}</div>
      <div style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>
        <span style={{ color: isProfit ? 'var(--up)' : 'var(--down)', fontWeight: 600 }}>
          {fmtSigned(trade.profit_amount ?? 0)}
        </span>
        <span style={{ color: 'var(--text-muted)', marginLeft: 6 }}>
          ({fmtPct(trade.profit_pct ?? 0)})
        </span>
      </div>
    </div>
  )
}

function TradeDetail({ r }) {
  const hasBest = r.best_trade != null
  const hasWorst = r.worst_trade != null
  const trades = Array.isArray(r.trades) ? r.trades : []

  return (
    <div style={{
      padding: '12px 14px',
      background: 'var(--bg)',
      borderBottom: '1px solid var(--border)',
    }}>
      {(hasBest || hasWorst) && (
        <div style={{
          display: 'grid',
          gridTemplateColumns: hasBest && hasWorst ? '1fr 1fr' : '1fr',
          gap: 8,
          marginBottom: 12,
        }}>
          {hasBest && (
            <TradeHighlight
              trade={r.best_trade}
              label="🏆 최고 거래"
              color="rgba(34,197,94,0.08)"
            />
          )}
          {hasWorst && (
            <TradeHighlight
              trade={r.worst_trade}
              label="최악 거래"
              color="rgba(239,68,68,0.08)"
            />
          )}
        </div>
      )}

      {trades.length > 0 ? (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border)' }}>
                {['종목명', '매수가→매도가', '수량', '손익금액', '손익률', '매도이유'].map(h => (
                  <th key={h} style={{
                    textAlign: 'left',
                    padding: '4px 8px',
                    fontWeight: 600,
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    whiteSpace: 'nowrap',
                  }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {trades.map((t, i) => {
                const isProfit = (t.profit_amount ?? 0) >= 0
                return (
                  <tr key={t.id ?? i} style={{ borderBottom: '1px solid var(--border)' }}>
                    <td style={{ padding: '5px 8px', fontWeight: 500 }}>
                      <div>{t.stock_name || t.stock_code}</div>
                      <div style={{ color: 'var(--text-muted)', fontSize: 10 }}>{t.stock_code}</div>
                    </td>
                    <td style={{ padding: '5px 8px', fontFamily: 'var(--font-mono)', whiteSpace: 'nowrap' }}>
                      {fmt(t.buy_price)}
                      <span style={{ color: 'var(--text-muted)', margin: '0 4px' }}>→</span>
                      {fmt(t.sell_price)}
                    </td>
                    <td style={{ padding: '5px 8px', fontFamily: 'var(--font-mono)' }}>
                      {t.qty ?? '—'}
                    </td>
                    <td style={{ padding: '5px 8px', fontFamily: 'var(--font-mono)', color: isProfit ? 'var(--up)' : 'var(--down)', fontWeight: 600 }}>
                      {fmtSigned(t.profit_amount ?? 0)}
                    </td>
                    <td style={{ padding: '5px 8px', fontFamily: 'var(--font-mono)', color: isProfit ? 'var(--up)' : 'var(--down)' }}>
                      {fmtPct(t.profit_pct ?? 0)}
                    </td>
                    <td style={{ padding: '5px 8px', color: 'var(--text-muted)', maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                      title={t.sell_reason || ''}>
                      {t.sell_reason || '—'}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      ) : (
        <div style={{ fontSize: 12, color: 'var(--text-muted)', textAlign: 'center', padding: '8px 0' }}>
          거래 내역이 없습니다
        </div>
      )}
    </div>
  )
}

export default function DailyReports() {
  const [reports, setReports] = useState([])
  const [loading, setLoading] = useState(true)
  const [expandedDate, setExpandedDate] = useState(null)

  useEffect(() => {
    const q = query(collection(db, 'daily_reports'), orderBy('date', 'desc'), limit(50))
    getDocs(q)
      .then(snap => setReports(snap.docs.map(d => transformReport(d))))
      .finally(() => setLoading(false))
  }, [])

  const totalPnl = reports.reduce((s, r) => s + (r.pnl ?? 0), 0)
  const avgWinRate = reports.length > 0
    ? reports.reduce((s, r) => s + (r.win_rate ?? 0), 0) / reports.length
    : 0
  const totalTrades = reports.reduce((s, r) => s + (r.trade_count ?? 0), 0)
  const avgPnlPct = reports.length > 0
    ? reports.reduce((s, r) => s + (r.pnl_pct ?? 0), 0) / reports.length
    : 0

  const maxAbs = Math.max(...reports.map(r => Math.abs(r.pnl ?? 0)), 1)

  return (
    <div>
      <div className="summary-grid">
        {[
          ['총 누적 손익', fmtSigned(totalPnl), totalPnl >= 0 ? 'var(--up)' : 'var(--down)'],
          ['평균 승률', `${avgWinRate.toFixed(1)}%`, 'var(--accent)'],
          ['평균 일 수익률', fmtPct(avgPnlPct), avgPnlPct >= 0 ? 'var(--up)' : 'var(--down)'],
          ['총 거래 횟수', `${totalTrades}건`, ''],
        ].map(([label, value, color]) => (
          <div key={label} className="stat-card">
            <div className="stat-label">{label}</div>
            <div className="stat-value mono" style={{ color: color || 'inherit' }}>{value}</div>
          </div>
        ))}
      </div>

      {reports.length > 0 && (
        <div className="card" style={{ marginBottom: 16 }}>
          <div className="card-header">
            <span className="card-title">최근 일별 손익</span>
          </div>
          <div className="card-body">
            <div className="bar-chart">
              {[...reports].reverse().slice(0, 30).map(r => {
                const h = Math.round((Math.abs(r.pnl ?? 0) / maxAbs) * 90)
                return (
                  <div key={r.date} className="bar-wrap">
                    <div className="bar"
                      style={{ height: Math.max(h, 2), background: (r.pnl ?? 0) >= 0 ? 'var(--up)' : 'var(--down)' }}
                      title={`${r.date}: ${fmtSigned(r.pnl ?? 0)}`}
                    ></div>
                    <div className="bar-label">{(r.date || '').slice(5)}</div>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}

      <div className="card">
        {loading ? (
          <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)' }}>로딩 중...</div>
        ) : (
          <>
            <div style={{ padding: '8px 14px', borderBottom: '1px solid var(--border)' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '120px 60px 80px 80px 1fr 80px', gap: 12, padding: '4px 0' }}>
                {['날짜', '거래', '승/패', '승률', '당일 손익', '손익률'].map(h => (
                  <div key={h} style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase' }}>{h}</div>
                ))}
              </div>
            </div>
            {reports.length === 0 ? (
              <div style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)' }}>일별 리포트가 없습니다</div>
            ) : reports.map(r => {
              const pnl = r.pnl ?? 0
              const winRate = r.win_rate ?? 0
              const wins = r.wins ?? 0
              const losses = r.losses ?? 0
              const tradeCount = r.trade_count ?? 0
              const pnlPct = r.pnl_pct ?? 0
              return (
                <div key={r.date}>
                  <div
                    className={`daily-row ${pnl >= 0 ? 'profit' : 'loss'}`}
                    onClick={() => setExpandedDate(expandedDate === r.date ? null : r.date)}>
                    <div className="mono" style={{ fontSize: 12 }}>{r.date}</div>
                    <div className="mono">{tradeCount}</div>
                    <div className="mono">
                      <span style={{ color: 'var(--up)' }}>{wins}</span>
                      <span className="muted">/</span>
                      <span style={{ color: 'var(--down)' }}>{losses}</span>
                    </div>
                    <div className="mono" style={{ color: winRate >= 60 ? 'var(--up)' : 'var(--down)' }}>
                      {winRate.toFixed(1)}%
                    </div>
                    <div className="mono" style={{ fontWeight: 600, color: pnl >= 0 ? 'var(--up)' : 'var(--down)' }}>
                      {fmtSigned(pnl)}
                    </div>
                    <div className="mono" style={{ color: pnlPct >= 0 ? 'var(--up)' : 'var(--down)' }}>
                      {fmtPct(pnlPct)}
                    </div>
                  </div>
                  {expandedDate === r.date && <TradeDetail r={r} />}
                </div>
              )
            })}
          </>
        )}
      </div>
    </div>
  )
}
