<script lang="ts">
  import { commandPaletteOpen, shortcutLabel } from "./store";

  // The visible way in. A shortcut alone would leave the palette as
  // undiscoverable as the pages it exists to surface.
</script>

<button
  type="button"
  class="bratrax-palette-trigger"
  title="Search settings and help ({shortcutLabel})"
  aria-label="Search settings and help"
  on:click={() => commandPaletteOpen.set(true)}
>
  <svg
    width="14"
    height="14"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <circle cx="11" cy="11" r="7" />
    <path d="m20 20-3.5-3.5" />
  </svg>
  <span class="bratrax-palette-trigger-label">Search</span>
  <kbd class="bratrax-palette-trigger-kbd">{shortcutLabel}</kbd>
</button>

<style>
  /* Same weight and border treatment as AddStoreButton in the header's right
     cluster. */
  .bratrax-palette-trigger {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    /* Twice its natural ~135px content width, so it reads as a search field
       rather than a button; the shortcut hint sits at the right edge. It is
       a flex item of ApplicationHeader's "center" slot, which squeezes it
       toward icon-only (36px) when both clusters crowd it. */
    flex: 0 1 270px;
    min-width: 36px;
    overflow: hidden;
    height: 32px;
    padding: 0 10px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-muted, var(--fg-muted));
    cursor: pointer;
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .bratrax-palette-trigger:hover {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
  }
  .bratrax-palette-trigger:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }

  .bratrax-palette-trigger svg {
    flex: none;
  }

  .bratrax-palette-trigger-label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bratrax-palette-trigger-kbd {
    flex: none;
    margin-left: auto;
    font-family: inherit;
    font-size: 10px;
    letter-spacing: 0.5px;
    white-space: nowrap;
    color: var(--color-text-muted, var(--fg-muted));
  }

  /* Icon-only on narrow headers; the title attribute still names it. */
  @media (max-width: 1100px) {
    .bratrax-palette-trigger {
      flex-basis: auto;
    }
    .bratrax-palette-trigger-label,
    .bratrax-palette-trigger-kbd {
      display: none;
    }
  }
</style>
