<script lang="ts">
  import { goto } from "$app/navigation";

  // Compact one-line callout shown on the onboarding checklist while any
  // connected source is still backfilling. A pointer, not a display: the
  // per-source detail lives on the Connectors page. Visibility is decided by
  // the parent (anyBackfilling); this component is presentational.
</script>

<div class="callout">
  <span class="icon" aria-hidden="true">&#9432;</span>
  <span class="text"
    >Historical data is still loading — this can take a few days.</span
  >
  <button class="link" on:click={() => goto("/connectors")}>
    See details on Connectors page →
  </button>
</div>

<style>
  .callout {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    margin-bottom: 24px;
    background: var(--color-surface);
    border-left: 3px solid var(--color-acid);
    font-family: "Outfit", sans-serif;
    font-size: 13px;
    color: var(--color-text);
  }
  .icon {
    flex-shrink: 0;
    color: var(--color-text-secondary);
    font-size: 14px;
  }
  .text {
    flex: 1 1 auto;
  }
  .link {
    flex-shrink: 0;
    white-space: nowrap;
    background: transparent;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: "Space Mono", monospace;
    font-size: 11px;
    letter-spacing: 0.5px;
    color: var(--color-text);
    text-decoration: underline;
    text-decoration-color: var(--color-acid);
    text-decoration-thickness: 2px;
  }
  .link:hover {
    color: var(--color-acid-text);
  }
</style>
