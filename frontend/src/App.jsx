import { useState, useEffect } from 'react'
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Positions from './pages/Positions'
import Orders from './pages/Orders'
import TradeReports from './pages/TradeReports'
import DailyReports from './pages/DailyReports'
import Settings from './pages/Settings'
import AdminDB from './pages/AdminDB'

const SCREEN_TITLES = {
  '/':               '대시보드',
  '/positions':      '포지션',
  '/orders':         '주문 내역',
  '/reports/trades': '리포트 › 거래별',
  '/reports/daily':  '리포트 › 일별',
  '/settings':       '설정',
  '/admin':          'DB Admin',
}

const SIDEBAR_ITEMS = [
  { path: '/',          icon: '◈', label: '대시보드' },
  { path: '/positions', icon: '⬡', label: '포지션' },
  { path: '/orders',    icon: '≡', label: '주문 내역' },
]
const SIDEBAR_REPORT_ITEMS = [
  { path: '/reports/trades', label: '거래별' },
  { path: '/reports/daily',  label: '일별' },
]
const SIDEBAR_BOTTOM_ITEMS = [
  { path: '/settings', icon: '◎', label: '설정' },
  { path: '/admin',    icon: '▦', label: 'DB Admin' },
]

const BOTTOM_TABS = [
  { path: '/',          icon: '◈', label: '대시보드' },
  { path: '/positions', icon: '⬡', label: '포지션' },
  { path: '/orders',    icon: '≡', label: '주문' },
  { path: '/settings',  icon: '◎', label: '설정' },
  { id: '__more',       icon: '☰', label: '더보기' },
]

const DRAWER_ITEMS = [
  { path: '/reports/trades', icon: '▦', label: '리포트 — 거래별' },
  { path: '/reports/daily',  icon: '▦', label: '리포트 — 일별' },
]

const MORE_PATHS = ['/reports/trades', '/reports/daily']

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const [lightMode, setLightMode] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [time, setTime] = useState(new Date())

  useEffect(() => {
    document.body.classList.toggle('light-mode', lightMode)
  }, [lightMode])

  useEffect(() => {
    const t = setInterval(() => setTime(new Date()), 1000)
    return () => clearInterval(t)
  }, [])

  const path = location.pathname
  const title = SCREEN_TITLES[path] || '대시보드'
  const moreActive = MORE_PATHS.includes(path)

  const kstStr = time.toLocaleString('ko-KR', {
    timeZone: 'Asia/Seoul',
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  })

  function goTo(target) {
    if (target === '__more') { setDrawerOpen(true); return }
    navigate(target)
    setDrawerOpen(false)
  }

  return (
    <div className="app">
      {/* ── Desktop Sidebar ── */}
      <nav className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
        <div className="sidebar-header">
          {!collapsed && <div className="sidebar-title">micro<span>-trader</span></div>}
        </div>

        <div className="sidebar-nav">
          {!collapsed ? (
            <>
              {SIDEBAR_ITEMS.map(item => (
                <div key={item.path}
                  className={`nav-item ${path === item.path ? 'active' : ''}`}
                  onClick={() => navigate(item.path)}>
                  <span className="nav-icon">{item.icon}</span>
                  <span className="nav-label">{item.label}</span>
                </div>
              ))}

              <div className="nav-section-label">리포트</div>
              <div className="nav-sub">
                {SIDEBAR_REPORT_ITEMS.map(item => (
                  <div key={item.path}
                    className={`nav-item ${path === item.path ? 'active' : ''}`}
                    onClick={() => navigate(item.path)}>
                    <span className="nav-icon" style={{ fontSize: 12 }}>—</span>
                    <span className="nav-label">{item.label}</span>
                  </div>
                ))}
              </div>

              {SIDEBAR_BOTTOM_ITEMS.map(item => (
                <div key={item.path}
                  className={`nav-item ${path === item.path ? 'active' : ''}`}
                  onClick={() => navigate(item.path)}>
                  <span className="nav-icon">{item.icon}</span>
                  <span className="nav-label">{item.label}</span>
                </div>
              ))}
            </>
          ) : (
            [...SIDEBAR_ITEMS,
             ...SIDEBAR_REPORT_ITEMS.map(i => ({ ...i, icon: '▦' })),
             ...SIDEBAR_BOTTOM_ITEMS,
            ].map(item => (
              <div key={item.path}
                className={`nav-item ${path === item.path ? 'active' : ''}`}
                style={{ justifyContent: 'center', padding: '12px' }}
                onClick={() => navigate(item.path)}>
                <span className="nav-icon" style={{ fontSize: 18 }}>{item.icon}</span>
              </div>
            ))
          )}
        </div>

        <div className="sidebar-footer">
          <button className="collapse-btn" onClick={() => setCollapsed(c => !c)}>
            {collapsed ? '→' : '←'}
          </button>
          {!collapsed && (
            <button className="theme-btn" onClick={() => setLightMode(l => !l)}>
              {lightMode ? 'DARK' : 'LIGHT'}
            </button>
          )}
        </div>
      </nav>

      {/* ── Main ── */}
      <div className="main">
        {/* Desktop topbar */}
        <div className="topbar">
          <div className="topbar-title">{title}</div>
          <div className="topbar-time">KST {kstStr}</div>
          {collapsed && (
            <button className="theme-btn" style={{ marginLeft: 8 }} onClick={() => setLightMode(l => !l)}>
              {lightMode ? 'DARK' : 'LIGHT'}
            </button>
          )}
        </div>

        {/* Mobile topbar */}
        <div className="mobile-topbar">
          <div className="mobile-topbar-brand">micro<span>-trader</span></div>
          <div className="mobile-topbar-title">{title}</div>
          <button className="theme-btn" onClick={() => setLightMode(l => !l)}>
            {lightMode ? 'DARK' : 'LIGHT'}
          </button>
        </div>

        <div className="content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/positions" element={<Positions />} />
            <Route path="/orders" element={<Orders />} />
            <Route path="/reports/trades" element={<TradeReports />} />
            <Route path="/reports/daily" element={<DailyReports />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/admin" element={<AdminDB />} />
            <Route path="*" element={<Dashboard />} />
          </Routes>
        </div>
      </div>

      {/* ── Mobile bottom tab bar ── */}
      <div className="bottom-tabbar">
        <div className="bottom-tabbar-inner">
          {BOTTOM_TABS.map(tab => {
            const isActive = tab.id === '__more' ? moreActive : path === tab.path
            return (
              <button key={tab.id || tab.path}
                className={`bottom-tab ${isActive ? 'active' : ''}`}
                onClick={() => goTo(tab.id || tab.path)}>
                <span className="bottom-tab-icon">{tab.icon}</span>
                <span className="bottom-tab-label">{tab.label}</span>
              </button>
            )
          })}
        </div>
      </div>

      {/* ── Mobile drawer ── */}
      {drawerOpen && (
        <div className="drawer-overlay" onClick={() => setDrawerOpen(false)}>
          <div className="drawer" onClick={e => e.stopPropagation()}>
            <div className="drawer-handle"></div>
            {DRAWER_ITEMS.map(item => (
              <div key={item.path}
                className={`drawer-item ${path === item.path ? 'active' : ''}`}
                onClick={() => goTo(item.path)}>
                <span className="drawer-item-icon">{item.icon}</span>
                <span>{item.label}</span>
              </div>
            ))}
            <div style={{ borderTop: '1px solid var(--border)', margin: '8px 0' }}></div>
            <div className="drawer-item" onClick={() => { setLightMode(l => !l); setDrawerOpen(false) }}>
              <span className="drawer-item-icon">{lightMode ? '○' : '◐'}</span>
              <span>{lightMode ? '다크 모드' : '라이트 모드'} 전환</span>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
