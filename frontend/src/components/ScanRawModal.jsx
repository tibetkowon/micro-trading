import useApi from '../hooks/useApi'
import { Modal, Badge, ProgressBar, Spinner } from './shared'

function KV({ label, value }) {
  return (
    <div className="raw-kv">
      <span className="raw-kv-label">{label}</span>
      <span className="raw-kv-val">{value ?? '—'}</span>
    </div>
  )
}

function Section({ title, children }) {
  return (
    <div className="raw-section">
      <div className="raw-section-title">{title}</div>
      <div className="raw-kv-grid">{children}</div>
    </div>
  )
}

function ScoreBar({ label, value }) {
  const v = value ?? 0
  return (
    <div className="score-bar-row">
      <span className="score-bar-label">{label}</span>
      <div style={{ flex: 1 }}>
        <ProgressBar value={v} height={7} />
      </div>
      <span className="score-bar-val">{v.toFixed(1)}</span>
    </div>
  )
}

function fmtNum(v, decimals = 2) {
  if (v == null || v === 0) return '—'
  return typeof v === 'number' ? v.toFixed(decimals) : v
}

function fmtVol(v) {
  if (!v || v === '0' || v === '') return '—'
  const n = parseInt(v, 10)
  return isNaN(n) ? v : n.toLocaleString('ko-KR')
}

function fmtPgb(v) {
  if (v == null || v === 0) return '—'
  return v.toLocaleString('ko-KR')
}

export default function ScanRawModal({ logId, stockCode, stockName, onClose }) {
  const { data, loading, error } = useApi(`/api/logs/scan/${logId}/raw`)

  const entry = data?.data?.find(e => e.code === stockCode)
  const info = entry?.info
  const scores = entry?.scores

  const renderBody = () => {
    if (loading) return <div style={{ textAlign: 'center', padding: '32px 0' }}><Spinner /></div>
    if (error === 'no_raw_data') return <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>이 로그는 구버전이라 원본 데이터가 없습니다.</div>
    if (error) return <div style={{ color: 'var(--down)', fontSize: 13 }}>불러오기 실패: {error}</div>
    if (!info) return <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>해당 종목 데이터를 찾을 수 없습니다.</div>

    const candles = info.recent_candles || []

    return (
      <>
        <Section title="가격 / 기본">
          <KV label="현재가" value={info.current_price ? parseInt(info.current_price).toLocaleString('ko-KR') + '원' : '—'} />
          <KV label="등락률" value={info.change_rate ? info.change_rate + '%' : '—'} />
          <KV label="시가" value={info.day_open ? parseInt(info.day_open).toLocaleString('ko-KR') : '—'} />
          <KV label="고가" value={info.day_high ? parseInt(info.day_high).toLocaleString('ko-KR') : '—'} />
          <KV label="저가" value={info.day_low ? parseInt(info.day_low).toLocaleString('ko-KR') : '—'} />
          <KV label="거래대금" value={info.trading_value > 0 ? (info.trading_value / 1e8).toFixed(1) + '억' : '—'} />
          <KV label="거래량" value={fmtVol(info.volume)} />
          <KV label="체결강도" value={info.strength > 0 ? info.strength.toFixed(1) : '—'} />
          <KV label="고점이격" value={info.high_price_diff != null ? info.high_price_diff.toFixed(2) + '%' : '—'} />
          <KV label="시가이격" value={info.open_price_diff != null ? info.open_price_diff.toFixed(2) + '%' : '—'} />
        </Section>

        <Section title="기술적 지표">
          <KV label="RSI(14)" value={info.rsi14 > 0 ? info.rsi14.toFixed(2) : '—'} />
          <KV label="MACD" value={fmtNum(info.macd_line)} />
          <KV label="Signal" value={fmtNum(info.macd_signal)} />
          <KV label="Histo" value={fmtNum(info.macd_histogram)} />
          <KV label="VWAP" value={info.vwap > 0 ? info.vwap.toFixed(0) : '—'} />
          <KV label="VWAP이격" value={info.vwap_diff != null ? info.vwap_diff.toFixed(2) + '%' : '—'} />
          <KV label="MA5" value={info.ma5 > 0 ? info.ma5.toFixed(0) : '—'} />
          <KV label="MA20" value={info.ma20 > 0 ? info.ma20.toFixed(0) : '—'} />
          <KV label="MA60" value={info.ma60 > 0 ? info.ma60.toFixed(0) : '—'} />
          <KV label="MA120" value={info.ma120 > 0 ? info.ma120.toFixed(0) : '—'} />
          <KV label="M5MA10" value={info.m5_ma10 > 0 ? info.m5_ma10.toFixed(0) : '—'} />
          <KV label="DisparityM5" value={info.disparity_m5 != null ? info.disparity_m5.toFixed(2) + '%' : '—'} />
        </Section>

        <Section title="호가 / 체결">
          <KV label="호가비율" value={info.bid_ask_ratio > 0 ? info.bid_ask_ratio.toFixed(3) : '—'} />
          <KV label="미시호가" value={info.micro_bid_ask_ratio > 0 ? info.micro_bid_ask_ratio.toFixed(3) : '—'} />
          <KV label="호가스프레드" value={info.bid_ask_spread > 0 ? info.bid_ask_spread.toFixed(3) + '%' : '—'} />
        </Section>

        <Section title="거래량">
          <KV label="거래량비율" value={info.vol_vs_3avg_ratio > 0 ? info.vol_vs_3avg_ratio.toFixed(3) : '—'} />
          <KV label="직전봉비율" value={info.prev_volume_ratio > 0 ? info.prev_volume_ratio.toFixed(3) : '—'} />
          <KV label="3봉기울기" value={fmtNum(info.vol_trend_3, 3)} />
          <KV label="고점봉거래량" value={info.vol_at_high > 0 ? info.vol_at_high.toLocaleString('ko-KR') : '—'} />
          <KV label="고점경과(분)" value={info.high_formed_mins_ago > 0 ? info.high_formed_mins_ago + '분' : '—'} />
        </Section>

        <Section title="프로그램 / VI">
          <KV label="프로그램순매수" value={fmtPgb(info.program_net_buy)} />
          <KV label="VI이격도" value={info.vi_disparity != null && info.vi_disparity !== 0 ? info.vi_disparity.toFixed(2) + '%' : '—'} />
        </Section>

        <div className="raw-section">
          <div className="raw-section-title">스코어 세부 (0 – 100)</div>
          {scores && (
            <>
              <ScoreBar label="체결강도" value={scores.strength} />
              <ScoreBar label="RSI" value={scores.rsi} />
              <ScoreBar label="MACD" value={scores.macd} />
              <ScoreBar label="호가비율" value={scores.bid_ask} />
              <ScoreBar label="VWAP" value={scores.vwap} />
              <ScoreBar label="거래량" value={scores.volume} />
              <ScoreBar label="프로그램" value={scores.program_buy} />
              <ScoreBar label="미시호가" value={scores.micro_bid_ask} />
              <ScoreBar label="VI이격" value={scores.vi_disparity} />
              <div style={{ borderTop: '1px solid var(--border)', marginTop: 6, paddingTop: 6 }}>
                <ScoreBar label="총점" value={scores.total} />
              </div>
            </>
          )}
        </div>

        {candles.length > 0 && (
          <div className="raw-section">
            <div className="raw-section-title">최근 캔들 ({candles.length}개, 구→신)</div>
            <div className="candle-snaps">
              {candles.map((c, i) => {
                const cls = c.dir === 'U' ? 'candle-snap-u' : c.dir === 'D' ? 'candle-snap-d' : 'candle-snap-e'
                return (
                  <span key={i} className={cls}>
                    {c.dir} {c.c?.toLocaleString('ko-KR')} ({c.v?.toLocaleString('ko-KR')})
                  </span>
                )
              })}
            </div>
          </div>
        )}
      </>
    )
  }

  return (
    <Modal
      title={`${stockName} 원본 데이터`}
      className="modal-wide"
      onClose={onClose}
      actions={<button className="btn btn-secondary" onClick={onClose}>닫기</button>}
    >
      <div style={{ marginBottom: 8, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span className="mono muted" style={{ fontSize: 11 }}>{stockCode}</span>
        {scores && <Badge color="blue">총점 {scores.total?.toFixed(1)}</Badge>}
      </div>
      {renderBody()}
    </Modal>
  )
}
