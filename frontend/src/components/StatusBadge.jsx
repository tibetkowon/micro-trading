import PropTypes from 'prop-types'

const colors = {
  FILLED:           'bg-emerald-500/15 text-emerald-400 border border-emerald-500/20',
  PARTIALLY_FILLED: 'bg-teal-500/15 text-teal-400 border border-teal-500/20',
  PENDING:          'bg-yellow-500/15 text-yellow-400 border border-yellow-500/20',
  CANCELLED:        'bg-zinc-700/50 text-zinc-400 border border-zinc-700',
  FAILED:           'bg-red-500/15 text-red-400 border border-red-500/20',
}

const labels = {
  PARTIALLY_FILLED: '부분체결',
  FILLED:           '체결',
  PENDING:          '대기',
  CANCELLED:        '취소',
  FAILED:           '실패',
}

export default function StatusBadge({ status }) {
  return (
    <span className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[status] || 'bg-zinc-700/50 text-zinc-400 border border-zinc-700'}`}>
      {labels[status] || status}
    </span>
  )
}

StatusBadge.propTypes = {
  status: PropTypes.string.isRequired,
}
