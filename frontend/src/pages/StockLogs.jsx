import { useState, useMemo, useEffect, useRef } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

function parseJSON(str) {
  try { return JSON.parse(str) } catch { return null }
}

function fmt(val, digits = 0) {
  if (val == null || val === '' || val === '0') return '-'
  const n = Number(val)
  if (isNaN(n)) return val
  return digits > 0 ? n.toFixed(digits) : n.toLocaleString()
}

/* ── 순위 조회 결과 섹션 ── */
function RankingResultSection({ log }) {
  const types = parseJSON(log.ranking_types) || []
  const resultStocks = parseJSON(log.result_stocks) || []
  const isOR = log.ranking_condition === 'OR'
  const separator = isOR ? '|' : '+'

  return (
    <div className="space-y-3">
      {/* 타입별 카운트 */}
      {!log.error_message && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
          {log.volume_count !== -1 && (
            <div className="bg-th-surface-low rounded-lg px-3 py-2">
              <p className="text-xs text-th-on-subtle">거래량</p>
              <p className="text-sm font-data font-medium text-th-on-surface mt-0.5">{log.volume_count}개</p>
            </div>
          )}
          {log.strength_count !== -1 && (
            <div className="bg-th-surface-low rounded-lg px-3 py-2">
              <p className="text-xs text-th-on-subtle">체결강도</p>
              <p className="text-sm font-data font-medium text-th-on-surface mt-0.5">{log.strength_count}개</p>
            </div>
          )}
          {log.exec_count_count !== -1 && (
            <div className="bg-th-surface-low rounded-lg px-3 py-2">
              <p className="text-xs text-th-on-subtle">대량체결</p>
              <p className="text-sm font-data font-medium text-th-on-surface mt-0.5">{log.exec_count_count}개</p>
            </div>
          )}
          {log.disparity_count !== -1 && (
            <div className="bg-th-surface-low rounded-lg px-3 py-2">
              <p className="text-xs text-th-on-subtle">이격도</p>
              <p className="text-sm font-data font-medium text-th-on-surface mt-0.5">{log.disparity_count}개</p>
            </div>
          )}
        </div>
      )}

      {/* 최종 결과 종목 */}
      {resultStocks.length > 0 ? (
        <div>
          <p className="text-xs text-th-on-subtle mb-1.5">
            {isOR ? 'OR 합집합' : 'AND 교집합'} — {types.map(rankingTypeKr).join(separator)} — {resultStocks.length}종목
          </p>
          <div className="flex flex-wrap gap-1.5">
            {resultStocks.map((s) => (
              <span key={s.stock_code} className="inline-flex items-center gap-1 text-xs bg-th-surface-high border border-black/10 dark:border-white/10 rounded-md px-2 py-0.5">
                <span className="font-data text-th-on-muted">{s.stock_code}</span>
                <span className="text-th-on-subtle">{s.stock_name}</span>
              </span>
            ))}
          </div>
        </div>
      ) : log.error_message ? (
        <p className="text-xs text-red-400">{log.error_message}</p>
      ) : (
        <p className="text-xs text-th-on-subtle">적합 종목 없음</p>
      )}
    </div>
  )
}
RankingResultSection.propTypes = {
  log: PropTypes.object.isRequired,
}

/* ── 하드필터 결과 섹션 ── */
function HardFilterSection({ filteredStocksJson }) {
  const filtered = parseJSON(filteredStocksJson) || []
  if (filtered.length === 0) {
    return <p className="text-xs text-th-on-subtle">하드필터 제거 종목 없음</p>
  }
  return (
    <div className="space-y-1.5">
      {filtered.map((f) => (
        <div key={f.stock_code} className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-data text-th-on-muted shrink-0">{f.stock_code}</span>
          {f.stock_name && <span className="text-xs text-th-on-subtle shrink-0">{f.stock_name}</span>}
          <span className="badge bg-th-warn/10 text-amber-400 border-th-warn/20 text-xs shrink-0">{f.filter_reason}</span>
        </div>
      ))}
    </div>
  )
}
HardFilterSection.propTypes = {
  filteredStocksJson: PropTypes.string,
}

/* ── LLM 선정 결과 섹션 ── */
function SelectionSection({ selLog, filteredStocks }) {
  if (!selLog) {
    if (filteredStocks && filteredStocks.length > 0) {
      return (
        <div className="space-y-2">
          <p className="text-xs text-amber-400">
            LLM에 전달된 후보 없음 — 하드 필터에서 {filteredStocks.length}종목 전체 제거
          </p>
          <div className="space-y-1.5">
            {filteredStocks.map((f) => (
              <div key={f.stock_code} className="flex items-center gap-2 flex-wrap">
                <span className="text-xs font-data text-th-on-muted shrink-0">{f.stock_code}</span>
                {f.stock_name && <span className="text-xs text-th-on-subtle shrink-0">{f.stock_name}</span>}
                <span className="badge bg-th-warn/10 text-amber-400 border-th-warn/20 text-xs shrink-0">{f.filter_reason}</span>
              </div>
            ))}
          </div>
        </div>
      )
    }
    return <p className="text-xs text-th-on-subtle">LLM 선정 단계 미실행 — 순위/필터 단계에서 후보 없음</p>
  }

  const candidates = parseJSON(selLog.candidates) || []
  const llmResult = parseJSON(selLog.llm_result) || []
  const hasSelected = selLog.selected_code !== ''
  const hasFailed = selLog.fail_reason !== ''

  return (
    <div className="space-y-3">
      {/* 결과 요약 */}
      <div className="flex items-center gap-2 flex-wrap">
        {hasSelected ? (
          <span className="badge bg-th-growth/10 text-emerald-400 border-th-growth/20">
            ✓ {selLog.selected_code} 선정
          </span>
        ) : hasFailed ? (
          <span className="badge bg-th-loss/10 text-red-400 border-th-loss/20">선정 실패</span>
        ) : (
          <span className="badge bg-th-warn/10 text-amber-400 border-th-warn/20">적합 종목 없음</span>
        )}
        <span className="text-xs text-th-on-subtle">후보 {selLog.sent_count}종목 전달</span>
      </div>

      {hasSelected && selLog.selected_reason && (
        <p className="text-sm text-th-on-muted">{selLog.selected_reason}</p>
      )}
      {hasFailed && (
        <div className="space-y-1">
          <p className="text-xs text-th-on-subtle">Hard Rejection Rule 적용 결과</p>
          <p className="text-xs text-red-400">{selLog.fail_reason}</p>
        </div>
      )}

      {/* Claude 순위 결과 */}
      {llmResult.length > 0 && (
        <div className="space-y-1.5">
          <p className="text-xs text-th-on-subtle">Claude 순위</p>
          {llmResult.map((item, idx) => (
            <div key={item.stock_code} className="flex items-start gap-2 text-xs">
              <span className="text-th-on-subtle w-4 shrink-0">{idx + 1}.</span>
              <span className="font-data text-th-primary shrink-0">{item.stock_code}</span>
              <span className="text-th-on-muted leading-relaxed">{item.reason}</span>
            </div>
          ))}
        </div>
      )}

      {/* 전달한 후보 종목 테이블 (접기) */}
      {candidates.length > 0 && (
        <details>
          <summary className="text-xs text-th-on-subtle cursor-pointer hover:text-th-on-muted select-none">
            전달한 후보 종목 ({candidates.length}개) 상세보기
          </summary>
          <div className="mt-2 overflow-x-auto">
            <table className="text-xs w-full border-collapse">
              <thead>
                <tr className="text-th-on-subtle border-b border-black/10 dark:border-white/10">
                  <th className="text-left py-1 pr-3 font-medium">코드</th>
                  <th className="text-left py-1 pr-3 font-medium">종목명</th>
                  <th className="text-right py-1 pr-3 font-medium">현재가</th>
                  <th className="text-right py-1 pr-3 font-medium">RSI</th>
                  <th className="text-right py-1 pr-3 font-medium">5분이격</th>
                  <th className="text-right py-1 pr-3 font-medium">고가대비%</th>
                  <th className="text-right py-1 pr-3 font-medium">시가대비%</th>
                  <th className="text-right py-1 pr-3 font-medium">체결강도</th>
                  <th className="text-right py-1 font-medium">거래량증가율</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-th-outline">
                {candidates.map((c) => {
                  const hpd = c.high_price_diff
                  const opd = c.open_price_diff
                  const dm5 = c.disparity_m5
                  return (
                    <tr key={c.stock_code} className="text-th-on-muted">
                      <td className="py-1 pr-3 font-data text-th-on-surface">{c.stock_code}</td>
                      <td className="py-1 pr-3 text-th-on-muted">{c.stock_name}</td>
                      <td className="py-1 pr-3 text-right font-data">{fmt(c.current_price)}</td>
                      <td className="py-1 pr-3 text-right font-data">{fmt(c.rsi14, 1)}</td>
                      <td className={`py-1 pr-3 text-right font-data ${dm5 > 3 ? 'text-red-400' : dm5 > 1.5 ? 'text-amber-400' : ''}`}>
                        {dm5 != null && dm5 !== 0 ? dm5.toFixed(1) + '%' : '-'}
                      </td>
                      <td className={`py-1 pr-3 text-right font-data ${hpd < -3 ? 'text-[#3B82F6]' : hpd > -0.5 ? 'text-red-400' : ''}`}>
                        {hpd != null && hpd !== 0 ? hpd.toFixed(1) + '%' : '-'}
                      </td>
                      <td className={`py-1 pr-3 text-right font-data ${opd > 10 ? 'text-red-400' : opd > 3 ? 'text-amber-400' : ''}`}>
                        {opd != null && opd !== 0 ? opd.toFixed(1) + '%' : '-'}
                      </td>
                      <td className="py-1 pr-3 text-right font-data">{fmt(c.strength)}</td>
                      <td className="py-1 text-right font-data">{fmt(c.vol_incr_rate)}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </details>
      )}
    </div>
  )
}
SelectionSection.propTypes = {
  selLog: PropTypes.object,
  filteredStocks: PropTypes.array,
}

/* ── 단계 패널 ── */
function StagePanel({ step, title, badge, children }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <span className="flex items-center justify-center w-5 h-5 rounded-full bg-th-surface-high text-th-on-subtle text-xs font-semibold shrink-0">
          {step}
        </span>
        <span className="text-xs font-semibold text-th-on-muted uppercase tracking-wide">{title}</span>
        {badge}
      </div>
      <div className="ml-7">{children}</div>
    </div>
  )
}
StagePanel.propTypes = {
  step: PropTypes.number.isRequired,
  title: PropTypes.string.isRequired,
  badge: PropTypes.node,
  children: PropTypes.node,
}

/* ── 순위 로그 카드 ── */
function RankingCard({ log, selLog }) {
  const [open, setOpen] = useState(false)
  const types = parseJSON(log.ranking_types) || []
  const filteredStocks = parseJSON(log.filtered_stocks) || []
  const isOR = log.ranking_condition === 'OR'
  const hasError = log.error_message !== ''
  const noMatch = !hasError && log.intersection_count === 0
  const separator = isOR ? '|' : '+'

  return (
    <div className="bg-th-surface border border-black/10 dark:border-white/10 rounded-xl overflow-hidden">
      {/* 헤더 — 클릭 시 펼치기 */}
      <button
        className="w-full px-4 py-3 flex items-center justify-between gap-3 hover:bg-th-surface-high transition-colors text-left"
        onClick={() => setOpen((v) => !v)}
      >
        <div className="flex items-center gap-2 flex-wrap min-w-0">
          {hasError ? (
            <span className="badge bg-th-loss/10 text-red-400 border-th-loss/20 shrink-0">오류</span>
          ) : noMatch ? (
            <span className="badge bg-th-warn/10 text-amber-400 border-th-warn/20 shrink-0">적합 없음</span>
          ) : (
            <span className="badge bg-th-growth/10 text-emerald-400 border-th-growth/20 font-data shrink-0">
              {isOR ? '합집합' : '교집합'} {log.intersection_count}종목
            </span>
          )}

          {filteredStocks.length > 0 && (
            <span className="badge bg-th-warn/10 text-amber-400 border-th-warn/20 font-data shrink-0">
              하드필터 {filteredStocks.length}종목 제거
            </span>
          )}

          {selLog && (
            selLog.selected_code ? (
              <span className="badge bg-th-primary/10 text-th-primary border-th-primary/20 shrink-0">
                ✓ {selLog.selected_code}
              </span>
            ) : selLog.fail_reason ? (
              <span className="badge bg-th-loss/10 text-red-400 border-th-loss/20 shrink-0">선정 실패</span>
            ) : null
          )}

          {types.length > 0 && (
            <span className="text-xs text-th-on-subtle font-data truncate">
              [{types.map(rankingTypeKr).join(separator)}]
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <span className="text-xs text-th-on-subtle">{fmtDate(log.timestamp)}</span>
          <svg
            className={`w-4 h-4 text-th-on-subtle transition-transform ${open ? 'rotate-180' : ''}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </button>

      {/* 상세 3단계 패널 */}
      {open && (
        <div className="px-4 pb-4 border-t border-black/10 dark:border-white/10 space-y-5 pt-4">
          <StagePanel
            step={1}
            title="순위 조회"
            badge={
              <span className="text-xs text-th-on-subtle font-data">
                {log.price_min && log.price_max
                  ? log.market === 'US'
                    ? `$${Number(log.price_min).toLocaleString()}~$${Number(log.price_max).toLocaleString()}`
                    : `${Number(log.price_min).toLocaleString()}~${Number(log.price_max).toLocaleString()}원`
                  : ''}
              </span>
            }
          >
            <RankingResultSection log={log} />
          </StagePanel>

          <div className="border-t border-black/10 dark:border-white/10" />

          <StagePanel
            step={2}
            title="하드 필터"
            badge={
              filteredStocks.length > 0
                ? <span className="badge bg-th-warn/10 text-amber-400 border-th-warn/20">{filteredStocks.length}종목 제거</span>
                : <span className="badge bg-th-growth/10 text-emerald-400 border-th-growth/20">전체 통과</span>
            }
          >
            <HardFilterSection filteredStocksJson={log.filtered_stocks} />
          </StagePanel>

          <div className="border-t border-black/10 dark:border-white/10" />

          <StagePanel
            step={3}
            title="LLM 선정"
            badge={
              selLog
                ? selLog.selected_code
                  ? <span className="badge bg-th-primary/10 text-th-primary border-th-primary/20">선정 완료</span>
                  : <span className="badge bg-th-loss/10 text-red-400 border-th-loss/20">미선정</span>
                : null
            }
          >
            <SelectionSection selLog={selLog} filteredStocks={filteredStocks} />
          </StagePanel>
        </div>
      )}
    </div>
  )
}
RankingCard.propTypes = {
  log: PropTypes.object.isRequired,
  selLog: PropTypes.object,
}

/* ── 순위 유형 한글 이름 ── */
const RANKING_TYPE_LABELS = {
  volume: '거래량',
  strength: '체결강도',
  exec_count: '대량체결',
  disparity: '이격도',
}
function rankingTypeKr(type) {
  return RANKING_TYPE_LABELS[type] || type
}

/* ── 자동 새로고침 옵션 ── */
const REFRESH_OPTIONS = [
  { value: 0, label: '끄기' },
  { value: 30, label: '30초' },
  { value: 60, label: '1분' },
  { value: 180, label: '3분' },
  { value: 300, label: '5분' },
]

export default function StockLogs() {
  const { data: rankingData, loading: rankingLoading, error: rankingError, refetch } = useApi('/api/logs/ranking?limit=100')
  const { data: selectionData } = useApi('/api/logs/selection?limit=200')

  const [refreshInterval, setRefreshInterval] = useState(0)
  const intervalRef = useRef(null)

  useEffect(() => {
    if (intervalRef.current) clearInterval(intervalRef.current)
    if (refreshInterval > 0) {
      intervalRef.current = setInterval(refetch, refreshInterval * 1000)
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [refreshInterval, refetch])

  // ranking_log_id → selection log 맵
  const selByRankingId = useMemo(() => {
    const selectionLogs = selectionData?.logs || []
    const m = {}
    for (const s of selectionLogs) {
      if (s.ranking_log_id > 0) {
        m[s.ranking_log_id] = s
      }
    }
    return m
  }, [selectionData])

  const filtered = useMemo(() => {
    return rankingData?.logs || []
  }, [rankingData])

  return (
    <div className="space-y-5">
      {/* 헤더 */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold text-th-on-surface">종목 로그</h1>
          <p className="text-xs text-th-on-muted mt-0.5">순위 조회 → 하드 필터 → LLM 선정 3단계 흐름</p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={refreshInterval}
            onChange={(e) => setRefreshInterval(Number(e.target.value))}
            className="text-xs px-2 py-2 bg-th-surface border border-black/10 dark:border-white/10 rounded-lg text-th-on-muted focus:outline-none focus:ring-1 focus:ring-orange-500/50"
          >
            {REFRESH_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
          <button
            onClick={refetch}
            className="text-sm px-3 py-2 bg-th-surface hover:bg-th-surface-high border border-black/10 dark:border-white/10 rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface"
          >
            새로고침
          </button>
        </div>
      </div>

      {rankingError && (
        <div className="bg-th-loss/10 border border-th-loss/20 text-red-400 rounded-xl p-4 text-sm">{rankingError}</div>
      )}

      {rankingLoading ? (
        <p className="text-th-on-subtle text-sm">로딩 중...</p>
      ) : filtered.length === 0 ? (
        <div className="bg-th-surface border border-black/10 dark:border-white/10 rounded-xl p-12 text-center">
          <p className="text-th-on-muted font-medium">기록된 종목 로그가 없습니다</p>
          <p className="text-xs text-th-on-subtle mt-2">트레이딩 엔진이 실행되면 자동 기록됩니다.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {filtered.map((log) => (
            <RankingCard
              key={log.id}
              log={log}
              selLog={selByRankingId[log.id] || null}
            />
          ))}
        </div>
      )}
    </div>
  )
}
