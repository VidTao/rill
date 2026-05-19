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
    ORDER_TIMELINE_METRICS_VIEW,
    sortTimelineRows,
  } from "./api";
  import OrderTimelineRow from "./OrderTimelineRow.svelte";
  import type { OrderTimelineRow as TimelineRow } from "./types";

  export let open = false;
  export let orderId: string;

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

  $: rawRows = ($rowsQuery.data?.data as
    | V1MetricsViewRowsResponseDataItem[]
    | undefined) as TimelineRow[] | undefined;
  $: rows = sortTimelineRows(rawRows);
  $: winner = findWinner(rawRows);

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

  // Group resolver_evidence rows under their parent touchpoint by activity_id
  // (resolver_evidence.activity_id maps to attribution_touchpoint.activity_id).
  // Touchpoint rows without matching evidence get an empty array.
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

  // Visible timeline rows: behavior_events, touchpoints, conversion.
  // Resolver evidence is hidden from the top level and surfaced under its
  // parent touchpoint.
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
  <Dialog.Content class="max-w-3xl">
    <Dialog.Header>
      <Dialog.Title>
        Order #{summary?.order_number ?? orderId}
      </Dialog.Title>
      {#if summary}
        <Dialog.Description>
          {fmtDateTime(summary.conversion_ts)} · {fmtMoney(summary.revenue)}
        </Dialog.Description>
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
          <div class="banner-label">Winner (last-touch)</div>
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

      <div class="timeline-wrap">
        <ul class="timeline">
          {#each visibleRows as row (row.row_type + ":" + (row.activity_id ?? "") + ":" + (row.event_ts ?? "") + ":" + (row.event_type ?? ""))}
            <OrderTimelineRow
              {row}
              evidence={row.row_type === "attribution_touchpoint" && row.activity_id
                ? (evidenceByTouchpoint.get(row.activity_id) ?? [])
                : []}
            />
          {/each}
        </ul>
      </div>
    {/if}
  </Dialog.Content>
</Dialog.Root>

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
  .timeline-wrap {
    max-height: 55vh;
    overflow-y: auto;
    padding-right: 4px;
  }
  .timeline {
    margin: 0;
    padding: 0;
    list-style: none;
  }
</style>
