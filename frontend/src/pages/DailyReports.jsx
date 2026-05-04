import { useState, useEffect } from 'react'
import { collection, query, orderBy, limit, getDocs } from 'firebase/firestore'
import { fmtPct, fmtSigned } from '../components/shared'
import { db } from '../lib/firebase'

function transformReport(d) {
  const r = { _docId: d.id, ...d.data() }
  const totalTrades = r.total_trades || 0
  const winningTrades = r.winning_trades || 0
  return {
    _docId: r._docId,
    date: r.date,
    trade_count: totalTrades,
    wins: winningTrades,
    losses: r.losing_trades || 0,
    pnl: r.total_profit_amount || 0,
    pnl_pct: r.avg_profit_pct || 0,
    win_rate: totalTrades > 0 ? (winningTrades / totalTrades) * 100 : 0,
    report_summary: r.trade_summary || '',
  }
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
                  {expandedDate === r.date && (
                    <div style={{
                      padding: '12px 14px',
                      background: 'var(--bg)',
                      borderBottom: '1px solid var(--border)',
                      fontSize: 12,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}>
                      {r.report_summary || `${r.date} 요약: 총 ${tradeCount}건 거래 중 ${wins}건 이익, ${losses}건 손실. 승률 ${winRate.toFixed(1)}%, 당일 손익 ${fmtSigned(pnl)} (${fmtPct(pnlPct)})`}
                    </div>
                  )}
                </div>
              )
            })}
          </>
        )}
      </div>
    </div>
  )
}
