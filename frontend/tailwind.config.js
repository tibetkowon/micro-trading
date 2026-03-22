/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Semantic theme tokens — mapped to CSS custom properties
        'th-bg':              'var(--th-bg)',
        'th-surface':         'var(--th-surface)',
        'th-surface-high':    'var(--th-surface-high)',
        'th-surface-highest': 'var(--th-surface-highest)',
        'th-surface-low':     'var(--th-surface-low)',
        'th-sidebar':         'var(--th-sidebar)',
        'th-on-surface':      'var(--th-on-surface)',
        'th-on-muted':        'var(--th-on-muted)',
        'th-on-subtle':       'var(--th-on-subtle)',
        'th-outline':         'var(--th-outline)',
        'th-primary':         'var(--th-primary)',
        'th-primary-dim':     'var(--th-primary-dim)',
        'th-growth':          'var(--th-growth)',
        'th-growth-dim':      'var(--th-growth-dim)',
        'th-loss':            'var(--th-loss)',
        'th-loss-dim':        'var(--th-loss-dim)',
        'th-warn':            'var(--th-warn)',
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        data: ['Manrope', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
