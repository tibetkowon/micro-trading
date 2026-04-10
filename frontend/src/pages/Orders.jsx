import { useState, useEffect, useRef, useCallback } from 'react'
import { useApi } from '../hooks/useApi'

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
  FILLED: 'bg-emerald-500/10 text-emerald-400',
  PARTIALLY_FILLED: 'bg-amber-500/10 text-amber-400',
  PENDING: 'bg-orange-500/10 text-orange-400',
  CANCELLED: 'bg-white/5 text-gray-500',
  FAILED: 'bg-red-500/10 text-red-400',
}

const TYPE_LABELS = { ALL: '전체', BUY: '매수', SELL: '매도' }
const STATUS_FILTER_LABELS = { ALL: '전체', PENDING: '대기', FILLED: '체결', CANCELLED: '취소', FAILED: '실패' }

const PAGE_SIZE = 50

export default function Orders() {
  const [offset, setOffset] = useState(0)
  const [allOrders, setAllOrders] = useState([])
  const [hasMore, setHasMore] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  const [filterType, setFilterType] = useState('ALL')
  const [filterStatus, setFilterStatus] = useState('ALL')

  const [syncing, setSyncing] = useState(false)
  const [syncMsg, setSyncMsg] = useState(null)
  const [syncDays, setSyncDays] = useState(1)
  const [cancellingIds, setCancellingIds] = useState(new Set())
  const [deletingIds, setDeletingIds] = useState(new Set())

  const observerRef = useRef(null)
  const sentinelRef = useRef(null)

  const { data, loading, error, refetch } = useApi(`/api/orders?limit=${PAGE_SIZE}&offset=0`)

  useEffect(() => {
    if (data?.orders) {
      setAllOrders(data.orders)
      setOffset(PAGE_SIZE)
      setHasMore(data.orders.length === PAGE_SIZE)
    }
  }, [data])

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

  const filtered = allOrders.filter((o) => {
    if (filterType !== 'ALL' && o.order_type !== filterType) return false
    if (filterStatus !== 'ALL') {
      if (filterStatus === 'FILLED' && o.status !== 'FILLED' && o.status !== 'PARTIALLY_FILLED') return false
      if (filterStatus !== 'FILLED' && o.status !== filterStatus) return false
    }
    return true
  })

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
    <div className="space-y-6">
      {/* 헤더 */}
      <div className="flex items-center justify-between flex-wrap gap-3 pt-2">
        <div>
          <h1 className="text-2xl font-bold text-th-on-surface tracking-tight">주문 내역</h1>
          <p className="text-xs text-th-on-muted mt-0.5 uppercase tracking-widest">전체 매수·매도 이력</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <select
            value={syncDays}
            onChange={(e) => setSyncDays(Number(e.target.value))}
            className="text-xs px-2 py-2 bg-th-surface rounded-lg text-th-on-muted focus:outline-none focus:ring-1 focus:ring-orange-500/50"
          >
            {[1, 3, 7, 14, 30, 90].map((d) => (
              <option key={d} value={d}>{d}일</option>
            ))}
          </select>
          <button
            onClick={handleSync}
            disabled={syncing}
            className="text-xs px-3 py-2 bg-th-surface hover:bg-th-surface-high rounded-lg disabled:opacity-50 transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            {syncing ? '동기화 중...' : 'KIS 동기화'}
          </button>
          <button
            onClick={refetch}
            className="flex items-center gap-1.5 text-xs px-3 py-2 bg-th-surface hover:bg-th-surface-high rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            <span className="material-symbols-outlined text-[16px]">refresh</span>
            새로고침
          </button>
        </div>
      </div>

      {syncMsg && (
        <div className={`rounded-xl p-3 text-sm ${syncMsg.ok ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
          {syncMsg.text}
        </div>
      )}
      {error && (
        <div className="bg-red-500/10 rounded-xl p-4 text-sm text-red-400">{error}</div>
      )}

      {/* 필터 */}
      <div className="flex flex-wrap gap-2">
        <div className="flex items-center gap-0.5 bg-th-surface rounded-lg p-1">
          {Object.entries(TYPE_LABELS).map(([k, v]) => (
            <button
              key={k}
              onClick={() => setFilterType(k)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterType === k
                  ? k === 'BUY'
                    ? 'bg-red-500/15 text-red-400 font-medium'
                    : k === 'SELL'
                    ? 'bg-blue-500/15 text-blue-400 font-medium'
                    : 'bg-th-surface-high text-th-on-surface font-medium'
                  : 'text-th-on-muted hover:text-th-on-surface'
              }`}
            >
              {v}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-0.5 bg-th-surface rounded-lg p-1">
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

      {/* 목록 */}
      {loading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : filtered.length === 0 ? (
        <div className="bg-th-surface rounded-xl p-8 text-center">
          <span className="material-symbols-outlined text-[36px] text-th-on-subtle block mb-2">receipt_long</span>
          <p className="text-th-on-muted text-sm">주문 내역이 없습니다.</p>
        </div>
      ) : (
        <>
          {/* 모바일 카드 */}
          <div className="sm:hidden space-y-2">
            {filtered.map((o) => {
              const isFilled = o.status === 'FILLED' || o.status === 'PARTIALLY_FILLED'
              const isPending = o.status === 'PENDING'
              const isCancel = cancellingIds.has(o.id)
              const isDelete = deletingIds.has(o.id)
              return (
                <div key={o.id} className="bg-th-surface rounded-xl px-4 py-3 space-y-2">
                  {/* 상단: 종목 + 유형/상태 뱃지 */}
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <span className="font-medium text-th-on-surface text-sm">{o.stock_name || o.stock_code}</span>
                      {o.stock_name && (
                        <span className="ml-1.5 text-xs text-th-on-subtle font-data">{o.stock_code}</span>
                      )}
                      {o.source === 'MANUAL' && (
                        <span className="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] bg-white/5 text-gray-500">수동</span>
                      )}
                    </div>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${
                        o.order_type === 'BUY' ? 'bg-red-500/10 text-red-400' : 'bg-blue-500/10 text-blue-400'
                      }`}>
                        {o.order_type === 'BUY' ? '매수' : '매도'}
                      </span>
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${STATUS_STYLE[o.status] || 'bg-white/5 text-gray-500'}`}>
                        {STATUS_LABEL[o.status] || o.status}
                      </span>
                    </div>
                  </div>

                  {/* 중단: 가격 + 수량 + 시각 */}
                  <div className="flex items-center justify-between text-xs">
                    <div className="font-data">
                      {isFilled && o.filled_price > 0 ? (
                        <span className="text-amber-400 font-medium">{fmtPrice(o.filled_price, o.market)}</span>
                      ) : o.price > 0 ? (
                        <span className="text-th-on-muted">{fmtPrice(o.price, o.market)}</span>
                      ) : (
                        <span className="text-th-on-subtle">시장가</span>
                      )}
                      <span className="text-th-on-subtle ml-1.5">{(o.qty || 0).toLocaleString()}주</span>
                    </div>
                    <span className="text-th-on-subtle">{fmtDate(o.created_at)}</span>
                  </div>

                  {/* 매도사유 */}
                  {o.order_type === 'SELL' && o.sell_reason && (
                    <p className="text-xs text-th-on-muted">{o.sell_reason}</p>
                  )}

                  {/* 액션 */}
                  <div className="flex items-center gap-1 pt-0.5">
                    {isPending && (
                      <button
                        onClick={() => handleCancel(o.id)}
                        disabled={isCancel}
                        className="text-xs px-2.5 py-1 text-orange-400 hover:bg-orange-500/10 rounded-lg disabled:opacity-40 transition-colors"
                      >
                        {isCancel ? '...' : '주문 취소'}
                      </button>
                    )}
                    <button
                      onClick={() => handleDelete(o.id)}
                      disabled={isDelete}
                      className="text-xs px-2.5 py-1 text-th-on-muted hover:text-red-400 hover:bg-red-500/10 rounded-lg disabled:opacity-40 transition-colors"
                    >
                      {isDelete ? '...' : '삭제'}
                    </button>
                  </div>
                </div>
              )
            })}
          </div>

          {/* 데스크탑 테이블 */}
          <div className="hidden sm:block bg-th-surface-low rounded-xl overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-[10px] text-th-on-subtle uppercase tracking-widest">
                    <th className="text-left px-4 py-3.5 font-medium hidden sm:table-cell">ID</th>
                    <th className="text-left px-4 py-3.5 font-medium">종목</th>
                    <th className="text-left px-4 py-3.5 font-medium hidden sm:table-cell">시장</th>
                    <th className="text-left px-4 py-3.5 font-medium">유형</th>
                    <th className="text-right px-4 py-3.5 font-medium hidden sm:table-cell">수량</th>
                    <th className="text-right px-4 py-3.5 font-medium">주문가 / 체결가</th>
                    <th className="text-left px-4 py-3.5 font-medium hidden md:table-cell">매도사유</th>
                    <th className="text-left px-4 py-3.5 font-medium">상태</th>
                    <th className="text-left px-4 py-3.5 font-medium hidden sm:table-cell">주문시각</th>
                    <th className="px-4 py-3.5"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/[0.04]">
                  {filtered.map((o) => {
                    const isFilled = o.status === 'FILLED' || o.status === 'PARTIALLY_FILLED'
                    const isPending = o.status === 'PENDING'
                    const isCancel = cancellingIds.has(o.id)
                    const isDelete = deletingIds.has(o.id)
                    return (
                      <tr key={o.id} className="hover:bg-white/[0.02] transition-colors">
                        <td className="px-4 py-3.5 text-th-on-subtle text-xs font-data hidden sm:table-cell">{o.id}</td>
                        <td className="px-4 py-3.5">
                          <span className="font-medium text-th-on-surface">{o.stock_name || o.stock_code}</span>
                          {o.stock_name && (
                            <span className="ml-2 text-xs text-th-on-subtle font-data">{o.stock_code}</span>
                          )}
                          {o.source === 'MANUAL' && (
                            <span className="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] bg-white/5 text-gray-500">수동</span>
                          )}
                        </td>
                        <td className="px-4 py-3.5 hidden sm:table-cell">
                          {o.market === 'US' ? (
                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-orange-500/10 text-orange-400">해외</span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] bg-blue-500/10 text-blue-400">국내</span>
                          )}
                        </td>
                        <td className="px-4 py-3.5">
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${
                            o.order_type === 'BUY'
                              ? 'bg-red-500/10 text-red-400'
                              : 'bg-blue-500/10 text-blue-400'
                          }`}>
                            {o.order_type === 'BUY' ? '매수' : '매도'}
                          </span>
                        </td>
                        <td className="px-4 py-3.5 text-right text-th-on-muted font-data hidden sm:table-cell">
                          {(o.qty || 0).toLocaleString()}
                        </td>
                        <td className="px-4 py-3.5 text-right font-data">
                          {isFilled && o.filled_price > 0 ? (
                            <span className="text-amber-400 font-medium">{fmtPrice(o.filled_price, o.market)}</span>
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
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${STATUS_STYLE[o.status] || 'bg-white/5 text-gray-500'}`}>
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
                                className="text-xs px-2 py-0.5 text-orange-400 hover:bg-orange-500/10 rounded disabled:opacity-40 transition-colors"
                              >
                                {isCancel ? '...' : '취소'}
                              </button>
                            )}
                            <button
                              onClick={() => handleDelete(o.id)}
                              disabled={isDelete}
                              className="text-xs px-2 py-0.5 text-th-on-muted hover:text-red-400 hover:bg-red-500/10 rounded disabled:opacity-40 transition-colors"
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
        </>
      )}

      <div ref={sentinelRef} className="h-4" />
      {loadingMore && <p className="text-center text-th-on-subtle text-xs py-2">불러오는 중...</p>}
      {!hasMore && allOrders.length > 0 && (
        <p className="text-center text-th-on-subtle text-xs py-2">모든 내역을 불러왔습니다.</p>
      )}
    </div>
  )
}
