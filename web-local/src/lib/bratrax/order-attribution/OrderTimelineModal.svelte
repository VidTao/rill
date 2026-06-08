<script lang="ts">
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import {
    createQueryServiceMetricsViewRows,
    type V1MetricsViewRowsResponseDataItem,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    buildOrderTimelineWhere,
    findWinner,
    formatAttributionModel,
    formatOrderLabel,
    ORDER_TIMELINE_METRICS_VIEW,
    sortTimelineRows,
    winnerColumnFor,
  } from "./api";
  import OrderEventDetailModal from "./OrderEventDetailModal.svelte";
  import OrderPathGraph from "./OrderPathGraph.svelte";
  import OrderTimelineRowComponent from "./OrderTimelineRow.svelte";
  import type { OrderTimelineRow as TimelineRow } from "./types";
  import { autoMarkOnce } from "$lib/bratrax/onboarding/checklist";

  export let open = false;
  export let orderId: string;

  // Onboarding checklist (Section 4): mark "Drill into one order's customer
  // journey" the first time this modal opens for an order. Best-effort, once.
  $: if (open && orderId) {
    autoMarkOnce("drilled_into_customer_journey");
  }
  // Attribution model from the parent cell — picks which `is_*_winner` flag
  // selects the winning touchpoint for the banner. Defaults to last_touch
  // to match the dashboard's default filter.
  export let attributionModel: string | undefined = undefined;

  // Two ways to read the same data: the swimlane path graph (default) and the
  // original chronological list. Users pick via the tab strip.
  let view: "graph" | "timeline" = "graph";

  // Modal 3 (event detail) is owned here so both views open the same one — the
  // graph reports clicks via its onSelectRow callback, timeline rows via theirs.
  let detailRow: TimelineRow | null = null;
  let detailOpen = false;
  function openDetail(row: TimelineRow) {
    detailRow = row;
    detailOpen = true;
  }

  $: ({ instanceId } = $runtime);

  $: rowsQuery = createQueryServiceMetricsViewRows(
    instanceId,
    ORDER_TIMELINE_METRICS_VIEW,
    {
      where: buildOrderTimelineWhere(orderId),
      // No time bounds — orders can span up to 30 days of pre-conversion
      // history per the order_timeline_v1 view definition.
      limit: 10000,
    },
    {
      query: {
        enabled: open && !!instanceId && !!orderId,
      },
    },
  );

  $: rawRows = $rowsQuery.data?.data as
    | V1MetricsViewRowsResponseDataItem[]
    | undefined as TimelineRow[] | undefined;
  $: rows = sortTimelineRows(rawRows);
  $: winner = findWinner(rawRows, attributionModel);
  $: modelLabel = formatAttributionModel(attributionModel);

  function errorMessage(err: unknown): string {
    if (!err) return "unknown error";
    const e = err as {
      response?: { data?: { message?: string } | string };
      message?: string;
    };
    if (e.response?.data) {
      if (typeof e.response.data === "string") return e.response.data;
      if (e.response.data.message) return e.response.data.message;
    }
    if (e.message) return e.message;
    try {
      return JSON.stringify(err);
    } catch {
      return String(err);
    }
  }

  $: if ($rowsQuery.isError) {
    // eslint-disable-next-line no-console
    console.error("[order-drilldown] OrderTimelineModal query error", {
      error: $rowsQuery.error,
      orderId,
    });
  }

  // First conversion row carries the order summary (number, revenue, ts).
  $: summary = rows.find((r) => r.row_type === "conversion") ?? rows[0];

  // Timeline view: group resolver_evidence rows under their parent touchpoint
  // by activity_id; surface behavior/touchpoint/conversion at the top level.
  $: evidenceByTouchpoint = (() => {
    const map = new Map<string, TimelineRow[]>();
    for (const r of rows) {
      if (r.row_type === "resolver_evidence" && r.activity_id) {
        const arr = map.get(r.activity_id) ?? [];
        arr.push(r);
        map.set(r.activity_id, arr);
      }
    }
    return map;
  })();

  $: visibleRows = rows.filter((r) => r.row_type !== "resolver_evidence");

  function fmtDateTime(iso: string | undefined): string {
    if (!iso) return "—";
    const d = new Date(iso);
    return Number.isNaN(d.getTime())
      ? iso
      : d.toLocaleString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        });
  }

  function fmtMoney(n: number | undefined): string {
    if (n == null) return "—";
    return n.toLocaleString(undefined, {
      style: "currency",
      currency: "USD",
      maximumFractionDigits: 2,
    });
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="w-[92vw] max-w-[1500px]">
    <Dialog.Header>
      <Dialog.Title>
        Order {formatOrderLabel(summary?.order_number, orderId)}
      </Dialog.Title>
      {#if summary}
        <Dialog.Description>
          {fmtDateTime(summary.conversion_ts)} · {fmtMoney(summary.revenue)}
        </Dialog.Description>
        {#if summary.email}
          <Dialog.Description>{summary.email}</Dialog.Description>
        {/if}
      {/if}
    </Dialog.Header>

    {#if $rowsQuery.isLoading}
      <div class="state">Loading timeline…</div>
    {:else if $rowsQuery.isError}
      <div class="state error">
        Couldn't load timeline: {errorMessage($rowsQuery.error)}
      </div>
    {:else if rows.length === 0}
      <div class="state">No timeline rows for this order.</div>
    {:else}
      {#if winner}
        <div class="winner-banner">
          <div class="banner-label">Winner ({modelLabel})</div>
          <div class="banner-body">
            <div class="banner-line">
              <strong>{winner.channel_group ?? "—"}</strong>
              {#if winner.campaign}
                <span class="sep">·</span>
                {winner.campaign}
              {/if}
            </div>
            <div class="banner-sub">
              {#if winner.adset}
                {winner.adset}
              {/if}
              {#if winner.adset && winner.ad}
                <span class="sep">·</span>
              {/if}
              {#if winner.ad}
                {winner.ad}
              {/if}
            </div>
            <div class="banner-reason">
              {#if winner.resolution_confidence}
                <span class="confidence conf-{winner.resolution_confidence}">
                  {winner.resolution_confidence}
                </span>
              {/if}
              {#if winner.resolution_reason}
                <span class="reason">{winner.resolution_reason}</span>
              {/if}
            </div>
          </div>
        </div>
      {/if}

      <div class="tabs" role="tablist">
        <button
          type="button"
          role="tab"
          class="tab"
          class:active={view === "graph"}
          aria-selected={view === "graph"}
          on:click={() => (view = "graph")}
        >
          Path graph
        </button>
        <button
          type="button"
          role="tab"
          class="tab"
          class:active={view === "timeline"}
          aria-selected={view === "timeline"}
          on:click={() => (view = "timeline")}
        >
          Timeline
        </button>
      </div>

      {#if view === "graph"}
        <OrderPathGraph
          {rows}
          winnerColumn={winnerColumnFor(attributionModel)}
          onSelectRow={openDetail}
        />
      {:else}
        <div class="timeline-wrap">
          <ul class="timeline">
            {#each visibleRows as row (row.row_type + ":" + (row.activity_id ?? "") + ":" + (row.event_ts ?? "") + ":" + (row.event_type ?? ""))}
              <OrderTimelineRowComponent
                {row}
                evidence={row.row_type === "attribution_touchpoint" &&
                row.activity_id
                  ? (evidenceByTouchpoint.get(row.activity_id) ?? [])
                  : []}
                onSelect={openDetail}
              />
            {/each}
          </ul>
        </div>
      {/if}
    {/if}
  </Dialog.Content>
</Dialog.Root>

<OrderEventDetailModal bind:open={detailOpen} row={detailRow} />

<style>
  .state {
    padding: 32px 8px;
    text-align: center;
    color: var(--color-text-muted, #6b7280);
    font-size: 14px;
  }
  .state.error {
    color: #b91c1c;
  }
  .winner-banner {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    padding: 12px 14px;
    margin: 8px 0;
    background: var(--color-acid-dim, #f5f5dc);
    border-left: 3px solid var(--color-acid, #d4ff00);
    border-radius: 4px;
  }
  .banner-label {
    flex-shrink: 0;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    padding-top: 2px;
    min-width: 80px;
  }
  .banner-body {
    flex: 1;
    min-width: 0;
  }
  .banner-line {
    font-size: 14px;
  }
  .banner-sub {
    margin-top: 2px;
    font-size: 12px;
    color: var(--color-text-muted, #6b7280);
  }
  .banner-reason {
    margin-top: 6px;
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }
  .confidence {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    padding: 2px 6px;
    border-radius: 2px;
  }
  .conf-high {
    background: #16a34a;
    color: white;
  }
  .conf-medium {
    background: #f59e0b;
    color: white;
  }
  .conf-low {
    background: #6b7280;
    color: white;
  }
  .reason {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
  }
  .sep {
    margin: 0 2px;
    color: var(--color-text-muted, #9ca3af);
  }
  .tabs {
    display: flex;
    gap: 4px;
    margin: 10px 0 0 0;
    border-bottom: 1px solid var(--color-bratrax-border, #e5e7eb);
  }
  .tab {
    appearance: none;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    padding: 6px 12px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    cursor: pointer;
  }
  .tab:hover {
    color: var(--color-text, #1a1a18);
  }
  .tab.active {
    color: var(--color-text, #1a1a18);
    border-bottom-color: var(--color-acid, #d4ff00);
  }
  .timeline-wrap {
    max-height: 55vh;
    overflow-y: auto;
    padding-right: 4px;
    margin-top: 8px;
  }
  .timeline {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  :global(.dark) .tabs {
    border-bottom-color: #2e2e2e;
  }
  :global(.dark) .tab {
    color: #a39d90;
  }
  :global(.dark) .tab:hover,
  :global(.dark) .tab.active {
    color: #ece7dd;
  }
  /* Banner text: same dialog theme-boundary pinning as the nodes/timeline. */
  :global(.dark) .banner-line {
    color: #ece7dd;
  }
  :global(.dark) .banner-label,
  :global(.dark) .banner-sub,
  :global(.dark) .reason {
    color: #a39d90;
  }
</style>
