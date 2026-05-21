import { useState, useEffect } from 'react'
import { apiFetch } from '../utils/api'

export default function AdminDB() {
  const [adminKey, setAdminKey] = useState('')
  const [authenticated, setAuthenticated] = useState(false)
  const [authError, setAuthError] = useState(false)
  const [tables, setTables] = useState([])
  const [selected, setSelected] = useState(null)
  const [rows, setRows] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)

  const headers = { 'X-Admin-Key': adminKey }

  function handleAuth(e) {
    e.preventDefault()
    apiFetch('/api/admin/tables', { headers })
      .then(res => {
        if (res.status === 401) { setAuthError(true); return }
        return res.json()
      })
      .then(d => {
        if (!d) return
        setTables(d.tables || [])
        setAuthenticated(true)
        setAuthError(false)
      })
  }

  useEffect(() => {
    if (!authenticated || !selected) return
    apiFetch(`/api/admin/tables/${selected}?page=${page}&limit=50`, { headers })
      .then(res => res.json())
      .then(d => { setRows(d.rows ?? []); setTotal(d.total ?? 0) })
  }, [selected, page, authenticated])

  if (!authenticated) {
    return (
      <div className="card" style={{ maxWidth: 360, margin: '80px auto', padding: 32 }}>
        <h2 style={{ marginBottom: 16 }}>DB Admin</h2>
        <form onSubmit={handleAuth} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <input
            type="password"
            placeholder="Admin Key"
            value={adminKey}
            onChange={e => setAdminKey(e.target.value)}
            className="input"
            autoFocus
          />
          {authError && <p style={{ color: 'var(--red)', fontSize: 13, margin: 0 }}>인증 실패. 키를 확인하세요.</p>}
          <button type="submit" className="btn btn-primary">접속</button>
        </form>
      </div>
    )
  }

  return (
    <div className="card">
      <div style={{ padding: 16, borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between' }}>
        <h1 style={{ fontSize: 18, margin: 0 }}>DB Admin</h1>
        <button className="btn btn-outline btn-sm" onClick={() => setAuthenticated(false)}>로그아웃</button>
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
