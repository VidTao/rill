# `metric_tree` Canvas component

A native Rill Canvas component that renders a **metric tree**: a top-level business
outcome (e.g. Profit) connected to the drivers underneath it (Revenue, Costs →
Ad spend, Discounts, Refunds, …). Each node shows a value + status; clicking a
node opens a detail panel with the observation, suggested test, success metric,
confidence, and limitation.

It is **data-driven**: Bratrax models generate rows, Rill renders them. It is not
a graph editor — no drag/drop, no manual layout.

- **Origin spec:** `bratrax/docs/RILL_TREE_COMPONENT.md` (requested by Brat, assignee Drasko).
- **Status:** v1 built + verified end-to-end on the `vyne` client (CH-direct). Final
  in-browser click-test pending. Expect iteration — see "Possible changes" below.
- **Downstream:** a separate Bratrax task generates the real `metric_tree_nodes`
  models + per-client dashboards that feed this primitive.

---

## Architecture

Two clean seams. The data layer is framework-free and unit-tested; everything
below `MetricTreeGraph` is pure presentation.

```
MetricTreeDisplay.svelte            registered renderer (gets the component instance)
  ├─ validateMetricTreeSchema()  ──► selector.ts      valid / loading / error
  ├─ createQueryServiceMetricsViewAggregation  ─► rows (gated on $visible)
  ├─ buildMetricTree(rows, cols) ──► build-tree.ts    pure: rows → {nodes, edges, …}
  ├─ <MetricTreeGraph>           ──► SvelteFlow + dagre layout + resize/fitView
  │     └─ <MetricTreeNode>      ──► custom node (label / value / status)
  └─ <MetricTreeDetail>          ──► right in-tile slide-over
```

How it plugs into Rill: `CanvasComponent.svelte` mounts the registered renderer via
`<svelte:component this={component.component} {component} />`. `MetricTreeComponent`
(in `index.ts`) sets `component = MetricTreeDisplay` and `type = "metric_tree"`.
`createComponent()` in `../util.ts` dispatches on the `renderer` string against
`COMPONENT_CLASS_MAP`.

## Files

| File | Responsibility |
|---|---|
| `columns.ts` | `MetricTreeSpec` (the YAML config), role→column mapping with convention defaults, `resolveTreeColumns()`, `MEASURE_ROLES`, `REQUIRED_ROLES`. |
| `build-tree.ts` | Pure `buildMetricTree(rows, cols, {tree})` → `{nodes, edges, rootIds, warnings, isEmpty}`. Handles multiple roots, orphans (promote to root), cycles (visited-set guard), `sort_order` sibling ordering, dup/null id, tree filtering, label fallback. |
| `build-tree.spec.ts` | 15 vitest cases for `buildMetricTree` + `resolveTreeColumns`. |
| `selector.ts` | `validateMetricTreeSchema(ctx, spec)` → `Readable<{isValid,error?,isLoading?}>`. Requires `node_id`/`parent_node_id`/`label` to resolve to dimensions; `value` (if set) to a measure. |
| `types.ts` | `MetricTreeNodeData` — the render contract the graph/node/detail consume. |
| `status.ts` | `MetricTreeStatus` (`good`/`warning`/`bad`/`neutral`) → light+dark color theme. |
| `layout.ts` | `buildMetricTreeLayout(nodes, edges, {orientation})` (dagre). `NODE_W=220`, `NODE_H=96`. **Sizing constants live here** and must match the node CSS. |
| `index.ts` | `MetricTreeComponent extends BaseCanvasComponent`, `inputParams()` (sidebar), `newComponentSpec()` (auto-wire on drop). Re-exports `columns.ts`. |
| `MetricTreeDisplay.svelte` | Registered renderer: data + states (loading / `ComponentError` / empty / graph). Owns `selectedNode`. |
| `MetricTreeGraph.svelte` | `<SvelteFlow>` surface: dagre layout, `colorMode` from `themeControl`, `ResizeObserver` + `{#key}` remount for fitView, `on:nodeclick`. Selection folded in without remounting. |
| `MetricTreeNode.svelte` | Custom xyflow node. Status accent (left border + dot), label (2-line clamp), value+unit, optional delta. `:global(.dark)` explicit colors. |
| `MetricTreeDetail.svelte` | Right slide-over (`absolute inset-y-0 right-0`, `min(340px,85%)`, scrim). Sections render only when the field is present. |
| `../../icons/MetricTreeIcon.svelte` | Menu/inspector icon. |

## Data contract

One metrics view, **one row per node**. `node_id` is unique; `parent_node_id`
links to the parent (empty/null = root). The view is a **current-state snapshot**
— the component does **not** thread the dashboard time range (no time-filter UI).

Roles map to metrics-view columns. All have convention defaults
(`METRIC_TREE_DEFAULT_COLUMNS`), so a view that follows the Bratrax
`metric_tree_nodes` shape needs almost no config.

| Role | Default column | Kind | Required |
|---|---|---|---|
| `node_id` | `node_id` | dimension | ✅ |
| `parent_node_id` | `parent_node_id` | dimension | ✅ |
| `label` | `label` | dimension | ✅ |
| `value` | `metric_value` | **measure** | — |
| `delta_value` | `delta_value` | **measure** | — |
| `unit` | `metric_unit` | dimension | — |
| `status` | `status` | dimension | — |
| `sort_order` | `sort_order` | dimension or measure | — |
| `confidence` / `limitation` / `observation` / `suggested_test` / `success_metric` | same name | dimension | — |
| `tree_column` | `tree_name` | dimension | — (only when filtering) |

`status` vocabulary → colors: `good` green, `warning` amber, `bad` red, `neutral`
gray (unknown → neutral). Light/dark in `status.ts`.

The query splits requested columns into `dimensions` vs `measures` by what the
metrics view actually models (so `value`/`sort_order` work whether modeled as a
measure or dimension). `limit: "1000"` (trees are small).

### `tree_name` is a value, not a column

Every other config key names a *column*; `tree_name` is the *value* to filter on
(matching the origin spec's example YAML). The column it filters is `tree_column`
(default `"tree_name"`). Omit `tree_name` for a single-tree view.

## YAML config (dashboard)

```yaml
rows:
  - height: 520px
    items:
      - width: 12
        metric_tree:
          metrics_view: metric_tree_nodes_metrics
          tree_name: Profit          # value filtered on tree_column
          # orientation: TB          # TB (default) or LR
          # everything below is optional — defaults match the convention columns:
          # node_id: node_id
          # parent_node_id: parent_node_id
          # label: label
          # value: metric_value
          # unit: metric_unit
          # status: status
          # observation: observation
          # suggested_test: suggested_test
          # success_metric: success_metric
          # confidence: confidence
          # limitation: limitation
          title: Profit tree
```

## Registration sites (for adding any future Canvas component)

The frontend is the single source of truth. To register `metric_tree` we touched:

1. `../types.ts` — `"metric_tree"` in the `CanvasComponentType` union + `MetricTreeSpec` in `ComponentWithMetricsView`.
2. `../util.ts` — import `MetricTreeComponent` + `MetricTreeIcon`; add to `NON_CHART_TYPES`, `baseComponentMap`, `IconMap`, `baseDisplayMap`.
3. `../../layout-util.ts` — `metric_tree: 360` in `initialHeights` (an **exhaustive** `Record<CanvasComponentType,number>` — a compile-time guard that forces registration).
4. `./menu-items.svelte` — Add-Component menu entry.
5. `../../AddComponentDropdown.svelte` — the second Add-Component menu entry.

**No Go/runtime changes.** `runtime/parser/parse_component.go` treats renderer
props as untyped and its JSON-schema validation is commented out
(`// TODO: Activate validation later`). `runtime/parser/data/component-template-v1.json`
is dormant — leaderboard isn't in it either. Adding a `MetricTreeTemplateT` there
is optional future-proofing only.

## Decisions (v1)

- **Orientation:** top-down (TB) default; `orientation: TB|LR` override.
- **Detail surface:** right in-tile slide-over (clips inside the tile, survives resize).
- **Time:** snapshot — no global time range threaded, time-filter UI hidden.
- **`value`:** numeric measure; the node formats `value + unit` (`percent`/`usd`/`count`/generic).
- **Click target:** every node is clickable; the panel adapts to whichever fields exist.
- **Delta on node face:** sign arrow in muted color (not green/red) to avoid "refunds down is good" confusion.

## Verifying / extending on a client (CH-direct, no DuckDB)

The Bratrax fleet is CH-direct (`rill_external_sources`); the test data lives in
ClickHouse. The throwaway harness used for `vyne` and how to recreate it is in
**`bratrax/docs/RILL_TREE_COMPONENT_IMPLEMENTATION.md`** (CH table DDL + sample
rows, the 3 generated YAMLs, rebuild/restart, reconcile checks, cleanup).

Key rule when authoring the metrics view: a CH-backed measure name **cannot equal
an underlying column name**. Store numeric columns with a `_v` suffix
(`metric_value_v`) and name the measure `metric_value` (`max(metric_value_v)`), so
the component's `value: metric_value` resolves. Dimensions may share their column name.

## Testing & quality

```bash
npx vitest run web-common/src/features/canvas/components/metric-tree/   # 15 tests
cd web-common && npx eslint src/features/canvas/components/metric-tree/  # clean
npx svelte-check --workspace web-common --no-tsconfig                    # 0 findings here
```

A web-common change only reaches the running multi-tenant Rill after a frontend
rebuild: `make cli` (or `/home/bratrax/bin/rill-rebuild.sh`, which also restarts).

## Possible changes / follow-ups

- **Warnings surfacing:** `buildMetricTree` returns `warnings` (orphans, cycles,
  dropped rows) but the Display doesn't show them yet — add a dismissible notice.
- **`component-template-v1.json`:** add a `MetricTreeTemplateT` if/when the runtime
  turns its renderer-schema validation back on.
- **AI canvas generation:** `runtime/server/generate_canvas_dashboard.go` has no
  `metric_tree` prompt; add one if the AI assistant should generate these.
- **Delta semantics / value formatting:** unit handling is minimal
  (`percent`/`usd`/`count`); extend as real `metric_unit` values appear.
- **Real data:** wire the downstream `metric_tree_nodes` model generator; this
  primitive is ready to consume it.
