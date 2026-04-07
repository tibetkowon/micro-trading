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
import StockList from './pages/StockList'
import TradeReports from './pages/TradeReports'
import DailyReports from './pages/DailyReports'

const navItems = [
  { to: '/', label: '대시보드', end: true, icon: 'dashboard' },
  { to: '/monitor', label: '모니터', icon: 'monitor_heart' },
  { to: '/orders', label: '주문 내역', icon: 'receipt_long' },
  { to: '/reports/trades', label: '거래 리포트', icon: 'analytics' },
  { to: '/reports/daily', label: '일별 리포트', icon: 'bar_chart' },
  { to: '/logs', label: '에러 로그', icon: 'report' },
  { to: '/stock-logs', label: '종목 로그', icon: 'candlestick_chart' },
  { to: '/stock-list', label: '종목 목록', icon: 'format_list_bulleted' },
  { to: '/settings', label: '설정', icon: 'settings' },
]

function NavItem({ to, label, end, icon, onClick }) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onClick}
      className={({ isActive }) =>
        `flex items-center gap-3 px-4 py-2.5 rounded-r-xl transition-all duration-150 ${
          isActive
            ? 'text-orange-500 bg-th-surface border-l-4 border-orange-500 font-semibold translate-x-0'
            : 'text-th-on-muted hover:text-th-on-surface hover:bg-th-surface border-l-4 border-transparent'
        }`
      }
    >
      <span className="material-symbols-outlined text-[20px] shrink-0">{icon}</span>
      <span className="text-xs uppercase tracking-widest font-medium">{label}</span>
    </NavLink>
  )
}
NavItem.propTypes = {
  to: PropTypes.string.isRequired,
  label: PropTypes.string.isRequired,
  end: PropTypes.bool,
  icon: PropTypes.string.isRequired,
  onClick: PropTypes.func,
}

function Sidebar({ onNavigate }) {
  return (
    <aside className="fixed left-0 top-0 h-full w-64 z-40 bg-th-sidebar flex flex-col py-8 px-0">
      {/* 로고 */}
      <div className="px-8 mb-10">
        <span className="text-orange-500 font-bold text-lg tracking-tight">Micro</span>
        <span className="text-th-on-surface font-bold text-lg tracking-tight"> Trading</span>
        <p className="text-th-on-subtle text-[10px] uppercase tracking-widest mt-0.5">AI Auto Trader</p>
      </div>

      {/* 네비게이션 */}
      <nav className="flex flex-col gap-0.5 flex-1 pr-4">
        {navItems.map((item) => (
          <NavItem
            key={item.to}
            to={item.to}
            label={item.label}
            end={item.end}
            icon={item.icon}
            onClick={onNavigate}
          />
        ))}
      </nav>

      {/* 하단 */}
      <div className="px-8 pt-6 border-t border-black/5 dark:border-white/5 space-y-3">
        <ThemeToggle />
        <div>
          <p className="text-th-on-subtle text-[10px] uppercase tracking-widest">KIS API</p>
          <p className="text-th-on-muted text-xs mt-0.5">Korea Investment</p>
        </div>
      </div>
    </aside>
  )
}
Sidebar.propTypes = { onNavigate: PropTypes.func }

function ThemeToggle() {
  const { isDark, toggle } = useTheme()
  return (
    <button
      onClick={toggle}
      className="flex items-center gap-2 text-th-on-muted hover:text-th-on-surface transition-colors w-full"
      title={isDark ? '라이트 모드로 전환' : '다크 모드로 전환'}
    >
      <span className="material-symbols-outlined text-[18px]">
        {isDark ? 'light_mode' : 'dark_mode'}
      </span>
      <span className="text-[11px] uppercase tracking-widest">{isDark ? 'Light Mode' : 'Dark Mode'}</span>
    </button>
  )
}

function AppInner() {
  const [drawerOpen, setDrawerOpen] = useState(false)

  function closeDrawer() { setDrawerOpen(false) }

  return (
    <div className="min-h-screen bg-th-bg">
      {/* 데스크탑 사이드바 */}
      <div className="hidden md:block">
        <Sidebar />
      </div>

      {/* 모바일 상단바 */}
      <div className="md:hidden fixed top-0 left-0 right-0 z-50 flex items-center gap-3 px-4 py-3 bg-th-sidebar border-b border-black/5 dark:border-white/5">
        <button
          onClick={() => setDrawerOpen((o) => !o)}
          className="text-th-on-muted hover:text-th-on-surface p-1.5 rounded-lg hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
          aria-label="메뉴 열기"
        >
          <span className="material-symbols-outlined text-[22px]">
            {drawerOpen ? 'close' : 'menu'}
          </span>
        </button>
        <div>
          <span className="text-orange-500 font-bold text-sm">Micro</span>
          <span className="text-th-on-surface font-bold text-sm"> Trading</span>
        </div>
      </div>

      {/* 모바일 드로어 */}
      {drawerOpen && (
        <>
          <div
            className="md:hidden fixed inset-0 z-40 bg-black/60"
            onClick={closeDrawer}
          />
          <div className="md:hidden fixed top-0 left-0 h-full w-64 z-50 bg-th-sidebar flex flex-col py-8 px-0 shadow-2xl">
            <div className="px-8 mb-10 mt-2">
              <span className="text-orange-500 font-bold text-lg tracking-tight">Micro</span>
              <span className="text-th-on-surface font-bold text-lg tracking-tight"> Trading</span>
            </div>
            <nav className="flex flex-col gap-0.5 flex-1 pr-4">
              {navItems.map((item) => (
                <NavItem
                  key={item.to}
                  to={item.to}
                  label={item.label}
                  end={item.end}
                  icon={item.icon}
                  onClick={closeDrawer}
                />
              ))}
            </nav>
            <div className="px-8 pt-6 border-t border-black/5 dark:border-white/5">
              <ThemeToggle />
            </div>
          </div>
        </>
      )}

      {/* 메인 콘텐츠 */}
      <main className="md:ml-64 min-h-screen pt-14 md:pt-0">
        <div className="p-4 md:p-8 max-w-screen-xl mx-auto">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/monitor" element={<Monitor />} />
            <Route path="/orders" element={<Orders />} />
            <Route path="/reports/trades" element={<TradeReports />} />
            <Route path="/reports/daily" element={<DailyReports />} />
            <Route path="/logs" element={<ErrorLogs />} />
            <Route path="/stock-logs" element={<StockLogs />} />
            <Route path="/stock-list" element={<StockList />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </div>
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
