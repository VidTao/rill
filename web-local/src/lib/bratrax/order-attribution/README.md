# Order attribution drilldown

Drilldown UI for the campaign deep-dive canvas dashboard. Clicking an
**Attributed Orders** cell opens a list of the orders behind that number; clicking
an order opens its full touchpoint journey. A parallel **Profiles** drilldown
reuses the same plumbing for identity exploration.

The feature is opt-in per client. It is mounted by `OrderDrilldownProvider`,
which is wrapped around the dashboard area at the workspace layout level only for
enabled clients.

## Flow

```
Attributed Orders cell ──click──▶ Modal 1 (OrderListModal)   list of orders
        order row       ──click──▶ Modal 2 (OrderTimelineModal) touchpoint paths

Profiles cell (grouped) ──click──▶ ProfileListModal           list of profiles
Profiles cell (single)  ──click──▶ ProfileGraphModal          identity graph
```

`OrderDrilldownProvider` registers a handler on the `ORDER_DRILLDOWN_CONTEXT`
Svelte context and owns every modal's open state. The pivot table calls
`context.open(ctx)` with the clicked cell's filters; the provider routes to the
right modal based on `measureName` (`metric_attributed_orders` vs `profiles`).

The active **attribution model** is captured from the cell filter at click time
(`handleOrderSelect`) and passed into Modal 2, so the winner stays consistent
with what Modal 1 showed even if the user changes the dashboard filter while the
timeline is open.

## Data

Both order modals query the `order_timeline_metrics` metrics view (one row per
event in an order's pre-conversion history, up to ~30 days). See
`buildOrderListWhere` / `buildOrderTimelineWhere` in `api.ts` for the filters.

Each row has a `row_type`:

| row_type                 | meaning                                  | in graph? |
| ------------------------ | ---------------------------------------- | --------- |
| `behavior_event`         | page view / identify / checkout event    | yes       |
| `attribution_touchpoint` | a touchpoint attribution considered      | yes       |
| `resolver_evidence`      | why the resolver matched a touchpoint    | no        |
| `conversion`             | the order itself (terminal node)         | yes       |

`event_source` is the axis the graph splits on — `shopify_browser_pixel`,
`shopify_journey` / `shopify_customer_journey`, `klaviyo`, `dim_orders`
(the conversion + referrer), etc. The winner is the touchpoint flagged in the
`is_*_winner` column matching the active model (`winnerColumnFor` maps model →
column; default `last_touch`).

`attribution_model` lives on the parent `cross_metrics` dashboard, **not** on
`order_timeline_metrics`. `buildOrderListWhere` strips it from the forwarded cell
filter and translates it to the matching `is_*_winner` flag; other dimensions not
present on the timeline view are pruned by `pruneFilterToOrderTimeline`.

## Modal 2: touchpoint path graph

Modal 2 renders the order's journey as a **swimlane path graph** rather than a
flat list: one horizontal lane per `event_source`, each a `step → step → step`
path connected by arrows, all converging into a single conversion node on the
right. The winning touchpoint and the edge leaving it are highlighted.

Built on `@xyflow/svelte` (SvelteFlow) + `@dagrejs/dagre`:

- `order-path-layout.ts` — pure layout. Groups rows by source into lanes, runs
  dagre left-to-right for step (X) ordering, then **snaps each node's Y into its
  lane band** so the sources read as parallel swimlanes. Returns SvelteFlow
  nodes/edges. Also owns the source label/color maps.
- `OrderPathNode.svelte` — the touchpoint / behavior / conversion card.
- `OrderPathLane.svelte` — the per-source label at the left of each lane.
- `OrderPathGraph.svelte` — the SvelteFlow canvas wrapper (zoom/pan + Controls).

### Gotchas

- **Card dimensions live in two places.** `NODE_W` / `NODE_H` in
  `order-path-layout.ts` are what dagre reserves; the `width` / `height` in
  `OrderPathNode.svelte` / `OrderPathLane.svelte` are what renders. Change both
  together or the boxes and the layout drift apart.
- **Dark mode needs explicit colors.** The canvas dashboard's theme boundary
  pins `--color-text` to its light value inside the dashboard, so raw
  `var(--color-text)` fallbacks render near-black text on dark cards. Text/
  background colors are therefore set explicitly under `:global(.dark)` in the
  node components, and `OrderPathGraph` passes `colorMode` (from `themeControl`)
  into `<SvelteFlow>` so its edges/Controls match.
- **`resolver_evidence` is dropped from the graph.** It was the collapsible
  "Show evidence" section in the old list view (`OrderTimelineRow.svelte`, now
  unused by the modal). If it needs to come back, a node-click popover is the
  natural home.

## Files

| File                          | Role                                             |
| ----------------------------- | ------------------------------------------------ |
| `OrderDrilldownProvider.svelte` | Context registration + modal orchestration     |
| `OrderListModal.svelte`       | Modal 1 — orders behind a cell                   |
| `OrderTimelineModal.svelte`   | Modal 2 — header/winner banner + path graph      |
| `OrderPathGraph.svelte`       | SvelteFlow canvas for the path graph             |
| `OrderPathNode.svelte`        | Touchpoint / behavior / conversion node card     |
| `OrderPathLane.svelte`        | Per-source lane label                            |
| `order-path-layout.ts`        | Swimlane layout (dagre) + source label/color maps |
| `OrderTimelineRow.svelte`     | Legacy flat-list row (no longer used by Modal 2) |
| `ProfileListModal.svelte`     | Profiles drilldown — list                        |
| `ProfileGraphModal.svelte`    | Profiles drilldown — identity graph              |
| `api.ts`                      | Query builders + response reshaping helpers      |
| `types.ts`                    | Row / summary shapes for the timeline view       |
