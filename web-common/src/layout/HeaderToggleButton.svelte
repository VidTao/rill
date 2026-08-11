<script lang="ts">
  /**
   * The header icon-cluster button: theme, AI and support all render through
   * this so the cluster reads as one control group rather than three unrelated
   * controls.
   *
   * `active` means "the panel this button controls is open", and it is the only
   * state that earns acid. A button that performs an action rather than opening
   * a panel — the theme switch — leaves it false and stays muted for good.
   *
   * The AI toggle previously used the shared `Button` with `type="secondary"`,
   * which resolves to `border/text-accent-primary-action`; the Bratrax dark
   * theme maps that to acid, so it read as permanently active next to its two
   * muted neighbours, and `compact` made it shorter than their 32px besides.
   * `Button` is still the right call for the AI toggle inside the Explore and
   * Canvas preview toolbars, where it sits among other shared buttons — hence
   * this styling lives here rather than in ChatToggle.
   */
  export let active = false;
  export let title: string;
  export let onClick: () => void;
</script>

<button
  type="button"
  class="header-toggle"
  class:active
  on:click={onClick}
  {title}
  {...$$restProps}
>
  <slot />
</button>

<style lang="postcss">
  .header-toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    cursor: pointer;
    /* Matches the Tier 1 nav links, so a text label ("AI") sits in the cluster
       at the same weight as the surrounding chrome. Icon slots ignore it. */
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    line-height: 1;
    text-transform: uppercase;
    transition:
      background-color 120ms ease,
      color 120ms ease,
      border-color 120ms ease;
  }

  .header-toggle:hover {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
  }

  /* The one acid state in the cluster. --color-acid-text rather than
     --color-acid because plain acid on the light theme's cream fails
     contrast; the token resolves to dark olive there and full acid on
     near-black. */
  .header-toggle.active {
    border-color: var(--color-acid, #d4ff00);
    color: var(--color-acid-text, var(--color-acid, #d4ff00));
  }

  .header-toggle:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
</style>
