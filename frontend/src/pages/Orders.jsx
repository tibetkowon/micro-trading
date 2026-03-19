import { useState, useEffect, useRef, useCallback } from 'react'
import { useApi } from '../hooks/useApi'

/* ── 유틸 ── */
function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}
function fmtPrice(price, market) {
  if (price == null || price === '') return '-'
  if (market === 'US') return '$' + Number(price).toFixed(2)
  return Number(price).toLocaleString('ko-KR') + '원'
}

const STATUS_LABEL = {
  FILLED: '체결',
  PARTIALLY_FILLED: '부분체결',
  PENDING: '대기',
  CANCELLED: '취소',
  FAILED: '실패',
}
const STATUS_STYLE = {
  FILLED: 'bg-th-growth/10 text-th-growth border-th-growth/20',
  PARTIALLY_FILLED: 'bg-th-warn/10 text-th-warn border-th-warn/20',
  PENDING: 'bg-th-primary/10 text-th-primary border-th-primary/20',
  CANCELLED: 'bg-th-surface-high text-th-on-muted border-th-outline',
  FAILED: 'bg-th-loss/10 text-th-loss border-th-loss/20',
}

const MARKET_LABELS = { ALL: '전체', KR: '국장', US: '미장' }
const TYPE_LABELS = { ALL: '전체', BUY: '매수', SELL: '매도' }
const STATUS_FILTER_LABELS = { ALL: '전체', PENDING: '대기', FILLED: '체결', CANCELLED: '취소', FAILED: '실패' }

const PAGE_SIZE = 50

export default function Orders() {
  const [offset, setOffset] = useState(0)
  const [allOrders, setAllOrders] = useState([])
  const [hasMore, setHasMore] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  const [filterMarket, setFilterMarket] = useState('ALL')
  const [filterType, setFilterType] = useState('ALL')
  const [filterStatus, setFilterStatus] = useState('ALL')

  const [syncing, setSyncing] = useState(false)
  const [syncMsg, setSyncMsg] = useState(null)
  const [syncDays, setSyncDays] = useState(1)
  const [cancellingIds, setCancellingIds] = useState(new Set())
  const [deletingIds, setDeletingIds] = useState(new Set())

  const observerRef = useRef(null)
  const sentinelRef = useRef(null)

  /* 초기 / 필터 변경 시 리셋 */
  const { data, loading, error, refetch } = useApi(`/api/orders?limit=${PAGE_SIZE}&offset=0`)

  useEffect(() => {
    if (data?.orders) {
      setAllOrders(data.orders)
      setOffset(PAGE_SIZE)
      setHasMore(data.orders.length === PAGE_SIZE)
    }
  }, [data])

  /* 무한스크롤 추가 로드 */
  const loadMore = useCallback(async () => {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    try {
      const res = await fetch(`/api/orders?limit=${PAGE_SIZE}&offset=${offset}`)
      const body = await res.json()
      const next = body.orders || []
      setAllOrders((prev) => [...prev, ...next])
      setOffset((prev) => prev + PAGE_SIZE)
      setHasMore(next.length === PAGE_SIZE)
    } finally {
      setLoadingMore(false)
    }
  }, [loadingMore, hasMore, offset])

  useEffect(() => {
    if (!sentinelRef.current) return
    const obs = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting) loadMore() },
      { threshold: 0.1 }
    )
    obs.observe(sentinelRef.current)
    observerRef.current = obs
    return () => obs.disconnect()
  }, [loadMore])

  /* 필터 적용 */
  const filtered = allOrders.filter((o) => {
    if (filterMarket !== 'ALL' && (o.market || 'KR') !== filterMarket) return false
    if (filterType !== 'ALL' && o.order_type !== filterType) return false
    if (filterStatus !== 'ALL') {
      if (filterStatus === 'FILLED' && o.status !== 'FILLED' && o.status !== 'PARTIALLY_FILLED') return false
      if (filterStatus !== 'FILLED' && o.status !== filterStatus) return false
    }
    return true
  })

  /* KIS 동기화 */
  async function handleSync() {
    setSyncing(true)
    setSyncMsg(null)
    try {
      const res = await fetch(`/api/orders?sync=true&days=${syncDays}&limit=1`)
      const body = await res.json()
      if (body.sync_error) {
        setSyncMsg({ ok: false, text: body.sync_error })
      } else {
        setSyncMsg({ ok: true, text: `${syncDays}일 동기화 완료` })
      }
    } catch (e) {
      setSyncMsg({ ok: false, text: e.message })
    } finally {
      setSyncing(false)
      refetch()
    }
  }

  /* KIS 취소 */
  async function handleCancel(id) {
    if (!confirm('미체결 주문을 취소하시겠습니까?')) return
    setCancellingIds((prev) => new Set(prev).add(id))
    try {
      const res = await fetch(`/api/orders/${id}/cancel`, { method: 'POST' })
      if (!res.ok) {
        const body = await res.json()
        alert(body.error || '취소 실패')
      }
      refetch()
    } finally {
      setCancellingIds((prev) => { const n = new Set(prev); n.delete(id); return n })
    }
  }

  /* DB 삭제 */
  async function handleDelete(id) {
    if (!confirm('주문 내역을 삭제하시겠습니까?')) return
    setDeletingIds((prev) => new Set(prev).add(id))
    try {
      await fetch(`/api/orders/${id}`, { method: 'DELETE' })
      refetch()
    } finally {
      setDeletingIds((prev) => { const n = new Set(prev); n.delete(id); return n })
    }
  }

  return (
    <div className="space-y-5">
      {/* 헤더 */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold text-th-on-surface">주문 내역</h1>
          <p className="text-xs text-th-on-muted mt-0.5">전체 매수·매도 이력</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          {/* 동기화 날짜 선택 */}
          <select
            value={syncDays}
            onChange={(e) => setSyncDays(Number(e.target.value))}
            className="text-sm px-2 py-2 bg-th-surface border border-th-outline rounded-lg text-th-on-muted focus:outline-none focus:border-th-primary"
          >
            {[1, 3, 7, 14, 30, 90].map((d) => (
              <option key={d} value={d}>{d}일</option>
            ))}
          </select>
          <button
            onClick={handleSync}
            disabled={syncing}
            className="text-sm px-4 py-2 bg-th-surface hover:bg-th-surface-high border border-th-outline rounded-lg disabled:opacity-50 transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            {syncing ? '동기화 중...' : 'KIS 동기화'}
          </button>
          <button
            onClick={refetch}
            className="text-sm px-3 py-2 bg-th-surface hover:bg-th-surface-high border border-th-outline rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            새로고침
          </button>
        </div>
      </div>

      {syncMsg && (
        <div className={`rounded-xl p-3 text-sm border ${syncMsg.ok ? 'bg-th-growth/10 border-th-growth/20 text-th-growth' : 'bg-th-loss/10 border-th-loss/20 text-th-loss'}`}>
          {syncMsg.text}
        </div>
      )}
      {error && (
        <div className="bg-th-loss/10 border border-th-loss/20 text-th-loss rounded-xl p-4 text-sm">{error}</div>
      )}

      {/* 필터 */}
      <div className="flex flex-wrap gap-3">
        {/* 시장 */}
        <div className="flex items-center gap-1 bg-th-surface border border-th-outline rounded-lg p-1">
          {Object.entries(MARKET_LABELS).map(([k, v]) => (
            <button
              key={k}
              onClick={() => setFilterMarket(k)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterMarket === k ? 'bg-th-surface-high text-th-on-surface font-medium' : 'text-th-on-muted hover:text-th-on-surface'
              }`}
            >
              {v}
            </button>
          ))}
        </div>
        {/* 유형 */}
        <div className="flex items-center gap-1 bg-th-surface border border-th-outline rounded-lg p-1">
          {Object.entries(TYPE_LABELS).map(([k, v]) => (
            <button
              key={k}
              onClick={() => setFilterType(k)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterType === k
                  ? k === 'BUY'
                    ? 'bg-th-loss/15 text-th-loss font-medium'
                    : k === 'SELL'
                    ? 'bg-[#3B82F6]/15 text-[#3B82F6] font-medium'
                    : 'bg-th-surface-high text-th-on-surface font-medium'
                  : 'text-th-on-muted hover:text-th-on-surface'
              }`}
            >
              {v}
            </button>
          ))}
        </div>
        {/* 상태 */}
        <div className="flex items-center gap-1 bg-th-surface border border-th-outline rounded-lg p-1">
          {Object.entries(STATUS_FILTER_LABELS).map(([k, v]) => (
            <button
              key={k}
              onClick={() => setFilterStatus(k)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterStatus === k ? 'bg-th-surface-high text-th-on-surface font-medium' : 'text-th-on-muted hover:text-th-on-surface'
              }`}
            >
              {v}
            </button>
          ))}
        </div>
        <span className="text-xs text-th-on-subtle self-center">{filtered.length}건</span>
      </div>

      {/* 테이블 */}
      {loading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : filtered.length === 0 ? (
        <div className="bg-th-surface border border-th-outline rounded-xl p-8 text-center text-th-on-subtle text-sm">
          주문 내역이 없습니다.
        </div>
      ) : (
        <div className="bg-th-surface border border-th-outline rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-th-outline text-xs text-th-on-subtle">
                  <th className="text-left px-4 py-3 font-medium hidden sm:table-cell">ID</th>
                  <th className="text-left px-4 py-3 font-medium">종목</th>
                  <th className="text-left px-4 py-3 font-medium hidden sm:table-cell">시장</th>
                  <th className="text-left px-4 py-3 font-medium">유형</th>
                  <th className="text-right px-4 py-3 font-medium hidden sm:table-cell">수량</th>
                  <th className="text-right px-4 py-3 font-medium">주문가 / 체결가</th>
                  <th className="text-left px-4 py-3 font-medium hidden md:table-cell">매도사유</th>
                  <th className="text-left px-4 py-3 font-medium">상태</th>
                  <th className="text-left px-4 py-3 font-medium hidden sm:table-cell">주문시각</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-th-outline">
                {filtered.map((o) => {
                  const isFilled = o.status === 'FILLED' || o.status === 'PARTIALLY_FILLED'
                  const isPending = o.status === 'PENDING'
                  const isCancel = cancellingIds.has(o.id)
                  const isDelete = deletingIds.has(o.id)
                  return (
                    <tr key={o.id} className="hover:bg-th-surface-high transition-colors">
                      <td className="px-4 py-3.5 text-th-on-subtle text-xs font-data hidden sm:table-cell">{o.id}</td>
                      <td className="px-4 py-3.5">
                        <span className="font-medium text-th-on-surface">{o.stock_name || o.stock_code}</span>
                        {o.stock_name && (
                          <span className="ml-1.5 text-xs text-th-on-subtle font-data">{o.stock_code}</span>
                        )}
                        {o.source === 'MANUAL' && (
                          <span className="ml-1.5 badge bg-th-surface-high text-th-on-subtle border-th-outline text-[10px]">수동</span>
                        )}
                      </td>
                      <td className="px-4 py-3.5 hidden sm:table-cell">
                        {o.market === 'US' ? (
                          <span className="badge bg-[#7C3AED]/10 text-[#7C3AED] border-[#7C3AED]/20 dark:bg-[#7C3AED]/15 dark:text-[#A78BFA] dark:border-[#7C3AED]/30">미장</span>
                        ) : (
                          <span className="badge bg-th-surface-high text-th-on-muted border-th-outline">국장</span>
                        )}
                      </td>
                      <td className="px-4 py-3.5">
                        <span className={`badge font-medium ${
                          o.order_type === 'BUY'
                            ? 'bg-th-loss/10 text-th-loss border-th-loss/20'
                            : 'bg-[#3B82F6]/10 text-[#3B82F6] border-[#3B82F6]/20'
                        }`}>
                          {o.order_type === 'BUY' ? '매수' : '매도'}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 text-right text-th-on-muted font-data hidden sm:table-cell">
                        {(o.qty || 0).toLocaleString()}
                      </td>
                      <td className="px-4 py-3.5 text-right font-data">
                        {isFilled && o.filled_price > 0 ? (
                          <span className="text-th-warn font-medium">{fmtPrice(o.filled_price, o.market)}</span>
                        ) : o.price > 0 ? (
                          <span className="text-th-on-muted">{fmtPrice(o.price, o.market)}</span>
                        ) : (
                          <span className="text-th-on-subtle text-xs">시장가</span>
                        )}
                      </td>
                      <td className="px-4 py-3.5 hidden md:table-cell">
                        {o.order_type === 'SELL' && o.sell_reason ? (
                          <span className="text-xs text-th-on-muted">{o.sell_reason}</span>
                        ) : (
                          <span className="text-th-on-subtle text-xs">-</span>
                        )}
                      </td>
                      <td className="px-4 py-3.5">
                        <span className={`badge ${STATUS_STYLE[o.status] || 'bg-th-surface-high text-th-on-muted border-th-outline'}`}>
                          {STATUS_LABEL[o.status] || o.status}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 text-th-on-subtle text-xs hidden sm:table-cell">{fmtDate(o.created_at)}</td>
                      <td className="px-4 py-3.5">
                        <div className="flex items-center gap-1">
                          {isPending && (
                            <button
                              onClick={() => handleCancel(o.id)}
                              disabled={isCancel}
                              className="text-xs px-2 py-0.5 text-th-primary hover:bg-th-primary/10 rounded border border-transparent hover:border-th-primary/20 disabled:opacity-40 transition-colors"
                            >
                              {isCancel ? '...' : '취소'}
                            </button>
                          )}
                          <button
                            onClick={() => handleDelete(o.id)}
                            disabled={isDelete}
                            className="text-xs px-2 py-0.5 text-th-on-subtle hover:text-th-loss hover:bg-th-loss/10 rounded border border-transparent hover:border-th-loss/20 disabled:opacity-40 transition-colors"
                          >
                            {isDelete ? '...' : '삭제'}
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* 무한스크롤 센티넬 */}
      <div ref={sentinelRef} className="h-4" />
      {loadingMore && <p className="text-center text-th-on-subtle text-xs py-2">불러오는 중...</p>}
      {!hasMore && allOrders.length > 0 && (
        <p className="text-center text-th-on-subtle text-xs py-2">모든 내역을 불러왔습니다.</p>
      )}
    </div>
  )
}
