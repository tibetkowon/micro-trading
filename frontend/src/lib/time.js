export function fmtTs(ts) {
  if (!ts) return '—'
  const d = ts instanceof Date ? ts : new Date(ts)
  return d.toLocaleString('sv-SE', { timeZone: 'Asia/Seoul' }).replace('T', ' ')
}
