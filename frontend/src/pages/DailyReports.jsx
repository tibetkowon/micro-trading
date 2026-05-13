import { useState, useEffect } from 'react'
import { collection, query, orderBy, limit, getDocs } from 'firebase/firestore'
import { fmtPct, fmtSigned, fmt } from '../components/shared'
import { db } from '../lib/firebase'

const parseSafe = (s) => { try { return JSON.parse(s || 'null') } catch { return null } }
const apiBase = import.meta.env.VITE_API_BASE_URL || ''
const todayKST = () => new Date(Date.now() + 9 * 60 * 60 * 1000).toISOString().slice(0, 10)

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
                  <tr key={`${t.stock_code ?? ''}-${i}`} style={{ borderBottom: '1px solid var(--border)' }}>
                    <td style={{ padding: '5px 8px', fontWeight: 500 }}>
                      <div>{t.stock_name || t.stock_code}</div>
                      {t.stock_name && (
                        <div style={{ color: 'var(--text-muted)', fontSize: 10 }}>{t.stock_code}</div>
                      )}
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
  const [exportModal, setExportModal] = useState(false)
  const [exportFrom, setExportFrom] = useState('')
  const [exportTo, setExportTo] = useState('')
  const [exportLoading, setExportLoading] = useState(false)
  const [exportError, setExportError] = useState('')
  const [simResult, setSimResult] = useState(null)
  const [simLoading, setSimLoading] = useState(false)
  const [simRunning, setSimRunning] = useState(false)
  const [simDate, setSimDate] = useState(todayKST())
  const [activeTab, setActiveTab] = useState('daily')

  useEffect(() => {
    const q = query(collection(db, 'daily_reports'), orderBy('date', 'desc'), limit(50))
    getDocs(q)
      .then(snap => setReports(snap.docs.map(d => transformReport(d))))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (reports.length > 0 && !exportFrom && !exportTo) {
      const dates = reports.map(r => r.date).filter(Boolean)
      setExportFrom(dates[dates.length - 1] || '')
      setExportTo(dates[0] || '')
      setSimDate(dates[0] || todayKST())
    }
  }, [reports, exportFrom, exportTo])

  const fetchSimulation = async (date) => {
    setSimLoading(true)
    try {
      const res = await fetch(`${apiBase}/api/simulation/${date}`)
      if (!res.ok) {
        setSimResult(null)
        return
      }
      setSimResult(await res.json())
    } catch {
      setSimResult(null)
    } finally {
      setSimLoading(false)
    }
  }

  const runSimulation = async (date) => {
    setSimRunning(true)
    try {
      await fetch(`${apiBase}/api/simulation/run?date=${date}`, { method: 'POST' })
      setTimeout(() => fetchSimulation(date), 30000)
    } finally {
      setSimRunning(false)
    }
  }

  const handleExport = async (action) => {
    if (!exportFrom || !exportTo) {
      setExportError('시작일과 종료일을 모두 입력해주세요.')
      return
    }
    setExportLoading(true)
    setExportError('')
    try {
      const res = await fetch(`${apiBase}/api/reports/export?from=${exportFrom}&to=${exportTo}`)
      if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || '내보내기 실패')
      }
      const data = await res.json()
      const json = JSON.stringify(data, null, 2)

      if (action === 'download') {
        const blob = new Blob([json], { type: 'application/json' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `trading-report-${exportFrom}-${exportTo}.json`
        a.click()
        URL.revokeObjectURL(url)
      } else {
        await navigator.clipboard.writeText(json)
        alert('클립보드에 복사되었습니다.')
      }
      setExportModal(false)
    } catch (e) {
      setExportError(e.message)
    } finally {
      setExportLoading(false)
    }
  }

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

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 8, borderBottom: '1px solid var(--border)', flex: 1 }}>
          <button
            onClick={() => setActiveTab('daily')}
            style={{
              padding: '8px 12px',
              border: 0,
              borderBottom: activeTab === 'daily' ? '2px solid var(--accent)' : '2px solid transparent',
              background: 'transparent',
              color: activeTab === 'daily' ? 'var(--accent)' : 'var(--text-muted)',
              fontSize: 13,
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            일별 리포트
          </button>
          <button
            onClick={() => { setActiveTab('simulation'); fetchSimulation(simDate || todayKST()) }}
            style={{
              padding: '8px 12px',
              border: 0,
              borderBottom: activeTab === 'simulation' ? '2px solid var(--accent)' : '2px solid transparent',
              background: 'transparent',
              color: activeTab === 'simulation' ? 'var(--accent)' : 'var(--text-muted)',
              fontSize: 13,
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            시뮬레이션
          </button>
        </div>
        <button
          onClick={() => setExportModal(true)}
          style={{
            padding: '7px 12px',
            border: '1px solid var(--accent)',
            borderRadius: 6,
            background: 'var(--accent)',
            color: '#fff',
            fontSize: 12,
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          LLM Export
        </button>
      </div>

      {activeTab === 'daily' && reports.length > 0 && (
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

      {activeTab === 'daily' && <div className="card">
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
      </div>}

      {activeTab === 'simulation' && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">시뮬레이션</span>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input
                type="date"
                value={simDate}
                onChange={e => setSimDate(e.target.value)}
                style={{
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  padding: '5px 8px',
                  background: 'var(--bg)',
                  color: 'var(--text)',
                  fontSize: 12,
                }}
              />
              <button
                onClick={() => fetchSimulation(simDate || todayKST())}
                disabled={simLoading}
                style={{
                  padding: '6px 10px',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  background: 'var(--surface)',
                  color: 'var(--text)',
                  fontSize: 12,
                  cursor: 'pointer',
                  opacity: simLoading ? 0.6 : 1,
                }}
              >
                조회
              </button>
            </div>
          </div>
          <div className="card-body">
            {simLoading && <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)' }}>시뮬레이션 결과 로딩 중...</div>}
            {!simLoading && !simResult && (
              <div style={{ textAlign: 'center', padding: '32px 0' }}>
                <div style={{ color: 'var(--text-muted)', fontSize: 13, marginBottom: 12 }}>이 날짜의 시뮬레이션 결과가 없습니다.</div>
                <button
                  onClick={() => runSimulation(simDate || todayKST())}
                  disabled={simRunning}
                  style={{
                    padding: '8px 14px',
                    border: 0,
                    borderRadius: 6,
                    background: 'var(--accent)',
                    color: '#fff',
                    fontSize: 13,
                    fontWeight: 600,
                    cursor: 'pointer',
                    opacity: simRunning ? 0.6 : 1,
                  }}
                >
                  {simRunning ? '실행 중...' : '지금 시뮬레이션 실행'}
                </button>
                {simRunning && <div style={{ marginTop: 8, fontSize: 11, color: 'var(--text-muted)' }}>30초 후 자동으로 결과를 불러옵니다.</div>}
              </div>
            )}
            {!simLoading && simResult && (
              <div style={{ display: 'grid', gap: 16 }}>
                <div style={{
                  background: 'rgba(99,102,241,0.08)',
                  border: '1px solid rgba(99,102,241,0.25)',
                  borderRadius: 6,
                  padding: 14,
                }}>
                  <div style={{ fontSize: 15, fontWeight: 700, color: 'var(--accent)', marginBottom: 4 }}>
                    추천 설정: {simResult.recommended?.label || '현재 설정'}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 10 }}>
                    {simResult.recommended?.reason || '추천 사유가 없습니다.'}
                  </div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 12 }}>
                    {simResult.recommended?.params?.take_profit_pct > 0 && (
                      <span style={{ fontSize: 11, padding: '4px 7px', border: '1px solid var(--border)', borderRadius: 6, background: 'var(--surface)' }}>
                        목표가 {simResult.recommended.params.take_profit_pct}%
                      </span>
                    )}
                    {simResult.recommended?.params?.stop_loss_pct > 0 && (
                      <span style={{ fontSize: 11, padding: '4px 7px', border: '1px solid var(--border)', borderRadius: 6, background: 'var(--surface)' }}>
                        손절 {simResult.recommended.params.stop_loss_pct}%
                      </span>
                    )}
                    {simResult.recommended?.params?.trailing_trigger_pct > 0 && (
                      <span style={{ fontSize: 11, padding: '4px 7px', border: '1px solid var(--border)', borderRadius: 6, background: 'var(--surface)' }}>
                        트레일링 트리거 {simResult.recommended.params.trailing_trigger_pct}%
                      </span>
                    )}
                    {simResult.recommended?.params?.trailing_stop_pct > 0 && (
                      <span style={{ fontSize: 11, padding: '4px 7px', border: '1px solid var(--border)', borderRadius: 6, background: 'var(--surface)' }}>
                        트레일링 스탑 {simResult.recommended.params.trailing_stop_pct}%
                      </span>
                    )}
                  </div>
                  <button
                    onClick={async () => {
                      const p = simResult.recommended?.params || {}
                      await fetch(`${apiBase}/api/settings`, {
                        method: 'PATCH',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                          take_profit_pct: p.take_profit_pct,
                          stop_loss_pct: p.stop_loss_pct,
                          trailing_trigger_pct: p.trailing_trigger_pct,
                          trailing_stop_pct: p.trailing_stop_pct,
                        }),
                      })
                      alert('설정이 적용되었습니다.')
                    }}
                    style={{
                      padding: '6px 10px',
                      border: 0,
                      borderRadius: 6,
                      background: 'var(--accent)',
                      color: '#fff',
                      fontSize: 12,
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    이 설정 적용
                  </button>
                </div>

                <div style={{ overflowX: 'auto' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                    <thead>
                      <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-muted)' }}>
                        <th style={{ textAlign: 'left', padding: '8px 10px' }}>시나리오</th>
                        <th style={{ textAlign: 'right', padding: '8px 10px' }}>총 손익률</th>
                        <th style={{ textAlign: 'right', padding: '8px 10px' }}>승률</th>
                        <th style={{ textAlign: 'right', padding: '8px 10px' }}>실제 대비</th>
                        <th style={{ textAlign: 'right', padding: '8px 10px' }}>평균 보유(분)</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(simResult.scenarios || []).map((s, i) => (
                        <tr key={`${s.label}-${i}`} style={{
                          borderBottom: '1px solid var(--border)',
                          background: s.label === '현재 설정' ? 'rgba(148,163,184,0.10)' : 'transparent',
                          fontWeight: s.label === '현재 설정' ? 600 : 400,
                        }}>
                          <td style={{ padding: '8px 10px' }}>{s.label}</td>
                          <td className="mono" style={{ padding: '8px 10px', textAlign: 'right', color: s.total_pnl_pct >= 0 ? 'var(--up)' : 'var(--down)' }}>
                            {s.total_pnl_pct >= 0 ? '+' : ''}{Number(s.total_pnl_pct || 0).toFixed(2)}%
                          </td>
                          <td className="mono" style={{ padding: '8px 10px', textAlign: 'right' }}>{Number(s.win_rate_pct || 0).toFixed(1)}%</td>
                          <td className="mono" style={{ padding: '8px 10px', textAlign: 'right', color: s.delta_vs_actual_pct > 0 ? 'var(--up)' : s.delta_vs_actual_pct < 0 ? 'var(--down)' : 'inherit' }}>
                            {s.delta_vs_actual_pct > 0 ? '+' : ''}{Number(s.delta_vs_actual_pct || 0).toFixed(2)}%
                          </td>
                          <td className="mono" style={{ padding: '8px 10px', textAlign: 'right' }}>{Number(s.avg_holding_minutes || 0).toFixed(0)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {exportModal && (
        <div style={{
          position: 'fixed',
          inset: 0,
          background: 'rgba(0,0,0,0.5)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 50,
          padding: 16,
        }}>
          <div style={{
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: 20,
            width: 384,
            maxWidth: '100%',
            boxShadow: '0 20px 50px rgba(0,0,0,0.25)',
          }}>
            <h2 style={{ fontSize: 18, fontWeight: 700, marginBottom: 6 }}>LLM Export</h2>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 14 }}>
              선택한 기간의 거래 리포트와 현재 설정을 JSON으로 내보냅니다.
            </p>
            <div style={{ display: 'grid', gap: 10 }}>
              <label style={{ display: 'grid', gap: 5, fontSize: 12, fontWeight: 600 }}>
                시작일
                <input
                  type="date"
                  value={exportFrom}
                  onChange={e => setExportFrom(e.target.value)}
                  style={{ border: '1px solid var(--border)', borderRadius: 6, padding: '7px 9px', background: 'var(--bg)', color: 'var(--text)' }}
                />
              </label>
              <label style={{ display: 'grid', gap: 5, fontSize: 12, fontWeight: 600 }}>
                종료일
                <input
                  type="date"
                  value={exportTo}
                  onChange={e => setExportTo(e.target.value)}
                  style={{ border: '1px solid var(--border)', borderRadius: 6, padding: '7px 9px', background: 'var(--bg)', color: 'var(--text)' }}
                />
              </label>
              {exportError && <div style={{ fontSize: 12, color: 'var(--down)' }}>{exportError}</div>}
              <div style={{ display: 'flex', gap: 8, paddingTop: 6 }}>
                <button
                  onClick={() => handleExport('download')}
                  disabled={exportLoading}
                  style={{ flex: 1, padding: '8px 10px', border: 0, borderRadius: 6, background: 'var(--accent)', color: '#fff', fontSize: 12, fontWeight: 600, cursor: 'pointer', opacity: exportLoading ? 0.6 : 1 }}
                >
                  {exportLoading ? '처리 중...' : 'JSON 다운로드'}
                </button>
                <button
                  onClick={() => handleExport('copy')}
                  disabled={exportLoading}
                  style={{ flex: 1, padding: '8px 10px', border: '1px solid var(--border)', borderRadius: 6, background: 'var(--bg)', color: 'var(--text)', fontSize: 12, fontWeight: 600, cursor: 'pointer', opacity: exportLoading ? 0.6 : 1 }}
                >
                  클립보드 복사
                </button>
                <button
                  onClick={() => { setExportModal(false); setExportError('') }}
                  style={{ padding: '8px 10px', border: '1px solid var(--border)', borderRadius: 6, background: 'transparent', color: 'var(--text)', fontSize: 12, cursor: 'pointer' }}
                >
                  취소
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
