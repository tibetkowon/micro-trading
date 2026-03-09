import { useApi } from '../hooks/useApi'

function fmtDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('ko-KR')
}

function parseJSON(str) {
  try {
    return JSON.parse(str)
  } catch {
    return null
  }
}

export default function SelectionLogs() {
  const { data, loading, error, refetch } = useApi('/api/logs/selection?limit=20')

  const logs = data?.logs || []

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">LLM 종목 선정 로그</h1>
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
        <p className="text-gray-500">기록된 선정 로그가 없습니다.</p>
      ) : (
        <div className="space-y-3">
          {logs.map((log) => {
            const llmResult = parseJSON(log.llm_result) || []
            const candidates = parseJSON(log.candidates) || []
            const hasSelected = log.selected_code !== ''

            return (
              <div
                key={log.id}
                className="bg-gray-900 border border-gray-700 rounded-lg p-4"
              >
                {/* 헤더 */}
                <div className="flex items-center justify-between gap-4 flex-wrap">
                  <div className="flex items-center gap-2">
                    {hasSelected ? (
                      <span className="text-xs bg-green-900/60 text-green-300 px-2 py-0.5 rounded font-mono">
                        {log.selected_code}
                      </span>
                    ) : (
                      <span className="text-xs bg-gray-700 text-gray-400 px-2 py-0.5 rounded">
                        미체결
                      </span>
                    )}
                    <span className="text-xs text-gray-400">
                      후보 {log.sent_count}종목 전달
                    </span>
                  </div>
                  <span className="text-xs text-gray-500">{fmtDate(log.timestamp)}</span>
                </div>

                {/* 선정 이유 */}
                {hasSelected && log.selected_reason && (
                  <p className="text-sm text-gray-300 mt-2">{log.selected_reason}</p>
                )}

                {/* LLM 순위 결과 */}
                {llmResult.length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs text-gray-500 mb-1">Claude 순위 결과</p>
                    <div className="space-y-1">
                      {llmResult.map((item, idx) => (
                        <div key={item.stock_code} className="flex items-start gap-2 text-xs">
                          <span className="text-gray-600 w-4 shrink-0">{idx + 1}.</span>
                          <span className="font-mono text-blue-400 shrink-0">{item.stock_code}</span>
                          <span className="text-gray-400">{item.reason}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* 전달 종목 전체 (펼치기) */}
                {candidates.length > 0 && (
                  <details className="mt-3">
                    <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-300">
                      전달 종목 전체 ({candidates.length}개) 보기
                    </summary>
                    <div className="mt-2 overflow-x-auto">
                      <table className="text-xs w-full border-collapse">
                        <thead>
                          <tr className="text-gray-500 border-b border-gray-700">
                            <th className="text-left py-1 pr-3">순위</th>
                            <th className="text-left py-1 pr-3">코드</th>
                            <th className="text-left py-1 pr-3">종목명</th>
                            <th className="text-right py-1 pr-3">현재가</th>
                            <th className="text-right py-1 pr-3">RSI</th>
                            <th className="text-right py-1">MACD</th>
                          </tr>
                        </thead>
                        <tbody>
                          {candidates.map((c) => (
                            <tr key={c.stock_code} className="border-b border-gray-800 text-gray-400">
                              <td className="py-1 pr-3">{c.data_rank}</td>
                              <td className="py-1 pr-3 font-mono text-gray-300">{c.stock_code}</td>
                              <td className="py-1 pr-3">{c.stock_name}</td>
                              <td className="py-1 pr-3 text-right">{Number(c.current_price).toLocaleString()}</td>
                              <td className="py-1 pr-3 text-right">{c.rsi14 ? c.rsi14.toFixed(1) : '-'}</td>
                              <td className="py-1 text-right">
                                {c.macd_line != null ? c.macd_line.toFixed(2) : '-'}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </details>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
