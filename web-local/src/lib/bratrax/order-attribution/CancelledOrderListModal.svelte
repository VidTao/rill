<script lang="ts">
  import type { MeasureCellClickContext } from "@rilldata/web-common/features/canvas/components/pivot/drilldown-context";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import {
    createQueryServiceMetricsViewRows,
    type V1MetricsViewRowsResponseDataItem,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    CANCELLED_ORDER_ITEMS_METRICS_VIEW,
    extractCellLabels,
    formatAttributionModel,
    formatOrderLabel,
    groupCancelledOrderItems,
  } from "./api";
  import type { CancelledOrderItemRow } from "./types";

  export let open = false;
  export let cellContext: MeasureCellClickContext;
  export let onSelectOrder: (orderId: string) => void;
  const LIMIT = 2000;

  $: ({ instanceId } = $runtime);
  $: rowsQuery = createQueryServiceMetricsViewRows(
    instanceId,
    CANCELLED_ORDER_ITEMS_METRICS_VIEW,
    {
      where: cellContext.filters,
      timeStart: cellContext.timeRange.start,
      timeEnd: cellContext.timeRange.end,
      limit: LIMIT,
    },
    { query: { enabled: open && !!instanceId } },
  );
  $: rawRows = $rowsQuery.data?.data as
    | V1MetricsViewRowsResponseDataItem[]
    | undefined as CancelledOrderItemRow[] | undefined;
  $: orders = groupCancelledOrderItems(rawRows);
  $: hitRowLimit = (rawRows?.length ?? 0) >= LIMIT;
  $: itemLines = orders.reduce((total, order) => total + order.items.length, 0);
  $: units = orders.reduce(
    (total, order) =>
      total + order.items.reduce((sum, item) => sum + item.quantity, 0),
    0,
  );
  $: cellLabels = extractCellLabels(cellContext.filters);
  $: subtitleParts = ["channel_group", "campaign", "adset", "ad"]
    .map((dimension) => cellLabels[dimension])
    .filter((value): value is string => !!value);
  $: modelLabel = formatAttributionModel(cellLabels["attribution_model"]);
  $: subtitle = subtitleParts.length
    ? `${subtitleParts.join(" / ")} / ${modelLabel}`
    : modelLabel;

  function errorMessage(err: unknown): string {
    if (!err) return "unknown error";
    const error = err as {
      response?: { data?: { message?: string } | string };
      message?: string;
    };
    if (typeof error.response?.data === "string") return error.response.data;
    if (error.response?.data?.message) return error.response.data.message;
    return error.message ?? String(err);
  }
  function fmtDate(iso?: string): string {
    if (!iso) return "-";
    const date = new Date(iso);
    return Number.isNaN(date.getTime())
      ? iso
      : date.toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
        });
  }
  function fmtMoney(value: number, currency: string): string {
    return value.toLocaleString(undefined, {
      style: "currency",
      currency: currency || "USD",
      maximumFractionDigits: 2,
    });
  }
  function selectOrder(orderId: string, event: KeyboardEvent | MouseEvent) {
    if (
      event instanceof KeyboardEvent &&
      event.key !== "Enter" &&
      event.key !== " "
    )
      return;
    if (event instanceof KeyboardEvent) event.preventDefault();
    onSelectOrder(orderId);
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-[96vw]">
    <Dialog.Header>
      <Dialog.Title>Cancelled Orders</Dialog.Title>
      <Dialog.Description
        >{subtitle} / grouped by order and SKU</Dialog.Description
      >
    </Dialog.Header>
    {#if $rowsQuery.isLoading}
      <div class="state">Loading cancelled order items...</div>
    {:else if $rowsQuery.isError}
      <div class="state error">
        Couldn't load cancelled order items: {errorMessage($rowsQuery.error)}
      </div>
    {:else if orders.length === 0}
      <div class="state">
        No cancelled orders match this cell and order-date range.
      </div>
    {:else}
      <div class="summary" aria-label="Cancelled order item summary">
        <div><strong>{orders.length}</strong><span>Orders</span></div>
        <div><strong>{itemLines}</strong><span>SKU lines</span></div>
        <div><strong>{units}</strong><span>Units</span></div>
      </div>
      {#if hitRowLimit}
        <div class="banner">
          Showing the first {LIMIT} item rows. Tighten the pivot cell or date range
          to see the full set.
        </div>
      {/if}
      <div class="table-wrap">
        <table>
          <thead
            ><tr
              ><th>Order</th><th>Ordered</th><th>Cancelled</th><th>Reason</th
              ><th>Product</th><th>Variant</th><th>SKU</th><th class="num"
                >Qty</th
              ><th class="num">Item value</th><th class="num">Order total</th
              ><th class="num">Weight</th></tr
            ></thead
          >
          {#each orders as order (order.order_id)}
            <tbody class="order-group">
              {#each order.items as item, index (item.line_item_id || `${item.sku}-${index}`)}
                <tr
                  tabindex="0"
                  on:click={(event) => selectOrder(order.order_id, event)}
                  on:keydown={(event) => selectOrder(order.order_id, event)}
                >
                  {#if index === 0}
                    <td rowspan={order.items.length} class="order-cell"
                      ><span class="order-link"
                        >{formatOrderLabel(
                          order.order_number,
                          order.order_id,
                        )}</span
                      ><span class="email">{order.email || "-"}</span></td
                    >
                    <td rowspan={order.items.length}
                      >{fmtDate(order.order_created_at)}</td
                    >
                    <td rowspan={order.items.length}
                      >{fmtDate(order.cancelled_at)}</td
                    >
                    <td rowspan={order.items.length} class="reason"
                      >{order.cancel_reason || "-"}</td
                    >
                  {/if}
                  <td class="product">{item.product_title || "-"}</td>
                  <td>{item.variant_title || "-"}</td>
                  <td class="sku">{item.sku || "-"}</td>
                  <td class="num">{item.quantity || "-"}</td>
                  <td class="num"
                    >{fmtMoney(item.item_value, order.currency)}</td
                  >
                  {#if index === 0}
                    <td rowspan={order.items.length} class="num"
                      >{fmtMoney(order.order_total, order.currency)}</td
                    >
                    <td rowspan={order.items.length} class="num weight"
                      >{order.attribution_weight.toFixed(2)}</td
                    >
                  {/if}
                </tr>
              {/each}
            </tbody>
          {/each}
        </table>
      </div>
    {/if}
  </Dialog.Content>
</Dialog.Root>

<style>
  .state {
    padding: 36px 12px;
    text-align: center;
    color: color-mix(in srgb, currentColor 65%, transparent);
    font-size: 14px;
  }
  .state.error {
    color: #b91c1c;
  }
  .summary {
    display: flex;
    gap: 1px;
    margin-bottom: 10px;
    background: color-mix(in srgb, currentColor 14%, transparent);
    border: 1px solid color-mix(in srgb, currentColor 14%, transparent);
  }
  .summary > div {
    display: flex;
    align-items: baseline;
    gap: 8px;
    min-width: 120px;
    padding: 8px 12px;
    background: color-mix(in srgb, currentColor 4%, transparent);
  }
  .summary strong {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 17px;
  }
  .summary span {
    color: color-mix(in srgb, currentColor 62%, transparent);
    font-size: 11px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .banner {
    padding: 8px 12px;
    background: color-mix(in srgb, #d4ff00 12%, transparent);
    border-left: 3px solid #b7df00;
    font-size: 12px;
  }
  .table-wrap {
    max-height: 62vh;
    overflow: auto;
    border-top: 1px solid color-mix(in srgb, currentColor 14%, transparent);
  }
  table {
    width: 100%;
    min-width: 1320px;
    border-collapse: separate;
    border-spacing: 0;
    font-size: 12px;
  }
  thead th {
    position: sticky;
    top: 0;
    z-index: 2;
    padding: 8px 10px;
    background: var(--color-bratrax-surface, #fff);
    border-bottom: 1px solid color-mix(in srgb, currentColor 18%, transparent);
    color: color-mix(in srgb, currentColor 64%, transparent);
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    letter-spacing: 0.08em;
    text-align: left;
    text-transform: uppercase;
    white-space: nowrap;
  }
  td {
    padding: 8px 10px;
    border-bottom: 1px solid color-mix(in srgb, currentColor 8%, transparent);
    vertical-align: top;
  }
  .order-group tr {
    cursor: pointer;
  }
  .order-group tr:hover td {
    background: color-mix(in srgb, #d4ff00 10%, transparent);
  }
  .order-group tr:focus-visible {
    outline: 2px solid #b7df00;
    outline-offset: -2px;
  }
  .order-group tr:first-child td {
    border-top: 1px solid color-mix(in srgb, currentColor 18%, transparent);
  }
  .order-cell {
    min-width: 150px;
  }
  .order-link {
    display: block;
    color: #2563eb;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    text-decoration: underline;
  }
  .email {
    display: block;
    max-width: 180px;
    margin-top: 3px;
    overflow: hidden;
    color: color-mix(in srgb, currentColor 58%, transparent);
    font-size: 11px;
    text-overflow: ellipsis;
  }
  .reason,
  .product {
    max-width: 220px;
  }
  .sku,
  .weight {
    font-family: "Space Mono", "JetBrains Mono", monospace;
  }
  .sku,
  .num {
    white-space: nowrap;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  :global(.dark) thead th {
    background: #18181b;
  }
  :global(.dark) .order-link {
    color: #8ab4ff;
  }
  @media (max-width: 720px) {
    .summary {
      overflow-x: auto;
    }
    .summary > div {
      min-width: 105px;
    }
  }
</style>
