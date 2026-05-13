import { useState, useEffect } from 'react'
import { doc, getDoc, setDoc } from 'firebase/firestore'
import { Toggle } from '../components/shared'
import { db } from '../lib/firebase'

function pf(v, d) { const n = parseFloat(v); return isNaN(n) ? d : n }
function pi(v, d) { const n = parseInt(v, 10); return isNaN(n) ? d : n }
function pb(v, d) { return v == null ? d : v === true || v === 'true' }
function pj(v, d) { if (!v) return d; if (Array.isArray(v)) return v; try { return JSON.parse(v) } catch { return d } }

function transformSettings(raw) {
  if (!raw) return null
  const FILTER_KEYS = ['rsi_upper_limit', 'strength_lower', 'vwap_min', 'vwap_max',
    'high_disparity', 'open_rise_limit', 'high_elapsed_min', 'volume_ratio_lower']
  const filters = {}
  for (const k of FILTER_KEYS) {
    filters[k] = {
      enabled: pb(raw[`filter_${k}_enabled`], false),
      value: pf(raw[`filter_${k}_value`], 0),
    }
  }
  const weights = {}
  for (const k of ['strength', 'rsi', 'macd', 'bidask', 'vwap', 'volume', 'program_buy', 'micro_bidask', 'vi_disparity']) {
    weights[k] = pi(raw[`score_weight_${k}`] ?? raw[`weight_${k}`], 0)
  }
  return {
    max_positions: pi(raw.max_positions, 1),
    order_amount_pct: pi(raw.order_amount_pct, 95),
    take_profit_pct: pf(raw.take_profit_pct, 3),
    stop_loss_pct: pf(raw.stop_loss_pct, 2),
    etf_take_profit_pct: pf(raw.etf_take_profit_pct, 2),
    etf_stop_loss_pct: pf(raw.etf_stop_loss_pct, 1),
    daily_loss_limit_pct: pf(raw.daily_max_loss_pct, 3),
    indicator_rsi_sell_enabled: raw.indicator_rsi_sell_enabled != null
      ? pb(raw.indicator_rsi_sell_enabled, false)
      : pf(raw.indicator_rsi_sell_threshold, 0) > 0,
    indicator_macd_bearish_sell: pb(raw.indicator_macd_bearish_sell, false),
    hard_ma60_support_enabled: pb(raw.hard_ma60_support_enabled, false),
    hard_ma120_support_enabled: pb(raw.hard_ma120_support_enabled, false),
    hard_program_buy_min: pf(raw.hard_program_buy_min, 0),
    buy_pause_start: raw.buy_pause_start || '',
    buy_pause_end: raw.buy_pause_end || '',
    ranking_types: pj(raw.ranking_types, []),
    ranking_condition: raw.ranking_condition || 'OR',
    ranking_top_n: pi(raw.ranking_top_n, 30),
    ranking_price_min: raw.ranking_price_min || '5000',
    ranking_price_max: raw.ranking_price_max || '200000',
    ranking_exchanges: pj(raw.ranking_exchanges, ['0001', '1001']),
    ranking_exclude_cls: raw.ranking_exclude_cls || '1111111111',
    min_score: pf(raw.min_score_threshold, 0),
    weights,
    filters,
    schedule: {
      trade_start: raw.trading_start_time || '09:15',
      trade_end: raw.trading_end_time || '15:15',
      scan_interval: pi(raw.scan_interval, 1),
      indicator_check_interval: pi(raw.indicator_check_interval_min, 5),
    },
    sell_on_upper_limit: pb(raw.sell_on_upper_limit, false),
    max_consecutive_losses: pi(raw.max_consecutive_losses, 0),
    consecutive_loss_reset_on_profit: pb(raw.consecutive_loss_reset_on_profit, true),
    max_bidask_spread_pct: pf(raw.max_bidask_spread_pct, 0),
    trailing: {
      trigger_pct: pf(raw.trailing_trigger_pct, 0),
      stop_pct: pf(raw.trailing_stop_pct, 0),
    },
    stagnation: {
      threshold_pct: pf(raw.stagnation_threshold_pct, 0),
      duration_min: pi(raw.stagnation_duration_min, 0),
      partial_exit_enabled: pb(raw.stagnation_partial_exit_enabled, false),
      bidask_sell_threshold: pf(raw.stagnation_bidask_sell_threshold, 1.0),
    },
    sell_conditions: pj(raw.sell_conditions, []),
    block_reentry_on_loss: pb(raw.block_reentry_on_loss, true),
    reentry_score_penalty: pf(raw.reentry_score_penalty, 10),
    reentry_cooldown_min: pi(raw.reentry_cooldown_min, 15),
    buy_order_type: raw.buy_order_type || 'limit',
    stream: {
      bypass_enabled: pb(raw.stream_bypass_enabled, true),
      big_trade_amount: pf(raw.stream_big_trade_amount, 30000000),
      velocity_threshold: pf(raw.stream_velocity_threshold, 5.0),
    },
  }
}

function DonutChart({ weights }) {
  const entries = Object.entries(weights)
  const total = entries.reduce((s, [, v]) => s + v, 0)
  const colors = ['#EA6C10', '#3B82F6', '#22C55E', '#F59E0B', '#8B5CF6', '#EC4899']
  const r = 40, cx = 50, cy = 50, stroke = 14
  const circ = 2 * Math.PI * r
  let cumPct = 0
  return (
    <svg viewBox="0 0 100 100" style={{ width: 120, height: 120 }}>
      {entries.map(([key, val], i) => {
        const pct = total > 0 ? val / total : 0
        const offset = circ * (1 - cumPct)
        const dash = circ * pct
        cumPct += pct
        return (
          <circle key={key} cx={cx} cy={cy} r={r} fill="none"
            stroke={colors[i % colors.length]}
            strokeWidth={stroke}
            strokeDasharray={`${dash} ${circ - dash}`}
            strokeDashoffset={offset}
            style={{ transform: 'rotate(-90deg)', transformOrigin: '50px 50px' }}>
            <title>{key}: {val}%</title>
          </circle>
        )
      })}
      <text x="50" y="53" textAnchor="middle"
        style={{ fill: 'var(--text)', fontSize: 12, fontFamily: 'var(--font-mono)', fontWeight: 700 }}>
        {total}%
      </text>
    </svg>
  )
}

const HARD_FILTERS = [
  { key: 'rsi_upper_limit',    label: 'RSI 상한선',          unit: '' },
  { key: 'strength_lower',     label: '체결강도 하한선',      unit: '' },
  { key: 'vwap_min',           label: 'VWAP 이격 최소',       unit: '%' },
  { key: 'vwap_max',           label: 'VWAP 이격 최대',       unit: '%' },
  { key: 'high_disparity',     label: '당일 고가 이격',        unit: '%' },
  { key: 'open_rise_limit',    label: '시가 대비 상승률 상한', unit: '%' },
  { key: 'high_elapsed_min',   label: '고점 경과 시간',        unit: '분' },
  { key: 'volume_ratio_lower', label: '거래량 비율 하한',      unit: '' },
]

const WEIGHT_LABELS = [
  ['strength',     '체결강도 (Strength)'],
  ['rsi',          'RSI'],
  ['macd',         'MACD'],
  ['bidask',       '10호가비율 (BidAsk)'],
  ['vwap',         'VWAP 이격'],
  ['volume',       '거래량 증가율'],
  ['program_buy',  '프로그램 순매수 비중'],
  ['micro_bidask', '1~3호가 매수/매도 비율'],
  ['vi_disparity', 'VI 정적 이격도 (1~2%)'],
]

export default function Settings() {
  const [tab, setTab] = useState('거래조건')
  const [settings, setSettings] = useState(null)
  const [loading, setLoading] = useState(true)
  const [saved, setSaved] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    getDoc(doc(db, 'settings', 'config'))
      .then(snap => setSettings(transformSettings(snap.exists() ? snap.data() : {})))
      .finally(() => setLoading(false))
  }, [])

  function set(path, val) {
    setSettings(prev => {
      const next = { ...prev }
      const keys = path.split('.')
      let obj = next
      for (let i = 0; i < keys.length - 1; i++) {
        obj[keys[i]] = { ...obj[keys[i]] }
        obj = obj[keys[i]]
      }
      obj[keys[keys.length - 1]] = val
      return next
    })
  }

  async function save() {
    setSaving(true)
    try {
      const flat = {}
      flat.max_positions = String(settings.max_positions ?? 1)
      flat.order_amount_pct = String(settings.order_amount_pct ?? 95)
      flat.take_profit_pct = String(settings.take_profit_pct ?? 3)
      flat.stop_loss_pct = String(settings.stop_loss_pct ?? 2)
      flat.etf_take_profit_pct = String(settings.etf_take_profit_pct ?? 2)
      flat.etf_stop_loss_pct = String(settings.etf_stop_loss_pct ?? 1)
      flat.daily_max_loss_pct = String(settings.daily_loss_limit_pct ?? 3)
      flat.indicator_rsi_sell_enabled = String(settings.indicator_rsi_sell_enabled ?? false)
      flat.indicator_macd_bearish_sell = String(settings.indicator_macd_bearish_sell ?? false)
      flat.hard_ma60_support_enabled = String(settings.hard_ma60_support_enabled ?? false)
      flat.hard_ma120_support_enabled = String(settings.hard_ma120_support_enabled ?? false)
      flat.hard_program_buy_min = String(settings.hard_program_buy_min ?? 0)
      flat.buy_pause_start = settings.buy_pause_start || ''
      flat.buy_pause_end = settings.buy_pause_end || ''
      flat.ranking_types = JSON.stringify(settings.ranking_types ?? [])
      flat.ranking_condition = settings.ranking_condition || 'OR'
      flat.ranking_top_n = String(settings.ranking_top_n ?? 30)
      flat.ranking_price_min = String(settings.ranking_price_min ?? '5000')
      flat.ranking_price_max = String(settings.ranking_price_max ?? '200000')
      flat.ranking_exchanges = JSON.stringify(settings.ranking_exchanges ?? [])
      flat.ranking_exclude_cls = settings.ranking_exclude_cls || '1111111111'
      flat.min_score_threshold = String(settings.min_score ?? 0)
      flat.trading_start_time = settings.schedule?.trade_start || '09:15'
      flat.trading_end_time = settings.schedule?.trade_end || '15:15'
      flat.scan_interval = String(settings.schedule?.scan_interval ?? 1)
      flat.indicator_check_interval_min = String(settings.schedule?.indicator_check_interval ?? 5)
      for (const k of ['strength', 'rsi', 'macd', 'bidask', 'vwap', 'volume', 'program_buy', 'micro_bidask', 'vi_disparity'])
        flat[`score_weight_${k}`] = String(settings.weights?.[k] ?? 0)
      for (const k of ['rsi_upper_limit', 'strength_lower', 'vwap_min', 'vwap_max',
          'high_disparity', 'open_rise_limit', 'high_elapsed_min', 'volume_ratio_lower']) {
        flat[`filter_${k}_enabled`] = String(settings.filters?.[k]?.enabled ?? false)
        flat[`filter_${k}_value`] = String(settings.filters?.[k]?.value ?? 0)
      }
      flat.sell_on_upper_limit = String(settings.sell_on_upper_limit ?? false)
      flat.max_consecutive_losses = String(settings.max_consecutive_losses ?? 0)
      flat.consecutive_loss_reset_on_profit = String(settings.consecutive_loss_reset_on_profit ?? true)
      flat.max_bidask_spread_pct = String(settings.max_bidask_spread_pct ?? 0)
      flat.trailing_trigger_pct = String(settings.trailing?.trigger_pct ?? 0)
      flat.trailing_stop_pct = String(settings.trailing?.stop_pct ?? 0)
      flat.stagnation_threshold_pct = String(settings.stagnation?.threshold_pct ?? 0)
      flat.stagnation_duration_min = String(settings.stagnation?.duration_min ?? 0)
      flat.stagnation_partial_exit_enabled = String(settings.stagnation?.partial_exit_enabled ?? false)
      flat.stagnation_bidask_sell_threshold = String(settings.stagnation?.bidask_sell_threshold ?? 1.0)
      flat.sell_conditions = JSON.stringify(settings.sell_conditions ?? [])
      flat.block_reentry_on_loss = String(settings.block_reentry_on_loss ?? true)
      flat.reentry_score_penalty = String(settings.reentry_score_penalty ?? 10)
      flat.reentry_cooldown_min = String(settings.reentry_cooldown_min ?? 15)
      flat.buy_order_type = settings.buy_order_type || 'limit'
      flat.stream_bypass_enabled = String(settings.stream?.bypass_enabled ?? true)
      flat.stream_big_trade_amount = String(settings.stream?.big_trade_amount ?? 30000000)
      flat.stream_velocity_threshold = String(settings.stream?.velocity_threshold ?? 5.0)
      await setDoc(doc(db, 'settings', 'config'), flat, { merge: true })
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
    } catch (err) {
      console.error('settings save error:', err)
      alert('저장 실패: ' + (err?.message || String(err)))
    } finally {
      setSaving(false)
    }
  }

  if (loading || !settings) {
    return <div style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)' }}>로딩 중...</div>
  }

  const weights = settings.weights || {}
  const totalWeight = Object.values(weights).reduce((s, v) => s + (v || 0), 0)
  const filters = settings.filters || {}
  const schedule = settings.schedule || {}

  return (
    <div>
      <div className="tab-bar">
        {['거래조건', '하이재킹', '순위조회', '하드필터', '점수시스템', '스케줄', '매도관리'].map(t => (
          <div key={t} className={`tab-item ${tab === t ? 'active' : ''}`} onClick={() => setTab(t)}>{t}</div>
        ))}
      </div>

      {tab === '거래조건' && (
        <div style={{ maxWidth: 600 }}>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">최대 동시 포지션 수</label>
              <input className="form-input" type="number" min="1" max="10"
                value={settings.max_positions ?? 1}
                onChange={e => set('max_positions', +e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">주문 금액 비율 (%)</label>
              <input className="form-input" type="number" min="50" max="100"
                value={settings.order_amount_pct ?? 95}
                onChange={e => set('order_amount_pct', +e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group" style={{ flex: '0 0 100%' }}>
              <label className="form-label">매수 주문 방식</label>
              <select className="form-input"
                value={settings.buy_order_type ?? 'limit'}
                onChange={e => set('buy_order_type', e.target.value)}
                style={{ cursor: 'pointer' }}>
                <option value="limit">현재가 지정가 (기본값)</option>
                <option value="ask1">매도 1호가 지정가 (즉각 체결 + 슬리피지 통제)</option>
                <option value="ask2">매도 2호가 지정가 (1호가 잔량 부족 시 보완)</option>
                <option value="market">순수 시장가 (최속 체결, 슬리피지 위험)</option>
              </select>
              {(settings.buy_order_type === 'ask1' || settings.buy_order_type === 'ask2') && (
                <div className="muted" style={{ fontSize: 11, marginTop: 4 }}>
                  주문 시점의 {settings.buy_order_type === 'ask1' ? '매도 1호가' : '매도 2호가'}로 지정가 주문 → 즉각 체결되면서 체결가 상한 통제
                </div>
              )}
              {settings.buy_order_type === 'market' && (
                <div className="muted" style={{ fontSize: 11, marginTop: 4, color: '#f59e0b' }}>
                  ⚠ 호가창이 얇은 종목에서 슬리피지 대참사 가능. 유동성 충분한 종목에만 사용 권장
                </div>
              )}
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">목표 수익률 (%)</label>
              <input className="form-input" type="number" step="0.1"
                value={settings.take_profit_pct ?? 3}
                onChange={e => set('take_profit_pct', +e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">손절률 (%)</label>
              <input className="form-input" type="number" step="0.1"
                value={settings.stop_loss_pct ?? 2}
                onChange={e => set('stop_loss_pct', +e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">ETF 목표 수익률 (%)</label>
              <input className="form-input" type="number" step="0.1"
                value={settings.etf_take_profit_pct ?? 2}
                onChange={e => set('etf_take_profit_pct', +e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">ETF 손절률 (%)</label>
              <input className="form-input" type="number" step="0.1"
                value={settings.etf_stop_loss_pct ?? 1}
                onChange={e => set('etf_stop_loss_pct', +e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">일일 손실 한도 (%)</label>
              <input className="form-input" type="number" step="0.1"
                value={settings.daily_loss_limit_pct ?? 3}
                onChange={e => set('daily_loss_limit_pct', +e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">익절 종목 재진입 페널티 (점)</label>
              <input className="form-input" type="number" min="0" step="1"
                value={settings.reentry_score_penalty ?? 10}
                onChange={e => set('reentry_score_penalty', +e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">익절 종목 재진입 쿨타임 (분)</label>
              <input className="form-input" type="number" min="0" step="1"
                value={settings.reentry_cooldown_min ?? 15}
                onChange={e => set('reentry_cooldown_min', +e.target.value)} />
            </div>
          </div>
          <div className="filter-row">
            <span className="filter-label">손절 종목 당일 재매수 금지</span>
            <Toggle
              checked={settings.block_reentry_on_loss ?? true}
              onChange={v => set('block_reentry_on_loss', v)} />
          </div>
          <div className="filter-row">
            <span className="filter-label">RSI 과매수 자동 매도</span>
            <Toggle
              checked={settings.indicator_rsi_sell_enabled ?? false}
              onChange={v => set('indicator_rsi_sell_enabled', v)} />
          </div>
          <div className="filter-row">
            <span className="filter-label">MACD 데드크로스 자동 매도</span>
            <Toggle
              checked={settings.indicator_macd_bearish_sell ?? false}
              onChange={v => set('indicator_macd_bearish_sell', v)} />
          </div>
        </div>
      )}

      {tab === '하이재킹' && (
        <div style={{ maxWidth: 600 }}>
          <div style={{ marginBottom: 24 }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              WebSocket 실시간 바이패스(Hijacking)
            </div>
            <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
              순위조회를 통과한 관심 종목에 대해 10초간 실시간 체결을 추적하여 대량 체결 및 체결 속도 급등 시 30초 스캔 주기를 무시하고 즉각 우선 매수합니다 (단, 하드필터는 거칩니다).
            </div>
            <div className="filter-row">
              <span className="filter-label">하이재킹 기능 활성화</span>
              <Toggle
                checked={settings.stream?.bypass_enabled ?? true}
                onChange={v => set('stream.bypass_enabled', v)} />
            </div>
          </div>
          
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Big Trade 기준 금액 (원)</label>
              <input className="form-input" type="number" step="1000000" min="0"
                value={settings.stream?.big_trade_amount ?? 30000000}
                onChange={e => set('stream.big_trade_amount', +e.target.value)} />
              <div className="muted" style={{ fontSize: 11, marginTop: 4 }}>해당 금액 이상의 건이 10초 내 2회 이상 발생 시 발동</div>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">체결 속도 임계치 (건/초)</label>
              <input className="form-input" type="number" step="1" min="1"
                value={settings.stream?.velocity_threshold ?? 5.0}
                onChange={e => set('stream.velocity_threshold', +e.target.value)} />
              <div className="muted" style={{ fontSize: 11, marginTop: 4 }}>최근 10초간 초당 체결 건수가 이 값을 초과하면 발동</div>
            </div>
          </div>
        </div>
      )}

      {tab === '순위조회' && (
        <div style={{ maxWidth: 600 }}>
          {/* 순위 타입 */}
          <div className="form-group" style={{ marginBottom: 16 }}>
            <label className="form-label">순위 타입 (복수 선택 가능)</label>
            <div style={{ display: 'flex', gap: 8, marginTop: 6, flexWrap: 'wrap' }}>
              {[['volume', '거래량'], ['strength', '체결강도'], ['fluctuation', '등락률']].map(([val, label]) => {
                const types = settings.ranking_types || []
                const active = types.includes(val)
                return (
                  <button key={val}
                    className={`btn ${active ? 'btn-primary' : 'btn-outline'} btn-sm`}
                    onClick={() => {
                      const next = active ? types.filter(t => t !== val) : [...types, val]
                      set('ranking_types', next)
                    }}>
                    {label}
                  </button>
                )
              })}
            </div>
          </div>

          {/* 선정 조건 */}
          <div className="form-group" style={{ marginBottom: 16 }}>
            <label className="form-label">종목 선정 조건</label>
            <div style={{ display: 'flex', gap: 8, marginTop: 6 }}>
              {['OR', 'AND'].map(m => (
                <button key={m}
                  className={`btn ${settings.ranking_condition === m ? 'btn-primary' : 'btn-outline'} btn-sm`}
                  onClick={() => set('ranking_condition', m)}>
                  {m === 'OR' ? '합집합 (OR)' : '교집합 (AND)'}
                </button>
              ))}
            </div>
            <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
              OR: 어느 한 순위에라도 등장한 종목 / AND: 모든 순위에 동시 등장한 종목만
            </div>
          </div>

          {/* 상위 N개 + 가격 범위 */}
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">상위 N개 (최대 30)</label>
              <input className="form-input" type="number" min="5" max="30"
                value={settings.ranking_top_n ?? 30}
                onChange={e => set('ranking_top_n', +e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">최소 가격 (원)</label>
              <input className="form-input" type="number" step="1000"
                value={settings.ranking_price_min ?? 5000}
                onChange={e => set('ranking_price_min', String(e.target.value))} />
            </div>
            <div className="form-group">
              <label className="form-label">최대 가격 (원)</label>
              <input className="form-input" type="number" step="10000"
                value={settings.ranking_price_max ?? 200000}
                onChange={e => set('ranking_price_max', String(e.target.value))} />
            </div>
          </div>

          {/* 거래소 */}
          <div className="form-group" style={{ marginBottom: 16 }}>
            <label className="form-label">거래소</label>
            <div style={{ display: 'flex', gap: 8, marginTop: 6 }}>
              {[['0001', 'KOSPI'], ['1001', 'KOSDAQ']].map(([val, label]) => {
                const exs = settings.ranking_exchanges || []
                const active = exs.includes(val)
                return (
                  <button key={val}
                    className={`btn ${active ? 'btn-primary' : 'btn-outline'} btn-sm`}
                    onClick={() => {
                      const next = active ? exs.filter(e => e !== val) : [...exs, val]
                      set('ranking_exchanges', next)
                    }}>
                    {label}
                  </button>
                )
              })}
            </div>
          </div>

          {/* 제외 종목 유형 */}
          <div className="form-group" style={{ marginBottom: 16 }}>
            <label className="form-label">제외 종목 유형 (10자리)</label>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '6px 16px', marginTop: 8 }}>
              {[
                [0, '투자위험종목'],
                [1, '투자경고종목'],
                [2, '투자주의종목'],
                [3, '관리종목'],
                [4, '정리매매종목'],
                [5, '불성실공시'],
                [6, '우선주'],
                [7, '거래정지'],
                [8, 'ETF'],
                [9, 'ETN'],
              ].map(([idx, label]) => {
                const cls = (settings.ranking_exclude_cls ?? '1111111111').split('')
                const excluded = cls[idx] === '1'
                return (
                  <label key={idx} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, cursor: 'pointer' }}>
                    <input type="checkbox" checked={excluded} onChange={e => {
                      const arr = (settings.ranking_exclude_cls ?? '1111111111').split('')
                      arr[idx] = e.target.checked ? '1' : '0'
                      set('ranking_exclude_cls', arr.join(''))
                    }} />
                    {label}
                  </label>
                )
              })}
            </div>
          </div>
        </div>
      )}

      {tab === '하드필터' && (
        <div style={{ maxWidth: 600 }}>
          <div className="filter-row" style={{ marginBottom: 12 }}>
            <span className="filter-label">최대 호가 스프레드 (%)<span className="muted" style={{ fontSize: 11, marginLeft: 6 }}>0=비활성</span></span>
            <input className="form-input" type="number" step="0.1" min="0"
              value={settings.max_bidask_spread_pct ?? 0}
              onChange={e => set('max_bidask_spread_pct', +e.target.value)}
              style={{ width: 100 }} />
          </div>
          {HARD_FILTERS.map(f => {
            const filter = filters[f.key] || {}
            return (
              <div className="filter-row" key={f.key}>
                <span className="filter-label">{f.label}</span>
                <Toggle
                  checked={filter.enabled ?? false}
                  onChange={v => set(`filters.${f.key}.enabled`, v)} />
                <input className="form-input" type="number" step="0.5"
                  value={filter.value ?? 0}
                  onChange={e => set(`filters.${f.key}.value`, +e.target.value)}
                  style={{ opacity: filter.enabled ? 1 : 0.4 }}
                  disabled={!filter.enabled} />
                {f.unit && <span className="muted" style={{ fontSize: 12, textAlign: 'right' }}>{f.unit}</span>}
              </div>
            )
          })}
          <div className="filter-row" style={{ marginTop: 12 }}>
            <span className="filter-label">당일 프로그램 순매수 하한선 (수량)</span>
            <input className="form-input" type="number" step="100"
              value={settings.hard_program_buy_min ?? 0}
              onChange={e => set('hard_program_buy_min', +e.target.value)}
              style={{ width: 120 }} />
          </div>
          <div className="filter-row">
            <span className="filter-label">MA60 지지 (현재가 ≥ MA60)</span>
            <Toggle
              checked={settings.hard_ma60_support_enabled ?? false}
              onChange={v => set('hard_ma60_support_enabled', v)} />
          </div>
          <div className="filter-row">
            <span className="filter-label">MA120 지지 (현재가 ≥ MA120)</span>
            <Toggle
              checked={settings.hard_ma120_support_enabled ?? false}
              onChange={v => set('hard_ma120_support_enabled', v)} />
          </div>
          <div style={{ marginTop: 16 }}>
            <button className="btn btn-outline btn-sm">↺ 필터 초기화</button>
          </div>
        </div>
      )}

      {tab === '점수시스템' && (
        <div style={{ maxWidth: 600 }}>
          <div style={{ display: 'flex', gap: 32, alignItems: 'flex-start' }}>
            <div style={{ flex: 1 }}>
              {WEIGHT_LABELS.map(([key, label]) => (
                <div className="weight-row" key={key}>
                  <span className="weight-label">{label}</span>
                  <input type="range" className="range-input" min="0" max="100"
                    value={weights[key] ?? 0}
                    onChange={e => set(`weights.${key}`, +e.target.value)}
                    style={{ flex: 1 }} />
                  <span className="weight-val">{weights[key] ?? 0}%</span>
                </div>
              ))}
              <div style={{
                marginTop: 8, padding: '8px 12px', borderRadius: 6,
                background: totalWeight === 100 ? 'rgba(34,197,94,0.1)' : 'rgba(37,99,235,0.1)',
                color: totalWeight === 100 ? 'var(--up)' : 'var(--red)',
                fontSize: 12, fontFamily: 'var(--font-mono)',
              }}>
                합계: {totalWeight}% {totalWeight !== 100 && '(⚠ 합계가 100이 되어야 합니다)'}
              </div>
              <div className="form-group" style={{ marginTop: 16 }}>
                <label className="form-label">최소 합산 점수 (0–100)</label>
                <input className="form-input" type="number" min="0" max="100"
                  value={settings.min_score ?? 0}
                  onChange={e => set('min_score', +e.target.value)}
                  style={{ width: 120 }} />
              </div>
            </div>
            <DonutChart weights={weights} />
          </div>
        </div>
      )}

      {tab === '스케줄' && (
        <div style={{ maxWidth: 600 }}>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">거래 시작 시각</label>
              <input className="form-input" type="time"
                value={schedule.trade_start ?? '09:15'}
                onChange={e => set('schedule.trade_start', e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">거래 종료 시각</label>
              <input className="form-input" type="time"
                value={schedule.trade_end ?? '15:15'}
                onChange={e => set('schedule.trade_end', e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">스캔 주기 (분)</label>
              <input className="form-input" type="number" min="1"
                value={schedule.scan_interval ?? 1}
                onChange={e => set('schedule.scan_interval', +e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">지표 체크 주기 (분)</label>
              <input className="form-input" type="number" min="1"
                value={schedule.indicator_check_interval ?? 5}
                onChange={e => set('schedule.indicator_check_interval', +e.target.value)} />
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">매수 중단 시간</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input className="form-input" type="time"
                value={settings.buy_pause_start ?? ''}
                onChange={e => set('buy_pause_start', e.target.value)}
                style={{ flex: 1 }} />
              <span className="muted">~</span>
              <input className="form-input" type="time"
                value={settings.buy_pause_end ?? ''}
                onChange={e => set('buy_pause_end', e.target.value)}
                style={{ flex: 1 }} />
            </div>
          </div>
        </div>
      )}

      {tab === '매도관리' && (
        <div style={{ maxWidth: 600 }}>

          {/* ── 상한가 매도 ── */}
          <div style={{ marginBottom: 24 }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              상한가 도달 시 매도
            </div>
            <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
              당일 상한가(+30%)에 도달하면 즉시 익절. 매수 등록 시 KIS API에서 상한가 자동 조회.
            </div>
            <div className="filter-row">
              <span className="filter-label">상한가 자동 매도 활성화</span>
              <Toggle
                checked={settings.sell_on_upper_limit ?? false}
                onChange={v => set('sell_on_upper_limit', v)} />
            </div>
          </div>

          {/* ── 연속 손절 제한 ── */}
          <div style={{ marginBottom: 24 }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              연속 손절 제한
            </div>
            <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
              N회 연속 손절 시 당일 매매 중지. 0 = 비활성
            </div>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">최대 연속 손절 횟수</label>
                <input className="form-input" type="number" min="0" max="20"
                  value={settings.max_consecutive_losses ?? 0}
                  onChange={e => set('max_consecutive_losses', +e.target.value)} />
              </div>
            </div>
            <div className="filter-row">
              <span className="filter-label">익절 시 연속 손절 카운터 리셋</span>
              <Toggle
                checked={settings.consecutive_loss_reset_on_profit ?? true}
                onChange={v => set('consecutive_loss_reset_on_profit', v)} />
            </div>
          </div>

          {/* ── 트레일링 스탑 ── */}
          <div style={{ marginBottom: 24 }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              트레일링 스탑
            </div>
            <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
              목표가 전이라도 수익이 발생했다가 최고점 대비 일정 % 하락 시 즉시 익절. 0 = 비활성
            </div>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">활성화 기준 수익률 (%)</label>
                <input className="form-input" type="number" step="0.1" min="0"
                  value={settings.trailing?.trigger_pct ?? 0}
                  onChange={e => set('trailing.trigger_pct', +e.target.value)} />
              </div>
              <div className="form-group">
                <label className="form-label">최고점 대비 하락 허용폭 (%)</label>
                <input className="form-input" type="number" step="0.1" min="0"
                  value={settings.trailing?.stop_pct ?? 0}
                  onChange={e => set('trailing.stop_pct', +e.target.value)} />
              </div>
            </div>
          </div>

          {/* ── 횡보 탐지 ── */}
          <div style={{ marginBottom: 24, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              횡보 탐지 청산
            </div>
            <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
              가격이 기준 변동폭 안에서 일정 시간 이상 횡보하면 청산. 0 = 비활성
            </div>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">횡보 판단 변동폭 (%)</label>
                <input className="form-input" type="number" step="0.1" min="0"
                  value={settings.stagnation?.threshold_pct ?? 0}
                  onChange={e => set('stagnation.threshold_pct', +e.target.value)} />
              </div>
              <div className="form-group">
                <label className="form-label">횡보 지속 기준 (분)</label>
                <input className="form-input" type="number" min="0"
                  value={settings.stagnation?.duration_min ?? 0}
                  onChange={e => set('stagnation.duration_min', +e.target.value)} />
              </div>
            </div>
            <div className="filter-row">
              <span className="filter-label">단계적 청산 (1차 50% → 2차 전량)</span>
              <Toggle
                checked={settings.stagnation?.partial_exit_enabled ?? false}
                onChange={v => set('stagnation.partial_exit_enabled', v)} />
            </div>
            <div className="form-row" style={{ marginTop: 8 }}>
              <div className="form-group">
                <label className="form-label">호가비율 즉시 전량청산 기준 (미만)</label>
                <input className="form-input" type="number" step="0.1" min="0"
                  value={settings.stagnation?.bidask_sell_threshold ?? 1.0}
                  onChange={e => set('stagnation.bidask_sell_threshold', +e.target.value)} />
              </div>
            </div>
          </div>

          {/* ── 매도 조건 순위 ── */}
          <div style={{ paddingTop: 16, borderTop: '1px solid var(--border)' }}>
            <div className="form-label" style={{ fontSize: 13, fontWeight: 700, marginBottom: 12, color: 'var(--accent)' }}>
              지표 기반 매도 조건 순위
            </div>
            <div className="muted" style={{ fontSize: 12, marginBottom: 10 }}>
              활성화된 조건을 순서대로 평가. ↑↓ 로 우선순위 변경.
            </div>
            {(() => {
              const ALL_CONDITIONS = [
                { key: 'rsi_overbought', label: 'RSI 과매수' },
                { key: 'macd_bearish',   label: 'MACD 데드크로스' },
                { key: 'stagnation',     label: '횡보 탐지' },
              ]
              const active = settings.sell_conditions ?? []
              const inactive = ALL_CONDITIONS.filter(c => !active.includes(c.key))
              const activeItems = active
                .map(k => ALL_CONDITIONS.find(c => c.key === k))
                .filter(Boolean)

              const move = (idx, dir) => {
                const next = [...active]
                const swap = idx + dir
                if (swap < 0 || swap >= next.length) return
                ;[next[idx], next[swap]] = [next[swap], next[idx]]
                set('sell_conditions', next)
              }
              const toggle = (key, enabled) => {
                if (enabled) {
                  set('sell_conditions', [...active, key])
                } else {
                  set('sell_conditions', active.filter(k => k !== key))
                }
              }

              return (
                <div>
                  {activeItems.map((c, i) => (
                    <div key={c.key} style={{
                      display: 'flex', alignItems: 'center', gap: 10,
                      padding: '8px 12px', marginBottom: 6,
                      background: 'var(--bg-card)', borderRadius: 6,
                      border: '1px solid var(--border)',
                    }}>
                      <span style={{ fontSize: 11, color: 'var(--text-muted)', width: 18, textAlign: 'center', fontFamily: 'var(--font-mono)' }}>{i + 1}</span>
                      <span style={{ flex: 1, fontSize: 13 }}>{c.label}</span>
                      <button className="btn btn-outline btn-sm" style={{ padding: '2px 8px' }}
                        onClick={() => move(i, -1)} disabled={i === 0}>↑</button>
                      <button className="btn btn-outline btn-sm" style={{ padding: '2px 8px' }}
                        onClick={() => move(i, 1)} disabled={i === activeItems.length - 1}>↓</button>
                      <button className="btn btn-sm" style={{ padding: '2px 8px', background: 'var(--down)', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}
                        onClick={() => toggle(c.key, false)}>✕</button>
                    </div>
                  ))}
                  {inactive.length > 0 && (
                    <div style={{ marginTop: 10 }}>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>비활성 조건 (클릭하여 추가)</div>
                      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                        {inactive.map(c => (
                          <button key={c.key} className="btn btn-outline btn-sm"
                            onClick={() => toggle(c.key, true)}>
                            + {c.label}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )
            })()}
          </div>
        </div>
      )}

      <div style={{ marginTop: 28, paddingTop: 20, borderTop: '1px solid var(--border)', maxWidth: 600 }}>
        <button className="btn btn-primary" style={{ width: '100%', justifyContent: 'center', padding: '12px' }}
          onClick={save} disabled={saving}>
          {saving ? '저장 중...' : saved ? '✓ 저장 완료' : '저장'}
        </button>
      </div>
    </div>
  )
}
