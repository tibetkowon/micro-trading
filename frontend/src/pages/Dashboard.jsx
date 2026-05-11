import { useState, useCallback } from 'react'
import useApi from '../hooks/useApi'
import { fmt, fmtPct, fmtSigned, Badge, BotStateBadge, PriceProgressBar } from '../components/shared'
import ScanRawModal from '../components/ScanRawModal'

export default function Dashboard() {
  const { data: balance, loading: balLoading } = useApi('/api/balance')
  const { data: positions } = useApi('/api/monitor/positions', { pollInterval: 3000 })
  const { data: traderStatus } = useApi('/api/server/status')
  const { data: scanLogs } = useApi('/api/logs/scan?limit=5')

  const [expanded, setExpanded] = useState({})
  const [rawModal, setRawModal] = useState(null)

  const toggleExpand = useCallback((id) => {
    setExpanded(prev => ({ ...prev, [id]: !prev[id] }))
  }, [])

  const stats = balance?.data || {}
  const pos = positions?.data || []
  const botState = traderStatus?.trader_state || 'IDLE'
  const logs = scanLogs?.data || []

  const todayPnl = stats.today_pnl ?? 0
  const totalAssets = stats.total_assets ?? 0
  const availableCash = stats.available_cash ?? 0
  const winRate = stats.win_rate ?? 0
  const totalAssetsChange = stats.total_assets_change_pct ?? 0

  return (
    <div>
      {/* Stat cards */}
      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-label">총 자산</div>
          <div className="stat-value">{balLoading ? '—' : fmt(totalAssets)}</div>
          <div className="stat-change" style={{ color: totalAssetsChange >= 0 ? 'var(--up)' : 'var(--down)' }}>
            {fmtPct(totalAssetsChange)} 오늘
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">가용 현금</div>
          <div className="stat-value">{balLoading ? '—' : fmt(availableCash)}</div>
          <div className="stat-change muted">즉시 사용 가능</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">오늘 손익</div>
          <div className="stat-value" style={{ color: todayPnl >= 0 ? 'var(--up)' : 'var(--down)' }}>
            {fmtSigned(todayPnl)}
          </div>
          <div className="stat-change" style={{ color: todayPnl >= 0 ? 'var(--up)' : 'var(--down)' }}>
            {fmtPct(stats.today_pnl_pct ?? 0)}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">승률 (오늘)</div>
          <div className="stat-value accent">{winRate.toFixed(1)}%</div>
          <div className="stat-change muted">체결 완료 기준</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">봇 상태</div>
          <div style={{ marginTop: 8 }}>
            <BotStateBadge state={botState} />
          </div>
        </div>
      </div>

      {/* Active positions summary */}
      <div className="card" style={{ marginBottom: 16 }}>
        <div className="card-header">
          <span className="card-title">보유 포지션</span>
          <Badge color="orange">{pos.length}종목</Badge>
        </div>
        <div className="table-wrap">
          {pos.length === 0 ? (
            <div style={{ padding: '24px 16px', textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
              현재 보유 중인 포지션이 없습니다
            </div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>종목</th>
                  <th>수량</th>
                  <th>현재가</th>
                  <th>수익률</th>
                </tr>
              </thead>
              <tbody>
                {pos.map(p => {
                  const range = (p.target_price ?? 0) - (p.stop_price ?? 0)
                  const nearStop = range > 0 && ((p.current_price - p.stop_price) / range) < 0.15
                  const pnlPct = p.pnl_pct ?? 0
                  return (
                    <tr key={p.stock_code} className={nearStop ? 'row-near-stop' : ''}>
                      <td>
                        <div style={{ fontWeight: 600, fontSize: 13 }}>{p.stock_name}</div>
                        <div className="mono muted" style={{ fontSize: 11 }}>{p.stock_code}</div>
                      </td>
                      <td className="mono">{p.quantity}주</td>
                      <td className="mono">{fmt(p.current_price)}</td>
                      <td>
                        <div className="mono" style={{ fontWeight: 600, color: pnlPct >= 0 ? 'var(--up)' : 'var(--down)' }}>
                          {fmtPct(pnlPct)}
                        </div>
                        {p.stop_price && p.target_price && (
                          <div style={{ marginTop: 4, width: 80 }}>
                            <PriceProgressBar
                              stop={p.stop_price}
                              avg={p.avg_price}
                              target={p.target_price}
                              current={p.current_price}
                            />
                          </div>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Scan logs */}
      <div className="card">
        <div className="card-header">
          <span className="card-title">최근 스캔 로그</span>
          <span className="muted" style={{ fontSize: 12 }}>최근 5회</span>
        </div>
        <div style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: 8 }}>
          {logs.length === 0 ? (
            <div style={{ padding: '16px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
              스캔 로그가 없습니다
            </div>
          ) : (
            logs.map(log => {
              const isOpen = expanded[log.id]
              const stocks = log.stocks || []
              return (
                <div className="scan-card" key={log.id}>
                  <div className="scan-card-header" onClick={() => toggleExpand(log.id)}>
                    <span className="mono muted" style={{ fontSize: 11, width: 160, flexShrink: 0 }}>
                      {log.scanned_at || log.created_at}
                    </span>
                    <span className="muted" style={{ fontSize: 12, flex: 1 }}>
                      평가 <strong style={{ color: 'var(--text)' }}>{log.evaluated_count ?? 0}</strong>종목 →
                      하드필터 통과 <strong style={{ color: 'var(--text)' }}>{log.passed_hard_filter ?? 0}</strong> →
                      점수 산정 <strong style={{ color: 'var(--text)' }}>{log.scored_count ?? 0}</strong>
                    </span>
                    {log.selected_stock_code
                      ? <Badge color="green">선정: {log.selected_stock_name || log.selected_stock_code}</Badge>
                      : <Badge color="gray">미선정</Badge>}
                    <span style={{ color: 'var(--text-dim)', fontSize: 12, marginLeft: 8 }}>{isOpen ? '▲' : '▼'}</span>
                  </div>
                  {isOpen && (
                    <div className="scan-card-body">
                      {log.rejection_summary && (
                        <div style={{ marginBottom: 8, fontSize: 12, color: 'var(--text-muted)' }}>
                          <strong style={{ color: 'var(--red)' }}>제외 사유:</strong> {log.rejection_summary}
                        </div>
                      )}
                      {stocks.length > 0 && (
                        <div className="table-wrap">
                          <table>
                            <thead>
                              <tr>
                                <th>종목</th>
                                <th>거래량</th>
                                <th>체결강도</th>
                                <th>RSI</th>
                                <th>MACD</th>
                                <th>호가비율</th>
                                <th>미시호가</th>
                                <th>VWAP</th>
                                <th>거래량비율</th>
                                <th>순매수</th>
                                <th>VI이격</th>
                                <th>총점</th>
                                <th>상세</th>
                              </tr>
                            </thead>
                            <tbody>
                              {stocks.map(s => {
                                const volNum = s.volume ? parseInt(s.volume, 10) : 0
                                const pgb = s.program_net_buy ?? 0
                                return (
                                <tr key={s.stock_code}>
                                  <td>
                                    <span style={{ fontWeight: 600 }}>{s.stock_name !== s.stock_code ? s.stock_name : s.stock_code}</span>
                                    {s.stock_name !== s.stock_code && <span className="mono muted" style={{ fontSize: 11, marginLeft: 4 }}>{s.stock_code}</span>}
                                  </td>
                                  <td className="mono">{volNum > 0 ? volNum.toLocaleString('ko-KR') : '—'}</td>
                                  <td className="mono">{s.strength > 0 ? s.strength.toFixed(1) : '—'}</td>
                                  <td className="mono">{s.has_chart && s.rsi > 0 ? s.rsi.toFixed(1) : '—'}</td>
                                  <td>
                                    {s.has_chart
                                      ? <Badge color={s.macd_bullish ? 'green' : 'red'}>{s.macd_bullish ? 'BULL' : 'BEAR'}</Badge>
                                      : '—'}
                                  </td>
                                  <td className="mono">{s.bid_ask_ratio > 0 ? s.bid_ask_ratio.toFixed(2) : '—'}</td>
                                  <td className="mono">{s.micro_bid_ask > 0 ? s.micro_bid_ask.toFixed(2) : '—'}</td>
                                  <td className="mono">{s.has_chart && s.vwap_disparity != null ? s.vwap_disparity.toFixed(2) + '%' : '—'}</td>
                                  <td className="mono">{s.has_chart && s.volume_ratio > 0 ? s.volume_ratio.toFixed(2) : '—'}</td>
                                  <td className="mono" style={{ color: pgb > 0 ? 'var(--up)' : pgb < 0 ? 'var(--down)' : undefined }}>
                                    {pgb !== 0 ? pgb.toLocaleString('ko-KR') : '—'}
                                  </td>
                                  <td className="mono">{s.vi_disparity != null && s.vi_disparity !== 0 ? s.vi_disparity.toFixed(2) + '%' : '—'}</td>
                                  <td className="mono accent" style={{ fontWeight: 700 }}>{s.total_score?.toFixed(1) ?? '—'}</td>
                                  <td>
                                    {log.has_raw_data
                                      ? <button style={{ fontSize: 11, padding: '2px 8px', background: 'transparent', border: '1px solid var(--border)', borderRadius: 4, color: 'var(--accent)', cursor: 'pointer' }} onClick={e => { e.stopPropagation(); setRawModal({ logId: log.id, stockCode: s.stock_code, stockName: s.stock_name }) }}>상세</button>
                                      : <span className="muted" style={{ fontSize: 11 }}>—</span>}
                                  </td>
                                </tr>
                                )
                              })}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      </div>
      {rawModal && (
        <ScanRawModal
          logId={rawModal.logId}
          stockCode={rawModal.stockCode}
          stockName={rawModal.stockName}
          onClose={() => setRawModal(null)}
        />
      )}
    </div>
  )
}
