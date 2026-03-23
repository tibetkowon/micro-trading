import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

function parseTypes(str) {
  try {
    return JSON.parse(str)
  } catch {
    return []
  }
}

function parseJSON(str) {
  try { return JSON.parse(str) } catch { return null }
}


function TypeCount({ label, count }) {
  if (count === -1) return null
  return (
    <div className="flex items-center justify-between text-xs py-0.5">
      <span className="text-gray-400">{label}</span>
      <span className="font-mono text-gray-300">{count}개</span>
    </div>
  )
}

TypeCount.propTypes = {
  label: PropTypes.string.isRequired,
  count: PropTypes.number.isRequired,
}

export default function RankingLogs() {
  const { data, loading, error, refetch } = useApi('/api/logs/ranking?limit=50')

  const logs = data?.logs || []

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">순위 조회 로그</h1>
        <button
          onClick={refetch}
          className="text-sm px-3 py-1.5 bg-gray-800 hover:bg-gray-700 rounded"
        >
          새로고침
        </button>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-700 text-red-300 rounded p-4 mb-4 text-sm">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-gray-500">로딩 중...</p>
      ) : logs.length === 0 ? (
        <p className="text-gray-500">기록된 순위 조회 로그가 없습니다.</p>
      ) : (
        <div className="space-y-3">
          {logs.map((log) => {
            const types = parseTypes(log.ranking_types)
            const hasError = log.error_message !== ''
            const noMatch = !hasError && log.intersection_count === 0
            const isOR = log.ranking_condition === 'OR'
            const separator = isOR ? '|' : '+'
            const resultLabel = isOR ? '합집합' : '교집합'

            const resultStocks = parseJSON(log.result_stocks) || []

            const filteredStocks = parseJSON(log.filtered_stocks) || []

            return (
              <div
                key={log.id}
                className="bg-gray-900 border border-gray-700 rounded-lg p-4"
              >
                {/* 헤더 */}
                <div className="flex items-center justify-between gap-4 flex-wrap">
                  <div className="flex items-center gap-2 flex-wrap">
                    {log.market === 'US' ? (
                      <span className="text-xs bg-blue-900/50 text-blue-300 px-1.5 py-0.5 rounded font-semibold">미장</span>
                    ) : (
                      <span className="text-xs bg-gray-800 text-gray-400 px-1.5 py-0.5 rounded font-semibold">국장</span>
                    )}
                    {hasError ? (
                      <span className="text-xs bg-red-900/60 text-red-300 px-2 py-0.5 rounded">
                        오류
                      </span>
                    ) : noMatch ? (
                      <span className="text-xs bg-yellow-900/60 text-yellow-300 px-2 py-0.5 rounded">
                        적합 종목 없음
                      </span>
                    ) : (
                      <span className="text-xs bg-green-900/60 text-green-300 px-2 py-0.5 rounded font-mono">
                        {resultLabel} {log.intersection_count}종목
                      </span>
                    )}
                    {filteredStocks.length > 0 && (
                      <span className="text-xs bg-orange-900/50 text-orange-300 px-2 py-0.5 rounded font-mono">
                        하드필터 -{filteredStocks.length}
                      </span>
                    )}
                    {types.length > 0 && (
                      <span className="text-xs text-gray-400 font-mono">
                        [{types.join(separator)}]
                      </span>
                    )}
                    <span className="text-xs text-gray-500">
                      {log.price_min && log.price_max
                        ? log.market === 'US'
                          ? `$${Number(log.price_min).toLocaleString()}~$${Number(log.price_max).toLocaleString()}`
                          : `${Number(log.price_min).toLocaleString()}~${Number(log.price_max).toLocaleString()}원`
                        : ''}
                    </span>
                  </div>
                  <span className="text-xs text-gray-500">{fmtDate(log.timestamp)}</span>
                </div>

                {/* 에러 메시지 */}
                {hasError && (
                  <p className="text-xs text-red-400 mt-2">{log.error_message}</p>
                )}

                {/* 타입별 상세 (펼치기) */}
                {!hasError && (
                  <details className="mt-3">
                    <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-300">
                      타입별 필터 결과 보기
                    </summary>
                    <div className="mt-2 pl-2 border-l border-gray-700 space-y-0.5">
                      <TypeCount label="거래량 (volume)" count={log.volume_count} />
                      <TypeCount label="체결강도 (strength)" count={log.strength_count} />
                      <TypeCount label="대량체결 (exec_count)" count={log.exec_count_count} />
                      <TypeCount label="이격도 (disparity)" count={log.disparity_count} />
                      <div className="flex items-center justify-between text-xs py-0.5 mt-1 border-t border-gray-700 pt-1">
                        <span className="text-gray-300 font-medium">{isOR ? 'OR 합집합' : 'AND 교집합'}</span>
                        <span className={`font-mono font-bold ${log.intersection_count > 0 ? 'text-green-400' : 'text-yellow-400'}`}>
                          {log.intersection_count}개
                        </span>
                      </div>
                    </div>
                  </details>
                )}

                {/* 하드필터 제거 종목 */}
                {filteredStocks.length > 0 && (
                  <details className="mt-2">
                    <summary className="text-xs text-orange-400/80 cursor-pointer hover:text-orange-300">
                      하드필터 제거 종목 ({filteredStocks.length}개)
                    </summary>
                    <div className="mt-2 space-y-1">
                      {filteredStocks.map((s) => (
                        <div key={s.stock_code} className="flex items-center gap-2 text-xs pl-1">
                          <span className="font-mono text-gray-400 shrink-0">{s.stock_code}</span>
                          <span className="text-gray-500 shrink-0">{s.stock_name}</span>
                          <span className="text-orange-400/70">{s.filter_reason}</span>
                        </div>
                      ))}
                    </div>
                  </details>
                )}

                {/* 결과 종목 목록 */}
                {!hasError && resultStocks.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1.5">
                    {resultStocks.map((s) => (
                      <span key={s.stock_code} className="inline-flex items-center gap-1 text-xs bg-gray-800 border border-gray-700 rounded px-2 py-0.5">
                        <span className="font-mono text-gray-300">{s.stock_code}</span>
                        <span className="text-gray-500">{s.stock_name}</span>
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
