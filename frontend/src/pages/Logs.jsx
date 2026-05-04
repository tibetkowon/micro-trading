import { useState, useEffect, useRef } from 'react'
import { collection, query, orderBy, limit, onSnapshot, getDocs } from 'firebase/firestore'
import { apiFetch } from '../utils/api'
import { Badge, Toggle } from '../components/shared'
import { db, fmtTs } from '../lib/firebase'

const LEVEL_COLOR = { ERROR: 'var(--accent)', WARN: 'var(--yellow)', INFO: 'var(--text-muted)' }
const LEVEL_BADGE = { ERROR: 'red', WARN: 'yellow', INFO: 'gray' }

function fetchKisLogs(setKisLogs, setKisLoading) {
  setKisLoading(true)
  const q = query(collection(db, 'kis_api_logs'), orderBy('timestamp', 'desc'), limit(100))
  getDocs(q)
    .then(snap => setKisLogs(snap.docs.map(d => ({ _docId: d.id, ...d.data() }))))
    .finally(() => setKisLoading(false))
}

export default function Logs() {
  const [tab, setTab] = useState('KIS API')
  const [expandedId, setExpandedId] = useState(null)
  const [selectedIds, setSelectedIds] = useState(new Set())
  const [levelFilter, setLevelFilter] = useState('ALL')
  const [sourceFilter, setSourceFilter] = useState('ALL')
  const [autoScroll, setAutoScroll] = useState(true)
  const logRef = useRef(null)

  const [kisLogs, setKisLogs] = useState([])
  const [kisLoading, setKisLoading] = useState(true)
  const [serviceLogs, setServiceLogs] = useState([])
  const [svcLoading, setSvcLoading] = useState(true)

  useEffect(() => {
    if (tab !== 'KIS API') return
    fetchKisLogs(setKisLogs, setKisLoading)
  }, [tab])

  useEffect(() => {
    if (tab !== '서비스 로그') return
    setSvcLoading(true)
    const q = query(collection(db, 'service_logs'), orderBy('timestamp', 'desc'), limit(200))
    const unsub = onSnapshot(q, snap => {
      setServiceLogs(snap.docs.map(d => ({ _docId: d.id, ...d.data() })).reverse())
      setSvcLoading(false)
    })
    return unsub
  }, [tab])

  useEffect(() => {
    if (autoScroll && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight
    }
  }, [serviceLogs, autoScroll])

  function toggleSelect(id, e) {
    e.stopPropagation()
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  function toggleSelectAll() {
    setSelectedIds(prev =>
      prev.size === kisLogs.length ? new Set() : new Set(kisLogs.map(l => l.id))
    )
  }

  async function deleteSelected() {
    await Promise.all([...selectedIds].map(id =>
      apiFetch(`/api/logs/kis/${id}`, { method: 'DELETE' })
    ))
    setSelectedIds(new Set())
    fetchKisLogs(setKisLogs, setKisLoading)
  }

  const filteredService = serviceLogs.filter(l => {
    if (levelFilter !== 'ALL' && l.level !== levelFilter) return false
    if (sourceFilter !== 'ALL' && l.source !== sourceFilter) return false
    return true
  })

  return (
    <div>
      <div className="tab-bar">
        {['KIS API', '서비스 로그'].map(t => (
          <div key={t} className={`tab-item ${tab === t ? 'active' : ''}`} onClick={() => setTab(t)}>{t}</div>
        ))}
      </div>

      {tab === 'KIS API' && (
        <div>
          <div style={{ display: 'flex', gap: 12, marginBottom: 14, alignItems: 'center' }}>
            <button className="btn btn-danger btn-sm" style={{ marginLeft: 'auto' }}
              disabled={selectedIds.size === 0}
              onClick={deleteSelected}>
              선택 삭제 {selectedIds.size > 0 && `(${selectedIds.size})`}
            </button>
          </div>
          <div className="card">
            {kisLoading ? (
              <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)' }}>로딩 중...</div>
            ) : (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th style={{ width: 32 }}>
                        <input type="checkbox"
                          checked={selectedIds.size === kisLogs.length && kisLogs.length > 0}
                          onChange={toggleSelectAll}
                          style={{ cursor: 'pointer', accentColor: 'var(--accent)' }} />
                      </th>
                      <th>시각</th>
                      <th>엔드포인트</th>
                      <th>에러코드</th>
                      <th>메시지 요약</th>
                    </tr>
                  </thead>
                  <tbody>
                    {kisLogs.length === 0 ? (
                      <tr>
                        <td colSpan={5} style={{ textAlign: 'center', padding: 32, color: 'var(--text-muted)' }}>
                          KIS API 로그가 없습니다
                        </td>
                      </tr>
                    ) : kisLogs.map(l => (
                      <tr key={l._docId} style={{ cursor: 'pointer', background: selectedIds.has(l.id) ? 'rgba(234,108,16,0.07)' : '' }}>
                        <td onClick={e => toggleSelect(l.id, e)} style={{ cursor: 'default' }}>
                          <input type="checkbox" checked={selectedIds.has(l.id)} onChange={() => {}}
                            style={{ cursor: 'pointer', accentColor: 'var(--accent)' }} />
                        </td>
                        <td className="mono" style={{ fontSize: 11, whiteSpace: 'nowrap' }}
                          onClick={() => setExpandedId(expandedId === l._docId ? null : l._docId)}>
                          {fmtTs(l.timestamp)}
                        </td>
                        <td className="mono" style={{ fontSize: 11, color: 'var(--text-muted)', maxWidth: 280 }}
                          onClick={() => setExpandedId(expandedId === l._docId ? null : l._docId)}>
                          {l.endpoint}
                        </td>
                        <td onClick={() => setExpandedId(expandedId === l._docId ? null : l._docId)}>
                          {l.error_code
                            ? <Badge color="red">{l.error_code}</Badge>
                            : <span className="muted" style={{ fontSize: 11 }}>—</span>}
                        </td>
                        <td style={{ fontSize: 12, color: l.error_code ? 'var(--red)' : 'var(--text)' }}
                          onClick={() => setExpandedId(expandedId === l._docId ? null : l._docId)}>
                          <span style={{ color: 'var(--text-dim)', fontSize: 10, marginRight: 6 }}>
                            {expandedId === l._docId ? '▼' : '▶'}
                          </span>
                          {l.error_msg}
                        </td>
                      </tr>
                    ))}
                    {kisLogs.map(l => expandedId === l._docId ? (
                      <tr key={`expand-${l._docId}`}>
                        <td colSpan={5} style={{ padding: '0 12px 12px 40px' }}>
                          <div className="code-block">{l.raw_response || '(응답 없음)'}</div>
                        </td>
                      </tr>
                    ) : null)}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {tab === '서비스 로그' && (
        <div>
          <div style={{ display: 'flex', gap: 10, marginBottom: 14, alignItems: 'center', flexWrap: 'wrap' }}>
            <div className="chip-group">
              {['ALL', 'ERROR', 'WARN', 'INFO'].map(l => (
                <span key={l}
                  className={`chip ${levelFilter === l ? 'active' : ''}`}
                  onClick={() => setLevelFilter(l)}
                  style={l !== 'ALL' && levelFilter !== l ? { borderColor: LEVEL_COLOR[l], color: LEVEL_COLOR[l] } : {}}>
                  {l}
                </span>
              ))}
            </div>
            <div className="chip-group">
              {['ALL', 'TRADER', 'MONITOR', 'SYSTEM'].map(s => (
                <span key={s}
                  className={`chip ${sourceFilter === s ? 'active' : ''}`}
                  onClick={() => setSourceFilter(s)}>
                  {s}
                </span>
              ))}
            </div>
            <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 }}>
              <Toggle checked={autoScroll} onChange={setAutoScroll} />
              <span className="muted" style={{ fontSize: 12 }}>자동 스크롤</span>
            </div>
          </div>
          <div className="log-terminal" ref={logRef}>
            {svcLoading ? (
              <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>로딩 중...</div>
            ) : filteredService.length === 0 ? (
              <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>로그가 없습니다</div>
            ) : filteredService.map(l => (
              <div key={l._docId} style={{ display: 'flex', gap: 12, padding: '3px 0', borderBottom: '1px solid rgba(255,255,255,0.03)' }}>
                <span className="muted" style={{ fontSize: 11, whiteSpace: 'nowrap' }}>{fmtTs(l.timestamp)}</span>
                <span style={{ width: 44, flexShrink: 0 }}><Badge color={LEVEL_BADGE[l.level] || 'gray'}>{l.level}</Badge></span>
                <span className="muted" style={{ fontSize: 11, width: 60, flexShrink: 0 }}>{l.source}</span>
                <span style={{ fontSize: 12, color: LEVEL_COLOR[l.level] || 'var(--text)' }}>{l.message}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
