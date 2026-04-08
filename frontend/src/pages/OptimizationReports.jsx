import { useState } from 'react'
import PropTypes from 'prop-types'
import { useApi } from '../hooks/useApi'

const CATEGORY_LABEL = { settings: '설정 변경', feature: '기능 요청' }
const STATUS_STYLE = {
  PENDING: 'bg-yellow-500/10 text-yellow-400',
  APPLIED: 'bg-emerald-500/10 text-emerald-400',
  REJECTED: 'bg-white/5 text-th-on-muted',
}
const STATUS_LABEL = { PENDING: '대기', APPLIED: '적용됨', REJECTED: '무시됨' }

function StatusBadge({ status }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${STATUS_STYLE[status] || STATUS_STYLE.PENDING}`}>
      {STATUS_LABEL[status] || status}
    </span>
  )
}
StatusBadge.propTypes = { status: PropTypes.string }

function CategoryBadge({ category }) {
  const style = category === 'settings'
    ? 'bg-blue-500/10 text-blue-400'
    : 'bg-purple-500/10 text-purple-400'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${style}`}>
      {CATEGORY_LABEL[category] || category}
    </span>
  )
}
CategoryBadge.propTypes = { category: PropTypes.string }

function SuggestionCard({ suggestion, date, onAction }) {
  const [loading, setLoading] = useState(false)

  async function handleAction(action) {
    setLoading(true)
    try {
      const res = await fetch(
        `/api/reports/optimization/${date}/suggestions/${suggestion.id}/${action}`,
        { method: 'POST' }
      )
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        alert(`오류: ${body.error || res.status}`)
      } else {
        onAction()
      }
    } finally {
      setLoading(false)
    }
  }

  const isPending = suggestion.status === 'PENDING'

  return (
    <div className="p-4 rounded-lg bg-black/5 dark:bg-white/5 space-y-2">
      <div className="flex items-center gap-2 flex-wrap">
        <CategoryBadge category={suggestion.category} />
        <StatusBadge status={suggestion.status} />
        {suggestion.key && (
          <code className="text-xs bg-black/10 dark:bg-white/10 px-1.5 py-0.5 rounded font-mono text-th-on-muted">
            {suggestion.key}
          </code>
        )}
        {suggestion.name && (
          <span className="text-sm font-semibold text-th-on-surface">{suggestion.name}</span>
        )}
      </div>

      {suggestion.category === 'settings' && suggestion.current_value !== '' && (
        <div className="flex items-center gap-2 text-sm">
          <span className="text-th-on-muted line-through">{suggestion.current_value}</span>
          <span className="material-symbols-outlined text-[16px] text-orange-400">arrow_forward</span>
          <span className="font-semibold text-orange-400">{suggestion.suggested_value}</span>
        </div>
      )}

      <p className="text-xs text-th-on-muted leading-relaxed">{suggestion.comment}</p>

      {isPending && (
        <div className="flex gap-2 pt-1">
          <button
            onClick={() => handleAction('apply')}
            disabled={loading || suggestion.category === 'feature'}
            className="text-xs px-3 py-1.5 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded font-medium transition-colors"
          >
            {suggestion.category === 'feature' ? '적용 불가 (기능 개발 필요)' : '적용'}
          </button>
          <button
            onClick={() => handleAction('reject')}
            disabled={loading}
            className="text-xs px-3 py-1.5 bg-white/5 hover:bg-white/10 disabled:opacity-40 text-th-on-muted rounded font-medium transition-colors"
          >
            무시
          </button>
        </div>
      )}
    </div>
  )
}
SuggestionCard.propTypes = {
  suggestion: PropTypes.object,
  date: PropTypes.string,
  onAction: PropTypes.func,
}

function OptimizationCard({ report, onRefetch }) {
  const [tab, setTab] = useState('all')

  let suggestions = []
  try {
    suggestions = JSON.parse(report.suggestions) || []
  } catch { /* ignore */ }

  const filtered = tab === 'all'
    ? suggestions
    : suggestions.filter((s) => s.category === tab)

  const pendingCount = suggestions.filter((s) => s.status === 'PENDING').length

  return (
    <div className="bg-th-surface border border-black/10 dark:border-white/10 rounded-xl p-5 space-y-4">
      {/* 헤더 */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div>
          <h2 className="font-bold text-base">{report.date}</h2>
          <p className="text-xs text-th-on-muted mt-0.5">
            적용 모드: <span className="font-medium">{report.apply_mode_snapshot === 'all_auto' ? '전체 자동' : '전체 수동'}</span>
            {pendingCount > 0 && <span className="ml-2 text-yellow-400">({pendingCount}건 대기 중)</span>}
          </p>
        </div>
      </div>

      {/* 전체 평가 */}
      {report.overall_assessment && (
        <div className="p-3 rounded-lg bg-orange-500/5 border border-orange-500/20 text-sm text-th-on-surface leading-relaxed">
          {report.overall_assessment}
        </div>
      )}

      {/* 탭 */}
      <div className="flex gap-1">
        {[
          { key: 'all', label: `전체 (${suggestions.length})` },
          { key: 'settings', label: `설정 변경 (${suggestions.filter((s) => s.category === 'settings').length})` },
          { key: 'feature', label: `기능 요청 (${suggestions.filter((s) => s.category === 'feature').length})` },
        ].map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`text-xs px-3 py-1.5 rounded-lg transition-colors ${
              tab === key
                ? 'bg-orange-500 text-white font-medium'
                : 'bg-black/5 dark:bg-white/5 text-th-on-muted hover:bg-black/10 dark:hover:bg-white/10'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {/* 제안 목록 */}
      {filtered.length === 0 ? (
        <p className="text-sm text-th-on-muted">해당 카테고리의 제안이 없습니다.</p>
      ) : (
        <div className="space-y-3">
          {filtered.map((s) => (
            <SuggestionCard
              key={s.id}
              suggestion={s}
              date={report.date}
              onAction={onRefetch}
            />
          ))}
        </div>
      )}
    </div>
  )
}
OptimizationCard.propTypes = {
  report: PropTypes.object,
  onRefetch: PropTypes.func,
}

export default function OptimizationReports() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [analyzing, setAnalyzing] = useState(false)

  const params = new URLSearchParams()
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  params.set('limit', '20')

  const { data, loading, error, refetch } = useApi(`/api/reports/optimization?${params}`)
  const reports = data?.reports || []

  async function handleAnalyze() {
    setAnalyzing(true)
    try {
      const res = await fetch('/api/reports/optimization/analyze', { method: 'POST' })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        alert('분석 요청 실패: ' + (body.error || res.status))
      } else {
        alert('분석을 시작했습니다. 잠시 후 새로고침 해주세요.')
      }
    } finally {
      setAnalyzing(false)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6 flex-wrap gap-3">
        <h1 className="text-xl font-bold">AI 개선 제안</h1>
        <div className="flex gap-2">
          <button
            onClick={handleAnalyze}
            disabled={analyzing}
            className="text-sm px-3 py-1.5 bg-orange-500 hover:bg-orange-600 disabled:opacity-50 text-white rounded font-medium"
          >
            {analyzing ? '분석 요청 중...' : '오늘 분석 실행'}
          </button>
          <button onClick={refetch} className="text-sm px-3 py-1.5 bg-th-surface hover:bg-th-surface-high rounded text-th-on-muted hover:text-th-on-surface transition-colors">
            새로고침
          </button>
        </div>
      </div>

      {/* 날짜 필터 */}
      <div className="flex flex-wrap gap-3 mb-6">
        <div className="flex items-center gap-2 text-sm text-th-on-muted">
          <span>기간</span>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="bg-th-surface border border-black/10 dark:border-white/10 rounded px-2 py-1 text-th-on-surface"
          />
          <span>~</span>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="bg-th-surface border border-black/10 dark:border-white/10 rounded px-2 py-1 text-th-on-surface"
          />
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-700 text-red-300 rounded p-4 mb-4 text-sm">{error}</div>
      )}

      {loading ? (
        <p className="text-th-on-muted">로딩 중...</p>
      ) : reports.length === 0 ? (
        <div className="text-center py-12 space-y-3">
          <span className="material-symbols-outlined text-4xl text-th-on-subtle">auto_fix_high</span>
          <p className="text-th-on-muted text-sm">
            AI 개선 제안이 없습니다.<br />
            장 마감 후 자동 생성되거나, &ldquo;오늘 분석 실행&rdquo; 버튼으로 수동 실행할 수 있습니다.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {reports.map((r) => (
            <OptimizationCard key={r.id} report={r} onRefetch={refetch} />
          ))}
        </div>
      )}
    </div>
  )
}
