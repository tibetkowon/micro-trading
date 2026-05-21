import { useState, useEffect } from 'react'
import { apiFetch } from '../utils/api'

const adminHeaders = import.meta.env.VITE_ADMIN_API_KEY
  ? { 'X-Admin-Key': import.meta.env.VITE_ADMIN_API_KEY }
  : {}

export default function AdminDB() {
  const [tables, setTables] = useState([])
  const [selected, setSelected] = useState(null)
  const [rows, setRows] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)

  useEffect(() => {
    apiFetch('/api/admin/tables', { headers: adminHeaders })
      .then(res => res.json())
      .then(d => setTables(d.tables || []))
  }, [])

  useEffect(() => {
    if (!selected) return
    apiFetch(`/api/admin/tables/${selected}?page=${page}&limit=50`, { headers: adminHeaders })
      .then(res => res.json())
      .then(d => { setRows(d.rows ?? []); setTotal(d.total ?? 0) })
  }, [selected, page])

  return (
    <div className="card">
      <div style={{ padding: 16, borderBottom: '1px solid var(--border)' }}>
        <h1 style={{ fontSize: 18, margin: 0 }}>DB Admin</h1>
      </div>
      <div style={{ padding: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        {tables.map(t => (
          <button key={t}
            className={`btn btn-sm ${selected === t ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => { setSelected(t); setPage(1) }}>
            {t}
          </button>
        ))}
      </div>
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>{Object.keys(rows[0]).map(k => <th key={k}>{k}</th>)}</tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i}>
                  {Object.keys(rows[0]).map(k => (
                    <td key={k} className="mono" style={{ maxWidth: 260, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {String(row[k] ?? '')}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          <div style={{ padding: 12, display: 'flex', justifyContent: 'center', gap: 8, borderTop: '1px solid var(--border)' }}>
            <span className="muted">Total: {total} | Page: {page}</span>
            <button className="btn btn-outline btn-xs" onClick={() => setPage(p => Math.max(1, p - 1))}>Prev</button>
            <button className="btn btn-outline btn-xs" onClick={() => setPage(p => p + 1)}>Next</button>
          </div>
        </div>
      )}
    </div>
  )
}
