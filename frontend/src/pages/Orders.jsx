import { useState, useEffect, useCallback } from 'react'
import { apiFetch } from '../utils/api'
import { fmt, Badge, Spinner } from '../components/shared'
import { fmtTs } from '../lib/time'

const STATUS_COLOR = { FILLED: 'green', PENDING: 'yellow', CANCELLED: 'gray', FAILED: 'red', PARTIALLY_FILLED: 'yellow' }
const STATUS_LABEL = { FILLED: '체결', PENDING: '대기', CANCELLED: '취소', FAILED: '실패', PARTIALLY_FILLED: '부분체결' }
const RANGE_DAYS = { '오늘': 1, '7일': 7, '30일': 30 }

export default function Orders() {
  const [range, setRange] = useState('오늘')
  const [typeFilter, setTypeFilter] = useState('전체')
  const [statusFilter, setStatusFilter] = useState('전체')
  const [page, setPage] = useState(1)
  const [syncing, setSyncing] = useState(false)
  const [orders, setOrders] = useState([])
  const [loading, setLoading] = useState(true)

  const PAGE_SIZE = 50
  const days = RANGE_DAYS[range] || 1

  const fetchOrders = useCallback(async () => {
    setLoading(true)
    try {
      const res = await apiFetch(`/api/orders?days=${days}&limit=500`)
      const data = await res.json()
      setOrders(data.orders || data.data || [])
    } finally {
      setLoading(false)
    }
  }, [days])

  useEffect(() => {
    fetchOrders()
    setPage(1)
  }, [fetchOrders])

  const filtered = orders.filter(o => {
    const type = o.order_type === 'BUY' ? '매수' : '매도'
    if (typeFilter !== '전체' && type !== typeFilter) return false
    if (statusFilter !== '전체' && STATUS_LABEL[o.status] !== statusFilter) return false
    return true
  })

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function syncKIS() {
    setSyncing(true)
    try {
      await apiFetch(`/api/orders?sync=true&days=${days}&limit=1`)
      await fetchOrders()
    } finally {
      setSyncing(false)
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', gap: 12, alignItems: 'center', flexWrap: 'wrap', marginBottom: 16 }}>
        <div className="chip-group">
          {['오늘', '7일', '30일'].map(r => (
            <span key={r} className={`chip ${range === r ? 'active' : ''}`}
              onClick={() => { setRange(r); setPage(1) }}>{r}</span>
          ))}
        </div>
        <div style={{ width: 1, height: 20, background: 'var(--border)' }}></div>
        <div className="chip-group">
          {['전체', '매수', '매도'].map(t => (
            <span key={t} className={`chip ${typeFilter === t ? 'active' : ''}`}
              onClick={() => setTypeFilter(t)}>{t}</span>
          ))}
        </div>
        <div className="chip-group">
          {['전체', '체결', '대기', '취소'].map(s => (
            <span key={s} className={`chip ${statusFilter === s ? 'active' : ''}`}
              onClick={() => setStatusFilter(s)}>{s}</span>
          ))}
        </div>
        <button className="btn btn-outline btn-sm" onClick={syncKIS} disabled={syncing} style={{ marginLeft: 'auto' }}>
          {syncing ? <><Spinner /> 동기화 중...</> : '↻ KIS 동기화'}
        </button>
        <span className="muted" style={{ fontSize: 12 }}>{filtered.length}건</span>
      </div>

      <div className="card">
        {loading ? (
          <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)' }}>로딩 중...</div>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>주문시각</th>
                  <th>종목명(코드)</th>
                  <th>구분</th>
                  <th>수량</th>
                  <th>주문가</th>
                  <th>체결가</th>
                  <th>상태</th>
                  <th>출처</th>
                </tr>
              </thead>
              <tbody>
                {paged.length === 0 ? (
                  <tr>
                    <td colSpan={8} style={{ textAlign: 'center', padding: 32, color: 'var(--text-muted)' }}>
                      주문 내역이 없습니다
                    </td>
                  </tr>
                ) : paged.map(o => {
                  const isBuy = o.order_type === 'BUY'
                  return (
                    <tr key={o.id || o.kis_order_id}>
                      <td className="mono" style={{ fontSize: 12, whiteSpace: 'nowrap' }}>{fmtTs(o.created_at)}</td>
                      <td>
                        <div style={{ fontWeight: 600 }}>{o.stock_name}</div>
                        <div className="mono muted" style={{ fontSize: 11 }}>{o.stock_code}</div>
                      </td>
                      <td><Badge color={isBuy ? 'blue' : 'orange'}>{isBuy ? '매수' : '매도'}</Badge></td>
                      <td className="mono">{o.qty}주</td>
                      <td className="mono">{fmt(o.price)}</td>
                      <td className="mono">{o.filled_price ? fmt(o.filled_price) : <span className="muted">—</span>}</td>
                      <td><Badge color={STATUS_COLOR[o.status] || 'gray'}>{STATUS_LABEL[o.status] || o.status}</Badge></td>
                      <td><Badge color={o.source === 'AGENT' ? 'orange' : 'gray'}>{o.source}</Badge></td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
        <div style={{ padding: '12px 16px', display: 'flex', justifyContent: 'center', gap: 8, borderTop: '1px solid var(--border)' }}>
          <button className="btn btn-outline btn-xs" disabled={page === 1} onClick={() => setPage(p => p - 1)}>← 이전</button>
          <span className="muted" style={{ padding: '2px 8px', fontSize: 12 }}>{page} / {totalPages}</span>
          <button className="btn btn-outline btn-xs" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>다음 →</button>
        </div>
      </div>
    </div>
  )
}
