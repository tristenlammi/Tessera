/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      padding: {
        'safe-top': 'env(safe-area-inset-top)',
        'safe-bottom': 'env(safe-area-inset-bottom)',
        'safe-left': 'env(safe-area-inset-left)',
        'safe-right': 'env(safe-area-inset-right)',
      },
      colors: {
        notion: {
          // Light mode surfaces
          'bg': '#ffffff',
          'bg-secondary': '#f7f7f5',
          'sidebar': '#f7f7f5',
          // Dark mode surfaces
          'dark-bg': '#191919',
          'dark-bg-secondary': '#202020',
          'dark-sidebar': '#1e1e1e',
          'dark-hover': '#2f2f2f',
          'dark-active': '#2d2d2d',
          'dark-border': '#2d2d2d',
          // Text
          'text': '#37352f',
          'text-secondary': '#787774',
          'text-light': '#9b9a97',
          'dark-text': '#ebebeb',
          'dark-text-secondary': '#9b9b9b',
          'dark-text-light': '#7f7f7f',
          // Borders
          'border': '#e8e8e5',
          'hover': '#efefef',
        }
      }
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
