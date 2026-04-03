import { useState, useEffect, useCallback } from 'react'

const inputCls = 'px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50'

export default function StockList() {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [query, setQuery] = useState('')
  const [etfOnly, setEtfOnly] = useState(false)
  const [market, setMarket] = useState('')
  const [saving, setSaving] = useState(null) // code being toggled
  const [message, setMessage] = useState(null)

  const fetchStocks = useCallback(async (q, eo, mkt) => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (q) params.set('q', q)
      if (eo) params.set('etf_only', 'true')
      if (mkt) params.set('market', mkt)
      const res = await fetch(`/api/stocks?${params}`)
      const json = await res.json()
      if (res.ok) setItems(json.items || [])
    } catch (e) {
      setMessage({ ok: false, text: e.message })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStocks(query, etfOnly, market)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  function handleSearch(e) {
    e.preventDefault()
    fetchStocks(query, etfOnly, market)
  }

  async function toggleHardWatch(code, currently) {
    setSaving(code)
    setMessage(null)
    try {
      // Read current settings, patch hard_watch_symbols
      const settingsRes = await fetch('/api/settings')
      const settings = await settingsRes.json()
      const current = Array.isArray(settings.hard_watch_symbols) ? settings.hard_watch_symbols : []
      const next = currently
        ? current.filter(c => c !== code)
        : current.includes(code) ? current : [...current, code]

      const res = await fetch('/api/settings', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ hard_watch_symbols: next }),
      })
      if (!res.ok) {
        const json = await res.json()
        setMessage({ ok: false, text: json.error || '저장 실패' })
        return
      }
      // Update local state
      setItems(prev => prev.map(item =>
        item.stock_code === code ? { ...item, is_hard_watch: !currently } : item
      ))
      setMessage({ ok: true, text: currently ? `${code} 하드 감시 해제` : `${code} 하드 감시 등록` })
    } catch (e) {
      setMessage({ ok: false, text: e.message })
    } finally {
      setSaving(null)
    }
  }

  function assetLabel(item) {
    if (!item.is_etf) return '주식'
    if (item.is_domestic_equity_etf) return 'ETF (국내주식형)'
    return 'ETF'
  }

  function assetBadge(item) {
    if (!item.is_etf) return 'bg-blue-500/10 text-blue-400'
    if (item.is_domestic_equity_etf) return 'bg-emerald-500/10 text-emerald-400'
    return 'bg-orange-500/10 text-orange-400'
  }

  return (
    <div className="space-y-4 pb-20">
      {/* 헤더 */}
      <div className="sticky top-14 md:top-0 z-30 glass-panel -mx-4 md:-mx-8 px-4 md:px-8 py-3">
        <h1 className="text-2xl font-bold text-th-on-surface tracking-tight">종목 목록</h1>
        <p className="text-xs text-th-on-muted mt-0.5 uppercase tracking-widest">종목 마스터 검색 및 하드 감시 관리</p>
      </div>

      {/* 검색 폼 */}
      <form onSubmit={handleSearch} className="bg-th-surface rounded-xl p-4 space-y-3">
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="종목명 또는 코드 검색"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className={`flex-1 ${inputCls}`}
          />
          <button
            type="submit"
            className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-sm font-semibold rounded-lg transition-colors"
          >
            검색
          </button>
        </div>
        <div className="flex flex-wrap gap-3 items-center">
          <label className="flex items-center gap-1.5 cursor-pointer">
            <input
              type="checkbox"
              checked={etfOnly}
              onChange={(e) => setEtfOnly(e.target.checked)}
              className="w-4 h-4 rounded accent-orange-500"
            />
            <span className="text-xs text-th-on-surface">ETF만 보기</span>
          </label>
          <div className="flex gap-1.5">
            {[{ value: '', label: '전체' }, { value: 'KOSPI', label: 'KOSPI' }, { value: 'KOSDAQ', label: 'KOSDAQ' }].map(({ value, label }) => (
              <button
                key={value}
                type="button"
                onClick={() => setMarket(value)}
                className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors border ${
                  market === value
                    ? 'bg-th-surface-high text-th-on-surface border-black/10 dark:border-white/10 ring-1 ring-zinc-600'
                    : 'bg-transparent text-th-on-muted border-black/10 dark:border-white/10 hover:text-th-on-surface'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        {message && (
          <p className={`text-xs ${message.ok ? 'text-emerald-400' : 'text-red-400'}`}>{message.text}</p>
        )}
      </form>

      {/* 결과 테이블 */}
      <div className="bg-th-surface rounded-xl overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-th-on-muted text-sm">로딩 중...</div>
        ) : items.length === 0 ? (
          <div className="p-8 text-center text-th-on-muted text-sm">검색 결과가 없습니다.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-black/5 dark:border-white/5">
                  <th className="text-left px-4 py-3 text-xs text-th-on-muted font-medium">종목코드</th>
                  <th className="text-left px-4 py-3 text-xs text-th-on-muted font-medium">종목명</th>
                  <th className="text-left px-4 py-3 text-xs text-th-on-muted font-medium">거래소</th>
                  <th className="text-left px-4 py-3 text-xs text-th-on-muted font-medium">유형</th>
                  <th className="text-center px-4 py-3 text-xs text-th-on-muted font-medium">하드 감시</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.stock_code} className="border-b border-black/5 dark:border-white/5 hover:bg-th-surface-high/30 transition-colors">
                    <td className="px-4 py-2.5 font-data text-th-on-surface">{item.stock_code}</td>
                    <td className="px-4 py-2.5 text-th-on-surface">{item.stock_name}</td>
                    <td className="px-4 py-2.5 text-th-on-muted text-xs">{item.market_type || '-'}</td>
                    <td className="px-4 py-2.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${assetBadge(item)}`}>
                        {assetLabel(item)}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <button
                        type="button"
                        disabled={saving === item.stock_code}
                        onClick={() => toggleHardWatch(item.stock_code, item.is_hard_watch)}
                        className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors disabled:opacity-50 ${
                          item.is_hard_watch
                            ? 'bg-orange-500/20 text-orange-400 hover:bg-red-500/20 hover:text-red-400'
                            : 'bg-th-surface-high text-th-on-muted hover:text-th-on-surface'
                        }`}
                      >
                        {saving === item.stock_code ? '...' : item.is_hard_watch ? '감시 중' : '등록'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="text-xs text-th-on-subtle px-4 py-3">최대 200개 표시</p>
          </div>
        )}
      </div>
    </div>
  )
}
