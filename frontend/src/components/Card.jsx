import PropTypes from 'prop-types'

export default function Card({ title, value, sub, className = '' }) {
  return (
    <div className={`bg-zinc-900 border border-zinc-800 rounded-xl p-5 ${className}`}>
      <p className="text-xs text-zinc-500 mb-1.5">{title}</p>
      <p className="text-2xl font-semibold text-white">{value}</p>
      {sub && <p className="text-sm text-zinc-500 mt-1">{sub}</p>}
    </div>
  )
}

Card.propTypes = {
  title: PropTypes.string.isRequired,
  value: PropTypes.string,
  sub: PropTypes.string,
  className: PropTypes.string,
}
