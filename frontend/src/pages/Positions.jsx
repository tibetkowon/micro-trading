import { useState } from 'react'
import useApi from '../hooks/useApi'
import { apiFetch } from '../utils/api'
import { fmt, fmtPct, fmtSigned, Badge, PriceProgressBar, Modal, EmptyState } from '../components/shared'

export default function Positions() {
  const { data, loading, refetch } = useApi('/api/monitor/positions')
  const [confirmAll, setConfirmAll] = useState(false)
  const [confirmSell, setConfirmSell] = useState(null)
  const [acting, setActing] = useState(false)

  const positions = data?.data || []

  async function forceSell(code) {
    setActing(true)
    try {
      await apiFetch(`/api/monitor/positions/${code}/sell`, { method: 'POST' })
      refetch()
    } finally {
      setActing(false)
      setConfirmSell(null)
    }
  }

  async function liquidateAll() {
    setActing(true)
    try {
      await apiFetch('/api/monitor/liquidate-all', { method: 'POST' })
      refetch()
    } finally {
      setActing(false)
      setConfirmAll(false)
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <Badge color="orange">{positions.length}개 모니터링 중</Badge>
        <div style={{ flex: 1 }}></div>
        <button className="btn btn-danger" onClick={() => setConfirmAll(true)} disabled={positions.length === 0}>
          전체 청산
        </button>
      </div>

      {loading ? (
        <div className="card">
          <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)' }}>로딩 중...</div>
        </div>
      ) : positions.length === 0 ? (
        <div className="card">
          <EmptyState icon="📭" message="현재 모니터링 중인 포지션이 없습니다" />
        </div>
      ) : (
        <div className="pos-grid">
          {positions.map(p => {
            const range = (p.target_price ?? 0) - (p.stop_price ?? 0)
            const pct = range > 0 ? ((p.current_price - p.stop_price) / range) * 100 : 50
            const nearStop = pct < 15
            const pnlPct = p.pnl_pct ?? 0
            const pnlAmt = p.pnl_amount ?? 0
            const heldDays = p.held_days ?? 0

            return (
              <div key={p.stock_code} className={`pos-card ${nearStop ? 'near-stop' : ''}`}>
                <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 12 }}>
                  <div>
                    <div style={{ fontSize: 16, fontWeight: 700 }}>{p.stock_name}</div>
                    <div className="mono muted" style={{ fontSize: 12 }}>{p.stock_code}</div>
                  </div>
                  <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    <Badge color={p.status === 'MONITORING' ? 'green' : 'yellow'}>
                      {p.status === 'MONITORING' ? 'MONITORING' : 'PENDING SELL'}
                    </Badge>
                    {nearStop && <Badge color="red">⚠ 손절 근접</Badge>}
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'baseline', gap: 12, marginBottom: 12 }}>
                  <div className="mono" style={{ fontSize: 24, fontWeight: 700 }}>{fmt(p.current_price)}</div>
                  <div className="mono" style={{ fontSize: 18, fontWeight: 700, color: pnlPct >= 0 ? 'var(--up)' : 'var(--down)' }}>
                    {fmtPct(pnlPct)}
                  </div>
                </div>

                {p.stop_price && p.target_price && (
                  <div style={{ marginBottom: 12 }}>
                    <PriceProgressBar
                      stop={p.stop_price}
                      avg={p.avg_price}
                      target={p.target_price}
                      current={p.current_price}
                    />
                  </div>
                )}

                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8, marginBottom: 12 }}>
                  {[
                    ['평균단가', fmt(p.avg_price)],
                    ['수량', `${p.quantity}주`],
                    ['보유일', `${heldDays}일`],
                    ['손익금', fmtSigned(pnlAmt)],
                  ].map(([label, value]) => (
                    <div key={label}>
                      <div style={{ fontSize: 10, color: 'var(--text-dim)', marginBottom: 2, textTransform: 'uppercase' }}>{label}</div>
                      <div className="mono" style={{
                        fontSize: 12, fontWeight: 600,
                        color: label === '손익금' ? (pnlAmt >= 0 ? 'var(--up)' : 'var(--down)') : 'inherit',
                      }}>{value}</div>
                    </div>
                  ))}
                </div>

                {p.entry_time && (
                  <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 12 }}>
                    기준시각: <span className="mono">{p.entry_time}</span>
                  </div>
                )}

                <button className="btn btn-danger" style={{ width: '100%', justifyContent: 'center' }}
                  onClick={() => setConfirmSell(p)}>
                  강제 매도
                </button>
              </div>
            )
          })}
        </div>
      )}

      {confirmAll && (
        <Modal title="⚠ 전체 청산 확인" onClose={() => setConfirmAll(false)}
          actions={[
            <button key="c" className="btn btn-outline" onClick={() => setConfirmAll(false)}>취소</button>,
            <button key="d" className="btn btn-danger" onClick={liquidateAll} disabled={acting}>
              {acting ? '청산 중...' : '전체 청산 실행'}
            </button>,
          ]}>
          <p>현재 보유 중인 <strong>{positions.length}개 포지션</strong>을 모두 시장가로 매도합니다.</p>
          <p style={{ marginTop: 8 }}>이 작업은 되돌릴 수 없습니다. 계속하시겠습니까?</p>
        </Modal>
      )}

      {confirmSell && (
        <Modal title="강제 매도 확인" onClose={() => setConfirmSell(null)}
          actions={[
            <button key="c" className="btn btn-outline" onClick={() => setConfirmSell(null)}>취소</button>,
            <button key="d" className="btn btn-danger" onClick={() => forceSell(confirmSell.stock_code)} disabled={acting}>
              {acting ? '매도 중...' : '매도 실행'}
            </button>,
          ]}>
          <p><strong>{confirmSell.stock_name} ({confirmSell.stock_code})</strong>를 시장가로 강제 매도합니다.</p>
          <p style={{ marginTop: 8 }}>현재가 <span className="mono">{fmt(confirmSell.current_price)}</span> 기준 — 실제 체결가는 다를 수 있습니다.</p>
        </Modal>
      )}
    </div>
  )
}
