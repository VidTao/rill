// Swimlane path layout for Modal 2 (order attribution graph).
//
// Each order's timeline rows come from several touchpoint sources (Shopify
// browser pixel, Shopify customer journey, the dim_orders referrer, Klaviyo,
// ...). Rather than a flat list, we render one horizontal "path" per source —
// step -> step -> step, connected by arrows — and converge every lane into a
// single conversion node on the right. The winning touchpoint (per attribution
// model) and the edge leaving it are highlighted.
//
// Layout: Dagre lays the graph out left-to-right to get sensible step (X)
// ordering and edge routing; we then snap each node's Y into its source's lane
// band so the sources read as parallel swimlanes. The conversion node is pinned
// to the far right, vertically centered across the lanes.

import { graphlib, layout as dagreLayout } from "@dagrejs/dagre";
import { Position, type Edge, type Node } from "@xyflow/svelte";
import { formatOrderLabel } from "./api";
import type { OrderTimelineRow } from "./types";

export type PathNodeKind = "behavior" | "touchpoint" | "conversion" | "lane";

export interface PathNodeData {
  // Index signature required by @xyflow/svelte's Node<T extends Record<...>>.
  [key: string]: unknown;
  kind: PathNodeKind;
  // Raw event_source, used for color tinting. Empty for the conversion node.
  source: string;
  // Human-readable source name shown on lane labels and node source tags.
  sourceLabel: string;
  title: string;
  subtitle?: string;
  ts?: string;
  isWinner: boolean;
  laneIndex: number;
  // Lane-label nodes only: how many steps are in this lane.
  laneCount?: number;
  // The underlying timeline row, for the click-through detail modal. Absent on
  // lane-label nodes (they aren't clickable).
  row?: OrderTimelineRow;
}

export interface OrderPathLayout {
  nodes: Node<PathNodeData>[];
  edges: Edge[];
  width: number;
  height: number;
}

// Node + lane geometry. Kept here (not in the component) so Dagre and the
// rendered nodes agree on sizing.
const NODE_W = 248;
const NODE_H = 104;
const LANE_V_GAP = 44;
const LANE_BAND = NODE_H + LANE_V_GAP;
const LABEL_W = 160;
const LABEL_GAP = 28;
const RANKSEP = 84;

const NODE_TYPE = "order-path-node";
const LANE_TYPE = "order-path-lane";
const CONVERSION_ID = "conversion";

// Friendly names for the sources we know about. Unknown sources fall back to a
// title-cased version of the raw identifier.
const SOURCE_LABELS: Record<string, string> = {
  shopify_browser_pixel: "Shopify Browser Pixel",
  shopify_journey: "Shopify Customer Journey",
  shopify_customer_journey: "Shopify Customer Journey",
  klaviyo: "Klaviyo",
  dim_orders: "Order / Referrer",
};

const SOURCE_COLORS: Record<string, string> = {
  shopify_browser_pixel: "#3b82f6",
  shopify_journey: "#8b5cf6",
  shopify_customer_journey: "#8b5cf6",
  klaviyo: "#ec4899",
  dim_orders: "#16a34a",
  unknown: "#6b7280",
};

const PALETTE = [
  "#3b82f6",
  "#8b5cf6",
  "#ec4899",
  "#f59e0b",
  "#14b8a6",
  "#ef4444",
  "#6366f1",
  "#0ea5e9",
];

export function humanizeSource(src: string | undefined): string {
  if (!src) return "Unknown";
  if (SOURCE_LABELS[src]) return SOURCE_LABELS[src];
  return src.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

// Stable color per source: known sources are pinned; anything else is hashed
// into the palette so the same source keeps the same color across renders.
export function sourceColor(src: string | undefined): string {
  if (!src) return SOURCE_COLORS.unknown;
  if (SOURCE_COLORS[src]) return SOURCE_COLORS[src];
  let h = 0;
  for (let i = 0; i < src.length; i++) h = (h * 31 + src.charCodeAt(i)) >>> 0;
  return PALETTE[h % PALETTE.length];
}

function fmtMoney(n: number | undefined): string | undefined {
  if (n == null) return undefined;
  return n.toLocaleString(undefined, {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 2,
  });
}

function nodeDataFor(
  row: OrderTimelineRow,
  source: string,
  laneIndex: number,
  isWinner: boolean,
): PathNodeData {
  const sourceLabel = humanizeSource(source);
  if (row.row_type === "attribution_touchpoint") {
    const subtitle =
      [row.channel_group, row.campaign, row.adset, row.ad]
        .filter((v): v is string => !!v && v.length > 0)
        .join(" / ") ||
      row.referrer ||
      undefined;
    return {
      kind: "touchpoint",
      source,
      sourceLabel,
      title: row.channel_group || row.event_type || "Touchpoint",
      subtitle,
      ts: row.event_ts,
      isWinner,
      laneIndex,
      row,
    };
  }
  // behavior_event
  return {
    kind: "behavior",
    source,
    sourceLabel,
    title: row.event_type || "Event",
    subtitle: row.url || row.referrer || undefined,
    ts: row.event_ts,
    isWinner: false,
    laneIndex,
    row,
  };
}

function makeEdge(source: string, target: string, winner: boolean): Edge {
  return {
    id: `${source}->${target}`,
    source,
    target,
    type: "smoothstep",
    animated: winner,
    style: winner
      ? "stroke:#b59f00;stroke-width:2.5px"
      : "stroke:#cbd5e1;stroke-width:1.5px",
  } as Edge;
}

/**
 * Build the SvelteFlow nodes/edges for one order's touchpoint paths.
 *
 * @param rows         Timeline rows for the order, already sorted chronologically.
 * @param winnerColumn The `is_*_winner` column matching the active attribution
 *                     model; the touchpoint flagged in that column is the winner.
 */
export function buildOrderPathLayout(
  rows: OrderTimelineRow[],
  winnerColumn: string,
): OrderPathLayout {
  // Behavior events + touchpoints become path steps; resolver_evidence is
  // omitted and the single conversion row becomes the terminal node.
  const content = rows.filter(
    (r) =>
      r.row_type === "behavior_event" ||
      r.row_type === "attribution_touchpoint",
  );
  const conversion = rows.find((r) => r.row_type === "conversion");

  // Group into lanes by source, preserving chronological order within a lane.
  const laneRows = new Map<string, OrderTimelineRow[]>();
  for (const r of content) {
    const src = (r.event_source ?? "").toString() || "unknown";
    const arr = laneRows.get(src);
    if (arr) arr.push(r);
    else laneRows.set(src, [r]);
  }
  // Order lanes by the earliest event in each — the source that started the
  // journey sits on top.
  const laneOrder = Array.from(laneRows.keys()).sort((a, b) => {
    const ats = laneRows.get(a)![0].event_ts ?? "";
    const bts = laneRows.get(b)![0].event_ts ?? "";
    return ats.localeCompare(bts);
  });

  const g = new graphlib.Graph();
  g.setGraph({
    rankdir: "LR",
    nodesep: 24,
    ranksep: RANKSEP,
    edgesep: 10,
    marginx: 8,
    marginy: 8,
  });
  g.setDefaultEdgeLabel(() => ({}));

  const nodes: Node<PathNodeData>[] = [];
  const edges: Edge[] = [];
  let idCounter = 0;

  laneOrder.forEach((source, laneIndex) => {
    const lane = laneRows.get(source)!;
    let prevId: string | null = null;
    let prevWinner = false;

    for (const row of lane) {
      const id = `${row.row_type}:${row.activity_id ?? ""}:${row.event_ts ?? ""}:${idCounter++}`;
      const isWinner =
        row.row_type === "attribution_touchpoint" &&
        !!(row as Record<string, unknown>)[winnerColumn];

      g.setNode(id, { width: NODE_W, height: NODE_H });
      nodes.push({
        id,
        type: NODE_TYPE,
        position: { x: 0, y: 0 },
        width: NODE_W,
        height: NODE_H,
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
        data: nodeDataFor(row, source, laneIndex, isWinner),
      });

      if (prevId) {
        g.setEdge(prevId, id);
        edges.push(makeEdge(prevId, id, prevWinner));
      }
      prevId = id;
      prevWinner = isWinner;
    }

    // Converge the lane's last step into the conversion node.
    if (conversion && prevId) {
      g.setEdge(prevId, CONVERSION_ID);
      edges.push(makeEdge(prevId, CONVERSION_ID, prevWinner));
    }
  });

  if (conversion) {
    g.setNode(CONVERSION_ID, { width: NODE_W, height: NODE_H });
    nodes.push({
      id: CONVERSION_ID,
      type: NODE_TYPE,
      position: { x: 0, y: 0 },
      width: NODE_W,
      height: NODE_H,
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      data: {
        kind: "conversion",
        source: "dim_orders",
        sourceLabel: "Conversion",
        title: `Order ${formatOrderLabel(conversion.order_number, conversion.order_id)}`,
        subtitle: fmtMoney(conversion.revenue),
        ts: conversion.event_ts ?? conversion.conversion_ts,
        isWinner: false,
        laneIndex: -1,
        row: conversion,
      },
    });
  }

  dagreLayout(g);

  // Keep Dagre's X (step ordering) but snap Y into lane bands so the sources
  // render as parallel swimlanes.
  let contentMaxX = 0;
  let contentMinX = Number.POSITIVE_INFINITY;
  for (const n of nodes) {
    if (n.id === CONVERSION_ID) continue;
    const dn = g.node(n.id);
    const x = (dn?.x ?? 0) - NODE_W / 2;
    n.position = { x, y: n.data.laneIndex * LANE_BAND };
    contentMaxX = Math.max(contentMaxX, x + NODE_W);
    contentMinX = Math.min(contentMinX, x);
  }
  if (!Number.isFinite(contentMinX)) contentMinX = 0;

  // Pin the conversion node to the far right, centered across the lanes.
  const centerY = ((laneOrder.length - 1) * LANE_BAND) / 2;
  const conversionNode = nodes.find((n) => n.id === CONVERSION_ID);
  if (conversionNode) {
    conversionNode.position = { x: contentMaxX + RANKSEP, y: centerY };
  }

  // One lane-label node per source, left of all the steps.
  const labelX = contentMinX - LABEL_W - LABEL_GAP;
  laneOrder.forEach((source, laneIndex) => {
    nodes.push({
      id: `lane:${laneIndex}`,
      type: LANE_TYPE,
      position: { x: labelX, y: laneIndex * LANE_BAND },
      width: LABEL_W,
      height: NODE_H,
      selectable: false,
      draggable: false,
      connectable: false,
      data: {
        kind: "lane",
        source,
        sourceLabel: humanizeSource(source),
        title: humanizeSource(source),
        isWinner: false,
        laneIndex,
        laneCount: laneRows.get(source)!.length,
      },
    });
  });

  const rightEdge = conversionNode
    ? conversionNode.position.x + NODE_W
    : contentMaxX;
  return {
    nodes,
    edges,
    width: rightEdge - labelX,
    height: Math.max(LANE_BAND, laneOrder.length * LANE_BAND),
  };
}
