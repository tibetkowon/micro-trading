import { useState, useEffect } from 'react'
import { collection, onSnapshot } from 'firebase/firestore'
import { apiFetch } from '../utils/api'
import { fmt, Badge, Modal, EmptyState } from '../components/shared'
import { db, fmtTs } from '../lib/firebase'

export default function Positions() {
  const [positions, setPositions] = useState([])
  const [loading, setLoading] = useState(true)
  const [confirmAll, setConfirmAll] = useState(false)
  const [confirmSell, setConfirmSell] = useState(null)
  const [acting, setActing] = useState(false)

  useEffect(() => {
    const unsub = onSnapshot(collection(db, 'monitored_positions'), snap => {
      setPositions(snap.docs.map(d => ({ _docId: d.id, ...d.data() })))
      setLoading(false)
    })
    return unsub
  }, [])

  async function forceSell(code) {
    setActing(true)
    try {
      await apiFetch(`/api/monitor/positions/${code}/sell`, { method: 'POST' })
    } finally {
      setActing(false)
      setConfirmSell(null)
    }
  }

  async function liquidateAll() {
    setActing(true)
    try {
      await apiFetch('/api/monitor/liquidate-all', { method: 'POST' })
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
            return (
              <div key={p._docId} className="pos-card">
                <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 12 }}>
                  <div>
                    <div style={{ fontSize: 16, fontWeight: 700 }}>{p.stock_name}</div>
                    <div className="mono muted" style={{ fontSize: 12 }}>{p.stock_code}</div>
                  </div>
                  <Badge color="green">MONITORING</Badge>
                </div>

                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8, marginBottom: 12 }}>
                  {[
                    ['매수가', fmt(p.filled_price)],
                    ['목표가', fmt(p.target_price)],
                    ['손절가', fmt(p.stop_price)],
                    ['현재가', fmt(p.current_price)],
                  ].map(([label, value]) => (
                    <div key={label}>
                      <div style={{ fontSize: 10, color: 'var(--text-dim)', marginBottom: 2, textTransform: 'uppercase' }}>{label}</div>
                      <div className="mono" style={{ fontSize: 12, fontWeight: 600 }}>{value}</div>
                    </div>
                  ))}
                </div>

                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 12 }}>
                  진입시각: <span className="mono">{fmtTs(p.created_at)}</span>
                </div>

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
          <p style={{ marginTop: 8 }}>매수가 <span className="mono">{fmt(confirmSell.filled_price)}</span> 기준으로 주문합니다.</p>
        </Modal>
      )}
    </div>
  )
}
