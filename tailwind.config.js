/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './index-vite.html',
    './src/**/*.{js,ts,jsx,tsx}',
    './static/**/*.js',
  ],
  theme: {
    extend: {
      colors: {
        // WebClaw custom colors
        'webclaw': {
          'dark': '#0f172a',
          'darker': '#020617',
          'panel': '#1e293b',
          'border': '#334155',
        },
      },
      fontFamily: {
        mono: ['Fira Code', 'Monaco', 'Consolas', 'monospace'],
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
    },
  },
  plugins: [],
  // Dark mode is default for WebClaw
  darkMode: 'class',
}
