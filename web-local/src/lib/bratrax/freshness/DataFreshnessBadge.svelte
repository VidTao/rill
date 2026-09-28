<script lang="ts">
  import { onMount } from "svelte";
  import Tooltip from "@rilldata/web-common/components/tooltip/Tooltip.svelte";
  import TooltipContent from "@rilldata/web-common/components/tooltip/TooltipContent.svelte";
  import {
    fetchDataFreshness,
    freshnessView,
    type DataFreshness,
  } from "./freshness";

  /**
   * "Updated 23 min ago · Next update in 17 min" beside the canvas filter bar.
   * Self-contained: fetches for whichever client is active, renders nothing
   * until it has an answer (or ever, if the backend can't tell). A client
   * switch is a full page reload, so there is no identity to key on here.
   */

  // The data only moves on the client's hourly Rill tick, so polling is just a
  // safety net; the targeted refetch below is what flips the label promptly.
  const POLL_MS = 5 * 60_000;
  // Relabel "23 min ago" → "24 min ago" without refetching.
  const TICK_MS = 30_000;
  // Refetch this long after the promised update, giving the Rill reconcile
  // and any in-flight cascade a moment to land.
  const AFTER_UPDATE_MS = 2 * 60_000;

  let data: DataFreshness | null = null;
  let now = Date.now();
  let afterUpdateTimer: ReturnType<typeof setTimeout> | undefined;

  async function load() {
    data = await fetchDataFreshness();
    now = Date.now();
    clearTimeout(afterUpdateTimer);
    const next = data?.next_update_at
      ? new Date(data.next_update_at).getTime()
      : NaN;
    if (Number.isFinite(next)) {
      afterUpdateTimer = setTimeout(
        () => void load(),
        Math.max(0, next - now) + AFTER_UPDATE_MS,
      );
    }
  }

  function onVisibility() {
    if (document.visibilityState === "visible") void load();
  }

  onMount(() => {
    void load();
    const poll = setInterval(() => void load(), POLL_MS);
    const tick = setInterval(() => (now = Date.now()), TICK_MS);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      clearInterval(poll);
      clearInterval(tick);
      clearTimeout(afterUpdateTimer);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  });

  $: view = freshnessView(data, now);
</script>

{#if view}
  <Tooltip distance={8} location="bottom" alignment="end">
    <div class="freshness-badge" class:delayed={view.tone === "delayed"}>
      {#if view.tone === "delayed"}
        <span class="freshness-dot" aria-hidden="true"></span>
      {/if}
      <span>{view.updated}</span>
      {#if view.next}
        <span class="freshness-next">· {view.next}</span>
      {/if}
    </div>
    <TooltipContent slot="tooltip-content" maxWidth="300px">
      {#each view.tooltip as line}
        <p>{line}</p>
      {/each}
    </TooltipContent>
  </Tooltip>
{/if}

<style lang="postcss">
  /* Sized to sit beside .canvas-edit-link (26px, 13px). font-size lives here,
     not in a Tailwind class: bratrax-theme.css floors every sub-14px text
     utility with !important. */
  .freshness-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 4px;
    color: var(--color-text-muted, var(--fg-muted));
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    cursor: default;
  }

  .freshness-badge.delayed {
    color: var(--color-text, var(--fg-primary));
  }

  /* Same danger token as ReconnectPill — stale data is a fault to repair. */
  .freshness-dot {
    width: 7px;
    height: 7px;
    flex: none;
    background: var(--bratrax-tomato, #e63226);
  }

  /* The badge overlays the right end of the centred filter row; the ETA is
     the part that can go, and it stays available in the tooltip. */
  .freshness-next {
    display: none;
  }
  @media (min-width: 1280px) {
    .freshness-next {
      display: inline;
    }
  }
</style>
