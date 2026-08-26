<script lang="ts">
  import LoadingSpinner from "@rilldata/web-common/components/icons/LoadingSpinner.svelte";
  import { createQueryServiceMetricsViewAggregation } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import ComponentHeader from "../../ComponentHeader.svelte";
  import type { FinancialStatementComponent } from "./index";
  import { buildFinancialStatementWhere } from "./query";

  export let component: FinancialStatementComponent;

  type RawRow = Record<string, unknown>;
  type StatementRow = {
    sort: number;
    label: string;
    values: Record<string, number>;
    total: number;
    subtotal: boolean;
    cost: boolean;
    percentage: boolean;
  };

  $: ({ instanceId } = $runtime);
  $: ({ specStore, timeAndFilterStore, visible } = component);
  $: spec = $specStore;
  $: ({ timeRange, where, hasTimeSeries } = $timeAndFilterStore);

  let selectedGrain: "Daily" | "Monthly" = "Daily";
  let configuredGrain: string | undefined;
  $: if (spec.grain !== configuredGrain) {
    configuredGrain = spec.grain;
    selectedGrain = spec.grain === "Monthly" ? "Monthly" : "Daily";
  }

  $: queryWhere = buildFinancialStatementWhere(where);
  $: rowsQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.metrics_view ?? "",
    {
      dimensions: [
        { name: "statement_sort" },
        { name: "statement_line" },
        { name: "period_label" },
        { name: "is_subtotal" },
        { name: "profit_ready" },
      ],
      measures: [{ name: "metric_statement_amount" }],
      where: queryWhere,
      timeRange: hasTimeSeries ? timeRange : undefined,
      limit: "10000",
      priority: 20,
    },
    {
      query: {
        enabled: Boolean(
          $visible && spec.metrics_view && (!hasTimeSeries || timeRange),
        ),
      },
    },
  );
  $: ({ isLoading, data: queryData, error: queryError } = $rowsQuery);
  $: rawRows = (queryData?.data ?? []) as RawRow[];

  const COST_SORTS = new Set([70, 80, 90, 100, 110, 120, 140]);
  const ABSOLUTE_SORTS = new Set([20, 30]);

  function numeric(value: unknown): number {
    const n = typeof value === "number" ? value : Number(value ?? 0);
    return Number.isFinite(n) ? n : 0;
  }

  function truthy(value: unknown): boolean {
    return value === true || value === 1 || value === "1" || value === "true";
  }

  function textValue(value: unknown): string {
    return typeof value === "string" || typeof value === "number"
      ? String(value)
      : "";
  }

  function periodFor(label: string, grain: "Daily" | "Monthly"): string {
    return grain === "Monthly" ? label.slice(0, 7) : label;
  }

  function buildStatement(rows: RawRow[], grain: "Daily" | "Monthly") {
    const profitRows = rows.filter(
      (row) => numeric(row.statement_sort) === 180,
    );
    const profitReady =
      profitRows.length > 0 &&
      profitRows.every((row) => truthy(row.profit_ready));
    const bySort = new Map<number, StatementRow>();
    const periods = new Set<string>();

    for (const raw of rows) {
      const sort = numeric(raw.statement_sort);
      const rawLabel = textValue(raw.statement_line);
      const label =
        sort === 180
          ? profitReady
            ? "Net Profit"
            : "Measured Profit"
          : rawLabel;
      const period = periodFor(textValue(raw.period_label), grain);
      if (!period) continue;
      periods.add(period);
      const row = bySort.get(sort) ?? {
        sort,
        label,
        values: {},
        total: 0,
        subtotal: truthy(raw.is_subtotal),
        cost: COST_SORTS.has(sort),
        percentage: false,
      };
      const value = numeric(raw.metric_statement_amount);
      row.values[period] = (row.values[period] ?? 0) + value;
      row.total += value;
      bySort.set(sort, row);
    }

    const orderedPeriods = [...periods].sort();
    const statementRows = [...bySort.values()].sort((a, b) => a.sort - b.sort);
    const revenue = bySort.get(60);
    const profit = bySort.get(180);
    if (revenue && profit) {
      const values: Record<string, number> = {};
      for (const period of orderedPeriods) {
        const denominator = revenue.values[period] ?? 0;
        values[period] =
          denominator === 0 ? 0 : (profit.values[period] ?? 0) / denominator;
      }
      statementRows.push({
        sort: 190,
        label: `${profit.label} Margin`,
        values,
        total: revenue.total === 0 ? 0 : profit.total / revenue.total,
        subtotal: true,
        cost: false,
        percentage: true,
      });
    }
    return { periods: orderedPeriods, rows: statementRows };
  }

  $: statement = buildStatement(rawRows, selectedGrain);
  $: currency = new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: spec.currency || "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  $: percentage = new Intl.NumberFormat(undefined, {
    style: "percent",
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  });

  function displayValue(row: StatementRow, value: number): string {
    if (row.percentage) return percentage.format(value);
    if (row.cost) return `(${currency.format(Math.abs(value))})`;
    if (ABSOLUTE_SORTS.has(row.sort)) return currency.format(Math.abs(value));
    if (value < 0) return `(${currency.format(Math.abs(value))})`;
    return currency.format(value);
  }

  $: title = spec.title ?? "Profit & Loss";
  $: description = spec.description;
  $: filters = {
    time_filters: spec.time_filters,
    dimension_filters: spec.dimension_filters,
  };
  $: errorText = queryError ? "The P&L query could not be completed." : null;
</script>

<ComponentHeader
  {component}
  {title}
  {description}
  showDescriptionAsTooltip={spec.show_description_as_tooltip}
  {filters}
/>

<div class="statement-shell">
  <div class="statement-toolbar">
    <div>
      <span class="eyebrow">Financial statement</span>
      <strong
        >{selectedGrain === "Daily"
          ? "Daily periods"
          : "Monthly periods"}</strong
      >
    </div>
    <div class="grain-toggle" aria-label="Statement granularity">
      <button
        class:active={selectedGrain === "Daily"}
        on:click={() => (selectedGrain = "Daily")}>Daily</button
      >
      <button
        class:active={selectedGrain === "Monthly"}
        on:click={() => (selectedGrain = "Monthly")}>Monthly</button
      >
    </div>
  </div>

  {#if isLoading}
    <div class="state"><LoadingSpinner size="32px" /></div>
  {:else if errorText}
    <div class="state error">{errorText}</div>
  {:else if statement.rows.length === 0}
    <div class="state">No financial activity in this period.</div>
  {:else}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th class="line-heading">Line item</th>
            {#each statement.periods as period (period)}
              <th>{period}</th>
            {/each}
            <th class="total-heading">Total</th>
          </tr>
        </thead>
        <tbody>
          {#each statement.rows as row (row.sort)}
            <tr class:subtotal={row.subtotal} class:profit={row.sort >= 170}>
              <th scope="row">{row.label}</th>
              {#each statement.periods as period (period)}
                <td>{displayValue(row, row.values[period] ?? 0)}</td>
              {/each}
              <td class="total-cell">{displayValue(row, row.total)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style lang="postcss">
  /* Colours here are semantic tokens only — `.dark` on <html> reassigns the
     underlying CSS variables, so both themes come from one set of rules. This
     widget used to hardcode ~25 hex values plus a partial :global(.dark) block,
     which is why dark theme had a white grain toggle and invisible profit rows.
     See bratrax-theme.css lines 1-23 for the token strategy.

     Deliberately NOT converted: the `font-size` declarations below. Section 15
     of bratrax-theme.css floors every Tailwind text class at 14px !important,
     so `@apply text-xs` would enlarge the whole table and break column fit.
     Raw font-size bypasses that floor, which is the behaviour this table wants. */
  .statement-shell {
    @apply text-fg-primary;
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 0 16px 16px;
  }
  .statement-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 8px 2px 12px;
  }
  .statement-toolbar > div:first-child {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .eyebrow {
    @apply text-fg-muted;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .statement-toolbar strong {
    font-size: 13px;
    font-weight: 650;
  }
  /* Segmented control: recessed track, raised active segment.
     The track is --surface-background and the segment --surface-card so the lift
     survives BOTH themes. Do not reach for --surface-subtle here: in dark it and
     --surface-card both resolve to #141414, so the active segment would be
     invisible against its own track. */
  .grain-toggle {
    @apply border border-border bg-surface-background;
    display: inline-flex;
    padding: 3px;
    border-radius: 8px;
  }
  .grain-toggle button {
    @apply text-fg-secondary;
    border: 0;
    border-radius: 6px;
    background: transparent;
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    padding: 5px 10px;
  }
  .grain-toggle button:hover:not(.active) {
    @apply text-fg-primary;
  }
  /* --fg-accent, not --fg-primary: near-black on the light theme's white segment,
     acid on dark. Two near-black surfaces can't carry the active state on their
     own, so the accent does it — the fork's "acid only when active" convention
     (HeaderToggleButton.svelte, commit 86cd70c5b). */
  .grain-toggle button.active {
    @apply bg-surface-card text-fg-accent;
    /* currentColor at low alpha so the lift reads on a cream track and on a
       near-black one; a fixed rgba(15,23,42,…) only worked on the former. */
    box-shadow: 0 1px 3px color-mix(in oklab, currentColor 18%, transparent);
  }
  .table-wrap {
    @apply border border-border bg-surface-card;
    flex: 1;
    min-height: 0;
    overflow: auto;
    border-radius: 8px;
  }
  table {
    border-collapse: separate;
    border-spacing: 0;
    min-width: 100%;
    font-size: 12px;
  }
  th,
  td {
    @apply border-b border-border;
    padding: 9px 14px;
    text-align: right;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  /* Every sticky cell restates an opaque background token. bratrax-theme.css
     section 5: a body-level graph-paper overlay sits behind everything, so a
     sticky cell without one lets the grid show through as it scrolls. */
  thead th {
    @apply bg-surface-background text-fg-secondary;
    position: sticky;
    top: 0;
    z-index: 3;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.02em;
  }
  th[scope="row"],
  .line-heading {
    @apply bg-surface-card;
    position: sticky;
    left: 0;
    z-index: 2;
    min-width: 190px;
    text-align: left;
    font-weight: 500;
  }
  .line-heading {
    @apply bg-surface-background;
    z-index: 4;
  }
  .total-heading,
  .total-cell {
    @apply border-l border-border bg-surface-muted;
    position: sticky;
    right: 0;
    z-index: 2;
    font-weight: 700;
  }
  .total-heading {
    z-index: 4;
  }
  tr.subtotal th,
  tr.subtotal td {
    @apply border-t border-border bg-surface-muted;
    font-weight: 700;
  }
  tr.subtotal th[scope="row"] {
    @apply bg-surface-muted;
  }
  /* Green means profit in both themes, but the hue has to invert — the light
     value is invisible on the dark subtotal surface. See --color-profit. */
  tr.profit th,
  tr.profit td {
    color: var(--color-profit);
  }
  tbody tr:last-child th,
  tbody tr:last-child td {
    border-bottom: 0;
  }
  .state {
    @apply text-fg-muted;
    flex: 1;
    display: grid;
    place-items: center;
  }
  .state.error {
    @apply text-destructive;
  }
</style>
