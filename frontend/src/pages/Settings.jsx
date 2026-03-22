import { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

/* ── 읽기 전용 행 ── */
function Row({ label, children }) {
  return (
    <div className="flex justify-between items-center text-sm py-2.5 border-b border-black/5 dark:border-white/5 last:border-0">
      <span className="text-th-on-muted">{label}</span>
      <span className="text-th-on-surface">{children}</span>
    </div>
  )
}
Row.propTypes = { label: PropTypes.string, children: PropTypes.node }

function Badge({ ok, trueLabel = '설정됨', falseLabel = '미설정' }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${ok ? 'bg-emerald-500/10 text-emerald-400' : 'bg-white/5 text-th-on-muted'}`}>
      {ok ? trueLabel : falseLabel}
    </span>
  )
}
Badge.propTypes = { ok: PropTypes.bool, trueLabel: PropTypes.string, falseLabel: PropTypes.string }

function WsBadge({ connected }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${connected ? 'bg-blue-500/10 text-blue-400' : 'bg-white/5 text-th-on-muted'}`}>
      {connected ? '연결됨' : '미연결'}
    </span>
  )
}
WsBadge.propTypes = { connected: PropTypes.bool }

// FID_TRGT_EXLS_CLS_CODE 10자리 각 비트의 의미
const EXCL_LABELS = [
  '투자위험',
  '투자경고',
  '투자주의',
  '관리종목',
  '정리매매',
  '불성실공시',
  '우선주',
  '거래정지',
  'ETF',
  'ETN',
]

const SELL_CONDITIONS = [
  { value: 'target_pct', label: '목표가 도달 (WebSocket 실시간)' },
  { value: 'stop_pct', label: '손절가 도달 (WebSocket 실시간)' },
  { value: 'rsi_overbought', label: 'RSI 과매수' },
  { value: 'macd_bearish', label: 'MACD 데드크로스' },
  { value: 'stagnation', label: '횡보 감지 (가격 정체 자동 매도)' },
]

export default function Settings() {
  const { data, loading, error, refetch } = useApi('/api/settings')

  // ── 거래 제어 ──
  const [tradingEnabled, setTradingEnabled] = useState(true)
  const [tradingStartTime, setTradingStartTime] = useState('09:15')
  const [tradingEndTime, setTradingEndTime] = useState('15:15')

  // ── 종목 선정 (순위 조회) ──
  // 10개 체크박스 (각 index = 해당 비트, true = 제외)
  const [exclBits, setExclBits] = useState(Array(10).fill(true))
  const [rankingTypes, setRankingTypes] = useState(['volume', 'strength', 'exec_count', 'disparity'])
  const [rankingPriceMin, setRankingPriceMin] = useState('5000')
  const [rankingPriceMax, setRankingPriceMax] = useState('100000')
  const [rankingTopN, setRankingTopN] = useState('20')
  const [volumeMinIncrRate, setVolumeMinIncrRate] = useState('0')
  const [strengthMin, setStrengthMin] = useState('100')
  const [execCountNetBuyOnly, setExecCountNetBuyOnly] = useState(true)
  const [disparityD20Min, setDisparityD20Min] = useState('0')
  const [disparityD20Max, setDisparityD20Max] = useState('0')

  const [rankingCondition, setRankingCondition] = useState('AND')

  // ── 매수 설정 ──
  const [maxPositions, setMaxPositions] = useState('1')
  const [orderAmountPct, setOrderAmountPct] = useState('95')

  // ── 매도 설정 ──
  const [takeProfitPct, setTakeProfitPct] = useState('3.0')
  const [stopLossPct, setStopLossPct] = useState('2.0')
  const [sellConditions, setSellConditions] = useState(['target_pct', 'stop_pct'])
  const [indicatorIntervalMin, setIndicatorIntervalMin] = useState('5')
  const [rsiThreshold, setRsiThreshold] = useState('70')
  const [macdBearish, setMacdBearish] = useState(false)
  const [stagnationThresholdPct, setStagnationThresholdPct] = useState('1.0')
  const [stagnationDurationMin, setStagnationDurationMin] = useState('30')

  // ── 매수 품질 필터 ──
  const [minTradingValue, setMinTradingValue] = useState('0')
  const [buyPauseStart, setBuyPauseStart] = useState('11:00')
  const [buyPauseEnd, setBuyPauseEnd] = useState('14:00')

  // ── 트레일링 스탑 ──
  const [trailingTriggerPct, setTrailingTriggerPct] = useState('0')
  const [trailingStopPct, setTrailingStopPct] = useState('1.0')

  // ── 리스크 관리 ──
  const [dailyMaxLossPct, setDailyMaxLossPct] = useState('0')
  const [indexCodes, setIndexCodes] = useState([])

  // ── 하드 필터 (매수 품질) ──
  const [filterRsiMax, setFilterRsiMax] = useState('80')
  const [filterDisparityM5Max, setFilterDisparityM5Max] = useState('3.0')
  const [filterHighPriceDiffMin, setFilterHighPriceDiffMin] = useState('-5.0')
  const [filterOpenPriceDiffMax, setFilterOpenPriceDiffMax] = useState('20.0')
  const [indexDropThresholdPct, setIndexDropThresholdPct] = useState('-1.0')

  // ── 요일 스케줄 ──
  const [tradingDays, setTradingDays] = useState([1, 2, 3, 4, 5])

  // ── AI 매매 기준값 — 하드 리젝션 ──
  const [hardDisparityM5Min, setHardDisparityM5Min] = useState('-1.5')
  const [hardDisparityM5Max, setHardDisparityM5Max] = useState('3.0')
  const [hardHighPriceDiffMax, setHardHighPriceDiffMax] = useState('-0.5')
  const [hardHighPriceDiffMin, setHardHighPriceDiffMin] = useState('-5.0')
  const [hardPrevVolRatioMax, setHardPrevVolRatioMax] = useState('1.2')
  const [hardStrengthMin, setHardStrengthMin] = useState('100.0')
  const [hardRsiMax, setHardRsiMax] = useState('70.0')
  const [hardOpenPriceDiffMax, setHardOpenPriceDiffMax] = useState('15.0')

  // ── AI 매매 기준값 — 랭킹 기준 ──
  const [vwapDiffMin, setVwapDiffMin] = useState('0.0')
  const [vwapDiffMax, setVwapDiffMax] = useState('1.5')
  const [rsiBuyMin, setRsiBuyMin] = useState('40.0')
  const [rsiBuyMax, setRsiBuyMax] = useState('60.0')
  const [bidAskRatioMin, setBidAskRatioMin] = useState('1.2')

  // ── AI 설정 ──
  const [claudeModel, setClaudeModel] = useState('claude-sonnet-4-6')

  // ── 미장 설정 ──
  const [usTradingEnabled, setUsTradingEnabled] = useState(false)
  const [usDstEnabled, setUsDstEnabled] = useState(true)
  const [usTradingStartTime, setUsTradingStartTime] = useState('22:30')
  const [usTradingEndTime, setUsTradingEndTime] = useState('05:00')
  const [usRankingTypes, setUsRankingTypes] = useState(['volume'])
  const [usRankingExchange, setUsRankingExchange] = useState('NAS')
  const [usRankingPriceMin, setUsRankingPriceMin] = useState('10')
  const [usRankingPriceMax, setUsRankingPriceMax] = useState('500')
  const [usRankingVolRang, setUsRankingVolRang] = useState('0')
  const [usRankingTopN, setUsRankingTopN] = useState('20')
  const [usDailyMaxLossPct, setUsDailyMaxLossPct] = useState('0')
  const [usMinTradingValue, setUsMinTradingValue] = useState('0')

  const [saving, setSaving] = useState(false)
  const [saveResult, setSaveResult] = useState(null)

  // 서버에서 읽어온 값으로 초기화
  useEffect(() => {
    if (!data) return
    setTradingEnabled(data.trading_enabled !== false)
    const cls = data.ranking_excl_cls || '1111111111'
    setExclBits(cls.split('').map((ch) => ch === '1'))

    if (data.trading_start_time) setTradingStartTime(data.trading_start_time)
    if (data.trading_end_time) setTradingEndTime(data.trading_end_time)

    if (Array.isArray(data.ranking_types)) setRankingTypes(data.ranking_types)
    if (data.ranking_price_min) setRankingPriceMin(data.ranking_price_min)
    if (data.ranking_price_max) setRankingPriceMax(data.ranking_price_max)
    if (data.ranking_top_n != null) setRankingTopN(String(data.ranking_top_n))
    if (data.ranking_volume_min_incrrate != null) setVolumeMinIncrRate(String(data.ranking_volume_min_incrrate))
    if (data.ranking_strength_min != null) setStrengthMin(String(data.ranking_strength_min))
    if (data.ranking_execcount_net_buy_only != null) setExecCountNetBuyOnly(data.ranking_execcount_net_buy_only)
    if (data.ranking_disparity_d20_min != null) setDisparityD20Min(String(data.ranking_disparity_d20_min))
    if (data.ranking_disparity_d20_max != null) setDisparityD20Max(String(data.ranking_disparity_d20_max))
    if (data.ranking_condition === 'AND' || data.ranking_condition === 'OR') setRankingCondition(data.ranking_condition)

    if (data.max_positions != null) setMaxPositions(String(data.max_positions))
    if (data.order_amount_pct != null) setOrderAmountPct(String(data.order_amount_pct))

    if (data.take_profit_pct != null) setTakeProfitPct(String(data.take_profit_pct))
    if (data.stop_loss_pct != null) setStopLossPct(String(data.stop_loss_pct))
    if (Array.isArray(data.sell_conditions)) setSellConditions(data.sell_conditions)
    if (data.indicator_check_interval_min != null) setIndicatorIntervalMin(String(data.indicator_check_interval_min))
    if (data.indicator_rsi_sell_threshold != null) setRsiThreshold(String(data.indicator_rsi_sell_threshold))
    if (data.indicator_macd_bearish_sell != null) setMacdBearish(data.indicator_macd_bearish_sell)
    if (data.stagnation_threshold_pct != null) setStagnationThresholdPct(String(data.stagnation_threshold_pct))
    if (data.stagnation_duration_min != null) setStagnationDurationMin(String(data.stagnation_duration_min))

    if (data.min_trading_value != null) setMinTradingValue(String(data.min_trading_value))
    if (data.buy_pause_start) setBuyPauseStart(data.buy_pause_start)
    if (data.buy_pause_end) setBuyPauseEnd(data.buy_pause_end)
    if (data.trailing_trigger_pct != null) setTrailingTriggerPct(String(data.trailing_trigger_pct))
    if (data.trailing_stop_pct != null) setTrailingStopPct(String(data.trailing_stop_pct))
    if (data.daily_max_loss_pct != null) setDailyMaxLossPct(String(data.daily_max_loss_pct))
    if (Array.isArray(data.index_codes)) setIndexCodes(data.index_codes)

    if (data.claude_model) setClaudeModel(data.claude_model)

    if (Array.isArray(data.trading_days)) setTradingDays(data.trading_days)

    if (data.hard_disparity_m5_min != null) setHardDisparityM5Min(String(data.hard_disparity_m5_min))
    if (data.hard_disparity_m5_max != null) setHardDisparityM5Max(String(data.hard_disparity_m5_max))
    if (data.hard_high_price_diff_max != null) setHardHighPriceDiffMax(String(data.hard_high_price_diff_max))
    if (data.hard_high_price_diff_min != null) setHardHighPriceDiffMin(String(data.hard_high_price_diff_min))
    if (data.hard_prev_vol_ratio_max != null) setHardPrevVolRatioMax(String(data.hard_prev_vol_ratio_max))
    if (data.hard_strength_min != null) setHardStrengthMin(String(data.hard_strength_min))
    if (data.hard_rsi_max != null) setHardRsiMax(String(data.hard_rsi_max))
    if (data.hard_open_price_diff_max != null) setHardOpenPriceDiffMax(String(data.hard_open_price_diff_max))
    if (data.vwap_diff_min != null) setVwapDiffMin(String(data.vwap_diff_min))
    if (data.vwap_diff_max != null) setVwapDiffMax(String(data.vwap_diff_max))
    if (data.rsi_buy_min != null) setRsiBuyMin(String(data.rsi_buy_min))
    if (data.rsi_buy_max != null) setRsiBuyMax(String(data.rsi_buy_max))
    if (data.bid_ask_ratio_min != null) setBidAskRatioMin(String(data.bid_ask_ratio_min))

    if (data.filter_rsi_max != null) setFilterRsiMax(String(data.filter_rsi_max))
    if (data.filter_disparity_m5_max != null) setFilterDisparityM5Max(String(data.filter_disparity_m5_max))
    if (data.filter_high_price_diff_min != null) setFilterHighPriceDiffMin(String(data.filter_high_price_diff_min))
    if (data.filter_open_price_diff_max != null) setFilterOpenPriceDiffMax(String(data.filter_open_price_diff_max))
    if (data.index_drop_threshold_pct != null) setIndexDropThresholdPct(String(data.index_drop_threshold_pct))

    if (data.us_trading_enabled != null) setUsTradingEnabled(data.us_trading_enabled)
    if (data.us_dst_enabled != null) setUsDstEnabled(data.us_dst_enabled)
    if (data.us_trading_start_time) setUsTradingStartTime(data.us_trading_start_time)
    if (data.us_trading_end_time) setUsTradingEndTime(data.us_trading_end_time)
    if (Array.isArray(data.us_ranking_types)) setUsRankingTypes(data.us_ranking_types)
    if (data.us_ranking_exchange) setUsRankingExchange(data.us_ranking_exchange)
    if (data.us_ranking_price_min) setUsRankingPriceMin(data.us_ranking_price_min)
    if (data.us_ranking_price_max) setUsRankingPriceMax(data.us_ranking_price_max)
    if (data.us_ranking_vol_rang != null) setUsRankingVolRang(String(data.us_ranking_vol_rang))
    if (data.us_ranking_top_n != null) setUsRankingTopN(String(data.us_ranking_top_n))
    if (data.us_daily_max_loss_pct != null) setUsDailyMaxLossPct(String(data.us_daily_max_loss_pct))
    if (data.us_min_trading_value != null) setUsMinTradingValue(String(data.us_min_trading_value))
  }, [data])

  function toggleBit(i) {
    setExclBits((prev) => prev.map((v, idx) => (idx === i ? !v : v)))
  }

  function toggleRankingType(val) {
    setRankingTypes((prev) =>
      prev.includes(val) ? prev.filter((v) => v !== val) : [...prev, val]
    )
  }

  function toggleSellCondition(val) {
    setSellConditions((prev) =>
      prev.includes(val) ? prev.filter((v) => v !== val) : [...prev, val]
    )
  }

  function moveSellCondition(val, dir) {
    setSellConditions((prev) => {
      const idx = prev.indexOf(val)
      if (idx === -1) return prev
      const next = [...prev]
      const swap = idx + dir
      if (swap < 0 || swap >= next.length) return prev
      ;[next[idx], next[swap]] = [next[swap], next[idx]]
      return next
    })
  }

  async function handleSave(e) {
    e.preventDefault()
    setSaving(true)
    setSaveResult(null)

    const rankingExclCls = exclBits.map((b) => (b ? '1' : '0')).join('')
    const body = {
      trading_enabled: tradingEnabled,
      trading_start_time: tradingStartTime,
      trading_end_time: tradingEndTime,
      ranking_excl_cls: rankingExclCls,
      ranking_types: rankingTypes,
      ranking_price_min: rankingPriceMin,
      ranking_price_max: rankingPriceMax,
      ranking_top_n: parseInt(rankingTopN) || 20,
      ranking_volume_min_incrrate: parseFloat(volumeMinIncrRate) || 0,
      ranking_strength_min: parseFloat(strengthMin) || 0,
      ranking_execcount_net_buy_only: execCountNetBuyOnly,
      ranking_disparity_d20_min: parseFloat(disparityD20Min) || 0,
      ranking_disparity_d20_max: parseFloat(disparityD20Max) || 0,
      ranking_condition: rankingCondition,
      max_positions: parseInt(maxPositions) || 1,
      order_amount_pct: parseFloat(orderAmountPct) || 95,
      take_profit_pct: parseFloat(takeProfitPct) || 3.0,
      stop_loss_pct: parseFloat(stopLossPct) || 2.0,
      sell_conditions: sellConditions,
      indicator_check_interval_min: parseInt(indicatorIntervalMin) || 5,
      indicator_rsi_sell_threshold: parseFloat(rsiThreshold) || 70,
      indicator_macd_bearish_sell: macdBearish,
      stagnation_threshold_pct: parseFloat(stagnationThresholdPct) || 1.0,
      stagnation_duration_min: parseInt(stagnationDurationMin) || 30,
      min_trading_value: parseFloat(minTradingValue) || 0,
      buy_pause_start: buyPauseStart,
      buy_pause_end: buyPauseEnd,
      trailing_trigger_pct: parseFloat(trailingTriggerPct) || 0,
      trailing_stop_pct: parseFloat(trailingStopPct) || 1.0,
      daily_max_loss_pct: parseFloat(dailyMaxLossPct) || 0,
      index_codes: indexCodes,
      claude_model: claudeModel,
      us_trading_enabled: usTradingEnabled,
      us_dst_enabled: usDstEnabled,
      us_trading_start_time: usTradingStartTime,
      us_trading_end_time: usTradingEndTime,
      us_ranking_types: usRankingTypes,
      us_ranking_exchange: usRankingExchange,
      us_ranking_price_min: usRankingPriceMin,
      us_ranking_price_max: usRankingPriceMax,
      us_ranking_vol_rang: usRankingVolRang,
      us_ranking_top_n: parseInt(usRankingTopN) || 20,
      us_daily_max_loss_pct: parseFloat(usDailyMaxLossPct) || 0,
      us_min_trading_value: parseFloat(usMinTradingValue) || 0,
      filter_rsi_max: parseFloat(filterRsiMax) || 80,
      filter_disparity_m5_max: parseFloat(filterDisparityM5Max) || 3.0,
      filter_high_price_diff_min: parseFloat(filterHighPriceDiffMin) || -5.0,
      filter_open_price_diff_max: parseFloat(filterOpenPriceDiffMax) || 20.0,
      index_drop_threshold_pct: parseFloat(indexDropThresholdPct) || -1.0,
      trading_days: tradingDays,
      hard_disparity_m5_min: parseFloat(hardDisparityM5Min),
      hard_disparity_m5_max: parseFloat(hardDisparityM5Max),
      hard_high_price_diff_max: parseFloat(hardHighPriceDiffMax),
      hard_high_price_diff_min: parseFloat(hardHighPriceDiffMin),
      hard_prev_vol_ratio_max: parseFloat(hardPrevVolRatioMax),
      hard_strength_min: parseFloat(hardStrengthMin),
      hard_rsi_max: parseFloat(hardRsiMax),
      hard_open_price_diff_max: parseFloat(hardOpenPriceDiffMax),
      vwap_diff_min: parseFloat(vwapDiffMin),
      vwap_diff_max: parseFloat(vwapDiffMax),
      rsi_buy_min: parseFloat(rsiBuyMin),
      rsi_buy_max: parseFloat(rsiBuyMax),
      bid_ask_ratio_min: parseFloat(bidAskRatioMin),
    }

    try {
      const res = await fetch('/api/settings', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const json = await res.json()
      if (!res.ok) {
        setSaveResult({ ok: false, text: json.error || '저장 실패' })
      } else {
        setSaveResult({ ok: true, text: json.message || '저장되었습니다.' })
        refetch()
      }
    } catch (err) {
      setSaveResult({ ok: false, text: err.message })
    } finally {
      setSaving(false)
    }
  }

  const stagnationActive = sellConditions.includes('stagnation')

  const inputCls = 'w-full px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50'
  const sectionCls = 'bg-th-surface rounded-xl p-5 space-y-4'
  const sectionTitle = 'text-sm font-semibold text-th-on-surface'
  const labelText = 'text-xs text-th-on-muted'
  const hintText = 'text-xs text-th-on-subtle'
  const divider = 'pt-3 border-t border-black/5 dark:border-white/5'

  return (
    <div className="space-y-6 pb-20">
      {/* 스티키 헤더 (저장 버튼 고정) */}
      <div className="sticky top-0 z-30 glass-panel -mx-4 md:-mx-8 px-4 md:px-8 py-3 flex items-center justify-between mb-2">
        <div>
          <h1 className="text-2xl font-bold text-th-on-surface tracking-tight">설정</h1>
          <p className="text-xs text-th-on-muted mt-0.5 uppercase tracking-widest">트레이딩 파라미터 및 서버 구성</p>
        </div>
        <div className="flex items-center gap-3">
          {saveResult && (
            <span className={`text-xs ${saveResult.ok ? 'text-emerald-400' : 'text-red-400'}`}>
              {saveResult.text}
            </span>
          )}
          <button
            type="submit"
            form="settings-form"
            disabled={saving}
            className="px-5 py-2 bg-orange-500 hover:bg-orange-600 disabled:opacity-50 rounded-lg text-sm font-semibold transition-colors text-white"
          >
            {saving ? '저장 중...' : '설정 저장'}
          </button>
        </div>
      </div>

      {/* ── 편집 폼 ── */}
      <form id="settings-form" onSubmit={handleSave} className="space-y-5">

        {/* ── 섹션 1: 거래 제어 ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>거래 제어</p>

          {/* ON/OFF 토글 */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-th-on-surface">Trading</p>
              <p className="text-xs text-th-on-muted mt-0.5">OFF 시 주문 API가 차단됩니다</p>
            </div>
            <button
              type="button"
              onClick={() => setTradingEnabled((v) => !v)}
              className={`relative inline-flex h-7 w-12 items-center rounded-full transition-colors focus:outline-none ${tradingEnabled ? 'bg-emerald-500' : 'bg-th-surface-high'}`}
            >
              <span
                className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${tradingEnabled ? 'translate-x-6' : 'translate-x-1'}`}
              />
            </button>
          </div>
          <p className="text-xs text-center font-semibold">
            {tradingEnabled
              ? <span className="text-emerald-400">거래 활성화 (ON)</span>
              : <span className="text-th-on-muted">거래 비활성화 (OFF)</span>
            }
          </p>

          {/* 거래 시간 */}
          <div className={`grid grid-cols-2 gap-4 ${divider}`}>
            <label className="space-y-1">
              <span className={labelText}>거래 시작 시간</span>
              <input
                type="time" step="60"
                value={tradingStartTime}
                onChange={(e) => setTradingStartTime(e.target.value)}
                className={inputCls}
              />
            </label>
            <label className="space-y-1">
              <span className={labelText}>거래 종료 시간</span>
              <input
                type="time" step="60"
                value={tradingEndTime}
                onChange={(e) => setTradingEndTime(e.target.value)}
                className={inputCls}
              />
            </label>
          </div>
          <p className={hintText}>기본값: 09:15 ~ 15:15. 변경 시 다음 거래일부터 적용됩니다.</p>

          {/* 매수 중단 시간 */}
          <div className={`grid grid-cols-2 gap-4 ${divider}`}>
            <label className="space-y-1">
              <span className={labelText}>매수 중단 시작</span>
              <input
                type="time" step="60"
                value={buyPauseStart}
                onChange={(e) => setBuyPauseStart(e.target.value)}
                className={inputCls}
              />
            </label>
            <label className="space-y-1">
              <span className={labelText}>매수 중단 종료</span>
              <input
                type="time" step="60"
                value={buyPauseEnd}
                onChange={(e) => setBuyPauseEnd(e.target.value)}
                className={inputCls}
              />
            </label>
          </div>
          <p className={hintText}>기본값: 11:00 ~ 14:00 매수 중단 (점심시간 유동성 저하 방지)</p>

          {/* 지수 필터 */}
          <div className={`space-y-2 ${divider}`}>
            <span className={`${labelText} block`}>지수 필터</span>
            <p className={hintText}>체크된 지수가 시가 대비 하락 임계값 이하로 하락 시 매수를 중단합니다.</p>
            <div className="flex gap-4">
              {[{ code: '0001', label: '코스피' }, { code: '1001', label: '코스닥' }].map(({ code, label }) => (
                <label key={code} className="flex items-center gap-1.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={indexCodes.includes(code)}
                    onChange={(e) => setIndexCodes(prev =>
                      e.target.checked ? [...prev, code] : prev.filter(c => c !== code)
                    )}
                    className="accent-orange-500"
                  />
                  <span className="text-sm text-th-on-surface">{label} ({code})</span>
                </label>
              ))}
            </div>
            <label className="space-y-1 block">
              <span className={labelText}>하락 임계값 (%, 이하 시 매수 중단)</span>
              <input
                type="number" step="0.1"
                value={indexDropThresholdPct}
                onChange={(e) => setIndexDropThresholdPct(e.target.value)}
                className="w-full md:w-48 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
              />
              <p className={hintText}>기본 -1.0 — 지수가 시가 대비 이 값 이하로 하락 시 매수 중단</p>
            </label>
          </div>

          {/* 거래 요일 */}
          <div className={`space-y-2 ${divider}`}>
            <span className={`${labelText} block`}>거래 요일</span>
            <p className={hintText}>체크된 요일에만 자동매매가 실행됩니다.</p>
            <div className="flex flex-wrap gap-3">
              {[
                { day: 1, label: '월' },
                { day: 2, label: '화' },
                { day: 3, label: '수' },
                { day: 4, label: '목' },
                { day: 5, label: '금' },
                { day: 6, label: '토' },
                { day: 0, label: '일' },
              ].map(({ day, label }) => (
                <label key={day} className="flex items-center gap-1.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={tradingDays.includes(day)}
                    onChange={(e) =>
                      setTradingDays((prev) =>
                        e.target.checked ? [...prev, day] : prev.filter((d) => d !== day)
                      )
                    }
                    className="accent-orange-500"
                  />
                  <span className="text-sm text-th-on-surface">{label}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        {/* ── 섹션 2: 종목 선정 (순위 조회) ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>종목 선정 (순위 조회)</p>

          {/* 제외 종목 필터 */}
          <div className="space-y-2">
            <p className={labelText}>순위조회 제외 종목</p>
            <p className={hintText}>체크된 항목은 순위조회 결과에서 제외됩니다</p>
            <div className="grid grid-cols-2 gap-2">
              {EXCL_LABELS.map((label, i) => (
                <label key={i} className="flex items-center gap-2 cursor-pointer group">
                  <input
                    type="checkbox"
                    checked={exclBits[i]}
                    onChange={() => toggleBit(i)}
                    className="w-4 h-4 rounded bg-th-surface-high accent-orange-500"
                  />
                  <span className="text-sm text-th-on-surface transition-colors">{label}</span>
                </label>
              ))}
            </div>
          </div>

          {/* 가격 범위 */}
          <div className={`space-y-3 ${divider}`}>
            <div className="grid grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className={labelText}>최소 주가 (원)</span>
                <input
                  type="number" step="1000" min="0"
                  value={rankingPriceMin}
                  onChange={(e) => setRankingPriceMin(e.target.value)}
                  className={inputCls}
                />
              </label>
              <label className="space-y-1">
                <span className={labelText}>최대 주가 (원)</span>
                <input
                  type="number" step="1000" min="0"
                  value={rankingPriceMax}
                  onChange={(e) => setRankingPriceMax(e.target.value)}
                  className={inputCls}
                />
              </label>
            </div>

            {/* 상위 N개 */}
            <label className="space-y-1">
              <span className={labelText}>각 순위별 상위 종목 수 (필터 통과 기준, 최대 30)</span>
              <input
                type="number" step="1" min="1" max="30"
                value={rankingTopN}
                onChange={(e) => setRankingTopN(e.target.value)}
                className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
              />
            </label>
          </div>

          {/* 순위 유형 선택 */}
          <div className={`space-y-3 ${divider}`}>
            {/* 거래량 순위 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <input type="checkbox" checked={rankingTypes.includes('volume')}
                  onChange={() => toggleRankingType('volume')}
                  className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                <span className="text-sm text-th-on-surface font-medium">거래량 순위</span>
              </div>
              {rankingTypes.includes('volume') && (
                <div className="ml-6">
                  <label className="space-y-1">
                    <span className={labelText}>전일대비 거래량 증가율 최솟값 (%, 0=필터없음)</span>
                    <input type="number" step="10" min="0"
                      value={volumeMinIncrRate}
                      onChange={(e) => setVolumeMinIncrRate(e.target.value)}
                      className="w-40 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
                    />
                  </label>
                </div>
              )}
            </div>

            {/* 체결강도 순위 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <input type="checkbox" checked={rankingTypes.includes('strength')}
                  onChange={() => toggleRankingType('strength')}
                  className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                <span className="text-sm text-th-on-surface font-medium">체결강도 순위</span>
              </div>
              {rankingTypes.includes('strength') && (
                <div className="ml-6">
                  <label className="space-y-1">
                    <span className={labelText}>최소 체결강도 (%, 100=매수우세 이상, 0=필터없음)</span>
                    <input type="number" step="5" min="0"
                      value={strengthMin}
                      onChange={(e) => setStrengthMin(e.target.value)}
                      className="w-40 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
                    />
                  </label>
                </div>
              )}
            </div>

            {/* 대량체결 순위 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <input type="checkbox" checked={rankingTypes.includes('exec_count')}
                  onChange={() => toggleRankingType('exec_count')}
                  className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                <span className="text-sm text-th-on-surface font-medium">대량체결 순위</span>
              </div>
              {rankingTypes.includes('exec_count') && (
                <div className="ml-6">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={execCountNetBuyOnly}
                      onChange={(e) => setExecCountNetBuyOnly(e.target.checked)}
                      className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                    <span className="text-sm text-th-on-surface">순매수 우세 종목만 (순매수체결량 &gt; 0)</span>
                  </label>
                </div>
              )}
            </div>

            {/* 이격도 순위 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <input type="checkbox" checked={rankingTypes.includes('disparity')}
                  onChange={() => toggleRankingType('disparity')}
                  className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                <span className="text-sm text-th-on-surface font-medium">이격도 순위</span>
              </div>
              {rankingTypes.includes('disparity') && (
                <div className="ml-6 flex items-center gap-3">
                  <label className="space-y-1">
                    <span className={labelText}>20일 이격도 최솟값 (0=필터없음)</span>
                    <input type="number" step="1" min="0"
                      value={disparityD20Min}
                      onChange={(e) => setDisparityD20Min(e.target.value)}
                      className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
                    />
                  </label>
                  <span className="text-th-on-muted mt-4">~</span>
                  <label className="space-y-1">
                    <span className={labelText}>최댓값 (0=필터없음)</span>
                    <input type="number" step="1" min="0"
                      value={disparityD20Max}
                      onChange={(e) => setDisparityD20Max(e.target.value)}
                      className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
                    />
                  </label>
                </div>
              )}
            </div>
          </div>

          {/* 순위 조건 (AND/OR) */}
          <div className={`space-y-2 ${divider}`}>
            <p className={labelText}>순위 조건</p>
            <p className={hintText}>AND: 모든 선택 순위에 공통으로 포함된 종목만 / OR: 하나 이상의 순위에 포함된 종목 모두</p>
            <div className="flex gap-2">
              {['AND', 'OR'].map((cond) => (
                <button
                  key={cond}
                  type="button"
                  onClick={() => setRankingCondition(cond)}
                  className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-colors border ${
                    rankingCondition === cond
                      ? 'bg-th-surface-high text-th-on-surface border-black/10 dark:border-white/10 ring-1 ring-zinc-600'
                      : 'bg-transparent text-th-on-muted border-black/10 dark:border-white/10 hover:text-th-on-surface'
                  }`}
                >
                  {cond}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* ── 섹션 3: 매수 설정 ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>매수 설정</p>

          <div className="grid grid-cols-2 gap-4">
            <label className="space-y-1">
              <span className={labelText}>최대 동시 보유 종목</span>
              <input
                type="number" step="1" min="1" max="10"
                value={maxPositions}
                onChange={(e) => setMaxPositions(e.target.value)}
                className={inputCls}
              />
            </label>
            <label className="space-y-1">
              <span className={labelText}>주문 금액 비율 (%)</span>
              <input
                type="number" step="1" min="1" max="100"
                value={orderAmountPct}
                onChange={(e) => setOrderAmountPct(e.target.value)}
                className={inputCls}
              />
            </label>
          </div>

        </div>

        {/* ── 섹션 4: 매도 설정 ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>매도 설정</p>

          {/* 익절/손절 기준 */}
          <div className="grid grid-cols-2 gap-4">
            <label className="space-y-1">
              <span className="text-xs text-red-400">익절 기준 (%)</span>
              <input
                type="number" step="0.1" min="0.1"
                value={takeProfitPct}
                onChange={(e) => setTakeProfitPct(e.target.value)}
                className={inputCls}
              />
            </label>
            <label className="space-y-1">
              <span className="text-xs text-[#3B82F6]">손절 기준 (%)</span>
              <input
                type="number" step="0.1" min="0.1"
                value={stopLossPct}
                onChange={(e) => setStopLossPct(e.target.value)}
                className={inputCls}
              />
            </label>
          </div>

          {/* 매도 조건 우선순위 */}
          <div className={`space-y-2 ${divider}`}>
            <p className={labelText}>매도 조건 우선순위</p>
            <p className={hintText}>위에서부터 순서대로 평가됩니다. 화살표로 우선순위를 변경하세요.</p>
            <div className="space-y-1">
              {sellConditions.map((val, idx) => {
                const item = SELL_CONDITIONS.find(c => c.value === val)
                if (!item) return null
                return (
                  <div key={val} className="flex items-center gap-2 bg-th-surface-high/60 rounded-lg px-3 py-2">
                    <span className="text-xs text-th-on-muted w-4">{idx + 1}</span>
                    <span className="flex-1 text-sm text-th-on-surface">{item.label}</span>
                    <button type="button" onClick={() => moveSellCondition(val, -1)} disabled={idx === 0}
                      className="text-th-on-muted hover:text-th-on-surface disabled:opacity-20 px-1">▲</button>
                    <button type="button" onClick={() => moveSellCondition(val, 1)} disabled={idx === sellConditions.length - 1}
                      className="text-th-on-muted hover:text-th-on-surface disabled:opacity-20 px-1">▼</button>
                    <button type="button" onClick={() => toggleSellCondition(val)}
                      className="text-th-on-subtle hover:text-red-400 px-1 text-xs">✕</button>
                  </div>
                )
              })}
              {SELL_CONDITIONS.filter(c => !sellConditions.includes(c.value)).map(({ value, label }) => (
                <div key={value} className="flex items-center gap-2 rounded-lg px-3 py-2 opacity-40">
                  <span className="text-xs text-th-on-muted w-4">-</span>
                  <span className="flex-1 text-sm text-th-on-muted">{label}</span>
                  <button type="button" onClick={() => toggleSellCondition(value)}
                    className="text-th-on-subtle hover:text-emerald-400 px-1 text-xs">＋</button>
                </div>
              ))}
            </div>
          </div>

          {/* 지표 설정 */}
          <div className={`space-y-3 ${divider}`}>
            <p className={labelText}>지표 확인 설정</p>
            <div className="grid grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className={labelText}>지표 확인 주기 (분)</span>
                <input
                  type="number" step="1" min="1"
                  value={indicatorIntervalMin}
                  onChange={(e) => setIndicatorIntervalMin(e.target.value)}
                  className={inputCls}
                />
              </label>
              <label className="space-y-1">
                <span className={labelText}>RSI 매도 기준값</span>
                <input
                  type="number" step="1" min="50" max="100"
                  value={rsiThreshold}
                  onChange={(e) => setRsiThreshold(e.target.value)}
                  className={inputCls}
                />
              </label>
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={macdBearish} onChange={(e) => setMacdBearish(e.target.checked)}
                className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
              <span className="text-sm text-th-on-surface">MACD 데드크로스 시 매도</span>
            </label>
          </div>

          {/* 트레일링 스탑 */}
          <div className={`space-y-3 ${divider}`}>
            <p className={labelText}>트레일링 스탑</p>
            <p className={hintText}>수익률이 활성화 기준에 도달하면, 이후 최고가 대비 일정 % 하락 시 자동 매도합니다. 활성화 기준 0=비활성.</p>
            <div className="grid grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className={labelText}>활성화 기준 수익률 (%, 0=비활성)</span>
                <input
                  type="number" step="0.1" min="0"
                  value={trailingTriggerPct}
                  onChange={(e) => setTrailingTriggerPct(e.target.value)}
                  className={inputCls}
                />
              </label>
              <label className="space-y-1">
                <span className={labelText}>최고가 대비 하락 허용폭 (%)</span>
                <input
                  type="number" step="0.1" min="0.1"
                  value={trailingStopPct}
                  onChange={(e) => setTrailingStopPct(e.target.value)}
                  className={inputCls}
                />
              </label>
            </div>
          </div>

          {/* 일일 최대 손실 */}
          <div className={`space-y-2 ${divider}`}>
            <p className={labelText}>일일 최대 손실 한도</p>
            <label className="space-y-1 block">
              <span className={labelText}>총자산 대비 최대 손실 (%, 0=제한없음)</span>
              <input
                type="number" step="0.1" min="0"
                value={dailyMaxLossPct}
                onChange={(e) => setDailyMaxLossPct(e.target.value)}
                className="w-40 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
              />
            </label>
            <p className={hintText}>당일 실현 손실이 한도 초과 시 매수를 중단합니다.</p>
          </div>

          {/* 횡보 감지 설정 (stagnation 조건 활성 시에만 표시) */}
          {stagnationActive && (
            <div className={`space-y-3 ${divider}`}>
              <p className={labelText}>횡보 감지 설정</p>
              <p className={hintText}>진입가 기준 ±N% 이내 변동이 M분 이상 지속되면 매도합니다.</p>
              <div className="grid grid-cols-2 gap-4">
                <label className="space-y-1">
                  <span className={labelText}>횡보 기준 변동폭 (%)</span>
                  <input
                    type="number" step="0.1" min="0.1"
                    value={stagnationThresholdPct}
                    onChange={(e) => setStagnationThresholdPct(e.target.value)}
                    className={inputCls}
                  />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>횡보 지속 기준 (분)</span>
                  <input
                    type="number" step="5" min="5"
                    value={stagnationDurationMin}
                    onChange={(e) => setStagnationDurationMin(e.target.value)}
                    className={inputCls}
                  />
                </label>
              </div>
            </div>
          )}
        </div>

        {/* ── 섹션 5b: 하드 필터 (매수 품질) ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>하드 필터 (매수 품질)</p>
          <p className={hintText}>LLM 호출 전 자동으로 제거되는 조건입니다. KR·US 공통 적용됩니다.</p>

          <div className={`space-y-1 ${divider}`}>
            <label className="space-y-1 block">
              <span className={labelText}>최소 거래대금 (원, 0=필터없음)</span>
              <input
                type="number" step="any" min="0"
                value={minTradingValue}
                onChange={(e) => setMinTradingValue(e.target.value)}
                className={inputCls}
              />
            </label>
            <p className={hintText}>
              예: 5000000000 = 50억원. 거래대금 미달 종목은 LLM 후보에서 제외됩니다.
            </p>
          </div>

          <div className="grid md:grid-cols-2 gap-4">
            <label className="space-y-1">
              <span className={labelText}>RSI 과열 임계값 (이상 제외)</span>
              <input
                type="number" step="1" min="50" max="100"
                value={filterRsiMax}
                onChange={(e) => setFilterRsiMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>기본 80 — RSI ≥ 이 값인 종목 제외</p>
            </label>

            <label className="space-y-1">
              <span className={labelText}>5분봉 이격도 최대값 (%) (초과 제외)</span>
              <input
                type="number" step="0.1" min="0"
                value={filterDisparityM5Max}
                onChange={(e) => setFilterDisparityM5Max(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>기본 3.0 — 5분봉 MA5 이격도 초과 시 제외</p>
            </label>

            <label className="space-y-1">
              <span className={labelText}>고가 대비 최솟값 (%) (미만 제외)</span>
              <input
                type="number" step="0.1"
                value={filterHighPriceDiffMin}
                onChange={(e) => setFilterHighPriceDiffMin(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>기본 -5.0 — 고가 대비 하락 폭이 이 값 미만인 종목 제외</p>
            </label>

            <label className="space-y-1">
              <span className={labelText}>시가 대비 최댓값 (%) (초과 제외)</span>
              <input
                type="number" step="0.1" min="0"
                value={filterOpenPriceDiffMax}
                onChange={(e) => setFilterOpenPriceDiffMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>기본 20.0 — 당일 상한가 영역 종목 제외</p>
            </label>
          </div>

        </div>

        {/* ── 섹션 5c: AI 매매 기준값 ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>AI 매매 기준값</p>
          <p className={hintText}>Claude에게 전달되는 하드 리젝션 룰과 랭킹 기준 수치입니다. 변경 시 즉시 다음 종목 선정에 반영됩니다.</p>

          {/* 하드 리젝션 룰 */}
          <div className="space-y-1">
            <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">하드 리젝션 룰 (ANY 해당 시 제외)</p>
          </div>
          <div className={`grid md:grid-cols-2 gap-4 ${divider}`}>
            <label className="space-y-1">
              <span className={labelText}>5분봉 이격도 하한 (%)</span>
              <input
                type="number" step="0.1"
                value={hardDisparityM5Min}
                onChange={(e) => setHardDisparityM5Min(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이하 → 칼날 하락 구간 (기본 -1.5)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>5분봉 이격도 상한 (%)</span>
              <input
                type="number" step="0.1"
                value={hardDisparityM5Max}
                onChange={(e) => setHardDisparityM5Max(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이상 → 과열 구간 (기본 3.0)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>고점 대비 상한 (%)</span>
              <input
                type="number" step="0.1"
                value={hardHighPriceDiffMax}
                onChange={(e) => setHardHighPriceDiffMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이상 → 고점 추격 위험 (기본 -0.5)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>고점 대비 하한 (%)</span>
              <input
                type="number" step="0.1"
                value={hardHighPriceDiffMin}
                onChange={(e) => setHardHighPriceDiffMin(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이하 + 거래량 급증 → 추세이탈 (기본 -5.0)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>하락 시 거래량 비율 상한</span>
              <input
                type="number" step="0.1" min="0"
                value={hardPrevVolRatioMax}
                onChange={(e) => setHardPrevVolRatioMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>하락 중 전 캔들 대비 거래량 비율 (기본 1.2)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>최소 체결강도</span>
              <input
                type="number" step="1" min="0"
                value={hardStrengthMin}
                onChange={(e) => setHardStrengthMin(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이하 → 매수세 소멸 (기본 100)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>RSI 과매수 상한</span>
              <input
                type="number" step="1" min="50" max="100"
                value={hardRsiMax}
                onChange={(e) => setHardRsiMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이상 → 과매수에서 꺾임 (기본 70)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>시가 대비 상승률 상한 (%)</span>
              <input
                type="number" step="0.5" min="0"
                value={hardOpenPriceDiffMax}
                onChange={(e) => setHardOpenPriceDiffMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이 값 이상 → 상한가 영역 (기본 15)</p>
            </label>
          </div>

          {/* 랭킹 기준 */}
          <div className={`space-y-1 ${divider}`}>
            <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">랭킹 우선 기준 (선호 구간)</p>
          </div>
          <div className={`grid md:grid-cols-2 gap-4`}>
            <label className="space-y-1">
              <span className={labelText}>VWAP 이격도 하한 (%)</span>
              <input
                type="number" step="0.1"
                value={vwapDiffMin}
                onChange={(e) => setVwapDiffMin(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>VWAP 지지선 위에서 매수 (기본 0.0)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>VWAP 이격도 상한 (%)</span>
              <input
                type="number" step="0.1"
                value={vwapDiffMax}
                onChange={(e) => setVwapDiffMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>VWAP 과리 제외 (기본 1.5)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>RSI 매수 구간 하한</span>
              <input
                type="number" step="1" min="0" max="100"
                value={rsiBuyMin}
                onChange={(e) => setRsiBuyMin(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이상적 RSI 매수 구간 (기본 40)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>RSI 매수 구간 상한</span>
              <input
                type="number" step="1" min="0" max="100"
                value={rsiBuyMax}
                onChange={(e) => setRsiBuyMax(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>이상적 RSI 매수 구간 (기본 60)</p>
            </label>
            <label className="space-y-1">
              <span className={labelText}>최소 매수호가 우세 비율</span>
              <input
                type="number" step="0.1" min="0"
                value={bidAskRatioMin}
                onChange={(e) => setBidAskRatioMin(e.target.value)}
                className={inputCls}
              />
              <p className={hintText}>매수잔량 / 매도잔량 최소 비율 (기본 1.2)</p>
            </label>
          </div>
        </div>

        {/* ── 섹션 6: 미장 (미국주식) 설정 ── */}
        <div className={sectionCls}>
          <p className={sectionTitle}>미장 (미국주식) 자동매매</p>

          {/* ON/OFF 토글 */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-th-on-surface">미장 자동매매</p>
              <p className="text-xs text-th-on-muted mt-0.5">미국 주식 시장 자동 거래 활성화</p>
            </div>
            <button type="button" onClick={() => setUsTradingEnabled(v => !v)}
              className={`relative inline-flex h-7 w-12 items-center rounded-full transition-colors focus:outline-none ${usTradingEnabled ? 'bg-emerald-500' : 'bg-th-surface-high'}`}>
              <span className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${usTradingEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>

          {usTradingEnabled && (
            <div className="space-y-4">
              {/* 서머타임 토글 */}
              <div className={`flex items-center justify-between pt-2 border-t border-black/10 dark:border-white/10`}>
                <div>
                  <p className="text-sm text-th-on-surface">서머타임 (DST)</p>
                  <p className="text-xs text-th-on-muted mt-0.5">ON: 22:30~05:00 / OFF: 23:30~06:00</p>
                </div>
                <button type="button" onClick={() => setUsDstEnabled(v => !v)}
                  className={`relative inline-flex h-7 w-12 items-center rounded-full transition-colors focus:outline-none ${usDstEnabled ? 'bg-zinc-600' : 'bg-th-surface-high'}`}>
                  <span className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${usDstEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
                </button>
              </div>

              {/* 거래 시간 */}
              <div className="grid grid-cols-2 gap-4 pt-2 border-t border-black/10 dark:border-white/10">
                <label className="space-y-1">
                  <span className={labelText}>미장 시작 시간 (KST)</span>
                  <input type="time" step="60" value={usTradingStartTime} onChange={e => setUsTradingStartTime(e.target.value)}
                    className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>미장 종료 시간 (KST)</span>
                  <input type="time" step="60" value={usTradingEndTime} onChange={e => setUsTradingEndTime(e.target.value)}
                    className={inputCls} />
                </label>
              </div>

              {/* 거래소 선택 */}
              <div className="pt-2 border-t border-black/10 dark:border-white/10">
                <p className={`${labelText} mb-2`}>거래소</p>
                <div className="flex gap-2">
                  {['NAS', 'NYS', 'AMS'].map(exch => (
                    <button key={exch} type="button" onClick={() => setUsRankingExchange(exch)}
                      className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors border ${
                        usRankingExchange === exch
                          ? 'bg-th-surface-high text-th-on-surface border-black/10 dark:border-white/10 ring-1 ring-zinc-600'
                          : 'bg-transparent text-th-on-muted border-black/10 dark:border-white/10 hover:text-th-on-surface'
                      }`}>
                      {exch === 'NAS' ? 'NASDAQ' : exch === 'NYS' ? 'NYSE' : 'AMEX'}
                    </button>
                  ))}
                </div>
              </div>

              {/* 가격 범위 (USD) */}
              <div className="grid grid-cols-2 gap-4 pt-2 border-t border-black/10 dark:border-white/10">
                <label className="space-y-1">
                  <span className={labelText}>최소 주가 (USD)</span>
                  <input type="number" step="1" min="0" value={usRankingPriceMin}
                    onChange={e => setUsRankingPriceMin(e.target.value)}
                    className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>최대 주가 (USD)</span>
                  <input type="number" step="1" min="0" value={usRankingPriceMax}
                    onChange={e => setUsRankingPriceMax(e.target.value)}
                    className={inputCls} />
                </label>
              </div>

              {/* 순위 유형 */}
              <div className="space-y-2 pt-2 border-t border-black/10 dark:border-white/10">
                <p className={labelText}>순위 조회 유형</p>
                {[
                  { value: 'volume', label: '거래량 순위' },
                ].map(({ value, label }) => (
                  <label key={value} className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox"
                      checked={usRankingTypes.includes(value)}
                      onChange={() => setUsRankingTypes(prev =>
                        prev.includes(value) ? prev.filter(v => v !== value) : [...prev, value]
                      )}
                      className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                    <span className="text-sm text-th-on-surface">{label}</span>
                  </label>
                ))}
              </div>

              {/* 거래량 필터 */}
              <div className="pt-2 border-t border-black/10 dark:border-white/10">
                <p className={`${labelText} mb-2`}>거래량 필터</p>
                <div className="flex gap-2 flex-wrap">
                  {[
                    { value: '0', label: '전체' },
                    { value: '1', label: '100주↑' },
                    { value: '2', label: '1000주↑' },
                    { value: '3', label: '10000주↑' },
                  ].map(({ value, label }) => (
                    <button key={value} type="button" onClick={() => setUsRankingVolRang(value)}
                      className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors border ${
                        usRankingVolRang === value
                          ? 'bg-th-surface-high text-th-on-surface border-black/10 dark:border-white/10 ring-1 ring-zinc-600'
                          : 'bg-transparent text-th-on-muted border-black/10 dark:border-white/10 hover:text-th-on-surface'
                      }`}>
                      {label}
                    </button>
                  ))}
                </div>
              </div>

              {/* 상위 N개 */}
              <label className="space-y-1 pt-2 border-t border-black/10 dark:border-white/10 block">
                <span className={labelText}>상위 종목 수</span>
                <input type="number" step="1" min="1" max="50"
                  value={usRankingTopN}
                  onChange={e => setUsRankingTopN(e.target.value)}
                  className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
              </label>

              {/* 미장 일일 최대 손실 한도 */}
              <div className="space-y-3 pt-2 border-t border-black/10 dark:border-white/10">
                <label className="space-y-1 block">
                  <span className={labelText}>미장 일일 최대 손실 한도 (%)</span>
                  <input type="number" step="0.1" min="0"
                    value={usDailyMaxLossPct}
                    onChange={e => setUsDailyMaxLossPct(e.target.value)}
                    className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
                  <p className={hintText}>가용 USD 대비 최대 손실 기준. 0 = 국장 손실 한도 공유.</p>
                </label>
                <label className="space-y-1 block">
                  <span className={labelText}>미장 최소 거래대금 (USD)</span>
                  <input type="number" step="1" min="0"
                    value={usMinTradingValue}
                    onChange={e => setUsMinTradingValue(e.target.value)}
                    className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
                  <p className={hintText}>0 = 국장 최소 거래대금(원) 설정 공유.</p>
                </label>
              </div>
            </div>
          )}
        </div>

      </form>

      {/* ── AI 설정 (폼 밖, 최하단) ── */}
      <div className={`${sectionCls} !space-y-3`}>
        <p className={sectionTitle}>AI 설정</p>
        <p className={hintText}>트레이딩 종목 선정에 사용할 Claude 모델을 지정합니다. ANTHROPIC_API_KEY 환경변수가 필요합니다.</p>
        <label className="space-y-1 block">
          <span className={labelText}>Claude 모델명</span>
          <input
            type="text"
            value={claudeModel}
            onChange={(e) => setClaudeModel(e.target.value)}
            className={`${inputCls} font-mono`}
            placeholder="claude-sonnet-4-6"
          />
        </label>
      </div>

      {/* ── 서버 정보 (읽기 전용) ── */}
      {error && (
        <div className="bg-red-500/10 text-red-400 rounded-xl p-4 text-sm">{error}</div>
      )}
      {!loading && data && (
        <div className="space-y-3">
          <p className="text-xs font-semibold text-th-on-subtle uppercase tracking-widest">서버 정보 (읽기 전용)</p>

          <div className="bg-th-surface rounded-xl p-5">
            <p className="text-xs text-th-on-muted font-medium mb-3">계좌 정보</p>
            <Row label="계좌번호"><span className="font-data">{data.account_no || '-'}</span></Row>
            <Row label="계좌 유형">
              {data.account_type === '01' ? '종합계좌 (01)' : data.account_type === '22' ? '선물옵션 (22)' : data.account_type || '-'}
            </Row>
            <Row label="KIS API 키"><Badge ok={data.kis_configured} /></Row>
            <Row label="Anthropic API 키"><Badge ok={data.anthropic_configured} /></Row>
          </div>

          <div className="bg-th-surface rounded-xl p-5">
            <p className="text-xs text-th-on-muted font-medium mb-3">실시간 연동</p>
            <Row label="KIS HTS ID">
              <Badge ok={data.hts_id_configured} falseLabel="미설정 (체결통보 비활성)" />
            </Row>
            <Row label="WebSocket 연결"><WsBadge connected={data.ws_connected} /></Row>
          </div>
        </div>
      )}

      <p className="text-xs text-th-on-subtle">
        KIS API 키, 계좌 정보 등 민감 정보는 서버의 .env 파일에서 관리합니다.
      </p>
    </div>
  )
}
