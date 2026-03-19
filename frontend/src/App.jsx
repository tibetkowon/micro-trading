import { useState } from 'react'
import { Routes, Route, NavLink } from 'react-router-dom'
import PropTypes from 'prop-types'
import { ThemeProvider, useTheme } from './contexts/ThemeContext'
import Dashboard from './pages/Dashboard'
import Orders from './pages/Orders'
import Monitor from './pages/Monitor'
import ErrorLogs from './pages/ErrorLogs'
import StockLogs from './pages/StockLogs'
import Settings from './pages/Settings'

const navItems = [
  { to: '/', label: '대시보드', end: true },
  { to: '/monitor', label: '모니터' },
  { to: '/orders', label: '주문 내역' },
  { to: '/logs', label: '에러 로그' },
  { to: '/stock-logs', label: '종목 로그' },
  { to: '/settings', label: '설정' },
]

function SunIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
    </svg>
  )
}

function ThemeToggle() {
  const { isDark, toggle } = useTheme()
  return (
    <button
      onClick={toggle}
      className="p-2 rounded-lg transition-colors text-th-on-muted hover:text-th-on-surface hover:bg-th-surface-high"
      aria-label={isDark ? '라이트 모드로 전환' : '다크 모드로 전환'}
      title={isDark ? '라이트 모드' : '다크 모드'}
    >
      {isDark ? <SunIcon /> : <MoonIcon />}
    </button>
  )
}

function MobileNavLink({ to, label, end, onClick }) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onClick}
      className={({ isActive }) =>
        `block px-4 py-3 text-sm font-medium transition-colors border-b border-th-outline ${
          isActive
            ? 'bg-th-surface-high text-th-on-surface'
            : 'text-th-on-muted hover:bg-th-surface-high hover:text-th-on-surface'
        }`
      }
    >
      {label}
    </NavLink>
  )
}
MobileNavLink.propTypes = {
  to: PropTypes.string.isRequired,
  label: PropTypes.string.isRequired,
  end: PropTypes.bool,
  onClick: PropTypes.func.isRequired,
}

function AppInner() {
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div className="min-h-screen flex flex-col bg-th-bg">
      <nav className="bg-th-surface border-b border-th-outline relative">
        <div className="px-4 py-3 flex items-center justify-between">
          <span className="text-th-on-surface font-semibold tracking-tight text-sm">
            Micro Trading
          </span>

          {/* 데스크탑 링크 */}
          <div className="hidden md:flex items-center gap-0.5 flex-wrap">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  `px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-th-surface-high text-th-on-surface'
                      : 'text-th-on-muted hover:text-th-on-surface hover:bg-th-surface-high'
                  }`
                }
              >
                {item.label}
              </NavLink>
            ))}
          </div>

          <div className="flex items-center gap-1">
            <ThemeToggle />
            {/* 햄버거 (모바일) */}
            <button
              className="md:hidden text-th-on-muted hover:text-th-on-surface p-2 rounded-lg"
              onClick={() => setMenuOpen((o) => !o)}
              aria-label="메뉴 열기"
            >
              {menuOpen ? (
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              ) : (
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              )}
            </button>
          </div>
        </div>

        {/* 모바일 드롭다운 */}
        {menuOpen && (
          <div className="md:hidden absolute top-full left-0 right-0 bg-th-surface border-b border-th-outline z-50 shadow-lg">
            {navItems.map((item) => (
              <MobileNavLink
                key={item.to}
                to={item.to}
                label={item.label}
                end={item.end}
                onClick={() => setMenuOpen(false)}
              />
            ))}
          </div>
        )}
      </nav>

      <main className="flex-1 p-4 md:p-6 max-w-screen-2xl mx-auto w-full">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/monitor" element={<Monitor />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/logs" element={<ErrorLogs />} />
          <Route path="/stock-logs" element={<StockLogs />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </main>
    </div>
  )
}

export default function App() {
  return (
    <ThemeProvider>
      <AppInner />
    </ThemeProvider>
  )
}
