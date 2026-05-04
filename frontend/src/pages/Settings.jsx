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
  for (const k of ['strength', 'rsi', 'macd', 'bidask', 'vwap', 'volume']) {
    weights[k] = pi(raw[`weight_${k}`], 0)
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
      scan_interval: pi(raw.scan_interval, 60),
      indicator_check_interval: pi(raw.indicator_check_interval_min, 5),
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
  ['strength', '체결강도 (Strength)'],
  ['rsi',      'RSI'],
  ['macd',     'MACD'],
  ['bidask',   '매수호가비율 (BidAsk)'],
  ['vwap',     'VWAP 이격'],
  ['volume',   '거래량 증가율'],
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
      flat.scan_interval = String(settings.schedule?.scan_interval ?? 60)
      flat.indicator_check_interval_min = String(settings.schedule?.indicator_check_interval ?? 5)
      for (const k of ['strength', 'rsi', 'macd', 'bidask', 'vwap', 'volume'])
        flat[`weight_${k}`] = String(settings.weights?.[k] ?? 0)
      for (const k of ['rsi_upper_limit', 'strength_lower', 'vwap_min', 'vwap_max',
          'high_disparity', 'open_rise_limit', 'high_elapsed_min', 'volume_ratio_lower']) {
        flat[`filter_${k}_enabled`] = String(settings.filters?.[k]?.enabled ?? false)
        flat[`filter_${k}_value`] = String(settings.filters?.[k]?.value ?? 0)
      }
      await setDoc(doc(db, 'settings', 'config'), flat, { merge: true })
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
    } catch {
      alert('저장 실패. 다시 시도하세요.')
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
        {['거래조건', '순위조회', '하드필터', '점수시스템', '스케줄'].map(t => (
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
              <label className="form-label">스캔 주기 (초)</label>
              <input className="form-input" type="number" min="10"
                value={schedule.scan_interval ?? 60}
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

      <div style={{ marginTop: 28, paddingTop: 20, borderTop: '1px solid var(--border)', maxWidth: 600 }}>
        <button className="btn btn-primary" style={{ width: '100%', justifyContent: 'center', padding: '12px' }}
          onClick={save} disabled={saving}>
          {saving ? '저장 중...' : saved ? '✓ 저장 완료' : '저장'}
        </button>
      </div>
    </div>
  )
}
