import { useState } from 'react'
import { Routes, Route, NavLink } from 'react-router-dom'
import PropTypes from 'prop-types'
import Dashboard from './pages/Dashboard'
import Orders from './pages/Orders'
import Monitor from './pages/Monitor'
import ErrorLogs from './pages/ErrorLogs'
import SelectionLogs from './pages/SelectionLogs'
import RankingLogs from './pages/RankingLogs'
import Settings from './pages/Settings'

const navItems = [
  { to: '/', label: '대시보드', end: true },
  { to: '/monitor', label: '모니터' },
  { to: '/orders', label: '주문 내역' },
  { to: '/logs', label: '에러 로그' },
  { to: '/selection-logs', label: '선정 로그' },
  { to: '/ranking-logs', label: '순위 조회 로그' },
  { to: '/settings', label: '설정' },
]

const desktopNavClass = ({ isActive }) =>
  `px-4 py-2 rounded text-sm font-medium transition-colors ${
    isActive
      ? 'bg-blue-600 text-white'
      : 'text-gray-400 hover:text-white hover:bg-gray-800'
  }`

function MobileNavLink({ to, label, end, onClick }) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onClick}
      className={({ isActive }) =>
        `block px-4 py-3 text-sm font-medium transition-colors border-b border-gray-800 ${
          isActive
            ? 'bg-blue-600/20 text-blue-400'
            : 'text-gray-300 hover:bg-gray-800 hover:text-white'
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

export default function App() {
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div className="min-h-screen flex flex-col">
      <nav className="bg-gray-900 border-b border-gray-800 relative">
        <div className="px-4 py-3 flex items-center justify-between">
          <span className="text-white font-bold">Micro Trading</span>

          {/* 데스크탑 링크 (md 이상) */}
          <div className="hidden md:flex items-center gap-1 flex-wrap">
            {navItems.map((item) => (
              <NavLink key={item.to} to={item.to} end={item.end} className={desktopNavClass}>
                {item.label}
              </NavLink>
            ))}
          </div>

          {/* 햄버거 버튼 (md 미만) */}
          <button
            className="md:hidden text-gray-400 hover:text-white p-2 rounded"
            onClick={() => setMenuOpen((o) => !o)}
            aria-label="메뉴 열기"
          >
            {menuOpen ? (
              <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            )}
          </button>
        </div>

        {/* 모바일 드롭다운 메뉴 */}
        {menuOpen && (
          <div className="md:hidden absolute top-full left-0 right-0 bg-gray-900 border-b border-gray-700 z-50 shadow-lg">
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

      <main className="flex-1 p-4 md:p-6">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/monitor" element={<Monitor />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/logs" element={<ErrorLogs />} />
          <Route path="/selection-logs" element={<SelectionLogs />} />
          <Route path="/ranking-logs" element={<RankingLogs />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </main>
    </div>
  )
}
