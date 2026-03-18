import { useState } from 'react'
import { useApi } from '../hooks/useApi'
import StatusBadge from '../components/StatusBadge'

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

function fmtPrice(price, market) {
  if (!price && price !== 0) return '-'
  if (market === 'US') {
    return '$' + Number(price).toFixed(2)
  }
  return Number(price).toLocaleString('ko-KR') + '원'
}

const FILLED_STATUSES = new Set(['FILLED', 'PARTIALLY_FILLED'])

export default function Orders() {
  const { data, loading, error, refetch } = useApi('/api/orders?limit=100')
  const [deletingIds, setDeletingIds] = useState(new Set())
  const [syncing, setSyncing] = useState(false)
  const [syncMsg, setSyncMsg] = useState(null)

  const orders = data?.orders || []

  async function handleSync() {
    setSyncing(true)
    setSyncMsg(null)
    try {
      const res = await fetch('/api/orders?sync=true&days=1&limit=1')
      const body = await res.json()
      if (body.sync_error) {
        setSyncMsg({ ok: false, text: body.sync_error })
      } else {
        setSyncMsg({ ok: true, text: '동기화 완료' })
      }
    } catch (e) {
      setSyncMsg({ ok: false, text: e.message })
    } finally {
      setSyncing(false)
      refetch()
    }
  }

  async function handleDelete(id) {
    setDeletingIds((prev) => new Set(prev).add(id))
    try {
      await fetch(`/api/orders/${id}`, { method: 'DELETE' })
      refetch()
    } finally {
      setDeletingIds((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-white">주문 내역</h1>
          <p className="text-sm text-zinc-500 mt-0.5">전체 매수·매도 이력</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleSync}
            disabled={syncing}
            className="text-sm px-4 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-lg border border-zinc-700 disabled:opacity-50 transition-colors"
          >
            {syncing ? '동기화 중...' : 'KIS 동기화'}
          </button>
          <button
            onClick={refetch}
            className="text-sm px-4 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
          >
            새로고침
          </button>
        </div>
      </div>

      {syncMsg && (
        <div className={`rounded-xl p-3 mb-4 text-sm border ${syncMsg.ok ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' : 'bg-red-500/10 border-red-500/20 text-red-400'}`}>
          {syncMsg.text}
        </div>
      )}

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl p-4 mb-4 text-sm">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-zinc-500">로딩 중...</p>
      ) : orders.length === 0 ? (
        <p className="text-zinc-500">주문 내역이 없습니다.</p>
      ) : (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-zinc-800 text-xs text-zinc-500">
                  <th className="text-left px-5 py-3 font-medium hidden sm:table-cell">ID</th>
                  <th className="text-left px-5 py-3 font-medium">종목</th>
                  <th className="text-left px-5 py-3 font-medium">시장</th>
                  <th className="text-left px-5 py-3 font-medium">유형</th>
                  <th className="text-right px-5 py-3 font-medium hidden sm:table-cell">수량</th>
                  <th className="text-right px-5 py-3 font-medium">주문가 / 체결가</th>
                  <th className="text-left px-5 py-3 font-medium hidden sm:table-cell">매도사유</th>
                  <th className="text-left px-5 py-3 font-medium">상태</th>
                  <th className="text-left px-5 py-3 font-medium hidden sm:table-cell">주문시각</th>
                  <th className="px-5 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/60">
                {orders.map((o) => {
                  const isFilled = FILLED_STATUSES.has(o.status)
                  const isDeleting = deletingIds.has(o.id)
                  return (
                    <tr key={o.id} className="hover:bg-zinc-800/40 transition-colors">
                      <td className="px-5 py-3.5 text-zinc-600 hidden sm:table-cell">{o.id}</td>
                      <td className="px-5 py-3.5">
                        <span className="font-medium text-white">{o.stock_name || o.stock_code}</span>
                        {o.stock_name && (
                          <span className="ml-1.5 text-xs text-zinc-500 font-mono">{o.stock_code}</span>
                        )}
                      </td>
                      <td className="px-5 py-3.5">
                        {o.market === 'US' ? (
                          <span className="inline-block px-2.5 py-0.5 rounded-full text-xs border bg-violet-500/15 text-violet-400 border-violet-500/20">미장</span>
                        ) : (
                          <span className="inline-block px-2.5 py-0.5 rounded-full text-xs border bg-zinc-700/50 text-zinc-400 border-zinc-700">국장</span>
                        )}
                      </td>
                      <td className="px-5 py-3.5">
                        {/* 한국식: 매수=빨강, 매도=파랑 */}
                        <span className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium border ${
                          o.order_type === 'BUY'
                            ? 'bg-red-500/15 text-red-400 border-red-500/20'
                            : 'bg-blue-500/15 text-blue-400 border-blue-500/20'
                        }`}>
                          {o.order_type === 'BUY' ? '매수' : '매도'}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-right text-zinc-300 hidden sm:table-cell">{o.qty.toLocaleString()}</td>
                      <td className="px-5 py-3.5 text-right">
                        {isFilled && o.filled_price > 0 ? (
                          <span className="text-yellow-400 font-medium">{fmtPrice(o.filled_price, o.market)}</span>
                        ) : o.price > 0 ? (
                          <span className="text-zinc-300">{fmtPrice(o.price, o.market)}</span>
                        ) : (
                          <span className="text-zinc-500 text-xs">시장가</span>
                        )}
                      </td>
                      <td className="px-5 py-3.5 text-xs text-zinc-500 hidden sm:table-cell">
                        {o.order_type === 'SELL' && o.sell_reason ? o.sell_reason : '-'}
                      </td>
                      <td className="px-5 py-3.5"><StatusBadge status={o.status} /></td>
                      <td className="px-5 py-3.5 text-zinc-500 text-xs hidden sm:table-cell">{fmtDate(o.created_at)}</td>
                      <td className="px-5 py-3.5">
                        <button
                          onClick={() => handleDelete(o.id)}
                          disabled={isDeleting}
                          className="text-xs px-2.5 py-1 text-zinc-500 hover:text-red-400 hover:bg-red-500/10 rounded-full disabled:opacity-40 transition-colors"
                        >
                          {isDeleting ? '...' : '삭제'}
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
