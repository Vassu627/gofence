/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: '#101416',
        panel: '#181e21',
        line: '#2a3438',
        mist: '#93a099',
        paper: '#e8eee9',
        amber: '#e0a425',
        signal: '#3cba8b',
        danger: '#e15a45',
        glide: '#7eb6d6',
      },
      fontFamily: {
        sans: ['Outfit', 'Segoe UI', 'sans-serif'],
        mono: ['IBM Plex Mono', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
}
