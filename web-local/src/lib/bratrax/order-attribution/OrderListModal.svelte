<script lang="ts">
  import type { MeasureCellClickContext } from "@rilldata/web-common/features/canvas/components/pivot/drilldown-context";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import {
    createQueryServiceMetricsViewRows,
    type V1MetricsViewRowsResponseDataItem,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    buildOrderListWhere,
    dedupeOrders,
    extractCellLabels,
    formatOrderLabel,
    humanizeResolutionReason,
    ORDER_TIMELINE_METRICS_VIEW,
  } from "./api";
  import type { OrderTimelineRow } from "./types";

  export let open = false;
  export let cellContext: MeasureCellClickContext;
  export let onSelectOrder: (orderId: string) => void;

  $: ({ instanceId } = $runtime);

  // 500 rows of *winning touchpoints* — most cells will be well below this.
  // For aggregated/coarse cells we'd dedupe down to fewer orders.
  const LIMIT = 500;

  $: whereExpr = buildOrderListWhere(cellContext.filters);

  $: rowsQuery = createQueryServiceMetricsViewRows(
    instanceId,
    ORDER_TIMELINE_METRICS_VIEW,
    {
      where: whereExpr,
      timeStart: cellContext.timeRange.start,
      timeEnd: cellContext.timeRange.end,
      limit: LIMIT,
    },
    {
      query: {
        enabled: open && !!instanceId,
      },
    },
  );

  $: rawRows = ($rowsQuery.data?.data as
    | V1MetricsViewRowsResponseDataItem[]
    | undefined) as OrderTimelineRow[] | undefined;
  $: orders = dedupeOrders(rawRows);
  $: hitRowLimit = (rawRows?.length ?? 0) >= LIMIT;

  // Dig the most informative message out of TanStack/runtime-client error
  // shapes. Rill's generated client wraps Orval responses, so the message
  // can live under .response.data.message, .response.data, or .message
  // depending on the failure mode.
  function errorMessage(err: unknown): string {
    if (!err) return "unknown error";
    const e = err as {
      response?: { data?: { message?: string } | string; status?: number };
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
    console.error("[order-drilldown] OrderListModal query error", {
      error: $rowsQuery.error,
      where: whereExpr,
      timeStart: cellContext.timeRange.start,
      timeEnd: cellContext.timeRange.end,
    });
  }

  $: cellLabels = extractCellLabels(cellContext.filters);
  $: subtitleParts = ["channel_group", "campaign", "adset", "ad"]
    .map((dim) => cellLabels[dim])
    .filter((v): v is string => !!v);
  $: subtitle = subtitleParts.length > 0 ? subtitleParts.join(" / ") : null;

  function fmtDate(iso: string | undefined): string {
    if (!iso) return "—";
    const d = new Date(iso);
    return Number.isNaN(d.getTime())
      ? iso
      : d.toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
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

  function fmtWeight(n: number | undefined): string {
    if (n == null) return "—";
    return n.toFixed(2);
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-4xl">
    <Dialog.Header>
      <Dialog.Title>Attributed Orders</Dialog.Title>
      {#if subtitle}
        <Dialog.Description>
          {subtitle} · last-touch
        </Dialog.Description>
      {/if}
    </Dialog.Header>

    {#if $rowsQuery.isLoading}
      <div class="state">Loading orders…</div>
    {:else if $rowsQuery.isError}
      <div class="state error">
        Couldn't load orders: {errorMessage($rowsQuery.error)}
      </div>
    {:else if orders.length === 0}
      <div class="state">No orders match this cell.</div>
    {:else}
      {#if hitRowLimit}
        <div class="banner">
          Showing the first {LIMIT} touchpoint rows ({orders.length} unique orders).
          Tighten the cell to see the full set.
        </div>
      {/if}
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Order</th>
              <th>Date</th>
              <th class="num">Revenue</th>
              <th class="num">Weight</th>
              <th>Confidence</th>
              <th>Reason</th>
              <th>How matched</th>
            </tr>
          </thead>
          <tbody>
            {#each orders as o (o.order_id)}
              <tr
                tabindex="0"
                on:click={() => onSelectOrder(o.order_id)}
                on:keydown={(e) =>
                  (e.key === "Enter" || e.key === " ") &&
                  onSelectOrder(o.order_id)}
              >
                <td class="mono">{formatOrderLabel(o.order_number, o.order_id)}</td>
                <td>{fmtDate(o.conversion_ts)}</td>
                <td class="num">{fmtMoney(o.revenue)}</td>
                <td class="num">{fmtWeight(o.last_touch_weight)}</td>
                <td>{o.resolution_confidence ?? "—"}</td>
                <td class="reason">{o.resolution_reason ?? "—"}</td>
                <td class="how-matched">
                  {humanizeResolutionReason(o.resolution_reason)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
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
  .banner {
    padding: 8px 12px;
    background: var(--color-acid-dim, #f5f5dc);
    border-left: 3px solid var(--color-acid, #d4ff00);
    font-size: 12px;
    color: var(--color-text-muted, #6b7280);
  }
  .table-wrap {
    max-height: 60vh;
    overflow-y: auto;
    border-top: 1px solid var(--color-bratrax-border, #e5e7eb);
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  thead th {
    position: sticky;
    top: 0;
    background: var(--color-bratrax-surface, #ffffff);
    text-align: left;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    padding: 8px 12px;
    border-bottom: 1px solid var(--color-bratrax-border, #e5e7eb);
  }
  tbody td {
    padding: 8px 12px;
    border-bottom: 1px solid var(--color-bratrax-border-faint, #f3f4f6);
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr:hover {
    background: var(--color-acid-dim, #f5f5dc);
  }
  tbody tr:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: -2px;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .mono {
    font-family: "Space Mono", "JetBrains Mono", monospace;
  }
  .reason {
    max-width: 240px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
  }
  .how-matched {
    max-width: 220px;
    white-space: nowrap;
  }
</style>
