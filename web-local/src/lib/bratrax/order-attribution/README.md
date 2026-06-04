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
Attributed Orders cell ──click──▶ Modal 1 (OrderListModal)        list of orders
        order row       ──click──▶ Modal 2 (OrderTimelineModal)    touchpoint paths
        node card       ──click──▶ Modal 3 (OrderEventDetailModal) full row detail

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

## Modal 2: two views (Path graph + Timeline)

Modal 2 carries a shared winner banner and a **two-tab switcher**:

- **Path graph** (default) — the swimlane graph described below.
- **Timeline** — the original chronological vertical list (`OrderTimelineRow`),
  with `resolver_evidence` collapsed under its parent touchpoint. This is the
  pre-graph view, kept as an alternative.

`view` (`"graph" | "timeline"`) in `OrderTimelineModal` holds the active tab.
In **both** views a click on a step opens Modal 3 (event detail): the graph
reports clicks via `OrderPathGraph`'s `onSelectRow` callback and timeline rows
via `OrderTimelineRow`'s `onSelect`. `OrderTimelineModal` owns the single
`OrderEventDetailModal` instance both feed.

### Path graph

The path-graph tab renders the order's journey as a **swimlane path graph**
rather than a flat list: one horizontal lane per `event_source`, each a
`step → step → step` path connected by arrows, all converging into a single
conversion node on the right. The winning touchpoint and the edge leaving it are
highlighted.

**Compact mode** (the `compact` prop on `OrderPathGraph`, default `true`) keeps
long journeys readable:

- Runs of ≥ 2 consecutive behavior events in a lane fold into one **`N × …`
  chip**; clicking it expands the run back into individual nodes (tracked in
  `expandedRuns` on the graph; run ids encode their contents so they don't leak
  across orders). Touchpoints and the conversion always stay full cards.
- Behavior events render as small chips, touchpoints/conversion as full cards.
- Initial view: short journeys fit the whole graph; once there are more than
  `FOCUS_THRESHOLD` nodes, the view frames just the **winner + conversion** at a
  readable zoom (`fitViewOptions.nodes`) and the user pans left for history —
  this is what stops a long journey from opening fully zoomed out.

Set `compact={false}` to get the original "every row is a full card, fit all"
linear layout (kept for easy comparison / revert).

Built on `@xyflow/svelte` (SvelteFlow) + `@dagrejs/dagre`:

- `order-path-layout.ts` — pure layout. Groups rows by source into lanes, runs
  dagre left-to-right for step (X) ordering, then **snaps each node's Y into its
  lane band** so the sources read as parallel swimlanes. Returns SvelteFlow
  nodes/edges. Also owns the source label/color maps.
- `OrderPathNode.svelte` — renders both shapes: full card (touchpoint /
  conversion) and compact chip (behavior / collapsed run), switched by
  `data.compact`.
- `OrderPathLane.svelte` — the per-source label at the left of each lane.
- `OrderPathGraph.svelte` — the SvelteFlow canvas wrapper (zoom/pan + Controls).
  Listens for `on:nodeclick` and opens Modal 3 with the clicked node's row.
- `OrderEventDetailModal.svelte` — Modal 3. Opened from either view (a graph
  node card or a timeline row), it shows the full underlying
  `order_timeline_metrics` row, grouped into sections (Event, Channel & campaign,
  Page, Order, Resolution, Attribution weights, Raw params). In the graph the
  full row rides on each node via `PathNodeData.row` (lane-label nodes carry no
  row and are inert); in the timeline each `OrderTimelineRow` passes its own row.

### Gotchas

- **Node dimensions live in two places.** `NODE_W`/`NODE_H` (cards) and
  `CHIP_W`/`CHIP_H` (chips) in `order-path-layout.ts` are what dagre reserves;
  the `width`/`height` in `OrderPathNode.svelte` / `OrderPathLane.svelte` are
  what renders. Change both together or the boxes and the layout drift apart.
- **Dark mode needs explicit colors.** The canvas dashboard's theme boundary
  pins `--color-text` to its light value inside the dashboard, so raw
  `var(--color-text)` fallbacks render near-black text on dark cards. Text/
  background colors are therefore set explicitly under `:global(.dark)` in the
  node components, and `OrderPathGraph` passes `colorMode` (from `themeControl`)
  into `<SvelteFlow>` so its edges/Controls match.
- **`resolver_evidence` only shows in the Timeline tab.** The path graph omits
  it; the Timeline tab (`OrderTimelineRow.svelte`) still surfaces it collapsed
  under its parent touchpoint. If you want it in the graph too, a node-click
  popover is the natural home.
- **Dark text inside the dialog.** Because the dashboard theme boundary pins
  `--color-*` to light values, every text-bearing piece inside this dialog (node
  cards, chips, lane labels, the winner banner, and the Timeline rows) sets its
  colors explicitly under `:global(.dark)`. New text in this modal should do the
  same rather than rely on `var(--color-text)` resolving correctly.

## Files

| File                          | Role                                             |
| ----------------------------- | ------------------------------------------------ |
| `OrderDrilldownProvider.svelte` | Context registration + modal orchestration     |
| `OrderListModal.svelte`       | Modal 1 — orders behind a cell                   |
| `OrderTimelineModal.svelte`   | Modal 2 — header/winner banner + path graph      |
| `OrderPathGraph.svelte`       | SvelteFlow canvas for the path graph             |
| `OrderPathNode.svelte`        | Touchpoint / behavior / conversion node card     |
| `OrderPathLane.svelte`        | Per-source lane label                            |
| `OrderEventDetailModal.svelte`| Modal 3 — full row detail for a clicked node     |
| `order-path-layout.ts`        | Swimlane layout (dagre) + source label/color maps |
| `OrderTimelineRow.svelte`     | Row for the Timeline tab (chronological list)    |
| `ProfileListModal.svelte`     | Profiles drilldown — list                        |
| `ProfileGraphModal.svelte`    | Profiles drilldown — identity graph              |
| `api.ts`                      | Query builders + response reshaping helpers      |
| `types.ts`                    | Row / summary shapes for the timeline view       |
