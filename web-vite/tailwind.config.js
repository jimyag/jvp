/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#383838',
          light: '#4a4a4a',
        },
        accent: {
          DEFAULT: '#6FC2FF',
          dark: '#2BA5FF',
        },
        background: {
          DEFAULT: '#F4EFEA',
          card: '#FFFFFF',
        },
        yellow: {
          DEFAULT: '#FFDE00',
        },
        coral: {
          DEFAULT: '#FF7169',
        },
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
}
