/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        studio: {
          bg: "rgb(var(--studio-bg) / <alpha-value>)",
          panel: "rgb(var(--studio-panel) / <alpha-value>)",
          border: "rgb(var(--studio-border) / <alpha-value>)",
          text: "rgb(var(--studio-text) / <alpha-value>)",
          muted: "rgb(var(--studio-muted) / <alpha-value>)",
          accent: "rgb(var(--studio-accent) / <alpha-value>)",
          "on-accent": "rgb(var(--studio-on-accent) / <alpha-value>)",
          ai: "rgb(var(--studio-ai) / <alpha-value>)",
          paper: "rgb(var(--studio-paper) / <alpha-value>)",
          ink: "rgb(var(--studio-ink) / <alpha-value>)",
          "cover-from": "rgb(var(--studio-cover-from) / <alpha-value>)",
          "cover-via": "rgb(var(--studio-cover-via) / <alpha-value>)",
          "cover-to": "rgb(var(--studio-cover-to) / <alpha-value>)",
        },
      },
      fontFamily: {
        sans: [
          "Noto Sans SC",
          "PingFang SC",
          "Microsoft YaHei",
          "system-ui",
          "sans-serif",
        ],
        serif: ["Source Han Serif SC", "Songti SC", "Georgia", "serif"],
      },
      boxShadow: {
        card: "var(--studio-shadow-card)",
      },
    },
  },
  plugins: [],
};
