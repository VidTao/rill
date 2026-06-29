<script lang="ts">
  import LoadingSpinner from "@rilldata/web-common/components/icons/LoadingSpinner.svelte";
  import ComponentError from "@rilldata/web-common/features/components/ComponentError.svelte";
  import {
    createAndExpression,
    createInExpression,
  } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
  import { createQueryServiceMetricsViewAggregation } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import ComponentHeader from "../../ComponentHeader.svelte";
  import { getCanvasStore } from "../../state-managers/state-managers";
  import {
    authoredTreeToRenderable,
    fetchAuthoredMetricTree,
    type AuthoredMetricNode,
    type AuthoredMetricTree,
  } from "./authored-tree";
  import { buildMetricTree } from "./build-tree";
  import { resolveTreeColumns } from "./columns";
  import { resolveMetricTreeLiveValues } from "./live-tree";
  import type { MetricTreeComponent } from "./index";
  import MetricTreeDetail from "./MetricTreeDetail.svelte";
  import MetricTreeGraph from "./MetricTreeGraph.svelte";
  import { validateMetricTreeSchema } from "./selector";
  import type { MetricTreeDetailData, MetricTreeNodeData } from "./types";

  export let component: MetricTreeComponent;

  type AuthoredEvent = {
    event_id: string;
    tree_id: string;
    node_id: string;
    event_type: string;
    payload?: unknown;
    created_at: string;
    created_by: string;
  };

  type AuthoredEvidence = {
    status: string;
    resultLabel?: string;
    queryBasis?: string;
    warnings?: string[];
    metrics?: Record<string, number | null>;
    dateRange?: { start?: string; end?: string };
    guardrails?: Array<{ nodeId: string; label: string; unit?: string; value?: number | null; savedValue?: number | null }>;
    message?: string;
  };

  type ReviewSummary = {
    stopReason: string;
    steps: Array<{ nodeId: string; label: string; delta: number | null; depth: number }>;
    topLevers: Array<{ nodeId: string; label: string; rootImpact: number | null }>;
  };

  type AuthoredReviewSession = {
    session_id: string;
    tree_id: string;
    selected_node_id: string;
    chosen_lever_node_id?: string;
    experiment_node_id?: string;
    action_type: string;
    note?: string;
    outcome?: string;
    next_action?: string;
    created_at: string;
    created_by: string;
    reviewWalk?: ReviewSummary["steps"];
    topLevers?: ReviewSummary["topLevers"];
    evidenceSnapshot?: Record<string, unknown>;
  };

  $: ({ instanceId } = $runtime);
  $: ({
    specStore,
    timeAndFilterStore,
    visible,
    parent: { name: canvasName },
  } = component);
  $: spec = $specStore;
  $: authoredMode =
    typeof spec.tree_id === "string" && spec.tree_id.trim() !== "";
  $: ({
    timeRange,
    comparisonTimeRange,
    where,
    hasTimeSeries,
    showTimeComparison,
    timeGrain,
  } = $timeAndFilterStore);

  let authoredTreeRaw: AuthoredMetricTree | null = null;
  let authoredTreeData: AuthoredMetricTree | null = null;
  let authoredTreeId: string | null = null;
  let authoredLoading = false;
  let authoredLiveLoading = false;
  let authoredError: string | null = null;
  let authoredRequest = 0;
  let authoredLiveRequest = 0;
  let authoredLiveKey: string | null = null;
  let authoredEvents: AuthoredEvent[] = [];
  let authoredEventsKey: string | null = null;
  let authoredEventsLoading = false;
  let authoredReviewSessions: AuthoredReviewSession[] = [];
  let authoredReviewSessionsKey: string | null = null;
  let authoredReviewSessionsLoading = false;
  let authoredEvidence: AuthoredEvidence | null = null;
  let authoredEvidenceKey: string | null = null;
  let authoredEvidenceLoading = false;
  let authoredOperatingError: string | null = null;
  let authoredOperatingSaving = false;

  async function loadAuthoredTree(treeId: string) {
    const requestId = ++authoredRequest;
    authoredTreeId = treeId;
    authoredLoading = true;
    authoredError = null;
    try {
      const data = await fetchAuthoredMetricTree(treeId);
      if (requestId === authoredRequest) {
        authoredTreeRaw = data;
        authoredTreeData = data;
        authoredLiveKey = null;
      }
    } catch (err) {
      if (requestId === authoredRequest) {
        authoredTreeRaw = null;
        authoredTreeData = null;
        authoredError =
          err instanceof Error
            ? err.message
            : "Metric tree could not be loaded";
      }
    } finally {
      if (requestId === authoredRequest) authoredLoading = false;
    }
  }

  $: if (
    authoredMode &&
    $visible &&
    spec.tree_id &&
    spec.tree_id !== authoredTreeId
  ) {
    void loadAuthoredTree(spec.tree_id);
  }
  $: if (!authoredMode) {
    authoredTreeRaw = null;
    authoredTreeData = null;
    authoredTreeId = null;
    authoredError = null;
    authoredLoading = false;
    authoredLiveLoading = false;
    authoredLiveKey = null;
    authoredEvents = [];
    authoredEventsKey = null;
    authoredReviewSessions = [];
    authoredReviewSessionsKey = null;
    authoredEvidence = null;
    authoredEvidenceKey = null;
    authoredOperatingError = null;
  }

  async function bratraxFetch<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(path, { credentials: "include", ...init });
    if (!res.ok) {
      let message = `Request failed (${res.status})`;
      try {
        const body = await res.json();
        message = body.error ?? body.message ?? message;
        if (body.validation?.errors?.[0]?.message) {
          message += `: ${body.validation.errors[0].message}`;
        }
      } catch {
        // Keep default message.
      }
      throw new Error(message);
    }
    return res.json() as Promise<T>;
  }

  function requestKey(tree: AuthoredMetricTree): string {
    return JSON.stringify({
      tree: tree.tree_id,
      updated: tree.nodes.map((n) => [n.id, n.value, n.metricBinding]),
      timeRange,
      comparisonTimeRange,
      showTimeComparison,
      timeGrain,
      where,
    });
  }

  function timeLabel(range: typeof timeRange): string | null {
    if (!range?.start || !range?.end) return null;
    return `${String(range.start).slice(0, 10)} to ${String(range.end).slice(0, 10)}`;
  }

  function filterLabel(): string | null {
    if (!where) return null;
    const expression = JSON.stringify(where);
    return expression === "{}" || expression === '{"cond":{}}'
      ? null
      : "Filtered";
  }

  async function resolveAuthoredTreeLive(tree: AuthoredMetricTree) {
    const key = requestKey(tree);
    if (key === authoredLiveKey) return;
    authoredLiveKey = key;
    const requestId = ++authoredLiveRequest;
    authoredLiveLoading = true;
    try {
      const resolved = await resolveMetricTreeLiveValues({
        tree,
        instanceId,
        timeRange,
        comparisonTimeRange,
        where,
        hasTimeSeries,
        showTimeComparison,
        dashboardTimeLabel: timeLabel(timeRange),
        dashboardFilterLabel: filterLabel(),
      });
      if (requestId === authoredLiveRequest)
        authoredTreeData = resolved ?? tree;
    } catch (err) {
      if (requestId === authoredLiveRequest) {
        authoredTreeData = tree;
        authoredError =
          err instanceof Error
            ? err.message
            : "Metric tree live data could not be resolved";
      }
    } finally {
      if (requestId === authoredLiveRequest) authoredLiveLoading = false;
    }
  }

  $: if (authoredMode && $visible && authoredTreeRaw && instanceId) {
    void resolveAuthoredTreeLive(authoredTreeRaw);
  }

  async function loadAuthoredEvents(treeId: string) {
    authoredEventsKey = treeId;
    authoredEventsLoading = true;
    try {
      const res = await bratraxFetch<{ data: AuthoredEvent[] }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/events`);
      authoredEvents = res.data ?? [];
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not load metric-tree events";
      authoredEvents = [];
    } finally {
      authoredEventsLoading = false;
    }
  }

  async function loadAuthoredReviewSessions(treeId: string) {
    authoredReviewSessionsKey = treeId;
    authoredReviewSessionsLoading = true;
    try {
      const res = await bratraxFetch<{ data: AuthoredReviewSession[] }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/review-sessions`);
      authoredReviewSessions = res.data ?? [];
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not load review sessions";
      authoredReviewSessions = [];
    } finally {
      authoredReviewSessionsLoading = false;
    }
  }

  async function loadAuthoredEvidence(treeId: string, nodeId: string, key: string) {
    authoredEvidenceKey = key;
    authoredEvidenceLoading = true;
    try {
      const res = await bratraxFetch<{ data: AuthoredEvidence }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/nodes/${encodeURIComponent(nodeId)}/evidence`);
      authoredEvidence = res.data;
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not load experiment evidence";
      authoredEvidence = null;
    } finally {
      authoredEvidenceLoading = false;
    }
  }

  async function replaceAuthoredTree(nextTree: AuthoredMetricTree, selectedId?: string) {
    authoredTreeRaw = nextTree;
    authoredTreeData = nextTree;
    authoredLiveKey = null;
    if (selectedId) {
      const rendered = authoredTreeToRenderable(nextTree).nodes.find((n) => n.id === selectedId);
      if (rendered) selectedNode = rendered;
    }
    if (nextTree.tree_id) {
      await loadAuthoredEvents(nextTree.tree_id);
      await loadAuthoredReviewSessions(nextTree.tree_id);
    }
  }

  async function updateAuthoredNode(nodeId: string, patch: Partial<AuthoredMetricNode>) {
    if (!authoredTreeRaw) return;
    authoredOperatingSaving = true;
    authoredOperatingError = null;
    try {
      const res = await bratraxFetch<{ data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(authoredTreeRaw.tree_id)}/nodes/${encodeURIComponent(nodeId)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      await replaceAuthoredTree(res.data, nodeId);
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not save node";
    } finally {
      authoredOperatingSaving = false;
    }
  }

  async function createAuthoredEvent(nodeId: string, payload: Record<string, unknown>) {
    if (!authoredTreeRaw) return;
    authoredOperatingSaving = true;
    authoredOperatingError = null;
    try {
      await bratraxFetch<{ success: boolean }>(`/bratrax/metric-trees/${encodeURIComponent(authoredTreeRaw.tree_id)}/nodes/${encodeURIComponent(nodeId)}/events`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      await loadAuthoredEvents(authoredTreeRaw.tree_id);
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not log event";
    } finally {
      authoredOperatingSaving = false;
    }
  }

  async function createReviewSession(payload: Record<string, unknown>) {
    if (!authoredTreeRaw) return;
    authoredOperatingSaving = true;
    authoredOperatingError = null;
    try {
      await bratraxFetch<{ data: AuthoredReviewSession }>(`/bratrax/metric-trees/${encodeURIComponent(authoredTreeRaw.tree_id)}/review-sessions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      await loadAuthoredTree(authoredTreeRaw.tree_id);
      await loadAuthoredEvents(authoredTreeRaw.tree_id);
      await loadAuthoredReviewSessions(authoredTreeRaw.tree_id);
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not save review";
    } finally {
      authoredOperatingSaving = false;
    }
  }

  async function createTrafficSourceTest(nodeId: string, payload: Record<string, unknown>) {
    if (!authoredTreeRaw) return;
    authoredOperatingSaving = true;
    authoredOperatingError = null;
    try {
      const res = await bratraxFetch<{ node_id: string; data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(authoredTreeRaw.tree_id)}/nodes/${encodeURIComponent(nodeId)}/traffic-source-tests`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      await replaceAuthoredTree(res.data, res.node_id);
    } catch (err) {
      authoredOperatingError = err instanceof Error ? err.message : "Could not create traffic-source test";
    } finally {
      authoredOperatingSaving = false;
    }
  }

  $: authoredEventTreeKey = authoredMode && authoredTreeRaw?.tree_id ? authoredTreeRaw.tree_id : null;
  $: if (authoredEventTreeKey && authoredEventTreeKey !== authoredEventsKey) {
    void loadAuthoredEvents(authoredEventTreeKey);
  }
  $: if (authoredEventTreeKey && authoredEventTreeKey !== authoredReviewSessionsKey) {
    void loadAuthoredReviewSessions(authoredEventTreeKey);
  }

  $: ctx = getCanvasStore(canvasName, instanceId);
  $: ({
    metricsView: { getDimensionsForMetricView, getMeasuresForMetricView },
  } = ctx.canvasEntity);

  $: schema = validateMetricTreeSchema(ctx, spec);
  $: ({ isValid, error: schemaError, isLoading: schemaLoading } = $schema);

  $: cols = resolveTreeColumns(spec);
  $: allDimensions = getDimensionsForMetricView(spec.metrics_view ?? "");
  $: allMeasures = getMeasuresForMetricView(spec.metrics_view ?? "");
  $: dimNameSet = new Set(
    $allDimensions.map((d) => (d.name || d.column) as string),
  );
  $: measNameSet = new Set($allMeasures.map((m) => m.name as string));

  // Every mapped column we want back, de-duped. tree_column is only needed when
  // filtering to a specific tree.
  $: wantedColumns = Array.from(
    new Set(
      [
        cols.node_id,
        cols.parent_node_id,
        cols.label,
        cols.value,
        cols.value2,
        cols.value3,
        cols.value2_label,
        cols.value3_label,
        cols.value2_unit,
        cols.value3_unit,
        cols.unit,
        cols.delta_value,
        cols.status,
        cols.sort_order,
        cols.confidence,
        cols.limitation,
        cols.observation,
        cols.suggested_test,
        cols.success_metric,
        cols.driver_type,
        cols.attribution_model,
        cols.comparison_model,
        cols.segment,
        cols.recommended_action,
        cols.evidence_label,
        cols.evidence_metric,
        cols.log_filter_key,
        cols.log_filter_value,
        cols.persona_filter_key,
        cols.persona_filter_value,
        spec.tree_name ? cols.tree_column : undefined,
      ].filter((c): c is string => !!c),
    ),
  );

  // Split into measures vs dimensions by what the metrics view actually models.
  $: dimensions = wantedColumns
    .filter((c) => dimNameSet.has(c))
    .map((name) => ({ name }));
  $: measures = wantedColumns
    .filter((c) => measNameSet.has(c))
    .map((name) => ({ name }));

  $: queryTimeRange = hasTimeSeries ? timeRange : undefined;

  $: rowsQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.metrics_view ?? "",
    {
      dimensions,
      measures,
      where,
      timeRange: queryTimeRange,
      limit: "1000",
      priority: 30,
    },
    {
      query: {
        enabled:
          !authoredMode &&
          isValid &&
          $visible &&
          dimensions.length > 0 &&
          (!hasTimeSeries || !!timeRange),
      },
    },
  );
  $: ({ isLoading: rowsLoading, data: rowsData } = $rowsQuery);

  $: rows = (rowsData?.data ?? []) as unknown as Record<string, unknown>[];
  $: authoredTree = authoredTreeToRenderable(authoredTreeData);
  $: fullTree = authoredMode
    ? authoredTree
    : buildMetricTree(rows, cols, { tree: spec.tree_name });

  // Drill-down: focus a node's subtree. Cleared when its node leaves the tree.
  let focusId: string | null = null;
  $: if (focusId && !fullTree.nodes.some((n) => n.id === focusId))
    focusId = null;
  $: focusNode = focusId
    ? (fullTree.nodes.find((n) => n.id === focusId) ?? null)
    : null;
  $: tree = focusId ? subtreeFrom(fullTree, focusId) : fullTree;

  function subtreeFrom(t: typeof fullTree, rootId: string): typeof fullTree {
    const keep = new Set<string>([rootId]);
    let added = true;
    while (added) {
      added = false;
      for (const e of t.edges) {
        if (keep.has(e.source) && !keep.has(e.target)) {
          keep.add(e.target);
          added = true;
        }
      }
    }
    const nodes = t.nodes
      .filter((n) => keep.has(n.id))
      .map((n) => (n.id === rootId ? { ...n, parentId: null } : n));
    const edges = t.edges.filter(
      (e) => keep.has(e.source) && keep.has(e.target),
    );
    return { ...t, nodes, edges, isEmpty: nodes.length === 0 };
  }

  function evaluateFormula(op: string, values: Array<number | null>): number | null {
    if (values.some((v) => v == null || !Number.isFinite(v))) return null;
    const nums = values as number[];
    if (op === "add") return nums.reduce((a, b) => a + b, 0);
    if (op === "multiply") return nums.reduce((a, b) => a * b, 1);
    if (op === "subtract") return nums.slice(1).reduce((a, b) => a - b, nums[0] ?? 0);
    if (op === "ratio") return nums[1] === 0 ? null : (nums[0] ?? 0) / nums[1];
    return null;
  }

  function rollupAuthored(tree: AuthoredMetricTree, overrides: Record<string, number> = {}) {
    const byId = new Map(tree.nodes.map((n) => [n.id, n]));
    const values: Record<string, number | null> = {};
    function valueFor(id: string): number | null {
      if (id in values) return values[id];
      if (id in overrides) return overrides[id];
      const node = byId.get(id);
      if (!node) return null;
      if (!node.formula || node.type === "surface" || node.type === "experiment") {
        values[id] = typeof node.value === "number" ? node.value : null;
        return values[id];
      }
      values[id] = evaluateFormula(node.formula.op, node.formula.operands.map(valueFor));
      return values[id];
    }
    for (const node of tree.nodes) valueFor(node.id);
    return values;
  }

  function rootNode(tree: AuthoredMetricTree): AuthoredMetricNode | null {
    return tree.nodes.find((n) => n.id === tree.root_node_id) ?? tree.nodes.find((n) => !n.parentId) ?? null;
  }

  function reviewSummaryFor(tree: AuthoredMetricTree | null): ReviewSummary | null {
    if (!tree) return null;
    const root = rootNode(tree);
    if (!root) return { stopReason: "missing root", steps: [], topLevers: [] };
    const steps: ReviewSummary["steps"] = [];
    let cursor: AuthoredMetricNode | undefined = root;
    while (cursor) {
      steps.push({ nodeId: cursor.id, label: cursor.label, delta: typeof cursor.delta === "number" ? cursor.delta : null, depth: steps.length });
      if (cursor.type === "lever") break;
      const children = tree.nodes.filter((n) => n.parentId === cursor?.id);
      const withDelta = children.filter((n) => typeof n.delta === "number" && Math.abs(n.delta) > 0);
      if (!children.length || !withDelta.length) break;
      cursor = [...withDelta].sort((a, b) => Math.abs(b.delta ?? 0) - Math.abs(a.delta ?? 0))[0];
    }
    const current = rollupAuthored(tree);
    const topLevers = tree.nodes
      .filter((n) => n.type === "lever")
      .map((n) => {
        const rootValue = current[root.id];
        const target = typeof n.target === "number" ? n.target : null;
        const projected = target == null ? null : rollupAuthored(tree, { [n.id]: target })[root.id];
        return { nodeId: n.id, label: n.label, rootImpact: rootValue != null && projected != null ? projected - rootValue : null };
      })
      .sort((a, b) => Math.abs(b.rootImpact ?? 0) - Math.abs(a.rootImpact ?? 0))
      .slice(0, 6);
    const last = steps[steps.length - 1];
    return { stopReason: last ? (tree.nodes.find((n) => n.id === last.nodeId)?.type === "lever" ? "owned lever reached" : "no child movement or missing live deltas") : "not reviewed", steps, topLevers };
  }

  // Detail panel selection. Drop it if the selected node leaves the full tree.
  let selectedNode: MetricTreeNodeData | null = null;
  $: if (
    selectedNode &&
    !fullTree.nodes.some((n) => n.id === selectedNode?.id)
  ) {
    selectedNode = null;
  }
  $: selectedHasChildren =
    !!selectedNode && fullTree.edges.some((e) => e.source === selectedNode?.id);
  $: authoredSelectedNode =
    authoredMode && selectedNode && authoredTreeRaw
      ? (authoredTreeRaw.nodes.find((n) => n.id === selectedNode?.id) ?? null)
      : null;
  $: authoredReviewSummary = authoredMode ? reviewSummaryFor(authoredTreeData) : null;
  $: authoredEvidenceRequestKey =
    authoredTreeRaw && authoredSelectedNode?.type === "experiment"
      ? `${authoredTreeRaw.tree_id}:${authoredSelectedNode.id}:${authoredSelectedNode.updatedAt ?? ""}`
      : null;
  $: if (authoredEvidenceRequestKey && authoredEvidenceRequestKey !== authoredEvidenceKey && authoredTreeRaw && authoredSelectedNode) {
    void loadAuthoredEvidence(authoredTreeRaw.tree_id, authoredSelectedNode.id, authoredEvidenceRequestKey);
  }
  $: if (!authoredEvidenceRequestKey && authoredEvidenceKey) {
    authoredEvidence = null;
    authoredEvidenceKey = null;
  }

  const RELATIONSHIP_DIMS = [
    "tree_name",
    "node_id",
    "related_node_id",
    "relationship_type",
    "relationship_label",
    "direction",
    "confidence",
    "explanation",
  ].map((name) => ({ name }));
  const RELATIONSHIP_MEASURES = ["sort_order", "relationship_value"].map(
    (name) => ({ name }),
  );

  const EVIDENCE_DIMS = [
    "tree_name",
    "node_id",
    "evidence_label",
    "evidence_source",
    "metric_name",
    "metric_unit",
    "attribution_model",
    "comparison_model",
    "filter_key",
    "filter_value",
    "confidence",
    "limitation",
  ].map((name) => ({ name }));
  const EVIDENCE_MEASURES = [
    "sort_order",
    "metric_value",
    "people",
    "revenue",
    "spend",
    "orders",
  ].map((name) => ({ name }));

  const PERSONA_DIMS = [
    "tree_name",
    "node_id",
    "segment",
    "persona_label",
    "recommended_action",
    "filter_key",
    "filter_value",
  ].map((name) => ({ name }));
  const PERSONA_MEASURES = ["sort_order", "profiles", "revenue", "orders"].map(
    (name) => ({ name }),
  );

  const LOG_DIMS = [
    "profile_id",
    "row_type",
    "event_type",
    "event_source",
    "title",
    "detail",
    "order_id",
    "channel_group",
    "source",
    "campaign",
    "activity_id",
    "event_ts",
  ].map((name) => ({ name }));
  const LOG_MEASURES = ["events", "metric_revenue"].map((name) => ({ name }));

  function selectedNodeWhere() {
    if (!selectedNode) return undefined;
    const clauses = [createInExpression("node_id", [selectedNode.id])];
    if (spec.tree_name)
      clauses.push(createInExpression("tree_name", [spec.tree_name]));
    return clauses.length === 1 ? clauses[0] : createAndExpression(clauses);
  }

  function selectedLogWhere() {
    if (!selectedNode?.logFilterKey || !selectedNode?.logFilterValue)
      return undefined;
    return createInExpression(selectedNode.logFilterKey, [
      selectedNode.logFilterValue,
    ]);
  }

  $: selectedRelatedWhere = selectedNodeWhere();
  $: selectedLogsWhere = selectedLogWhere();

  $: relationshipsQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.relationships_view ?? "",
    {
      dimensions: RELATIONSHIP_DIMS,
      measures: RELATIONSHIP_MEASURES,
      where: selectedRelatedWhere,
      limit: "100",
      priority: 35,
    },
    {
      query: {
        enabled:
          !authoredMode &&
          !!spec.relationships_view &&
          !!selectedNode &&
          $visible,
      },
    },
  );

  $: evidenceQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.evidence_view ?? "",
    {
      dimensions: EVIDENCE_DIMS,
      measures: EVIDENCE_MEASURES,
      where: selectedRelatedWhere,
      limit: "100",
      priority: 35,
    },
    {
      query: {
        enabled:
          !authoredMode && !!spec.evidence_view && !!selectedNode && $visible,
      },
    },
  );

  $: personasQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.personas_view ?? "",
    {
      dimensions: PERSONA_DIMS,
      measures: PERSONA_MEASURES,
      where: selectedRelatedWhere,
      limit: "100",
      priority: 35,
    },
    {
      query: {
        enabled:
          !authoredMode && !!spec.personas_view && !!selectedNode && $visible,
      },
    },
  );

  $: logsQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.logs_view ?? "",
    {
      dimensions: LOG_DIMS,
      measures: LOG_MEASURES,
      where: selectedLogsWhere,
      limit: "50",
      priority: 40,
    },
    {
      query: {
        enabled:
          !authoredMode &&
          !!spec.logs_view &&
          !!selectedNode &&
          !!selectedLogsWhere &&
          $visible,
      },
    },
  );

  $: detailData = {
    relationships: ($relationshipsQuery.data?.data ?? []) as Record<
      string,
      unknown
    >[],
    evidence: ($evidenceQuery.data?.data ?? []) as Record<string, unknown>[],
    personas: ($personasQuery.data?.data ?? []) as Record<string, unknown>[],
    logs: ($logsQuery.data?.data ?? []) as Record<string, unknown>[],
    loading:
      $relationshipsQuery.isLoading ||
      $evidenceQuery.isLoading ||
      $personasQuery.isLoading ||
      $logsQuery.isLoading,
  } satisfies MetricTreeDetailData;

  function handleSelect(n: MetricTreeNodeData) {
    selectedNode = n;
  }

  $: workspaceHref =
    authoredMode && spec.tree_id
      ? `/metric-trees?tree_id=${encodeURIComponent(spec.tree_id)}${selectedNode ? `&node_id=${encodeURIComponent(selectedNode.id)}` : ""}`
      : null;

  $: ({ title, description, show_description_as_tooltip } = spec);
  $: orientation = spec.orientation ?? "TB";
</script>

{#if schemaLoading}
  <div class="state"><LoadingSpinner size="36px" /></div>
{:else if !isValid}
  <ComponentError error={schemaError} />
{:else}
  <ComponentHeader
    {component}
    {title}
    {description}
    showDescriptionAsTooltip={show_description_as_tooltip}
  />

  {#if authoredMode && (authoredLoading || authoredLiveLoading) && !authoredTreeData}
    <div class="state grow"><LoadingSpinner size="36px" /></div>
  {:else if authoredMode && authoredError}
    <ComponentError error={authoredError} />
  {:else if !authoredMode && rowsLoading}
    <div class="state grow"><LoadingSpinner size="36px" /></div>
  {:else if tree.isEmpty}
    <div class="state grow">
      <div class="empty">
        No metric tree data{authoredMode && spec.tree_id
          ? ` for "${spec.tree_id}"`
          : spec.tree_name
            ? ` for "${spec.tree_name}"`
            : ""}.
      </div>
    </div>
  {:else}
    <div class="relative size-full grow">
      {#if focusId || selectedHasChildren}
        <div class="mt-toolbar">
          {#if focusId}
            <button class="mt-btn" on:click={() => (focusId = null)}>
              ← Full tree
            </button>
            <span class="mt-crumb">{focusNode?.label ?? ""}</span>
          {:else if selectedHasChildren}
            <button
              class="mt-btn"
              on:click={() => (focusId = selectedNode?.id ?? null)}
            >
              ⤢ Focus: {selectedNode?.label}
            </button>
          {/if}
        </div>
      {/if}
      <MetricTreeGraph
        nodes={tree.nodes}
        edges={tree.edges}
        {orientation}
        selectedId={selectedNode?.id ?? null}
        onSelect={handleSelect}
      />
      <MetricTreeDetail
        node={selectedNode}
        {detailData}
        {workspaceHref}
        authoredNode={authoredSelectedNode}
        authoredEvents={authoredEvents}
        authoredEvidence={authoredEvidence}
        authoredEvidenceLoading={authoredEvidenceLoading}
        authoredEventsLoading={authoredEventsLoading}
        authoredReviewSessions={authoredReviewSessions}
        authoredReviewSessionsLoading={authoredReviewSessionsLoading}
        authoredOperatingError={authoredOperatingError}
        authoredOperatingSaving={authoredOperatingSaving}
        authoredReviewSummary={authoredReviewSummary}
        onUpdateAuthoredNode={updateAuthoredNode}
        onCreateAuthoredEvent={createAuthoredEvent}
        onCreateReviewSession={createReviewSession}
        onCreateTrafficSourceTest={createTrafficSourceTest}
        onClose={() => (selectedNode = null)}
      />
    </div>
  {/if}
{/if}

<style lang="postcss">
  .state {
    @apply flex size-full items-center justify-center;
  }
  .empty {
    @apply rounded-lg border border-dashed border-gray-300 px-4 py-3 text-sm text-fg-muted;
  }
  .mt-toolbar {
    @apply absolute left-2 top-2 z-10 flex items-center gap-2;
  }
  .mt-btn {
    @apply rounded border border-gray-300 bg-surface px-2 py-1 text-xs font-medium text-fg shadow-sm hover:bg-gray-100;
  }
  .mt-crumb {
    @apply text-xs font-semibold text-fg-muted;
  }
</style>
