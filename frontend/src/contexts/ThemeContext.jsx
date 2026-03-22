import { createContext, useContext, useState, useEffect } from 'react'
import PropTypes from 'prop-types'

const ThemeContext = createContext(null)

export function ThemeProvider({ children }) {
  const [isDark, setIsDark] = useState(() => {
    const saved = localStorage.getItem('theme')
    if (saved) return saved === 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    const root = document.documentElement
    if (isDark) {
      root.classList.add('dark')
    } else {
      root.classList.remove('dark')
    }
    localStorage.setItem('theme', isDark ? 'dark' : 'light')
  }, [isDark])

  function toggle() {
    setIsDark((v) => !v)
  }

  return (
    <ThemeContext.Provider value={{ isDark, toggle }}>
      {children}
    </ThemeContext.Provider>
  )
}

ThemeProvider.propTypes = { children: PropTypes.node }

// eslint-disable-next-line react-refresh/only-export-components
export function useTheme() {
  return useContext(ThemeContext)
}
