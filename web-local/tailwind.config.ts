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
        // Bratrax tokens — theme-aware via CSS variables.
        // Values defined in bratrax-theme.css under :root (light) and :root.dark (dark).
        bratrax: {
          bg: "var(--bratrax-bg)",
          surface: "var(--bratrax-surface)",
          hover: "var(--bratrax-hover)",
          border: "var(--bratrax-border)",
          acid: "var(--bratrax-acid)",
          "acid-dim": "var(--bratrax-acid-dim)",
          "acid-muted": "var(--bratrax-acid-muted)",
          tomato: "var(--bratrax-tomato)",
          cyan: "var(--bratrax-cyan)",
          lavender: "var(--bratrax-lavender)",
          "text-headline": "var(--bratrax-text-headline)",
          "text-primary": "var(--bratrax-text-primary)",
          "text-body": "var(--bratrax-text-body)",
          "text-muted": "var(--bratrax-text-muted)",
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
