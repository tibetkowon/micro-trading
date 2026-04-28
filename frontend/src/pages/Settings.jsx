import { useState, useEffect, useCallback } from 'react'
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

const inputCls = 'w-full px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50'
const sectionCls = 'bg-th-surface rounded-xl p-5 space-y-4'
const sectionTitle = 'text-sm font-semibold text-th-on-surface'
const hintText = 'text-xs text-th-on-subtle'
const divider = 'pt-3 border-t border-black/5 dark:border-white/5'

function PresetPanel({
  market, presets, activePresetId, nameVal, descVal, setName, setDesc,
  presetSaving, presetApplying, presetMsg,
  handleSavePreset, handleApplyPreset, handleDeletePreset,
}) {
  const isKR = market === 'KR'
  const filteredPresets = presets.filter((p) => p.market === market)
  const emptyMsg = isKR ? '저장된 국장 프리셋이 없습니다.' : '저장된 미장 프리셋이 없습니다.'
  const saveBtnLabel = isKR ? '국장 설정 저장' : '미장 설정 저장'
  const placeholder = isKR ? '예: 공격적' : '예: 나스닥 변동성'

  return (
    <div className={`${sectionCls} !space-y-3`}>
      <div className="flex items-center gap-2">
        <p className={sectionTitle}>{isKR ? '국장' : '미장'} 프리셋</p>
        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium ${isKR ? 'bg-blue-500/10 text-blue-400' : 'bg-orange-500/10 text-orange-400'}`}>
          {isKR ? '국내' : '해외'}
        </span>
      </div>
      <p className={hintText}>
        {isKR
          ? '국장 설정(거래 제어, 종목 선정, 매수/매도, 필터 등)만 저장·적용합니다.'
          : '미장 설정(us_ 접두사 항목)만 저장·적용합니다. 국장 설정은 영향받지 않습니다.'}
      </p>
      {filteredPresets.length > 0 ? (
        <div className="space-y-2">
          {filteredPresets.map((p) => (
            <div key={p.id} className={`flex items-center gap-2 rounded-lg px-3 py-2 ${p.id === activePresetId ? 'bg-orange-500/10 ring-1 ring-orange-500/30' : 'bg-th-surface-high'}`}>
              <div className="flex-1 min-w-0">
                <span className="text-sm font-medium text-th-on-surface">{p.name}</span>
                {p.id === activePresetId && (
                  <span className="text-[10px] text-orange-400 ml-2 font-medium uppercase tracking-wider">적용 중</span>
                )}
                {p.description && (
                  <span className="text-xs text-th-on-muted ml-2">{p.description}</span>
                )}
              </div>
              <button
                type="button"
                onClick={() => handleApplyPreset(p.id)}
                disabled={presetApplying === p.id}
                className="shrink-0 px-3 py-1 text-xs rounded-md bg-orange-500/15 text-orange-400 hover:bg-orange-500/25 disabled:opacity-50 transition-colors"
              >
                {presetApplying === p.id ? '적용 중...' : '적용'}
              </button>
              <button
                type="button"
                onClick={() => handleDeletePreset(p.id, p.name)}
                className="shrink-0 p-1 text-th-on-subtle hover:text-red-400 transition-colors rounded"
                title="삭제"
              >
                <span className="material-symbols-outlined text-[16px]">delete</span>
              </button>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-xs text-th-on-subtle">{emptyMsg}</p>
      )}
      <div className={`flex flex-col sm:flex-row gap-2 ${divider}`}>
        <input
          type="text"
          placeholder={`프리셋 이름 (${placeholder})`}
          value={nameVal}
          onChange={(e) => setName(e.target.value)}
          className={`flex-1 ${inputCls}`}
        />
        <input
          type="text"
          placeholder="설명 (선택)"
          value={descVal}
          onChange={(e) => setDesc(e.target.value)}
          className={`flex-1 ${inputCls}`}
        />
        <button
          type="button"
          onClick={() => handleSavePreset(market)}
          disabled={presetSaving}
          className="shrink-0 px-4 py-1.5 bg-th-surface-high hover:bg-th-surface-highest text-th-on-surface text-xs font-semibold rounded-lg disabled:opacity-50 transition-colors"
        >
          {presetSaving ? '저장 중...' : saveBtnLabel}
        </button>
      </div>
      {presetMsg && (
        <p className={`text-xs ${presetMsg.ok ? 'text-emerald-400' : 'text-red-400'}`}>{presetMsg.text}</p>
      )}
    </div>
  )
}
PresetPanel.propTypes = {
  market: PropTypes.string,
  presets: PropTypes.array,
  activePresetId: PropTypes.number,
  nameVal: PropTypes.string,
  descVal: PropTypes.string,
  setName: PropTypes.func,
  setDesc: PropTypes.func,
  presetSaving: PropTypes.bool,
  presetApplying: PropTypes.any,
  presetMsg: PropTypes.object,
  handleSavePreset: PropTypes.func,
  handleApplyPreset: PropTypes.func,
  handleDeletePreset: PropTypes.func,
}

export default function Settings() {
  const { data, loading, error, refetch } = useApi('/api/settings')

  // ── 프리셋 ──
  const [presets, setPresets] = useState([])
  const [activePresetId, setActivePresetId] = useState(0)
  const [krPresetName, setKrPresetName] = useState('')
  const [krPresetDesc, setKrPresetDesc] = useState('')
  const [usPresetName, setUsPresetName] = useState('')
  const [usPresetDesc, setUsPresetDesc] = useState('')
  const [presetSaving, setPresetSaving] = useState(false)
  const [presetApplying, setPresetApplying] = useState(null)
  const [presetMsg, setPresetMsg] = useState(null)

  const fetchPresets = useCallback(async () => {
    try {
      const res = await fetch('/api/presets')
      const json = await res.json()
      if (res.ok) setPresets(json.presets || [])
    } catch (_) { /* ignore */ }
  }, [])

  useEffect(() => { fetchPresets() }, [fetchPresets])

  async function handleSavePreset(market) {
    const name = market === 'KR' ? krPresetName : usPresetName
    const desc = market === 'KR' ? krPresetDesc : usPresetDesc
    if (!name.trim()) { setPresetMsg({ ok: false, text: '프리셋 이름을 입력하세요' }); return }
    setPresetSaving(true); setPresetMsg(null)
    try {
      const res = await fetch('/api/presets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name.trim(), description: desc.trim(), market }),
      })
      const json = await res.json()
      if (!res.ok) { setPresetMsg({ ok: false, text: json.error || '저장 실패' }); return }
      setPresetMsg({ ok: true, text: json.message })
      if (market === 'KR') { setKrPresetName(''); setKrPresetDesc('') }
      else { setUsPresetName(''); setUsPresetDesc('') }
      fetchPresets()
    } catch (err) {
      setPresetMsg({ ok: false, text: err.message })
    } finally {
      setPresetSaving(false)
    }
  }

  async function handleApplyPreset(id) {
    setPresetApplying(id); setPresetMsg(null)
    try {
      const res = await fetch(`/api/presets/${id}/apply`, { method: 'POST' })
      const json = await res.json()
      if (!res.ok) { setPresetMsg({ ok: false, text: json.error || '적용 실패' }); return }
      setPresetMsg({ ok: true, text: json.message })
      refetch()
    } catch (err) {
      setPresetMsg({ ok: false, text: err.message })
    } finally {
      setPresetApplying(null)
    }
  }

  async function handleDeletePreset(id, name) {
    if (!window.confirm(`'${name}' 프리셋을 삭제하시겠습니까?`)) return
    try {
      const res = await fetch(`/api/presets/${id}`, { method: 'DELETE' })
      const json = await res.json()
      if (!res.ok) { setPresetMsg({ ok: false, text: json.error || '삭제 실패' }); return }
      setPresetMsg({ ok: true, text: json.message })
      fetchPresets()
    } catch (err) {
      setPresetMsg({ ok: false, text: err.message })
    }
  }

  // ── 거래 제어 ──
  const [tradingEnabled, setTradingEnabled] = useState(true)
  const [tradingStartTime, setTradingStartTime] = useState('09:15')
  const [tradingEndTime, setTradingEndTime] = useState('15:15')

  // ── 종목 선정 (순위 조회) ──
  const [exclBits, setExclBits] = useState(Array(10).fill(true))
  const [rankingTypes, setRankingTypes] = useState(['volume', 'strength'])
  const [rankingPriceMin, setRankingPriceMin] = useState('5000')
  const [rankingPriceMax, setRankingPriceMax] = useState('100000')
  const [rankingTopN, setRankingTopN] = useState('20')
  const [volumeMinIncrRate, setVolumeMinIncrRate] = useState('0')
  const [strengthMin, setStrengthMin] = useState('100')
  const [fluctuationMinRate, setFluctuationMinRate] = useState('0')
  const [fluctuationMaxRate, setFluctuationMaxRate] = useState('0')
  const [viKindCode, setViKindCode] = useState('')
  const [rankingCondition, setRankingCondition] = useState('AND')
  const [rankingExchanges, setRankingExchanges] = useState(['0001', '1001'])
  const [rankingVolumeBlngClsCodes, setRankingVolumeBlngClsCodes] = useState(['0', '1', '2', '3', '4'])

  // ── 매수 설정 ──
  const [maxPositions, setMaxPositions] = useState('1')
  const [orderAmountPct, setOrderAmountPct] = useState('95')

  // ── 매도 설정 ──
  const [takeProfitPct, setTakeProfitPct] = useState('3.0')
  const [stopLossPct, setStopLossPct] = useState('2.0')
  const [etfTakeProfitPct, setEtfTakeProfitPct] = useState('0.5')
  const [etfStopLossPct, setEtfStopLossPct] = useState('0.4')
  const [stockTakeProfitPct, setStockTakeProfitPct] = useState('1.5')
  const [stockStopLossPct, setStockStopLossPct] = useState('1.0')
  const [sellConditions, setSellConditions] = useState(['target_pct', 'stop_pct'])
  const [indicatorIntervalMin, setIndicatorIntervalMin] = useState('5')
  const [rsiThreshold, setRsiThreshold] = useState('70')
  const [macdBearish, setMacdBearish] = useState(false)
  const [stagnationThresholdPct, setStagnationThresholdPct] = useState('1.0')
  const [stagnationDurationMin, setStagnationDurationMin] = useState('30')
  const [stagnationPartialExitEnabled, setStagnationPartialExitEnabled] = useState(false)
  const [stagnationBidAskSellThreshold, setStagnationBidAskSellThreshold] = useState('1.0')
  const [momentumScoreMin, setMomentumScoreMin] = useState('0')
  // ── 부분 익절 ──
  const [partialTPEnabled, setPartialTPEnabled] = useState(false)
  const [partialTPPct, setPartialTPPct] = useState('1.0')
  const [partialTPRatio, setPartialTPRatio] = useState('0.5')
  const [partialTPRaiseStop, setPartialTPRaiseStop] = useState(true)
  // ── 복합 스코어링 가중치 ──
  const [scoringBidAskWeight, setScoringBidAskWeight] = useState('30')
  const [scoringStrengthWeight, setScoringStrengthWeight] = useState('25')
  const [scoringMACDWeight, setScoringMACDWeight] = useState('20')
  const [scoringRSIWeight, setScoringRSIWeight] = useState('15')
  const [scoringVWAPWeight, setScoringVWAPWeight] = useState('10')

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
  const [hardMacdBearishEnabled, setHardMacdBearishEnabled] = useState(false)
  const [hardHighFormedMinsMax, setHardHighFormedMinsMax] = useState('0')
  const [hardVolVs3AvgRatioMin, setHardVolVs3AvgRatioMin] = useState('0')
  const [hardRelativeStrengthMin, setHardRelativeStrengthMin] = useState('0')

  // ── Adaptive Threshold ──
  const [adaptiveThresholdEnabled, setAdaptiveThresholdEnabled] = useState(false)
  const [adaptiveThresholdTrigger, setAdaptiveThresholdTrigger] = useState('10')
  const [adaptiveRelaxPct, setAdaptiveRelaxPct] = useState('20')

  // ── Market Phase Detection ──
  const [marketPhaseRelaxEnabled, setMarketPhaseRelaxEnabled] = useState(false)
  const [marketPhaseIndexDropTrigger, setMarketPhaseIndexDropTrigger] = useState('-1.0')
  const [marketPhaseRelaxPct, setMarketPhaseRelaxPct] = useState('15')
  // ── Hard Rule Escalation ──
  const [escalationEnabled, setEscalationEnabled] = useState(false)
  const [escalationTrigger, setEscalationTrigger] = useState('20')
  const [escalationStepPct, setEscalationStepPct] = useState('10')
  const [escalationMaxStages, setEscalationMaxStages] = useState('5')
  // ── Hard Rule Feedback (룰별 자동 완화) ──
  const [hardRuleFeedbackEnabled, setHardRuleFeedbackEnabled] = useState(false)
  const [hardRuleFeedbackWindow, setHardRuleFeedbackWindow] = useState('10')
  const [hardRuleFeedbackThresholdPct, setHardRuleFeedbackThresholdPct] = useState('70')

  // ── AI 매매 기준값 — 랭킹 기준 ──
  const [vwapDiffMin, setVwapDiffMin] = useState('0.0')
  const [vwapDiffMax, setVwapDiffMax] = useState('1.5')
  const [rsiBuyMin, setRsiBuyMin] = useState('40.0')
  const [rsiBuyMax, setRsiBuyMax] = useState('60.0')
  const [bidAskRatioMin, setBidAskRatioMin] = useState('1.2')
  const [minMarketCap, setMinMarketCap] = useState('0')
  const [minExpectedProfitPct, setMinExpectedProfitPct] = useState('0')
  const [maxClaudeCandidates, setMaxClaudeCandidates] = useState('15')

  // ── AI 설정 ──
  const [claudeModel, setClaudeModel] = useState('claude-sonnet-4-6')
  const [optimizationApplyMode, setOptimizationApplyMode] = useState('all_manual')

  // ── 하드 감시 종목 ──
  const [hardWatchSymbols, setHardWatchSymbols] = useState([])
  const [rankLeaseDurationMin, setRankLeaseDurationMin] = useState('5')
  const [hardWatchInput, setHardWatchInput] = useState('')

  const [saving, setSaving] = useState(false)
  const [saveResult, setSaveResult] = useState(null)
  const [activeTab, setActiveTab] = useState('KR')

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
    if (data.ranking_fluctuation_min_rate != null) setFluctuationMinRate(String(data.ranking_fluctuation_min_rate))
    if (data.ranking_fluctuation_max_rate != null) setFluctuationMaxRate(String(data.ranking_fluctuation_max_rate))
    if (data.ranking_vi_kind_code != null) setViKindCode(String(data.ranking_vi_kind_code))
    if (data.ranking_condition === 'AND' || data.ranking_condition === 'OR') setRankingCondition(data.ranking_condition)
    if (Array.isArray(data.ranking_exchanges)) setRankingExchanges(data.ranking_exchanges)
    if (Array.isArray(data.ranking_volume_blng_cls_codes)) setRankingVolumeBlngClsCodes(data.ranking_volume_blng_cls_codes)

    if (data.max_positions != null) setMaxPositions(String(data.max_positions))
    if (data.order_amount_pct != null) setOrderAmountPct(String(data.order_amount_pct))

    if (data.take_profit_pct != null) setTakeProfitPct(String(data.take_profit_pct))
    if (data.stop_loss_pct != null) setStopLossPct(String(data.stop_loss_pct))
    if (data.etf_take_profit_pct != null) setEtfTakeProfitPct(String(data.etf_take_profit_pct))
    if (data.etf_stop_loss_pct != null) setEtfStopLossPct(String(data.etf_stop_loss_pct))
    if (data.stock_take_profit_pct != null) setStockTakeProfitPct(String(data.stock_take_profit_pct))
    if (data.stock_stop_loss_pct != null) setStockStopLossPct(String(data.stock_stop_loss_pct))
    if (Array.isArray(data.sell_conditions)) setSellConditions(data.sell_conditions)
    if (data.indicator_check_interval_min != null) setIndicatorIntervalMin(String(data.indicator_check_interval_min))
    if (data.indicator_rsi_sell_threshold != null) setRsiThreshold(String(data.indicator_rsi_sell_threshold))
    if (data.indicator_macd_bearish_sell != null) setMacdBearish(data.indicator_macd_bearish_sell)
    if (data.stagnation_threshold_pct != null) setStagnationThresholdPct(String(data.stagnation_threshold_pct))
    if (data.stagnation_duration_min != null) setStagnationDurationMin(String(data.stagnation_duration_min))
    if (data.stagnation_partial_exit_enabled != null) setStagnationPartialExitEnabled(data.stagnation_partial_exit_enabled)
    if (data.stagnation_bid_ask_sell_threshold != null) setStagnationBidAskSellThreshold(String(data.stagnation_bid_ask_sell_threshold))
    if (data.momentum_score_min != null) setMomentumScoreMin(String(data.momentum_score_min))
    if (data.partial_tp_enabled != null) setPartialTPEnabled(data.partial_tp_enabled)
    if (data.partial_tp_pct != null) setPartialTPPct(String(data.partial_tp_pct))
    if (data.partial_tp_ratio != null) setPartialTPRatio(String(data.partial_tp_ratio))
    if (data.partial_tp_raise_stop != null) setPartialTPRaiseStop(data.partial_tp_raise_stop)
    if (data.scoring_bidask_weight != null) setScoringBidAskWeight(String(data.scoring_bidask_weight))
    if (data.scoring_strength_weight != null) setScoringStrengthWeight(String(data.scoring_strength_weight))
    if (data.scoring_macd_weight != null) setScoringMACDWeight(String(data.scoring_macd_weight))
    if (data.scoring_rsi_weight != null) setScoringRSIWeight(String(data.scoring_rsi_weight))
    if (data.scoring_vwap_weight != null) setScoringVWAPWeight(String(data.scoring_vwap_weight))

    if (data.min_trading_value != null) setMinTradingValue(String(data.min_trading_value))
    if (data.buy_pause_start) setBuyPauseStart(data.buy_pause_start)
    if (data.buy_pause_end) setBuyPauseEnd(data.buy_pause_end)
    if (data.trailing_trigger_pct != null) setTrailingTriggerPct(String(data.trailing_trigger_pct))
    if (data.trailing_stop_pct != null) setTrailingStopPct(String(data.trailing_stop_pct))
    if (data.daily_max_loss_pct != null) setDailyMaxLossPct(String(data.daily_max_loss_pct))
    if (Array.isArray(data.index_codes)) setIndexCodes(data.index_codes)

    if (data.claude_model) setClaudeModel(data.claude_model)
    if (data.optimization_apply_mode) setOptimizationApplyMode(data.optimization_apply_mode)

    if (Array.isArray(data.hard_watch_symbols)) setHardWatchSymbols(data.hard_watch_symbols)
    if (data.rank_lease_duration_min != null) setRankLeaseDurationMin(String(data.rank_lease_duration_min))

    if (Array.isArray(data.trading_days)) setTradingDays(data.trading_days)

    if (data.hard_disparity_m5_min != null) setHardDisparityM5Min(String(data.hard_disparity_m5_min))
    if (data.hard_disparity_m5_max != null) setHardDisparityM5Max(String(data.hard_disparity_m5_max))
    if (data.hard_high_price_diff_max != null) setHardHighPriceDiffMax(String(data.hard_high_price_diff_max))
    if (data.hard_high_price_diff_min != null) setHardHighPriceDiffMin(String(data.hard_high_price_diff_min))
    if (data.hard_prev_vol_ratio_max != null) setHardPrevVolRatioMax(String(data.hard_prev_vol_ratio_max))
    if (data.hard_strength_min != null) setHardStrengthMin(String(data.hard_strength_min))
    if (data.hard_rsi_max != null) setHardRsiMax(String(data.hard_rsi_max))
    if (data.hard_open_price_diff_max != null) setHardOpenPriceDiffMax(String(data.hard_open_price_diff_max))
    if (data.hard_macd_bearish_enabled != null) setHardMacdBearishEnabled(data.hard_macd_bearish_enabled)
    if (data.hard_high_formed_mins_max != null) setHardHighFormedMinsMax(String(data.hard_high_formed_mins_max))
    if (data.hard_vol_vs_3avg_ratio_min != null) setHardVolVs3AvgRatioMin(String(data.hard_vol_vs_3avg_ratio_min))
    if (data.hard_relative_strength_min != null) setHardRelativeStrengthMin(String(data.hard_relative_strength_min))
    if (data.adaptive_threshold_enabled != null) setAdaptiveThresholdEnabled(data.adaptive_threshold_enabled)
    if (data.adaptive_threshold_trigger != null) setAdaptiveThresholdTrigger(String(data.adaptive_threshold_trigger))
    if (data.market_phase_relax_enabled != null) setMarketPhaseRelaxEnabled(data.market_phase_relax_enabled)
    if (data.market_phase_index_drop_trigger != null) setMarketPhaseIndexDropTrigger(String(data.market_phase_index_drop_trigger))
    if (data.market_phase_relax_pct != null) setMarketPhaseRelaxPct(String(data.market_phase_relax_pct))
    if (data.escalation_enabled != null) setEscalationEnabled(data.escalation_enabled)
    if (data.escalation_trigger != null) setEscalationTrigger(String(data.escalation_trigger))
    if (data.escalation_step_pct != null) setEscalationStepPct(String(data.escalation_step_pct))
    if (data.escalation_max_stages != null) setEscalationMaxStages(String(data.escalation_max_stages))
    if (data.hard_rule_feedback_enabled != null) setHardRuleFeedbackEnabled(data.hard_rule_feedback_enabled)
    if (data.hard_rule_feedback_window != null) setHardRuleFeedbackWindow(String(data.hard_rule_feedback_window))
    if (data.hard_rule_feedback_threshold_pct != null) setHardRuleFeedbackThresholdPct(String(data.hard_rule_feedback_threshold_pct))
    if (data.adaptive_relax_pct != null) setAdaptiveRelaxPct(String(data.adaptive_relax_pct))
    if (data.vwap_diff_min != null) setVwapDiffMin(String(data.vwap_diff_min))
    if (data.vwap_diff_max != null) setVwapDiffMax(String(data.vwap_diff_max))
    if (data.rsi_buy_min != null) setRsiBuyMin(String(data.rsi_buy_min))
    if (data.rsi_buy_max != null) setRsiBuyMax(String(data.rsi_buy_max))
    if (data.bid_ask_ratio_min != null) setBidAskRatioMin(String(data.bid_ask_ratio_min))
    if (data.min_market_cap != null) setMinMarketCap(String(data.min_market_cap))
    if (data.min_expected_profit_pct != null) setMinExpectedProfitPct(String(data.min_expected_profit_pct))
    if (data.max_claude_candidates != null) setMaxClaudeCandidates(String(data.max_claude_candidates))
    if (data.active_preset_id != null) setActivePresetId(Number(data.active_preset_id) || 0)

    if (data.filter_rsi_max != null) setFilterRsiMax(String(data.filter_rsi_max))
    if (data.filter_disparity_m5_max != null) setFilterDisparityM5Max(String(data.filter_disparity_m5_max))
    if (data.filter_high_price_diff_min != null) setFilterHighPriceDiffMin(String(data.filter_high_price_diff_min))
    if (data.filter_open_price_diff_max != null) setFilterOpenPriceDiffMax(String(data.filter_open_price_diff_max))
    if (data.index_drop_threshold_pct != null) setIndexDropThresholdPct(String(data.index_drop_threshold_pct))

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
      ranking_top_n: parseInt(rankingTopN),
      ranking_volume_min_incrrate: parseFloat(volumeMinIncrRate),
      ranking_strength_min: parseFloat(strengthMin),
      ranking_fluctuation_min_rate: parseFloat(fluctuationMinRate),
      ranking_fluctuation_max_rate: parseFloat(fluctuationMaxRate),
      ranking_vi_kind_code: viKindCode,
      ranking_condition: rankingCondition,
      ranking_exchanges: rankingExchanges,
      ranking_volume_blng_cls_codes: rankingVolumeBlngClsCodes,
      max_positions: parseInt(maxPositions),
      order_amount_pct: parseFloat(orderAmountPct),
      take_profit_pct: parseFloat(takeProfitPct),
      stop_loss_pct: parseFloat(stopLossPct),
      etf_take_profit_pct: parseFloat(etfTakeProfitPct),
      etf_stop_loss_pct: parseFloat(etfStopLossPct),
      stock_take_profit_pct: parseFloat(stockTakeProfitPct),
      stock_stop_loss_pct: parseFloat(stockStopLossPct),
      sell_conditions: sellConditions,
      indicator_check_interval_min: parseInt(indicatorIntervalMin),
      indicator_rsi_sell_threshold: parseFloat(rsiThreshold),
      indicator_macd_bearish_sell: macdBearish,
      stagnation_threshold_pct: parseFloat(stagnationThresholdPct),
      stagnation_duration_min: parseInt(stagnationDurationMin),
      stagnation_partial_exit_enabled: stagnationPartialExitEnabled,
      stagnation_bid_ask_sell_threshold: parseFloat(stagnationBidAskSellThreshold),
      momentum_score_min: parseFloat(momentumScoreMin),
      partial_tp_enabled: partialTPEnabled,
      partial_tp_pct: parseFloat(partialTPPct),
      partial_tp_ratio: parseFloat(partialTPRatio),
      partial_tp_raise_stop: partialTPRaiseStop,
      scoring_bidask_weight: parseInt(scoringBidAskWeight),
      scoring_strength_weight: parseInt(scoringStrengthWeight),
      scoring_macd_weight: parseInt(scoringMACDWeight),
      scoring_rsi_weight: parseInt(scoringRSIWeight),
      scoring_vwap_weight: parseInt(scoringVWAPWeight),
      min_trading_value: parseFloat(minTradingValue),
      buy_pause_start: buyPauseStart,
      buy_pause_end: buyPauseEnd,
      trailing_trigger_pct: parseFloat(trailingTriggerPct),
      trailing_stop_pct: parseFloat(trailingStopPct),
      daily_max_loss_pct: parseFloat(dailyMaxLossPct),
      index_codes: indexCodes,
      claude_model: claudeModel,
      optimization_apply_mode: optimizationApplyMode,
      hard_watch_symbols: hardWatchSymbols,
      rank_lease_duration_min: parseInt(rankLeaseDurationMin),
      filter_rsi_max: parseFloat(filterRsiMax),
      filter_disparity_m5_max: parseFloat(filterDisparityM5Max),
      filter_high_price_diff_min: parseFloat(filterHighPriceDiffMin),
      filter_open_price_diff_max: parseFloat(filterOpenPriceDiffMax),
      index_drop_threshold_pct: parseFloat(indexDropThresholdPct),
      trading_days: tradingDays,
      hard_disparity_m5_min: parseFloat(hardDisparityM5Min),
      hard_disparity_m5_max: parseFloat(hardDisparityM5Max),
      hard_high_price_diff_max: parseFloat(hardHighPriceDiffMax),
      hard_high_price_diff_min: parseFloat(hardHighPriceDiffMin),
      hard_prev_vol_ratio_max: parseFloat(hardPrevVolRatioMax),
      hard_strength_min: parseFloat(hardStrengthMin),
      hard_rsi_max: parseFloat(hardRsiMax),
      hard_open_price_diff_max: parseFloat(hardOpenPriceDiffMax),
      hard_macd_bearish_enabled: hardMacdBearishEnabled,
      hard_high_formed_mins_max: parseFloat(hardHighFormedMinsMax),
      hard_vol_vs_3avg_ratio_min: parseFloat(hardVolVs3AvgRatioMin),
      hard_relative_strength_min: parseFloat(hardRelativeStrengthMin),
      adaptive_threshold_enabled: adaptiveThresholdEnabled,
      adaptive_threshold_trigger: parseInt(adaptiveThresholdTrigger),
      market_phase_relax_enabled: marketPhaseRelaxEnabled,
      market_phase_index_drop_trigger: parseFloat(marketPhaseIndexDropTrigger),
      market_phase_relax_pct: parseFloat(marketPhaseRelaxPct),
      escalation_enabled: escalationEnabled,
      escalation_trigger: parseInt(escalationTrigger),
      escalation_step_pct: parseFloat(escalationStepPct),
      escalation_max_stages: parseInt(escalationMaxStages),
      hard_rule_feedback_enabled: hardRuleFeedbackEnabled,
      hard_rule_feedback_window: parseInt(hardRuleFeedbackWindow),
      hard_rule_feedback_threshold_pct: parseFloat(hardRuleFeedbackThresholdPct),
      adaptive_relax_pct: parseFloat(adaptiveRelaxPct),
      vwap_diff_min: parseFloat(vwapDiffMin),
      vwap_diff_max: parseFloat(vwapDiffMax),
      rsi_buy_min: parseFloat(rsiBuyMin),
      rsi_buy_max: parseFloat(rsiBuyMax),
      bid_ask_ratio_min: parseFloat(bidAskRatioMin),
      min_market_cap: parseFloat(minMarketCap),
      min_expected_profit_pct: parseFloat(minExpectedProfitPct),
      max_claude_candidates: parseInt(maxClaudeCandidates, 10),
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
  const scoringTotal = [scoringBidAskWeight, scoringStrengthWeight, scoringMACDWeight, scoringRSIWeight, scoringVWAPWeight]
    .map(Number).reduce((a, b) => a + b, 0)
  const labelText = 'text-xs text-th-on-muted'

  return (
    <div className="space-y-4 pb-20">
      {/* ── 스티키 헤더 ── */}
      <div className="sticky top-14 md:top-0 z-30 glass-panel -mx-4 md:-mx-8 px-4 md:px-8 py-3 flex items-center justify-between">
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
          {activeTab !== 'INFO' && (
            <button
              type="submit"
              form="settings-form"
              disabled={saving}
              className="px-5 py-2 bg-orange-500 hover:bg-orange-600 disabled:opacity-50 rounded-lg text-sm font-semibold transition-colors text-white"
            >
              {saving ? '저장 중...' : '설정 저장'}
            </button>
          )}
        </div>
      </div>

      {/* ── 탭 바 ── */}
      <div className="flex bg-th-surface rounded-xl p-1 gap-1">
        {[
          { id: 'KR', label: '국장', badge: '국내', badgeCls: 'bg-blue-500/10 text-blue-400' },
          { id: 'INFO', label: 'AI / 서버', badge: null, badgeCls: '' },
        ].map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 flex items-center justify-center gap-1.5 py-2 px-2 rounded-lg text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'bg-th-surface-high text-th-on-surface shadow-sm'
                : 'text-th-on-muted hover:text-th-on-surface'
            }`}
          >
            {tab.label}
            {tab.badge && (
              <span className={`hidden sm:inline-flex items-center px-1.5 py-0.5 rounded-full text-[9px] font-medium ${tab.badgeCls}`}>
                {tab.badge}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* ── form: KR + US 탭을 모두 포함 (hidden 으로 전환, 상태 유지) ── */}
      <form id="settings-form" onSubmit={handleSave}>

        {/* ════════════════ KR 탭 ════════════════ */}
        <div className={activeTab !== 'KR' ? 'hidden' : 'space-y-5'}>

          <PresetPanel
            market="KR"
            presets={presets}
            activePresetId={activePresetId}
            nameVal={krPresetName} descVal={krPresetDesc}
            setName={setKrPresetName} setDesc={setKrPresetDesc}
            presetSaving={presetSaving} presetApplying={presetApplying} presetMsg={presetMsg}
            handleSavePreset={handleSavePreset} handleApplyPreset={handleApplyPreset}
            handleDeletePreset={handleDeletePreset}
          />

          {/* ── 섹션 1: 거래 제어 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>거래 제어</p>

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
                <span className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${tradingEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
              </button>
            </div>
            <p className="text-xs text-center font-semibold">
              {tradingEnabled
                ? <span className="text-emerald-400">거래 활성화 (ON)</span>
                : <span className="text-th-on-muted">거래 비활성화 (OFF)</span>
              }
            </p>

            <div className={`grid grid-cols-2 gap-4 ${divider}`}>
              <label className="space-y-1">
                <span className={labelText}>거래 시작 시간</span>
                <input type="time" step="60" value={tradingStartTime} onChange={(e) => setTradingStartTime(e.target.value)} className={inputCls} />
              </label>
              <label className="space-y-1">
                <span className={labelText}>거래 종료 시간</span>
                <input type="time" step="60" value={tradingEndTime} onChange={(e) => setTradingEndTime(e.target.value)} className={inputCls} />
              </label>
            </div>
            <p className={hintText}>기본값: 09:15 ~ 15:15. 변경 시 다음 거래일부터 적용됩니다.</p>

            <div className={`grid grid-cols-2 gap-4 ${divider}`}>
              <label className="space-y-1">
                <span className={labelText}>매수 중단 시작</span>
                <input type="time" step="60" value={buyPauseStart} onChange={(e) => setBuyPauseStart(e.target.value)} className={inputCls} />
              </label>
              <label className="space-y-1">
                <span className={labelText}>매수 중단 종료</span>
                <input type="time" step="60" value={buyPauseEnd} onChange={(e) => setBuyPauseEnd(e.target.value)} className={inputCls} />
              </label>
            </div>
            <p className={hintText}>기본값: 11:00 ~ 14:00 매수 중단 (점심시간 유동성 저하 방지)</p>

            <div className={`space-y-2 ${divider}`}>
              <span className={`${labelText} block`}>지수 필터</span>
              <p className={hintText}>체크된 지수가 시가 대비 하락 임계값 이하로 하락 시 매수를 중단합니다.</p>
              <div className="flex gap-4">
                {[{ code: '0001', label: '코스피' }, { code: '1001', label: '코스닥' }].map(({ code, label }) => (
                  <label key={code} className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={indexCodes.includes(code)}
                      onChange={(e) => setIndexCodes(prev => e.target.checked ? [...prev, code] : prev.filter(c => c !== code))}
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

            <div className={`space-y-2 ${divider}`}>
              <span className={`${labelText} block`}>거래 요일</span>
              <p className={hintText}>체크된 요일에만 자동매매가 실행됩니다.</p>
              <div className="flex flex-wrap gap-3">
                {[{ day: 1, label: '월' }, { day: 2, label: '화' }, { day: 3, label: '수' }, { day: 4, label: '목' }, { day: 5, label: '금' }, { day: 6, label: '토' }, { day: 0, label: '일' }].map(({ day, label }) => (
                  <label key={day} className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={tradingDays.includes(day)}
                      onChange={(e) => setTradingDays((prev) => e.target.checked ? [...prev, day] : prev.filter((d) => d !== day))}
                      className="accent-orange-500"
                    />
                    <span className="text-sm text-th-on-surface">{label}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>

          {/* ── 섹션 2: 종목 선정 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>종목 선정 (순위 조회)</p>

            <div className="space-y-2">
              <p className={labelText}>순위조회 제외 종목</p>
              <p className={hintText}>체크된 항목은 순위조회 결과에서 제외됩니다</p>
              <div className="grid grid-cols-2 gap-2">
                {EXCL_LABELS.map((label, i) => (
                  <label key={i} className="flex items-center gap-2 cursor-pointer group">
                    <input type="checkbox" checked={exclBits[i]} onChange={() => toggleBit(i)}
                      className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                    <span className="text-sm text-th-on-surface transition-colors">{label}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className={`space-y-3 ${divider}`}>
              <div className="grid grid-cols-2 gap-4">
                <label className="space-y-1">
                  <span className={labelText}>최소 주가 (원)</span>
                  <input type="number" step="1000" min="0" value={rankingPriceMin} onChange={(e) => setRankingPriceMin(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>최대 주가 (원)</span>
                  <input type="number" step="1000" min="0" value={rankingPriceMax} onChange={(e) => setRankingPriceMax(e.target.value)} className={inputCls} />
                </label>
              </div>
              <label className="space-y-1">
                <span className={labelText}>각 순위별 상위 종목 수 (최대 30)</span>
                <input type="number" step="1" min="1" max="30" value={rankingTopN} onChange={(e) => setRankingTopN(e.target.value)}
                  className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
              </label>
            </div>

            <div className={`space-y-3 ${divider}`}>
              <div className="space-y-2">
                <p className={labelText}>조회 거래소 (복수 선택 시 합산)</p>
                <p className="text-xs text-th-on-muted">선택한 거래소별로 각각 순위를 조회한 후 종목을 합산합니다</p>
                <div className="flex flex-wrap gap-4">
                  {[
                    { code: '0001', label: 'KOSPI (거래소)' },
                    { code: '1001', label: 'KOSDAQ (코스닥)' },
                  ].map(({ code, label }) => (
                    <label key={code} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={rankingExchanges.includes(code)}
                        onChange={(e) =>
                          setRankingExchanges((prev) =>
                            e.target.checked ? [...prev, code] : prev.filter((c) => c !== code)
                          )
                        }
                        className="w-4 h-4 rounded bg-th-surface-high accent-orange-500"
                      />
                      <span className="text-sm text-th-on-surface">{label}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div className={`space-y-3 ${divider}`}>
              {/* 거래량 순위 */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <input type="checkbox" checked={rankingTypes.includes('volume')} onChange={() => toggleRankingType('volume')}
                    className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                  <span className="text-sm text-th-on-surface font-medium">거래량 순위</span>
                </div>
                {rankingTypes.includes('volume') && (
                  <div className="ml-6 space-y-3">
                    <div className="space-y-1">
                      <p className={labelText}>거래량 분류 코드 (복수 선택 시 합산)</p>
                      <div className="flex flex-wrap gap-3">
                        {[
                          { code: '1', label: '거래량증가율' },
                          { code: '3', label: '거래대금순' },
                        ].map(({ code, label }) => (
                          <label key={code} className="flex items-center gap-2 cursor-pointer">
                            <input
                              type="checkbox"
                              checked={rankingVolumeBlngClsCodes.includes(code)}
                              onChange={(e) =>
                                setRankingVolumeBlngClsCodes((prev) =>
                                  e.target.checked ? [...prev, code] : prev.filter((c) => c !== code)
                                )
                              }
                              className="w-4 h-4 rounded bg-th-surface-high accent-orange-500"
                            />
                            <span className="text-sm text-th-on-surface">{label}</span>
                          </label>
                        ))}
                      </div>
                    </div>
                    <label className="space-y-1">
                      <span className={labelText}>전일대비 거래량 증가율 최솟값 (%, 0=필터없음)</span>
                      <input type="number" step="10" min="0" value={volumeMinIncrRate} onChange={(e) => setVolumeMinIncrRate(e.target.value)}
                        className="w-40 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
                    </label>
                  </div>
                )}
              </div>
              {/* 체결강도 순위 */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <input type="checkbox" checked={rankingTypes.includes('strength')} onChange={() => toggleRankingType('strength')}
                    className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                  <span className="text-sm text-th-on-surface font-medium">체결강도 순위</span>
                </div>
                {rankingTypes.includes('strength') && (
                  <div className="ml-6">
                    <label className="space-y-1">
                      <span className={labelText}>최소 체결강도 (%, 100=매수우세 이상, 0=필터없음)</span>
                      <input type="number" step="5" min="0" value={strengthMin} onChange={(e) => setStrengthMin(e.target.value)}
                        className="w-40 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
                    </label>
                  </div>
                )}
              </div>
              {/* 등락률 순위 */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <input type="checkbox" checked={rankingTypes.includes('fluctuation')} onChange={() => toggleRankingType('fluctuation')}
                    className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                  <span className="text-sm text-th-on-surface font-medium">등락률 순위</span>
                </div>
                {rankingTypes.includes('fluctuation') && (
                  <div className="ml-6 flex flex-wrap gap-4">
                    <label className="space-y-1">
                      <span className={labelText}>최소 등락률 (%, 0=필터없음)</span>
                      <input type="number" step="0.5" min="0" value={fluctuationMinRate} onChange={(e) => setFluctuationMinRate(e.target.value)}
                        className="w-32 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
                    </label>
                    <label className="space-y-1">
                      <span className={labelText}>최대 등락률 (%, 0=제한없음)</span>
                      <input type="number" step="0.5" min="0" value={fluctuationMaxRate} onChange={(e) => setFluctuationMaxRate(e.target.value)}
                        className="w-32 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
                    </label>
                  </div>
                )}
              </div>
              {/* VI 발동현황 */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <input type="checkbox" checked={rankingTypes.includes('vi_status')} onChange={() => toggleRankingType('vi_status')}
                    className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                  <span className="text-sm text-th-on-surface font-medium">VI 발동현황 (해제 직후 반등)</span>
                </div>
                {rankingTypes.includes('vi_status') && (
                  <div className="ml-6">
                    <div className="flex gap-1.5">
                      {[{ value: '', label: '전체' }, { value: '1', label: '정적 VI만' }, { value: '2', label: '동적 VI만' }].map(({ value, label }) => (
                        <button key={value} type="button" onClick={() => setViKindCode(value)}
                          className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors border ${
                            viKindCode === value
                              ? 'bg-th-surface-high text-th-on-surface border-black/10 dark:border-white/10 ring-1 ring-zinc-600'
                              : 'bg-transparent text-th-on-muted border-black/10 dark:border-white/10 hover:text-th-on-surface'
                          }`}>
                          {label}
                        </button>
                      ))}
                    </div>
                    <p className={`mt-1 ${hintText}`}>정적 VI: 가격 급등락 발동 / 동적 VI: 단시간 속도 급변 발동</p>
                  </div>
                )}
              </div>
            </div>

            <div className={`space-y-2 ${divider}`}>
              <p className={labelText}>순위 조건</p>
              <p className={hintText}>AND: 모든 선택 순위에 공통으로 포함된 종목만 / OR: 하나 이상의 순위에 포함된 종목 모두</p>
              <div className="flex gap-2">
                {['AND', 'OR'].map((cond) => (
                  <button key={cond} type="button" onClick={() => setRankingCondition(cond)}
                    className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-colors border ${
                      rankingCondition === cond
                        ? 'bg-th-surface-high text-th-on-surface border-black/10 dark:border-white/10 ring-1 ring-zinc-600'
                        : 'bg-transparent text-th-on-muted border-black/10 dark:border-white/10 hover:text-th-on-surface'
                    }`}>
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
                <input type="number" step="1" min="1" max="10" value={maxPositions} onChange={(e) => setMaxPositions(e.target.value)} className={inputCls} />
              </label>
              <label className="space-y-1">
                <span className={labelText}>주문 금액 비율 (%)</span>
                <input type="number" step="1" min="1" max="100" value={orderAmountPct} onChange={(e) => setOrderAmountPct(e.target.value)} className={inputCls} />
              </label>
            </div>
          </div>

          {/* ── 섹션 4: 매도 설정 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>매도 설정</p>

            <p className={hintText}>기본값: 종목 유형 구분 없이 적용됩니다. ETF/주식 전용 값을 설정하면 해당 값이 우선 적용됩니다.</p>
            <div className="grid grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className="text-xs text-red-400">익절 기준 — 기본값 (%)</span>
                <input type="number" step="0.1" min="0.1" value={takeProfitPct} onChange={(e) => setTakeProfitPct(e.target.value)} className={inputCls} />
              </label>
              <label className="space-y-1">
                <span className="text-xs text-[#3B82F6]">손절 기준 — 기본값 (%)</span>
                <input type="number" step="0.1" min="0.1" value={stopLossPct} onChange={(e) => setStopLossPct(e.target.value)} className={inputCls} />
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">ETF 전용 수익/손절</p>
              <p className={hintText}>국내주식형 ETF 및 일반 ETF에 적용 (비과세/저세율 감안하여 낮게 설정 권장)</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className="text-xs text-red-400">ETF 익절 기준 (%)</span>
                <input type="number" step="0.1" min="0.1" value={etfTakeProfitPct} onChange={(e) => setEtfTakeProfitPct(e.target.value)} className={inputCls} />
              </label>
              <label className="space-y-1">
                <span className="text-xs text-[#3B82F6]">ETF 손절 기준 (%)</span>
                <input type="number" step="0.1" min="0.1" value={etfStopLossPct} onChange={(e) => setEtfStopLossPct(e.target.value)} className={inputCls} />
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">주식 전용 수익/손절</p>
              <p className={hintText}>일반 주식에 적용 (거래세 0.2% 감안하여 ETF보다 높게 설정 권장)</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className="text-xs text-red-400">주식 익절 기준 (%)</span>
                <input type="number" step="0.1" min="0.1" value={stockTakeProfitPct} onChange={(e) => setStockTakeProfitPct(e.target.value)} className={inputCls} />
              </label>
              <label className="space-y-1">
                <span className="text-xs text-[#3B82F6]">주식 손절 기준 (%)</span>
                <input type="number" step="0.1" min="0.1" value={stockStopLossPct} onChange={(e) => setStockStopLossPct(e.target.value)} className={inputCls} />
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <label className="space-y-1 block">
                <span className={labelText}>주식 세후 최소 기대수익 (%, 0=미사용)</span>
                <input type="number" step="0.1" min="0" value={minExpectedProfitPct} onChange={(e) => setMinExpectedProfitPct(e.target.value)} className={inputCls} />
              </label>
              <p className={hintText}>주식 진입 시 거래세(0.2%) 차감 후 이 수익률 이상 기대 안 되면 Claude가 거절. 0=미사용. 권장: 0.8</p>
            </div>

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

            <div className={`space-y-3 ${divider}`}>
              <p className={labelText}>지표 확인 설정</p>
              <div className="grid grid-cols-2 gap-4">
                <label className="space-y-1">
                  <span className={labelText}>지표 확인 주기 (분)</span>
                  <input type="number" step="1" min="1" value={indicatorIntervalMin} onChange={(e) => setIndicatorIntervalMin(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>RSI 매도 기준값</span>
                  <input type="number" step="1" min="50" max="100" value={rsiThreshold} onChange={(e) => setRsiThreshold(e.target.value)} className={inputCls} />
                </label>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={macdBearish} onChange={(e) => setMacdBearish(e.target.checked)}
                  className="w-4 h-4 rounded bg-th-surface-high accent-orange-500" />
                <span className="text-sm text-th-on-surface">MACD 데드크로스 시 매도</span>
              </label>
            </div>

            <div className={`space-y-3 ${divider}`}>
              <p className={labelText}>트레일링 스탑</p>
              <p className={hintText}>수익률이 활성화 기준에 도달하면, 이후 최고가 대비 일정 % 하락 시 자동 매도합니다. 활성화 기준 0=비활성.</p>
              <div className="grid grid-cols-2 gap-4">
                <label className="space-y-1">
                  <span className={labelText}>활성화 기준 수익률 (%, 0=비활성)</span>
                  <input type="number" step="0.1" min="0" value={trailingTriggerPct} onChange={(e) => setTrailingTriggerPct(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>최고가 대비 하락 허용폭 (%)</span>
                  <input type="number" step="0.1" min="0.1" value={trailingStopPct} onChange={(e) => setTrailingStopPct(e.target.value)} className={inputCls} />
                </label>
              </div>
            </div>

            <div className={`space-y-2 ${divider}`}>
              <p className={labelText}>일일 최대 손실 한도</p>
              <label className="space-y-1 block">
                <span className={labelText}>총자산 대비 최대 손실 (%, 0=제한없음)</span>
                <input type="number" step="0.1" min="0" value={dailyMaxLossPct} onChange={(e) => setDailyMaxLossPct(e.target.value)}
                  className="w-40 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
              </label>
              <p className={hintText}>당일 실현 손실이 한도 초과 시 매수를 중단합니다.</p>
            </div>

            {stagnationActive && (
              <div className={`space-y-3 ${divider}`}>
                <p className={labelText}>횡보 감지 설정</p>
                <p className={hintText}>진입가 기준 ±N% 이내 변동이 M분 이상 지속되면 매도합니다.</p>
                <div className="grid grid-cols-2 gap-4">
                  <label className="space-y-1">
                    <span className={labelText}>횡보 기준 변동폭 (%)</span>
                    <input type="number" step="0.1" min="0.1" value={stagnationThresholdPct} onChange={(e) => setStagnationThresholdPct(e.target.value)} className={inputCls} />
                  </label>
                  <label className="space-y-1">
                    <span className={labelText}>횡보 지속 기준 (분)</span>
                    <input type="number" step="5" min="5" value={stagnationDurationMin} onChange={(e) => setStagnationDurationMin(e.target.value)} className={inputCls} />
                  </label>
                </div>
                <div className="space-y-2 pt-1 border-t border-th-border/40">
                  <p className={`${labelText} font-medium`}>단계적 횡보 청산</p>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={stagnationPartialExitEnabled} onChange={(e) => setStagnationPartialExitEnabled(e.target.checked)}
                      className="w-4 h-4 rounded accent-orange-500" />
                    <span className={labelText}>활성화 (1차 횡보→절반 청산, 2차 횡보→전량 청산)</span>
                  </label>
                  {stagnationPartialExitEnabled && (
                    <label className="space-y-1 block">
                      <span className={labelText}>즉시 전량청산 bid_ask 임계값 (이 값 미만이면 매도우세로 즉시 전량 청산)</span>
                      <input type="number" step="0.1" min="0.1" value={stagnationBidAskSellThreshold}
                        onChange={(e) => setStagnationBidAskSellThreshold(e.target.value)} className={inputCls} />
                      <p className={hintText}>기본 1.0 — bid_ask_ratio가 이 값 미만이면 절반 청산 없이 즉시 전량 청산</p>
                    </label>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* ── 섹션 5b: 부분 익절 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>부분 익절 (Partial Take Profit)</p>
            <p className={hintText}>중간 목표가 도달 시 포지션 일부를 익절하여 수익을 확정합니다. 나머지는 최종 목표가까지 보유합니다.</p>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={partialTPEnabled}
                onChange={(e) => setPartialTPEnabled(e.target.checked)}
                className="w-4 h-4 rounded accent-orange-500" />
              <span className={labelText}>부분 익절 활성화</span>
            </label>
            {partialTPEnabled && (
              <div className={`space-y-3 pt-2 border-t border-th-border/40`}>
                <div className="grid grid-cols-2 gap-4">
                  <label className="space-y-1">
                    <span className={labelText}>중간 익절 기준 수익률 (%)</span>
                    <input type="number" step="0.1" min="0.1" value={partialTPPct}
                      onChange={(e) => setPartialTPPct(e.target.value)} className={inputCls} />
                    <p className={hintText}>기본 1.0 — 이 수익률 도달 시 부분 매도</p>
                  </label>
                  <label className="space-y-1">
                    <span className={labelText}>매도 비율 (0~1)</span>
                    <input type="number" step="0.05" min="0.05" max="0.95" value={partialTPRatio}
                      onChange={(e) => setPartialTPRatio(e.target.value)} className={inputCls} />
                    <p className={hintText}>기본 0.5 = 보유 수량의 50% 매도</p>
                  </label>
                </div>
                <label className="flex items-center gap-2 cursor-pointer pt-1 border-t border-th-border/40">
                  <input type="checkbox" checked={partialTPRaiseStop}
                    onChange={(e) => setPartialTPRaiseStop(e.target.checked)}
                    className="w-4 h-4 rounded accent-orange-500" />
                  <span className={labelText}>부분 익절 후 손절가를 진입가(BEP)로 올리기</span>
                </label>
                <p className={hintText}>활성화하면 부분 익절 후 손절가가 매수가로 상향되어 잔여 포지션 원금을 보호합니다.</p>
              </div>
            )}
          </div>

          {/* ── 섹션 5c: 하드 필터 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>하드 필터 (매수 품질)</p>
            <p className={hintText}>LLM 호출 전 자동으로 제거되는 조건입니다.</p>

            <div className={`space-y-2 ${divider}`}>
              <p className={`${labelText} font-medium`}>스코어링 가중치</p>
              <p className={hintText}>각 지표의 가중치를 조정합니다. 합계: {scoringTotal}pt (기본 합계=100)</p>
              <div className="grid grid-cols-2 gap-3">
                <label className="space-y-1">
                  <span className={labelText}>매수호가 우세 (BidAsk)</span>
                  <input type="number" step="1" min="0" value={scoringBidAskWeight}
                    onChange={(e) => setScoringBidAskWeight(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>체결강도 (Strength)</span>
                  <input type="number" step="1" min="0" value={scoringStrengthWeight}
                    onChange={(e) => setScoringStrengthWeight(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>MACD 방향성</span>
                  <input type="number" step="1" min="0" value={scoringMACDWeight}
                    onChange={(e) => setScoringMACDWeight(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>RSI 구간 [40–60]</span>
                  <input type="number" step="1" min="0" value={scoringRSIWeight}
                    onChange={(e) => setScoringRSIWeight(e.target.value)} className={inputCls} />
                </label>
                <label className="space-y-1">
                  <span className={labelText}>VWAP 이격도 [0–1.5%]</span>
                  <input type="number" step="1" min="0" value={scoringVWAPWeight}
                    onChange={(e) => setScoringVWAPWeight(e.target.value)} className={inputCls} />
                </label>
              </div>
              <p className={hintText}>가중치 합계가 100이면 기존과 동일 스케일(0~100pt). 합계와 momentum_score_min을 맞춰 설정하세요.</p>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <label className="space-y-1 block">
                <span className={labelText}>복합 모멘텀 스코어 최솟값 (0~합계pt, 0=비활성)</span>
                <input type="number" step="5" min="0" value={momentumScoreMin}
                  onChange={(e) => setMomentumScoreMin(e.target.value)} className={inputCls} />
              </label>
              <p className={hintText}>
                bid_ask({scoringBidAskWeight}pt) + 체결강도({scoringStrengthWeight}pt) + MACD({scoringMACDWeight}pt) + RSI({scoringRSIWeight}pt) + VWAP({scoringVWAPWeight}pt) = 최대 {scoringTotal}pt. 미달 종목은 Claude 전달 전 제거.
              </p>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <label className="space-y-1 block">
                <span className={labelText}>최소 거래대금 (원, 0=필터없음)</span>
                <input type="number" step="any" min="0" value={minTradingValue} onChange={(e) => setMinTradingValue(e.target.value)} className={inputCls} />
              </label>
              <p className={hintText}>예: 5000000000 = 50억원. 거래대금 미달 종목은 LLM 후보에서 제외됩니다.</p>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <label className="space-y-1 block">
                <span className={labelText}>최소 시가총액 (억원, 0=필터없음)</span>
                <input type="number" step="100" min="0" value={minMarketCap} onChange={(e) => setMinMarketCap(e.target.value)} className={inputCls} />
              </label>
              <p className={hintText}>잡주 1차 필터. MST 상장주식수 × 현재가 기준. 권장값: 500~1000억. MST 재다운로드 후 적용됩니다.</p>
            </div>

            <div className="grid md:grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className={labelText}>RSI 과열 임계값 (이상 제외)</span>
                <input type="number" step="1" min="50" max="100" value={filterRsiMax} onChange={(e) => setFilterRsiMax(e.target.value)} className={inputCls} />
                <p className={hintText}>기본 80 — RSI ≥ 이 값인 종목 제외</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>5분봉 이격도 최대값 (%) (초과 제외)</span>
                <input type="number" step="0.1" min="0" value={filterDisparityM5Max} onChange={(e) => setFilterDisparityM5Max(e.target.value)} className={inputCls} />
                <p className={hintText}>기본 3.0 — 5분봉 MA5 이격도 초과 시 제외</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>고가 대비 최솟값 (%) (미만 제외)</span>
                <input type="number" step="0.1" value={filterHighPriceDiffMin} onChange={(e) => setFilterHighPriceDiffMin(e.target.value)} className={inputCls} />
                <p className={hintText}>기본 -5.0 — 고가 대비 하락 폭이 이 값 미만인 종목 제외</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>시가 대비 최댓값 (%) (초과 제외)</span>
                <input type="number" step="0.1" min="0" value={filterOpenPriceDiffMax} onChange={(e) => setFilterOpenPriceDiffMax(e.target.value)} className={inputCls} />
                <p className={hintText}>기본 20.0 — 당일 상한가 영역 종목 제외</p>
              </label>
            </div>
          </div>

          {/* ── 섹션 5c: AI 매매 기준값 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>AI 매매 기준값</p>
            <p className={hintText}>Claude에게 전달되는 하드 리젝션 룰과 랭킹 기준 수치입니다. 변경 시 즉시 다음 종목 선정에 반영됩니다.</p>

            <div className="space-y-1">
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">하드 리젝션 룰 (ANY 해당 시 제외)</p>
            </div>
            <div className={`grid md:grid-cols-2 gap-4 ${divider}`}>
              <label className="space-y-1">
                <span className={labelText}>5분봉 이격도 하한 (%)</span>
                <input type="number" step="0.1" value={hardDisparityM5Min} onChange={(e) => setHardDisparityM5Min(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이하 → 칼날 하락 구간 (기본 -1.5)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>5분봉 이격도 상한 (%)</span>
                <input type="number" step="0.1" value={hardDisparityM5Max} onChange={(e) => setHardDisparityM5Max(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이상 → 과열 구간 (기본 3.0)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>고점 대비 상한 (%)</span>
                <input type="number" step="0.1" value={hardHighPriceDiffMax} onChange={(e) => setHardHighPriceDiffMax(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이상 → 고점 추격 위험 (기본 -0.5)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>고점 대비 하한 (%)</span>
                <input type="number" step="0.1" value={hardHighPriceDiffMin} onChange={(e) => setHardHighPriceDiffMin(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이하 + 거래량 급증 → 추세이탈 (기본 -5.0)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>하락 시 거래량 비율 상한</span>
                <input type="number" step="0.1" min="0" value={hardPrevVolRatioMax} onChange={(e) => setHardPrevVolRatioMax(e.target.value)} className={inputCls} />
                <p className={hintText}>하락 중 전 캔들 대비 거래량 비율 (기본 1.2)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>최소 체결강도</span>
                <input type="number" step="1" min="0" value={hardStrengthMin} onChange={(e) => setHardStrengthMin(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이하 → 매수세 소멸 (기본 100)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>RSI 과매수 상한</span>
                <input type="number" step="1" min="50" max="100" value={hardRsiMax} onChange={(e) => setHardRsiMax(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이상 → 과매수에서 꺾임 (기본 70)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>시가 대비 상승률 상한 (%)</span>
                <input type="number" step="0.5" min="0" value={hardOpenPriceDiffMax} onChange={(e) => setHardOpenPriceDiffMax(e.target.value)} className={inputCls} />
                <p className={hintText}>이 값 이상 → 상한가 영역 (기본 15)</p>
              </label>
              <label className="flex items-center gap-3 cursor-pointer py-2">
                <input type="checkbox" checked={hardMacdBearishEnabled} onChange={(e) => setHardMacdBearishEnabled(e.target.checked)} className="w-4 h-4 accent-th-primary" />
                <span className={labelText}>MACD 베어리시 진입 차단</span>
                <p className={`${hintText} ml-auto`}>MACD 선 &lt; 시그널 선이면 종목 제외</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>고점 경과 시간 상한 (분)</span>
                <input type="number" step="1" min="0" value={hardHighFormedMinsMax} onChange={(e) => setHardHighFormedMinsMax(e.target.value)} className={inputCls} />
                <p className={hintText}>0 = 비활성. 고점 형성 후 이 시간 초과 시 제외 (예: 60)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>거래량 회복 비율 최솟값</span>
                <input type="number" step="0.1" min="0" value={hardVolVs3AvgRatioMin} onChange={(e) => setHardVolVs3AvgRatioMin(e.target.value)} className={inputCls} />
                <p className={hintText}>0 = 비활성. 현재봉/직전 3봉 평균 거래량. 이 미만이면 제외 (예: 0.5)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>시장 대비 상대강도 최솟값 (%)</span>
                <input type="number" step="0.5" value={hardRelativeStrengthMin} onChange={(e) => setHardRelativeStrengthMin(e.target.value)} className={inputCls} />
                <p className={hintText}>0 = 비활성. 종목 등락률 − 시장 등락률. 이 미만이면 제외 (예: -2.0)</p>
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">Adaptive Threshold (Hard Rule 자동 완화)</p>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <label className="flex items-center gap-3 space-y-0">
                <input type="checkbox" checked={adaptiveThresholdEnabled} onChange={(e) => setAdaptiveThresholdEnabled(e.target.checked)} className="w-4 h-4" />
                <div>
                  <span className={labelText}>Hard Rule 자동 완화 활성</span>
                  <p className={hintText}>N회 연속 종목 선정 실패 시 hard rule을 일시 완화하고, 거래 성사 후 자동 복원</p>
                </div>
              </label>
              <label className="space-y-1">
                <span className={labelText}>발동 연속 실패 횟수</span>
                <input type="number" step="1" min="1" value={adaptiveThresholdTrigger} onChange={(e) => setAdaptiveThresholdTrigger(e.target.value)} className={inputCls} />
                <p className={hintText}>이 횟수 이상 연속 실패 시 완화 시작 (기본 10)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>완화 비율 (%)</span>
                <input type="number" step="5" min="5" max="50" value={adaptiveRelaxPct} onChange={(e) => setAdaptiveRelaxPct(e.target.value)} className={inputCls} />
                <p className={hintText}>hard rule 임계값을 이 비율만큼 완화 (기본 20%)</p>
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">Market Phase Detection (시장 국면 감지 완화)</p>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <label className="flex items-center gap-3 space-y-0">
                <input type="checkbox" checked={marketPhaseRelaxEnabled} onChange={(e) => setMarketPhaseRelaxEnabled(e.target.checked)} className="w-4 h-4" />
                <div>
                  <span className={labelText}>약세장 감지 시 Hard Rule 자동 완화</span>
                  <p className={hintText}>지수 전일 대비 하락률이 기준 이하이면 hard rule을 완화 (강세장에서는 엄격한 기준 유지)</p>
                </div>
              </label>
              <label className="space-y-1">
                <span className={labelText}>전일 대비 하락률 기준 (%)</span>
                <input type="number" step="0.5" value={marketPhaseIndexDropTrigger} onChange={(e) => setMarketPhaseIndexDropTrigger(e.target.value)} className={inputCls} />
                <p className={hintText}>KOSPI/KOSDAQ 중 어느 하나라도 이 이하이면 약세장 판정 (기본 -1.0%)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>완화 비율 (%)</span>
                <input type="number" step="5" min="5" max="50" value={marketPhaseRelaxPct} onChange={(e) => setMarketPhaseRelaxPct(e.target.value)} className={inputCls} />
                <p className={hintText}>약세장 판정 시 hard rule 임계값을 이 비율만큼 완화 (기본 15%)</p>
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">Hard Rule Escalation (단계적 자동 완화)</p>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <label className="flex items-center gap-3 space-y-0">
                <input type="checkbox" checked={escalationEnabled} onChange={(e) => setEscalationEnabled(e.target.checked)} className="w-4 h-4" />
                <div>
                  <span className={labelText}>단계적 완화 활성</span>
                  <p className={hintText}>연속 실패 횟수에 비례해 단계적으로 hard rule을 완화. AdaptiveThreshold와 동시 활성 시 더 큰 완화 비율(max-wins)이 적용됩니다.</p>
                </div>
              </label>
              <label className="space-y-1">
                <span className={labelText}>단계당 실패 횟수</span>
                <input type="number" step="1" min="1" value={escalationTrigger} onChange={(e) => setEscalationTrigger(e.target.value)} className={inputCls} />
                <p className={hintText}>이 횟수마다 1단계씩 상승 (기본 20). 예: 20회=1단계, 40회=2단계</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>단계당 완화 비율 (%)</span>
                <input type="number" step="5" min="5" max="50" value={escalationStepPct} onChange={(e) => setEscalationStepPct(e.target.value)} className={inputCls} />
                <p className={hintText}>1단계당 hard rule 임계값 완화 비율 (기본 10%). 5단계 시 최대 50% 완화</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>최대 단계</span>
                <input type="number" step="1" min="1" max="10" value={escalationMaxStages} onChange={(e) => setEscalationMaxStages(e.target.value)} className={inputCls} />
                <p className={hintText}>완화 상한 단계 (기본 5). 이 단계 이상은 더 이상 완화하지 않음</p>
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">룰별 자동 완화 (Hard Rule Feedback)</p>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <label className="flex items-center gap-3 space-y-0">
                <input type="checkbox" checked={hardRuleFeedbackEnabled} onChange={(e) => setHardRuleFeedbackEnabled(e.target.checked)} className="w-4 h-4" />
                <div>
                  <span className={labelText}>룰별 자동 완화 활성</span>
                  <p className={hintText}>특정 Hard Rule이 최근 N사이클에서 임계 비율 이상 후보를 차단하면 해당 룰의 임계값을 단계적으로 완화합니다.</p>
                </div>
              </label>
              <label className="space-y-1">
                <span className={labelText}>분석 사이클 수 (Window)</span>
                <input type="number" step="1" min="3" max="20" value={hardRuleFeedbackWindow} onChange={(e) => setHardRuleFeedbackWindow(e.target.value)} className={inputCls} />
                <p className={hintText}>최근 몇 사이클을 분석할지 (기본 10). 클수록 완화 기준이 엄격해집니다.</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>완화 발동 임계 비율 (%)</span>
                <input type="number" step="5" min="50" max="100" value={hardRuleFeedbackThresholdPct} onChange={(e) => setHardRuleFeedbackThresholdPct(e.target.value)} className={inputCls} />
                <p className={hintText}>Window 내 이 비율 이상 사이클에서 차단 발생 시 완화 (기본 70%)</p>
              </label>
            </div>

            <div className={`space-y-1 ${divider}`}>
              <p className="text-xs font-semibold text-th-on-muted uppercase tracking-widest">랭킹 우선 기준 (선호 구간)</p>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <label className="space-y-1">
                <span className={labelText}>VWAP 이격도 하한 (%)</span>
                <input type="number" step="0.1" value={vwapDiffMin} onChange={(e) => setVwapDiffMin(e.target.value)} className={inputCls} />
                <p className={hintText}>VWAP 지지선 위에서 매수 (기본 0.0)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>VWAP 이격도 상한 (%)</span>
                <input type="number" step="0.1" value={vwapDiffMax} onChange={(e) => setVwapDiffMax(e.target.value)} className={inputCls} />
                <p className={hintText}>VWAP 과리 제외 (기본 1.5)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>RSI 매수 구간 하한</span>
                <input type="number" step="1" min="0" max="100" value={rsiBuyMin} onChange={(e) => setRsiBuyMin(e.target.value)} className={inputCls} />
                <p className={hintText}>이상적 RSI 매수 구간 (기본 40)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>RSI 매수 구간 상한</span>
                <input type="number" step="1" min="0" max="100" value={rsiBuyMax} onChange={(e) => setRsiBuyMax(e.target.value)} className={inputCls} />
                <p className={hintText}>이상적 RSI 매수 구간 (기본 60)</p>
              </label>
              <label className="space-y-1">
                <span className={labelText}>최소 매수호가 우세 비율</span>
                <input type="number" step="0.1" min="0" value={bidAskRatioMin} onChange={(e) => setBidAskRatioMin(e.target.value)} className={inputCls} />
                <p className={hintText}>매수잔량 / 매도잔량 최소 비율 (기본 1.2)</p>
              </label>
            </div>
          </div>

          {/* ── 하드 감시 종목 ── */}
          <div className={sectionCls}>
            <p className={sectionTitle}>하드 감시 종목</p>
            <p className={hintText}>순위 API 결과와 무관하게 항상 선정 후보에 포함할 종목입니다. 종목 코드 6자리를 직접 입력하거나 종목 목록 페이지에서 등록할 수 있습니다.</p>

            <div className={`space-y-2 ${divider}`}>
              <label className="space-y-1">
                <span className={labelText}>순위 유지 시간 (분)</span>
                <input type="number" step="1" min="1" max="60" value={rankLeaseDurationMin} onChange={(e) => setRankLeaseDurationMin(e.target.value)}
                  className="w-28 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50" />
              </label>
              <p className={hintText}>순위에서 사라진 종목도 이 시간(분) 동안 후보로 유지합니다.</p>
            </div>

            <div className={`space-y-2 ${divider}`}>
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="종목코드 (예: 005930)"
                  value={hardWatchInput}
                  onChange={(e) => setHardWatchInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      const code = hardWatchInput.trim().replace(/\D/g, '').slice(0, 6)
                      if (code.length === 6 && !hardWatchSymbols.includes(code)) {
                        setHardWatchSymbols(prev => [...prev, code])
                      }
                      setHardWatchInput('')
                    }
                  }}
                  className="flex-1 px-3 py-1.5 bg-th-surface-high rounded-lg text-sm text-th-on-surface focus:outline-none focus:ring-1 focus:ring-orange-500/50"
                />
                <button
                  type="button"
                  onClick={() => {
                    const code = hardWatchInput.trim().replace(/\D/g, '').slice(0, 6)
                    if (code.length === 6 && !hardWatchSymbols.includes(code)) {
                      setHardWatchSymbols(prev => [...prev, code])
                    }
                    setHardWatchInput('')
                  }}
                  className="px-3 py-1.5 bg-orange-500/15 text-orange-400 hover:bg-orange-500/25 rounded-lg text-sm font-medium transition-colors"
                >추가</button>
              </div>
              {hardWatchSymbols.length === 0 ? (
                <p className="text-xs text-th-on-subtle">등록된 하드 감시 종목이 없습니다.</p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {hardWatchSymbols.map((code) => (
                    <span key={code} className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-th-surface-high rounded-lg text-sm font-data text-th-on-surface">
                      {code}
                      <button
                        type="button"
                        onClick={() => setHardWatchSymbols(prev => prev.filter(c => c !== code))}
                        className="text-th-on-subtle hover:text-red-400 transition-colors"
                      >×</button>
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>

        </div>{/* ── end KR 탭 ── */}

      </form>

      {/* ════════════════ INFO 탭 ════════════════ */}
      <div className={activeTab !== 'INFO' ? 'hidden' : 'space-y-5'}>

        {/* ── AI 설정 ── */}
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
          <label className="space-y-1 block">
            <span className={labelText}>Claude 최대 후보 종목 수</span>
            <input
              type="number"
              step="1"
              min="1"
              max="50"
              value={maxClaudeCandidates}
              onChange={(e) => setMaxClaudeCandidates(e.target.value)}
              className={inputCls}
            />
            <p className={hintText}>서버 필터 통과 후 Claude에 전달할 최대 종목 수. 많을수록 토큰 소모 증가. 기본값: 15</p>
          </label>
          <div className={`pt-3 border-t border-black/5 dark:border-white/5 space-y-1`}>
            <span className={labelText}>AI 개선 제안 자동 적용 모드</span>
            <p className={hintText}>장 마감 후 생성된 AI 제안을 어떻게 처리할지 설정합니다. 기능 요청은 항상 수동 처리됩니다.</p>
            <select
              value={optimizationApplyMode}
              onChange={(e) => setOptimizationApplyMode(e.target.value)}
              className={inputCls}
            >
              <option value="all_manual">전체 수동 — 모든 제안을 검토 후 직접 적용</option>
              <option value="all_auto">전체 자동 — 유효 범위 내 설정 제안 자동 적용</option>
            </select>
          </div>
          <button
            type="button"
            onClick={handleSave.bind(null, { preventDefault: () => {} })}
            disabled={saving}
            className="px-5 py-2 bg-orange-500 hover:bg-orange-600 disabled:opacity-50 rounded-lg text-sm font-semibold transition-colors text-white"
          >
            {saving ? '저장 중...' : '설정 저장'}
          </button>
          {saveResult && (
            <p className={`text-xs ${saveResult.ok ? 'text-emerald-400' : 'text-red-400'}`}>{saveResult.text}</p>
          )}
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

      </div>{/* ── end INFO 탭 ── */}

    </div>
  )
}
