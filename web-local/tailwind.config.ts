module.exports = {
  presets: [require("../web-common/tailwind.config.ts")],
  content: [
    "./src/**/*.{html,js,svelte,ts}",
    "../web-common/src/**/*.{svelte,ts}",
    "!../**/*.spec.ts",
    "!../**/*.test.ts",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Outfit"', '"Helvetica Neue"', "sans-serif"],
        display: ['"Outfit"', '"Helvetica Neue"', "sans-serif"],
        serif: ['"DM Serif Display"', "Georgia", "serif"],
        mono: ['"Space Mono"', '"JetBrains Mono"', "monospace"],
      },
      colors: {
        bratrax: {
          bg: "#0A0A0A",
          surface: "#141414",
          hover: "#1E1E1E",
          border: "#3A3A3A",
          acid: "#D4FF00",
          "acid-dim": "rgba(212, 255, 0, 0.7)",
          "acid-muted": "rgba(212, 255, 0, 0.08)",
          tomato: "#FF3B30",
          cyan: "#00D4FF",
          lavender: "#B4A0FF",
          "text-headline": "#FAFAF5",
          "text-primary": "#E8E4DC",
          "text-body": "#B0B0B0",
          "text-muted": "#858585",
        },
      },
      borderRadius: {
        DEFAULT: "0",
        none: "0",
        sm: "0",
        md: "0",
        lg: "0",
        xl: "0",
        "2xl": "0",
        "3xl": "0",
        full: "0",
      },
    },
  },
};
