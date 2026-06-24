<script lang="ts">
  // Connectors-page status pill that doubles as a backfill progress bar.
  //
  // - fillPercentage undefined or >= 100 → today's solid acid-green
  //   "CONNECTED" pill (visual no-op for completed / untracked sources).
  // - 0 <= fillPercentage < 100 → outlined pill whose background fills
  //   left→right with acid-green over a muted track, label
  //   "CONNECTED — N% BACKFILLED" (or "CONNECTED — BACKFILLING 0%" at 0).
  //
  // The text and the fill are independent: the fill is the data signal, the
  // text is the human-readable label. Dark text reads cleanly on both the
  // acid fill and the muted track.

  export let fillPercentage: number | undefined = undefined;

  // Muted track color for the unfilled portion (warm beige, close to the page
  // background — resolved in spec §"Resolved").
  const TRACK = "#E8E4D6";

  $: complete = fillPercentage === undefined || fillPercentage >= 100;
  $: pct = Math.max(0, Math.min(100, Math.round(fillPercentage ?? 0)));
  $: label =
    pct <= 0 ? "Connected — backfilling 0%" : `Connected — ${pct}% backfilled`;
  $: fillStyle = `background: linear-gradient(to right, var(--color-acid) 0%, var(--color-acid) ${pct}%, ${TRACK} ${pct}%, ${TRACK} 100%);`;
</script>

{#if complete}
  <span class="pill pill-complete">Connected</span>
{:else}
  <span class="pill pill-progress" style={fillStyle}>{label}</span>
{/if}

<style>
  .pill {
    font-family: "Space Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    padding: 3px 8px;
    color: #0a0a0a;
    white-space: nowrap;
  }
  .pill-complete {
    background: var(--color-acid);
  }
  .pill-progress {
    border: 1.5px solid var(--color-acid);
    /* inline `style` supplies the gradient fill */
  }
</style>
