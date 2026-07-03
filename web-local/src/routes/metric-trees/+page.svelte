<script lang="ts">
  import { onMount } from "svelte";
  import { browser } from "$app/environment";
  import { goto } from "$app/navigation";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { authoredTreeToRenderable } from "@rilldata/web-common/features/canvas/components/metric-tree/authored-tree";
  import MetricTreeGraph from "@rilldata/web-common/features/canvas/components/metric-tree/MetricTreeGraph.svelte";
  import MetricTreeReviewReadout from "@rilldata/web-common/features/canvas/components/metric-tree/MetricTreeReviewReadout.svelte";
  import { resolveMetricTreeLiveValues } from "@rilldata/web-common/features/canvas/components/metric-tree/live-tree";
  import {
    ownerCoverageLabel,
    ownerMatchesNode,
    ownerNodeText,
    ownershipRosterFor,
    type OwnershipPersona,
  } from "@rilldata/web-common/features/canvas/components/metric-tree/ownership";
  import type { MetricTreeNodeResolution } from "@rilldata/web-common/features/canvas/components/metric-tree/live-tree";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { getTeam } from "$lib/bratrax/settings/api";
  import type { TeamMember } from "$lib/bratrax/settings/types";
  import {
    createMetricNode,
    createMetricTreeEvent,
    createMetricTreeReviewSession,
    createStarterTree,
    deleteMetricNode,
    getMetricTree,
    getMetricTreeEvents,
    getMetricTreeEvidence,
    listMetricTreeReviewSessions,
    listMetricTreeTemplates,
    listMetricTrees,
    updateMetricNode,
    validateMetricTree,
    type MetricTreeTemplateSummary,
  } from "$lib/bratrax/metric-trees/api";
  import type {
    AuthoredMetricNode,
    AuthoredMetricTree,
    ExperimentAttributionModel,
    ExperimentMeasurementBinding,
    ExperimentStatus,
    FormulaOp,
    GoodDirection,
    MetricBindingFilter,
    MetricNodeType,
    CreateMetricTreeReviewSessionPayload,
    MetricTreeDecisionEventType,
    MetricTreeEvent,
    MetricTreeEvidence,
    MetricTreeReviewActionType,
    MetricTreeReviewDecision,
    MetricTreeReviewSession,
    MetricUnit,
  } from "$lib/bratrax/metric-trees/types";
  import {
    EXPERIMENT_STATUSES,
    FORMULA_OPS,
    LEGAL_CHILDREN,
    childrenOf,
    buildReviewWalk,
    formatValue,
    iceScore,
    nodeTypeLabel,
    propagate,
    rankExperimentsByRootImpact,
    rankLeversByRootImpact,
    rollupTree,
    validateTree,
    type LeverRootImpactRank,
    type ReviewWalk,
  } from "$lib/bratrax/metric-trees/engine";

  type TreeSummary = {
    tree_id: string;
    name: string;
    description: string;
    root_node_id: string;
    updated_at: string;
  };

  type TimePreset =
    | "last_7_days"
    | "last_30_days"
    | "last_90_days"
    | "year_to_date";

  type TimeContext = {
    key: string;
    label: string;
    timeRange: { start: string; end: string; timeZone: string };
    comparisonTimeRange: { start: string; end: string; timeZone: string };
  };

  type GuardrailLive = {
    value: number | null;
    status: "live" | "computed" | "fallback" | "unbound" | "error";
    error?: string | null;
  };

  type WorkspaceMode = "canvas" | "review";
  type WorkspaceView = "all" | "my_work";

  const UNITS: MetricUnit[] = [
    "currency",
    "count",
    "percent",
    "ratio",
    "days",
    "text",
  ];
  const GOOD_DIRECTIONS: GoodDirection[] = ["up", "down"];
  const EDITABLE_NODE_TYPES: MetricNodeType[] = [
    "output",
    "revenue_stream",
    "driver",
    "lever",
    "surface",
    "experiment",
  ];
  const ATTRIBUTION_MODELS: ExperimentAttributionModel[] = [
    "first_touch",
    "last_touch",
    "linear",
    "position_based",
    "time_decay",
  ];
  const TRAFFIC_FILTER_DIMENSIONS = [
    "source",
    "campaign",
    "channel_group",
    "adset",
    "ad",
  ];
  const DECISION_EVENT_TYPES: MetricTreeDecisionEventType[] = [
    "decision",
    "experiment_launch",
    "result_readout",
    "owner_change",
    "status_change",
  ];
  const TIME_PRESETS: Array<{ value: TimePreset; label: string }> = [
    { value: "last_7_days", label: "Last 7 days" },
    { value: "last_30_days", label: "Last 30 days" },
    { value: "last_90_days", label: "Last 90 days" },
    { value: "year_to_date", label: "Year to date" },
  ];

  let summaries: TreeSummary[] = [];
  let templates: MetricTreeTemplateSummary[] = [];
  let selectedTemplateId = "d2c_hybrid";
  let teamMembers: TeamMember[] = [];
  let tree: AuthoredMetricTree | null = null;
  let liveTree: AuthoredMetricTree | null = null;
  let liveLoading = false;
  let liveError = "";
  let liveKey = "";
  let timePreset: TimePreset = "last_30_days";
  let workspaceMode: WorkspaceMode = "canvas";
  let workspaceView: WorkspaceView = "all";
  let selectedId: string | null = null;
  let selectedDraft: AuthoredMetricNode | null = null;
  let loading = true;
  let saving = false;
  let error = "";
  let serverMessage = "";
  let evidence: MetricTreeEvidence | null = null;
  let evidenceLoading = false;
  let evidenceError = "";
  let evidenceKey = "";
  let guardrailLive: Record<string, GuardrailLive> = {};
  let guardrailLiveKey = "";
  let events: MetricTreeEvent[] = [];
  let eventsLoading = false;
  let reviewSessions: MetricTreeReviewSession[] = [];
  let selectedReviewSession: MetricTreeReviewSession | null = null;
  let reviewSessionsLoading = false;
  let ownerFilter: string | null = null;
  let reviewDecisionQueue: MetricTreeReviewDecision[] = [];
  let reviewSaving = false;
  let reviewAction: Exclude<MetricTreeReviewActionType, "none" | "guided_review"> = "log_decision";
  let reviewDecisionNote = "";
  let reviewNextAction = "";
  let reviewOutcome: "won" | "lost" | "shipped" = "won";
  let reviewTrafficLabel = "New traffic-source test";
  let reviewTrafficSource = "";
  let reviewTrafficCampaign = "";
  let reviewTrafficChannel = "";
  let reviewLeverOwner = "";
  let reviewLeverTarget: number | null = null;
  let reviewLeverTargetDate = "";
  let decisionEventType: MetricTreeDecisionEventType = "decision";
  let decisionNote = "";
  let decisionSaving = false;
  let newChildLabel = "";
  let newChildType: MetricNodeType = "driver";
  let urlSyncReady = false;
  let lastSyncedUrl = "";

  $: role = $bratraxUser?.role ?? null;
  $: canEditWorkspace = role === "admin" || role === "super_admin";
  $: instanceId = $runtime.instanceId;
  $: timeContext = buildTimeContext(timePreset);
  $: displayTree = normalizeTree(liveTree ?? tree);
  $: storedTree = normalizeTree(tree);
  $: storedNodes = storedTree?.nodes ?? [];
  $: nodes = displayTree?.nodes ?? [];
  $: root = nodes.find((n) => !n.parentId) ?? null;
  $: storedRoot = storedNodes.find((n) => !n.parentId) ?? null;
  $: graphTree = authoredTreeToRenderable(displayTree as never);
  $: liveWarnings = (
    (displayTree as unknown as { liveWarnings?: string[] } | null)
      ?.liveWarnings ?? []
  ).filter(Boolean);
  $: selectedNode = selectedId
    ? (nodes.find((n) => n.id === selectedId) ?? null)
    : root;
  $: selectedStoredNode = selectedId
    ? (storedNodes.find((n) => n.id === selectedId) ?? null)
    : storedRoot;
  $: if (selectedStoredNode && selectedDraft?.id !== selectedStoredNode.id)
    selectedDraft = cloneNode(selectedStoredNode);
  $: if (!selectedStoredNode) selectedDraft = null;
  $: localValidation = validateTree(storedNodes);
  $: rollup = displayTree ? rollupTree(displayTree) : { values: {}, drift: {} };
  $: reviewWalk = displayTree ? buildReviewWalk(displayTree) : null;
  $: rankedLevers = displayTree ? rankLeversByRootImpact(displayTree) : [];
  $: rankedExperiments = displayTree ? rankExperimentsByRootImpact(displayTree) : [];
  $: experiments = nodes
    .filter((n) => n.type === "experiment")
    .map((n) => ({
      node: n,
      score: iceScore(n),
      rootImpact: rankedExperiments.find((r) => r.nodeId === n.id)?.rootImpact ?? null,
      unsizedReason: rankedExperiments.find((r) => r.nodeId === n.id)?.unsizedReason ?? null,
    }))
    .sort((a, b) => {
      const aImpact = a.rootImpact == null ? null : Math.abs(a.rootImpact);
      const bImpact = b.rootImpact == null ? null : Math.abs(b.rootImpact);
      if (aImpact != null && bImpact == null) return -1;
      if (aImpact == null && bImpact != null) return 1;
      if (aImpact != null && bImpact != null && bImpact !== aImpact) return bImpact - aImpact;
      return (b.score ?? -1) - (a.score ?? -1);
    });
  $: ownershipRoster = ownershipRosterFor(storedNodes);
  $: currentUserId = $bratraxUser?.id ?? null;
  $: myOwnerKey = currentUserId != null ? `user:${currentUserId}` : null;
  $: myOwnership = myOwnerKey ? ownershipRoster.find((owner) => owner.ownerKey === myOwnerKey) ?? null : null;
  $: activeOwnerFilter = ownerFilter && ownershipRoster.some((owner) => owner.ownerKey === ownerFilter) ? ownerFilter : null;
  $: activeOwnerLabel = ownershipRoster.find((owner) => owner.ownerKey === activeOwnerFilter)?.owner ?? null;
  $: if (workspaceView === "my_work" && myOwnerKey && myOwnership && ownerFilter !== myOwnerKey) ownerFilter = myOwnerKey;
  $: if (workspaceView === "my_work" && storedTree && (!myOwnerKey || !myOwnership)) workspaceView = "all";
  $: myActionItemCount = myOwnership ? ownershipItemCount(myOwnership) : 0;
  $: isMyOperatingView = workspaceView === "my_work" && !!myOwnership;
  $: filteredRankedLevers = rankedLevers.filter((item) => ownerMatchesNode(activeOwnerFilter, nodeById(item.nodeId, nodes)));
  $: filteredExperiments = experiments.filter((item) => ownerMatchesNode(activeOwnerFilter, item.node));
  $: selectedRankedLever = selectedDraft?.type === "lever"
    ? rankedLevers.find((r) => r.nodeId === selectedDraft?.id) ?? null
    : null;
  $: suggestedReviewLever = reviewLeverCandidate(
    selectedNode,
    reviewWalk,
    rankedLevers,
    nodes,
  );
  $: reviewLeverExperiments = suggestedReviewLever
    ? experimentsUnderLever(suggestedReviewLever.id, nodes)
    : [];
  $: filteredReviewLeverExperiments = reviewLeverExperiments.filter((experiment) => ownerMatchesNode(activeOwnerFilter, experiment));
  $: reviewEvidenceWarnings = evidenceWarningsForReview(
    selectedNode,
    reviewWalk,
    suggestedReviewLever,
    evidence,
    evidenceLoading,
    evidenceError,
  );
  $: selectedMeasurementReadiness = selectedDraft?.type === "experiment"
    ? measurementReadiness(selectedDraft.measurementBinding)
    : null;
  $: leverProjection =
    tree && selectedDraft?.type === "lever" && selectedDraft.target != null
      ? propagate(tree, selectedDraft.id, Number(selectedDraft.target))
      : null;
  $: selectedChildren = selectedDraft
    ? childrenOf(storedNodes, selectedDraft.id)
    : [];
  $: formulaOperandsText = selectedDraft?.formula?.operands?.join(", ") ?? "";
  $: legalChildTypes = selectedDraft ? (LEGAL_CHILDREN[selectedDraft.type] ?? []) : [];
  $: if (!legalChildTypes.includes(newChildType) && legalChildTypes[0])
    newChildType = legalChildTypes[0];
  $: selectedEvidenceKey =
    tree && selectedNode?.type === "experiment"
      ? `${tree.tree_id}:${selectedNode.id}:${selectedNode.updatedAt ?? ""}`
      : "";
  $: if (selectedEvidenceKey && selectedEvidenceKey !== evidenceKey)
    void loadEvidence(selectedEvidenceKey);
  $: if (!selectedEvidenceKey && evidenceKey) clearEvidence();
  $: workspaceLiveKey =
    tree && instanceId
      ? `${tree.tree_id}:${tree.updated_at ?? ""}:${timeContext.key}:${(tree.nodes ?? []).map((n) => `${n.id}:${n.updatedAt ?? ""}:${JSON.stringify(n.metricBinding ?? {})}`).join("|")}`
      : "";
  $: if (workspaceLiveKey && workspaceLiveKey !== liveKey)
    void loadLiveTree(workspaceLiveKey);

  onMount(() => {
    const link = initialDeepLink();
    if (link.timePreset) timePreset = link.timePreset;
    if (link.mode) workspaceMode = link.mode;
    workspaceView = link.view;
    ownerFilter = link.owner;
    if (link.view === "my_work" && !link.mode) workspaceMode = "review";
    void loadTeamMembers();
    void loadSummaries(link.treeId ?? undefined, link.treeId ? link.nodeId : null).finally(
      () => {
        if (link.reviewSessionId) {
          const session = reviewSessions.find((item) => item.session_id === link.reviewSessionId);
          if (session) openReviewSession(session);
        }
        urlSyncReady = true;
      },
    );
  });

  $: if (urlSyncReady) syncUrl(
    tree?.tree_id ?? null,
    selectedId,
    workspaceMode,
    timePreset,
    activeOwnerFilter,
    selectedReviewSession?.session_id ?? null,
    workspaceView,
  );

  function cloneNode(node: AuthoredMetricNode): AuthoredMetricNode {
    return JSON.parse(JSON.stringify(node)) as AuthoredMetricNode;
  }

  function normalizeArray<T>(value: T[] | null | undefined): T[] {
    return Array.isArray(value) ? value : [];
  }

  function normalizeTree(nextTree: AuthoredMetricTree | null | undefined): AuthoredMetricTree | null {
    if (!nextTree) return null;
    const safeNodes = normalizeArray(nextTree.nodes).filter(
      (node): node is AuthoredMetricNode =>
        !!node && typeof node.id === "string" && typeof node.label === "string",
    );
    return { ...nextTree, nodes: safeNodes };
  }

  function isoDate(date: Date): string {
    return date.toISOString().slice(0, 10);
  }

  function startOfUtcDay(date: Date): Date {
    return new Date(
      Date.UTC(
        date.getUTCFullYear(),
        date.getUTCMonth(),
        date.getUTCDate(),
        0,
        0,
        0,
        0,
      ),
    );
  }

  function endOfUtcDay(date: Date): Date {
    return new Date(
      Date.UTC(
        date.getUTCFullYear(),
        date.getUTCMonth(),
        date.getUTCDate(),
        23,
        59,
        59,
        999,
      ),
    );
  }

  function addDays(date: Date, days: number): Date {
    const next = new Date(date);
    next.setUTCDate(next.getUTCDate() + days);
    return next;
  }

  function buildTimeContext(preset: TimePreset): TimeContext {
    const today = new Date();
    const end = startOfUtcDay(today);
    let start: Date;
    let label: string;
    if (preset === "last_7_days") {
      start = addDays(end, -6);
      label = "Last 7 days";
    } else if (preset === "last_90_days") {
      start = addDays(end, -89);
      label = "Last 90 days";
    } else if (preset === "year_to_date") {
      start = new Date(Date.UTC(end.getUTCFullYear(), 0, 1));
      label = "Year to date";
    } else {
      start = addDays(end, -29);
      label = "Last 30 days";
    }
    const days = Math.max(
      1,
      Math.round((end.getTime() - start.getTime()) / 86400000) + 1,
    );
    const previousEnd = addDays(start, -1);
    const previousStart = addDays(previousEnd, -(days - 1));
    const rangeEnd = endOfUtcDay(end);
    const previousRangeEnd = endOfUtcDay(previousEnd);
    return {
      key: `${preset}:${start.toISOString()}:${rangeEnd.toISOString()}`,
      label,
      timeRange: {
        start: start.toISOString(),
        end: rangeEnd.toISOString(),
        timeZone: "UTC",
      },
      comparisonTimeRange: {
        start: previousStart.toISOString(),
        end: previousRangeEnd.toISOString(),
        timeZone: "UTC",
      },
    };
  }

  function timeRangeLabel(
    range: { start?: string; end?: string } | undefined,
  ): string | null {
    if (!range?.start || !range?.end) return null;
    return `${range.start.slice(0, 10)} to ${range.end.slice(0, 10)}`;
  }

  async function loadLiveTree(nextKey: string) {
    if (!tree || !instanceId) return;
    liveKey = nextKey;
    liveLoading = true;
    liveError = "";
    try {
      const resolved = await resolveMetricTreeLiveValues({
        tree: tree as never,
        instanceId,
        timeRange: timeContext.timeRange,
        comparisonTimeRange: timeContext.comparisonTimeRange,
        hasTimeSeries: true,
        showTimeComparison: true,
        dashboardTimeLabel: timeContext.label,
      });
      liveTree = (resolved as unknown as AuthoredMetricTree) ?? tree;
    } catch (err) {
      liveTree = null;
      liveError =
        err instanceof Error
          ? err.message
          : "Could not resolve live metric tree values.";
    } finally {
      liveLoading = false;
    }
  }

  function initialDeepLink(): { treeId: string | null; nodeId: string | null; mode: WorkspaceMode | null; owner: string | null; reviewSessionId: string | null; timePreset: TimePreset | null; view: WorkspaceView } {
    if (!browser) return { treeId: null, nodeId: null, mode: null, owner: null, reviewSessionId: null, timePreset: null, view: "all" };
    const params = new URLSearchParams(window.location.search);
    const treeId = params.get("tree_id")?.trim() || null;
    const mode = params.get("mode") === "review" ? "review" : params.get("mode") === "canvas" ? "canvas" : null;
    const preset = params.get("time_preset") as TimePreset | null;
    const view = params.get("view") === "my_work" ? "my_work" : "all";
    return {
      treeId,
      nodeId: treeId ? params.get("node_id")?.trim() || null : null,
      mode,
      owner: params.get("owner")?.trim() || null,
      reviewSessionId: params.get("review_session_id")?.trim() || null,
      timePreset: TIME_PRESETS.some((item) => item.value === preset) ? preset : null,
      view,
    };
  }

  function rootNodeId(nextTree: AuthoredMetricTree): string | null {
    return (
      nextTree.root_node_id ??
      nextTree.nodes.find((n) => !n.parentId)?.id ??
      null
    );
  }

  function resolveSelectedId(
    nextTree: AuthoredMetricTree,
    preferredNodeId?: string | null,
  ): string | null {
    if (preferredNodeId && nextTree.nodes.some((n) => n.id === preferredNodeId))
      return preferredNodeId;
    return rootNodeId(nextTree);
  }

  function setSelectedId(nextId: string | null) {
    selectedId = nextId;
  }

  function syncUrl(
    treeId: string | null,
    nodeId: string | null,
    mode: WorkspaceMode,
    preset: TimePreset,
    ownerKey: string | null,
    reviewSessionId: string | null,
    view: WorkspaceView,
  ) {
    if (!browser) return;
    const url = new URL(window.location.href);
    if (!treeId) {
      url.searchParams.delete("tree_id");
      url.searchParams.delete("node_id");
      url.searchParams.delete("review_session_id");
    } else {
      url.searchParams.set("tree_id", treeId);
      if (nodeId) url.searchParams.set("node_id", nodeId);
      else url.searchParams.delete("node_id");
    }
    url.searchParams.set("mode", mode);
    url.searchParams.set("time_preset", preset);
    if (ownerKey) url.searchParams.set("owner", ownerKey);
    else url.searchParams.delete("owner");
    if (reviewSessionId) url.searchParams.set("review_session_id", reviewSessionId);
    else url.searchParams.delete("review_session_id");
    if (view === "my_work") url.searchParams.set("view", view);
    else url.searchParams.delete("view");
    const nextUrl = `${url.pathname}${url.search}`;
    if (
      nextUrl === lastSyncedUrl ||
      nextUrl === `${window.location.pathname}${window.location.search}`
    ) {
      lastSyncedUrl = nextUrl;
      return;
    }
    lastSyncedUrl = nextUrl;
    void goto(nextUrl, { replaceState: true, noScroll: true, keepFocus: true });
  }

  async function loadSummaries(
    preferredId?: string,
    preferredNodeId?: string | null,
  ) {
    loading = true;
    error = "";
    try {
      summaries = normalizeArray(await listMetricTrees());
      templates = normalizeArray(await listMetricTreeTemplates()).filter(
        (template) => !!template?.templateId,
      );
      const treeId = preferredId ?? tree?.tree_id ?? summaries?.[0]?.tree_id;
      if (treeId) await loadTree(treeId, preferredNodeId);
      else tree = null;
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Could not load metric trees.";
    } finally {
      loading = false;
    }
  }

  async function loadTree(treeId: string, preferredNodeId?: string | null) {
    error = "";
    tree = normalizeTree(await getMetricTree(treeId));
    liveTree = null;
    liveKey = "";
    setSelectedId(tree ? resolveSelectedId(tree, preferredNodeId) : null);
    serverMessage = "";
    clearEvidence();
    resetReviewDraft();
    selectedReviewSession = null;
    await loadEvents(treeId);
    await loadReviewSessions(treeId);
  }

  async function seedStarter() {
    saving = true;
    error = "";
    try {
      const seeded = normalizeTree(await createStarterTree(selectedTemplateId));
      if (!seeded) throw new Error("Starter tree response did not include a tree.");
      tree = seeded;
      setSelectedId(rootNodeId(seeded));
      await loadSummaries(seeded.tree_id);
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Could not seed starter tree.";
    } finally {
      saving = false;
    }
  }

  async function runServerValidation() {
    if (!tree) return;
    saving = true;
    error = "";
    try {
      tree.validation = await validateMetricTree(tree.tree_id);
      serverMessage = `Server validation: ${tree.validation.errors.length} errors, ${tree.validation.warnings.length} warnings.`;
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not validate tree.";
    } finally {
      saving = false;
    }
  }

  async function saveSelectedNode() {
    if (!tree || !selectedDraft) return;
    saving = true;
    error = "";
    try {
      const treeId = tree.tree_id;
      const payload = normalizeNode(selectedDraft);
      tree = normalizeTree(await updateMetricNode(treeId, payload.id, payload));
      if (!tree) throw new Error("Save response did not include a tree.");
      liveTree = null;
      liveKey = "";
      setSelectedId(payload.id);
      serverMessage = "Node saved.";
      clearEvidence();
      await loadEvents(treeId);
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not save node.";
    } finally {
      saving = false;
    }
  }

  async function removeSelectedNode() {
    if (!tree || !selectedDraft || !selectedDraft.parentId) return;
    saving = true;
    error = "";
    try {
      const treeId = tree.tree_id;
      const parentId = selectedDraft.parentId;
      tree = normalizeTree(await deleteMetricNode(treeId, selectedDraft.id));
      if (!tree) throw new Error("Delete response did not include a tree.");
      liveTree = null;
      liveKey = "";
      setSelectedId(parentId);
      serverMessage = "Node deleted.";
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not delete node.";
    } finally {
      saving = false;
    }
  }

  async function addChild() {
    if (!tree || !selectedDraft || !newChildLabel.trim()) return;
    saving = true;
    error = "";
    try {
      const treeId = tree.tree_id;
      const node = defaultChildNode(
        selectedDraft.id,
        newChildType,
        newChildLabel.trim(),
        selectedChildren.length,
      );
      tree = normalizeTree(await createMetricNode(treeId, node));
      if (!tree) throw new Error("Create response did not include a tree.");
      liveTree = null;
      liveKey = "";
      setSelectedId(
        node.id ?? tree.nodes[tree.nodes.length - 1]?.id ?? selectedDraft.id,
      );
      newChildLabel = "";
      serverMessage = "Child node added.";
      await loadEvents(treeId);
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not add child node.";
    } finally {
      saving = false;
    }
  }

  async function loadEvents(treeId: string) {
    eventsLoading = true;
    try {
      events = normalizeArray(await getMetricTreeEvents(treeId));
    } catch {
      events = [];
    } finally {
      eventsLoading = false;
    }
  }

  async function loadReviewSessions(treeId: string) {
    reviewSessionsLoading = true;
    try {
      reviewSessions = normalizeArray(await listMetricTreeReviewSessions(treeId));
    } catch {
      reviewSessions = [];
    } finally {
      reviewSessionsLoading = false;
    }
  }

  async function loadEvidence(nextKey: string) {
    if (!tree || !selectedNode || selectedNode.type !== "experiment") return;
    evidenceKey = nextKey;
    evidenceLoading = true;
    evidenceError = "";
    try {
      evidence = await getMetricTreeEvidence(tree.tree_id, selectedNode.id);
      await resolveEvidenceGuardrails(evidence);
    } catch (err) {
      evidence = null;
      evidenceError =
        err instanceof Error
          ? err.message
          : "Could not load experiment evidence.";
    } finally {
      evidenceLoading = false;
    }
  }

  async function loadTeamMembers() {
    try {
      const team = await getTeam();
      teamMembers = team.members ?? [];
    } catch {
      teamMembers = [];
    }
  }

  function ownerOptionLabel(member: TeamMember): string {
    return member.name || member.email;
  }

  function ownerSelectionValue(node: AuthoredMetricNode | null): string {
    if (!node) return "";
    if (node.ownerUserId != null) return `user:${node.ownerUserId}`;
    const label = (node.ownerLabel || node.owner || "").trim();
    return label ? `label:${label}` : "";
  }

  function applyOwnerSelection(value: string) {
    if (!selectedDraft) return;
    if (!value) {
      selectedDraft.ownerUserId = null;
      selectedDraft.ownerLabel = null;
      selectedDraft.owner = null;
    } else if (value.startsWith("user:")) {
      const userId = Number(value.slice(5));
      const member = teamMembers.find((item) => item.id === userId);
      selectedDraft.ownerUserId = Number.isFinite(userId) ? userId : null;
      selectedDraft.ownerLabel = member ? ownerOptionLabel(member) : selectedDraft.ownerLabel;
      selectedDraft.owner = selectedDraft.ownerLabel ?? null;
    } else if (value.startsWith("label:")) {
      const label = value.slice(6).trim();
      selectedDraft.ownerUserId = null;
      selectedDraft.ownerLabel = label || null;
      selectedDraft.owner = label || null;
    }
    selectedDraft = selectedDraft;
  }

  function setCustomOwnerLabel(label: string) {
    if (!selectedDraft) return;
    selectedDraft.ownerUserId = null;
    selectedDraft.ownerLabel = label.trim() || null;
    selectedDraft.owner = selectedDraft.ownerLabel;
    selectedDraft = selectedDraft;
  }

  function ownershipItemCount(owner: OwnershipPersona): number {
    return owner.leverCount + owner.experimentCount;
  }

  function selectMyOperatingView() {
    if (!myOwnerKey || !myOwnership) return;
    workspaceView = "my_work";
    workspaceMode = "review";
    ownerFilter = myOwnerKey;
  }

  function selectOwnerFilter(owner: string | null) {
    workspaceView = "all";
    ownerFilter = owner;
  }

  function resetReviewDraft() {
    reviewDecisionQueue = [];
    reviewAction = "log_decision";
    reviewDecisionNote = "";
    reviewNextAction = "";
    reviewOutcome = "won";
    reviewTrafficLabel = "New traffic-source test";
    reviewTrafficSource = "";
    reviewTrafficCampaign = "";
    reviewTrafficChannel = "";
    reviewLeverOwner = "";
    reviewLeverTarget = null;
    reviewLeverTargetDate = "";
  }

  function nodeById(
    id: string | null | undefined,
    sourceNodes: AuthoredMetricNode[] = nodes,
  ): AuthoredMetricNode | null {
    if (!id) return null;
    return sourceNodes.find((n) => n.id === id) ?? null;
  }

  function descendantsOf(
    parentId: string,
    sourceNodes: AuthoredMetricNode[] = nodes,
  ): AuthoredMetricNode[] {
    const out: AuthoredMetricNode[] = [];
    const stack = childrenOf(sourceNodes, parentId);
    while (stack.length) {
      const next = stack.shift();
      if (!next) continue;
      out.push(next);
      stack.push(...childrenOf(sourceNodes, next.id));
    }
    return out;
  }

  function reviewLeverCandidate(
    nextSelectedNode: AuthoredMetricNode | null,
    nextReviewWalk: ReviewWalk | null,
    nextRankedLevers: LeverRootImpactRank[] | null | undefined,
    sourceNodes: AuthoredMetricNode[],
  ): AuthoredMetricNode | null {
    if (nextSelectedNode?.type === "lever") return nextSelectedNode;
    if (nextReviewWalk?.stopReason === "lever") {
      const stopped = nextReviewWalk.steps?.[nextReviewWalk.steps.length - 1];
      const node = nodeById(stopped?.nodeId, sourceNodes);
      if (node?.type === "lever") return node;
    }
    const ranked = nodeById(nextRankedLevers?.[0]?.nodeId, sourceNodes);
    return ranked?.type === "lever" ? ranked : null;
  }

  function experimentsUnderLever(
    leverId: string,
    sourceNodes: AuthoredMetricNode[] = nodes,
  ): AuthoredMetricNode[] {
    return descendantsOf(leverId, sourceNodes).filter((n) => n.type === "experiment");
  }

  function evidenceWarningsForReview(
    nextSelectedNode: AuthoredMetricNode | null,
    nextReviewWalk: ReviewWalk | null,
    nextSuggestedReviewLever: AuthoredMetricNode | null,
    nextEvidence: MetricTreeEvidence | null,
    nextEvidenceLoading: boolean,
    nextEvidenceError: string,
  ): string[] {
    const warnings: string[] = [];
    if (nextSelectedNode?.type === "experiment") {
      if (nextSelectedNode.measurementBinding && nextEvidenceLoading) warnings.push("Experiment evidence is still loading.");
      if (nextSelectedNode.measurementBinding && nextEvidenceError) warnings.push(`Experiment evidence failed: ${nextEvidenceError}`);
      if (nextSelectedNode.measurementBinding && (!nextEvidence || nextEvidence.status !== "ok")) warnings.push("Experiment has a measurement binding but no loaded evidence snapshot.");
      if (!nextSelectedNode.measurementBinding) warnings.push("Selected experiment has no measurement binding.");
    }
    if (!nextReviewWalk?.steps?.length) warnings.push("Review walk is empty because live movement is unavailable.");
    if (!nextSuggestedReviewLever) warnings.push("No accountable lever selected.");
    return warnings;
  }

  function reviewEvidenceSnapshot(): Record<string, unknown> {
    return {
      selectedNodeId: selectedNode?.id ?? null,
      selectedNodeLabel: selectedNode?.label ?? null,
      evidence: evidence ?? null,
      guardrailLive,
      warnings: reviewEvidenceWarnings,
      capturedAt: new Date().toISOString(),
    };
  }

  function reviewSummaryPayload(): NonNullable<CreateMetricTreeReviewSessionPayload["summary"]> {
    return {
      rootNodeId: root?.id,
      rootLabel: root?.label,
      rootDelta: root?.delta ?? null,
      rootValue: root ? rollup.values[root.id] ?? root.value ?? null : null,
      chosenLeverId: suggestedReviewLever?.id,
      chosenLeverLabel: suggestedReviewLever?.label,
      chosenLeverOwner: suggestedReviewLever?.owner ?? null,
      chosenLeverTarget: suggestedReviewLever?.target ?? null,
      timeLabel: timeContext.label,
      stopReason: reviewWalk?.stopReason,
      warnings: reviewEvidenceWarnings,
    };
  }

  function resetReviewComposer() {
    reviewDecisionNote = "";
    reviewNextAction = "";
    reviewTrafficLabel = "New traffic-source test";
    reviewTrafficSource = "";
    reviewTrafficCampaign = "";
    reviewTrafficChannel = "";
    reviewLeverOwner = "";
    reviewLeverTarget = null;
    reviewLeverTargetDate = "";
  }

  function queueReviewDecision() {
    if (!tree) return;
    const lever = suggestedReviewLever;
    const note = reviewDecisionNote.trim();
    let decision: MetricTreeReviewDecision | null = null;
    if (reviewAction === "log_decision") {
      if (!note) return;
      decision = {
        actionType: "log_decision",
        nodeId: selectedNode?.id ?? lever?.id ?? root?.id,
        note,
        nextAction: reviewNextAction.trim() || undefined,
      };
    } else if (reviewAction === "update_lever_plan") {
      if (!lever) return;
      decision = {
        actionType: "update_lever_plan",
        leverNodeId: lever.id,
        note: note || "Updated lever plan",
        owner: reviewLeverOwner.trim() || lever.owner || null,
        target: reviewLeverTarget == null ? lever.target ?? null : Number(reviewLeverTarget),
        targetDate: reviewLeverTargetDate || lever.targetDate || null,
        nextAction: reviewNextAction.trim() || undefined,
      };
    } else if (reviewAction === "result_readout") {
      if (selectedNode?.type !== "experiment") return;
      if (selectedNode.measurementBinding && (!evidence || evidence.status !== "ok" || evidenceLoading || evidenceError)) return;
      decision = {
        actionType: "result_readout",
        experimentNodeId: selectedNode.id,
        note: note || `Marked ${reviewOutcome}`,
        outcome: reviewOutcome,
        nextAction: reviewNextAction.trim() || undefined,
        evidenceSnapshot: reviewEvidenceSnapshot(),
      };
    } else if (reviewAction === "create_traffic_source_test") {
      if (!lever) return;
      if (!reviewTrafficSource.trim() && !reviewTrafficCampaign.trim() && !reviewTrafficChannel.trim()) return;
      decision = {
        actionType: "create_traffic_source_test",
        leverNodeId: lever.id,
        note: note || "Create traffic-source test",
        nextAction: reviewNextAction.trim() || undefined,
        trafficSourceTest: {
          label: reviewTrafficLabel.trim() || "New traffic-source test",
          source: reviewTrafficSource.trim() || undefined,
          campaign: reviewTrafficCampaign.trim() || undefined,
          channel_group: reviewTrafficChannel.trim() || undefined,
          startDate: isoDate(new Date()),
          attributionModel: "first_touch",
          primaryMeasure: "metric_nc_cpa",
        },
      };
    }
    if (!decision) return;
    reviewDecisionQueue = [...reviewDecisionQueue, decision];
    resetReviewComposer();
  }

  function removeQueuedReviewDecision(index: number) {
    reviewDecisionQueue = reviewDecisionQueue.filter((_, i) => i !== index);
  }


  function nodeLabel(nodeId: string): string {
    return nodeById(nodeId)?.label ?? nodeId;
  }

  function reviewSessionWorkspaceHref(session: MetricTreeReviewSession | null): string | null {
    if (!session?.tree_id) return null;
    const nodeId = session.selected_node_id || session.chosen_lever_node_id || session.experiment_node_id || session.root_node_id;
    const params = new URLSearchParams({ tree_id: session.tree_id, mode: "review", review_session_id: session.session_id });
    if (nodeId) params.set("node_id", nodeId);
    const timePresetFromSession = (session.timeRange as { preset?: string } | undefined)?.preset;
    if (timePresetFromSession) params.set("time_preset", timePresetFromSession);
    return `/metric-trees?${params.toString()}`;
  }

  function reviewSessionPageHref(session: MetricTreeReviewSession | null): string | null {
    if (!session?.tree_id || !session.session_id) return null;
    return `/metric-trees/reviews/${encodeURIComponent(session.tree_id)}/${encodeURIComponent(session.session_id)}`;
  }

  function openReviewSession(session: MetricTreeReviewSession) {
    selectedReviewSession = session;
    workspaceMode = "review";
    setSelectedId(
      session.selected_node_id ||
        session.chosen_lever_node_id ||
        session.experiment_node_id ||
        session.root_node_id ||
        selectedId,
    );
  }

  function queuedDecisionLabel(decision: MetricTreeReviewDecision): string {
    if (decision.actionType === "log_decision") return decision.note || "Decision";
    if (decision.actionType === "update_lever_plan") return `Update ${nodeById(decision.leverNodeId)?.label ?? "lever plan"}`;
    if (decision.actionType === "result_readout") return `${nodeById(decision.experimentNodeId)?.label ?? "Experiment"}: ${decision.outcome}`;
    if (decision.actionType === "create_traffic_source_test") return String(decision.trafficSourceTest?.label ?? "Traffic-source test");
    return decision.actionType;
  }

  async function saveGuidedReviewSession() {
    if (!tree || !root || !suggestedReviewLever || !reviewDecisionQueue.length) return;
    reviewSaving = true;
    error = "";
    try {
      const savedSession = await createMetricTreeReviewSession(tree.tree_id, {
        actionType: "guided_review",
        status: "completed",
        selectedNodeId: selectedNode?.id ?? root.id,
        chosenLeverNodeId: suggestedReviewLever.id,
        experimentNodeId: selectedNode?.type === "experiment" ? selectedNode.id : undefined,
        timeRange: timeContext.timeRange,
        reviewWalk: reviewWalk?.steps ?? [],
        topLevers: rankedLevers.slice(0, 8),
        evidenceSnapshot: reviewEvidenceSnapshot(),
        summary: reviewSummaryPayload(),
        decisions: reviewDecisionQueue,
        note: reviewDecisionQueue[0]?.note ?? "Guided review",
        nextAction: reviewDecisionQueue[0]?.nextAction,
      });
      serverMessage = "Review session saved.";
      selectedReviewSession = savedSession;
      resetReviewDraft();
      await loadTree(tree.tree_id, selectedNode?.id ?? suggestedReviewLever.id);
      selectedReviewSession = savedSession;
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not save review session.";
    } finally {
      reviewSaving = false;
    }
  }

  async function logDecisionEvent() {
    if (!tree || !selectedDraft || !decisionNote.trim()) return;
    decisionSaving = true;
    error = "";
    try {
      await createMetricTreeEvent(tree.tree_id, selectedDraft.id, {
        eventType: decisionEventType,
        note: decisionNote.trim(),
      });
      decisionNote = "";
      serverMessage = "Decision logged.";
      await loadEvents(tree.tree_id);
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not log decision.";
    } finally {
      decisionSaving = false;
    }
  }

  function clearEvidence() {
    evidence = null;
    evidenceError = "";
    evidenceKey = "";
    guardrailLive = {};
    guardrailLiveKey = "";
  }

  async function resolveEvidenceGuardrails(
    nextEvidence: MetricTreeEvidence | null,
  ) {
    if (!tree || !instanceId || !nextEvidence?.guardrails?.length) {
      guardrailLive = {};
      return;
    }
    const ids = new Set(nextEvidence.guardrails.map((g) => g.nodeId));
    const guardrailNodes = tree.nodes.filter((n) => ids.has(n.id));
    const start = nextEvidence.dateRange?.start;
    const end = nextEvidence.dateRange?.end;
    const key = `${tree.tree_id}:${[...ids].join(",")}:${start ?? ""}:${end ?? ""}`;
    if (!guardrailNodes.length || key === guardrailLiveKey) return;
    guardrailLiveKey = key;
    try {
      const resolved = await resolveMetricTreeLiveValues({
        tree: { ...tree, nodes: guardrailNodes } as never,
        instanceId,
        timeRange: start && end ? { start, end } : undefined,
        hasTimeSeries: true,
        showTimeComparison: false,
        dashboardTimeLabel:
          timeRangeLabel(nextEvidence.dateRange) ?? "Experiment window",
      });
      const resolutions =
        (
          resolved as unknown as {
            nodeResolutions?: Record<string, MetricTreeNodeResolution>;
          }
        )?.nodeResolutions ?? {};
      guardrailLive = Object.fromEntries(
        guardrailNodes.map((node) => {
          const resolution = resolutions[node.id];
          return [
            node.id,
            {
              value: resolution?.value ?? node.value ?? null,
              status: resolution?.sourceStatus ?? "fallback",
              error: resolution?.sourceError ?? null,
            } satisfies GuardrailLive,
          ];
        }),
      );
    } catch {
      guardrailLive = {};
    }
  }

  function normalizeNode(node: AuthoredMetricNode): AuthoredMetricNode {
    const next = cloneNode(node);
    next.value = nullableNumber(next.value);
    next.delta = nullableNumber(next.delta ?? null);
    next.baseline = nullableNumber(next.baseline ?? null);
    next.target = nullableNumber(next.target ?? null);
    next.owner = emptyToNull(next.owner);
    next.ownerLabel = emptyToNull(next.ownerLabel ?? next.owner);
    next.ownerUserId = nullableNumber(next.ownerUserId);
    next.targetDate = emptyToNull(next.targetDate);
    next.hypothesis = emptyToNull(next.hypothesis);
    if (next.type === "experiment") {
      next.ice = {
        impact: clampIce(next.ice?.impact),
        confidence: clampIce(next.ice?.confidence),
        ease: clampIce(next.ice?.ease),
      };
      if (next.measurementBinding) {
        next.measurementBinding = normalizeMeasurementBinding(
          next.measurementBinding,
        );
      }
    }
    if (next.formula && !next.formula.operands.length) next.formula = null;
    return next;
  }

  function defaultChildNode(
    parentId: string,
    type: MetricNodeType,
    label: string,
    sortOrder: number,
  ): Partial<AuthoredMetricNode> {
    const id =
      label
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "_")
        .replace(/^_|_$/g, "") || `node_${Date.now()}`;
    const base: Partial<AuthoredMetricNode> = {
      id: `${parentId}_${id}_${Date.now().toString(36)}`,
      parentId,
      label,
      type,
      unit: type === "surface" || type === "experiment" ? "text" : "count",
      value: type === "surface" || type === "experiment" ? null : 0,
      goodDirection: "up",
      sortOrder,
    };
    if (type === "lever") {
      base.owner = "Unassigned";
      base.baseline = 0;
      base.target = 0;
      base.targetDate = new Date().toISOString().slice(0, 10);
    }
    if (type === "experiment") {
      base.status = "backlog";
      base.hypothesis = "Set hypothesis";
      base.ice = { impact: 5, confidence: 5, ease: 5 };
      base.measurementBinding = defaultMeasurementBinding();
    }
    return base;
  }

  function nullableNumber(value: unknown): number | null {
    if (value === "" || value == null) return null;
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }

  function emptyToNull(value: unknown): string | null {
    if (typeof value !== "string") return value == null ? null : String(value);
    const trimmed = value.trim();
    return trimmed ? trimmed : null;
  }

  function clampIce(value: unknown): number {
    const n = Number(value);
    if (!Number.isFinite(n)) return 5;
    return Math.min(10, Math.max(1, Math.round(n)));
  }

  function csvToValues(text: string): string[] {
    return text
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);
  }

  function valuesToCsv(values: string[] | undefined): string {
    return (values ?? []).join(", ");
  }

  function defaultMeasurementBinding(): ExperimentMeasurementBinding {
    return {
      type: "traffic_source_test",
      metricsView: "cross_metrics",
      primaryMeasure: "metric_nc_cpa",
      successMetricLabel:
        "NC CPA below target with nonzero new customer purchases",
      filters: [{ dimension: "source", op: "in", values: [] }],
      startDate: new Date().toISOString().slice(0, 10),
      attributionModel: "first_touch",
      guardrailNodeIds: ["conversion", "aov"],
      spendMeasure: "metric_spend",
      revenueMeasure: "metric_nc_revenue",
      ordersMeasure: "metric_nc_purchases",
      trafficMeasure: "metric_clicks",
    };
  }

  function normalizeMeasurementBinding(
    binding: ExperimentMeasurementBinding,
  ): ExperimentMeasurementBinding {
    return {
      ...binding,
      type: binding.type || "traffic_source_test",
      metricsView: binding.metricsView?.trim() || "cross_metrics",
      primaryMeasure: binding.primaryMeasure?.trim() || "metric_nc_cpa",
      successMetricLabel:
        binding.successMetricLabel?.trim() ||
        "Primary measure moved without breaking guardrails",
      filters: (binding.filters ?? [])
        .filter((f) => f.dimension && f.values?.length)
        .map((f) => ({
          dimension: f.dimension,
          op: "in",
          values: f.values.map((v) => v.trim()).filter(Boolean),
        })),
      startDate: binding.startDate || new Date().toISOString().slice(0, 10),
      endDate: binding.endDate || null,
      attributionModel: binding.attributionModel || "first_touch",
      guardrailNodeIds: binding.guardrailNodeIds ?? [],
      spendMeasure: binding.spendMeasure || null,
      revenueMeasure: binding.revenueMeasure || null,
      ordersMeasure: binding.ordersMeasure || null,
      trafficMeasure: binding.trafficMeasure || null,
    };
  }

  function ensureMeasurementBinding() {
    if (!selectedDraft) return;
    selectedDraft.measurementBinding =
      selectedDraft.measurementBinding ?? defaultMeasurementBinding();
    selectedDraft = selectedDraft;
  }

  function setMeasurementField<K extends keyof ExperimentMeasurementBinding>(
    field: K,
    value: ExperimentMeasurementBinding[K],
  ) {
    if (!selectedDraft) return;
    const current =
      selectedDraft.measurementBinding ?? defaultMeasurementBinding();
    selectedDraft.measurementBinding = { ...current, [field]: value };
    selectedDraft = selectedDraft;
  }

  function setMeasurementType(value: string) {
    if (
      ["traffic_source_test", "metric_filter_test", "manual"].includes(value)
    ) {
      setMeasurementField(
        "type",
        value as ExperimentMeasurementBinding["type"],
      );
    }
  }

  function setAttributionModel(value: string) {
    if (ATTRIBUTION_MODELS.includes(value as ExperimentAttributionModel)) {
      setMeasurementField(
        "attributionModel",
        value as ExperimentAttributionModel,
      );
    }
  }

  function filterValues(dimension: string): string {
    const filter = selectedDraft?.measurementBinding?.filters?.find(
      (f) => f.dimension === dimension,
    );
    return valuesToCsv(filter?.values);
  }

  function setFilterValues(dimension: string, text: string) {
    if (!selectedDraft) return;
    const current =
      selectedDraft.measurementBinding ?? defaultMeasurementBinding();
    const values = csvToValues(text);
    const otherFilters = (current.filters ?? []).filter(
      (f) => f.dimension !== dimension,
    );
    const nextFilters: MetricBindingFilter[] = values.length
      ? [...otherFilters, { dimension, op: "in", values }]
      : otherFilters;
    selectedDraft.measurementBinding = { ...current, filters: nextFilters };
    selectedDraft = selectedDraft;
  }

  function guardrailText(): string {
    return valuesToCsv(selectedDraft?.measurementBinding?.guardrailNodeIds);
  }

  function setGuardrails(text: string) {
    setMeasurementField("guardrailNodeIds", csvToValues(text));
  }

  function evidenceMetric(
    label: string,
    key: string,
    unit: MetricUnit | "currency" | "count" | "ratio" = "count",
  ) {
    return { label, value: evidence?.metrics?.[key], unit };
  }

  function measurementReadiness(binding: ExperimentMeasurementBinding | null | undefined): { ready: boolean; missing: string[] } {
    if (!binding) return { ready: false, missing: ["binding"] };
    const missing: string[] = [];
    if (!binding.metricsView?.trim()) missing.push("metrics view");
    if (!binding.primaryMeasure?.trim()) missing.push("primary measure");
    if (!binding.startDate) missing.push("start date");
    if (!(binding.filters ?? []).some((f) => f.dimension && f.values?.length)) missing.push("filter");
    if (!binding.attributionModel) missing.push("attribution");
    if (!binding.guardrailNodeIds?.length) missing.push("guardrails");
    return { ready: missing.length === 0, missing };
  }

  async function launchSelectedExperiment() {
    if (!selectedDraft || selectedDraft.type !== "experiment") return;
    selectedDraft.status = "running";
    await saveSelectedNode();
  }

  function labelStatus(value: unknown): string {
    return typeof value === "string" ? value.replaceAll("_", " ") : "-";
  }

  function shortPayload(event: MetricTreeEvent): string {
    const payload = event.payload;
    if (!payload || typeof payload !== "object") return "";
    const automatic = (payload as { automatic?: unknown }).automatic === true;
    if (automatic && event.event_type === "owner_change") {
      const owner = (payload as { owner?: { from?: unknown; to?: unknown } }).owner;
      return `Owner: ${labelStatus(owner?.from)} -> ${labelStatus(owner?.to)}`;
    }
    if (automatic && event.event_type === "status_change") {
      const status = (payload as { status?: { from?: unknown; to?: unknown } }).status;
      return `Status: ${labelStatus(status?.from)} -> ${labelStatus(status?.to)}`;
    }
    if (automatic && event.event_type === "experiment_launch") {
      const binding = (payload as { measurementBinding?: ExperimentMeasurementBinding }).measurementBinding;
      return `Launched: ${binding?.attributionModel?.replaceAll("_", " ") ?? "attribution"}`;
    }
    if (automatic && event.event_type === "result_readout") {
      const status = (payload as { status?: { to?: unknown } }).status;
      return `Readout: ${labelStatus(status?.to)}`;
    }
    const maybeNote = (payload as { note?: unknown }).note;
    if (typeof maybeNote === "string") return maybeNote;
    const maybeLabel =
      (payload as { label?: unknown; nodeLabel?: unknown }).label ??
      (payload as { nodeLabel?: unknown }).nodeLabel;
    if (typeof maybeLabel === "string") return maybeLabel;
    const maybeId = (payload as { id?: unknown }).id;
    if (typeof maybeId === "string") return maybeId;
    return "";
  }

  function guardrailDisplayValue(
    guardrail: NonNullable<MetricTreeEvidence["guardrails"]>[number],
  ): number | null | undefined {
    return (
      guardrailLive[guardrail.nodeId]?.value ??
      guardrail.value ??
      guardrail.savedValue
    );
  }

  function liveGuardrailLabel(status?: string): string {
    if (status === "live" || status === "computed") return "live";
    if (status === "error") return "saved - resolver error";
    if (status === "unbound") return "saved - unbound";
    return "saved";
  }

  function setFormulaEnabled(enabled: boolean) {
    if (!selectedDraft) return;
    selectedDraft.formula = enabled
      ? {
          op: selectedDraft.formula?.op ?? "add",
          operands: selectedDraft.formula?.operands ?? [],
        }
      : null;
    selectedDraft = selectedDraft;
  }

  function setFormulaOperands(text: string) {
    if (!selectedDraft) return;
    selectedDraft.formula = {
      op: selectedDraft.formula?.op ?? "add",
      operands: text
        .split(",")
        .map((v) => v.trim())
        .filter(Boolean),
    };
    selectedDraft = selectedDraft;
  }

  function selectGraphNode(node: { id: string }) {
    setSelectedId(node.id);
  }

  function metricBindingView(node: AuthoredMetricNode): string {
    return (
      node.metricBinding?.metricsView ?? node.metricBinding?.metrics_view ?? ""
    );
  }

  function reviewStopLabel(reason: string | undefined): string {
    if (reason === "lever") return "Stop: owned lever reached";
    if (reason === "leaf") return "Stop: leaf reached";
    if (reason === "missing_deltas") return "Stop: missing live deltas";
    if (reason === "no_child_movement") return "Stop: no child movement";
    if (reason === "missing_root") return "Stop: missing root";
    return "Stop: not reviewed";
  }

  function setMetricBindingField(
    field: "metricsView" | "measure" | "time_range",
    value: string,
  ) {
    if (!selectedDraft) return;
    selectedDraft.metricBinding = {
      ...(selectedDraft.metricBinding ?? {}),
      [field]: value.trim() || undefined,
    };
    selectedDraft = selectedDraft;
  }
</script>

<svelte:head>
  <title>Metric Trees</title>
</svelte:head>

<div class="metric-trees-page">
  <header class="page-header">
    <div>
      <h1>Metric Trees</h1>
      <p>
        {tree?.description ||
          "Decision trees for outputs, inputs, levers, surfaces, and experiments."}
      </p>
    </div>
    <div class="header-actions">
      <div class="mode-control" aria-label="Workspace mode">
        <button
          type="button"
          class:active={workspaceMode === "canvas"}
          on:click={() => (workspaceMode = "canvas")}>Canvas</button
        >
        <button
          type="button"
          class:active={workspaceMode === "review"}
          on:click={() => (workspaceMode = "review")}>Review</button
        >
      </div>
      <label class="time-control">
        Time
        <select bind:value={timePreset}>
          {#each TIME_PRESETS as preset}
            <option value={preset.value}>{preset.label}</option>
          {/each}
        </select>
      </label>
      <label class="time-control">
        Template
        <select bind:value={selectedTemplateId}>
          {#each templates.length ? templates : [{ templateId: "d2c_hybrid", name: "D2C hybrid" }] as template}
            <option value={template.templateId}>{template.name}</option>
          {/each}
        </select>
      </label>
      <button
        type="button"
        on:click={seedStarter}
        disabled={saving || !canEditWorkspace}>Seed template</button
      >
      <button
        type="button"
        on:click={runServerValidation}
        disabled={!tree || saving}>Validate</button
      >
    </div>
  </header>

  {#if error}
    <div class="alert error">{error}</div>
  {/if}
  {#if serverMessage}
    <div class="alert success">{serverMessage}</div>
  {/if}
  {#if role && !canEditWorkspace}
    <div class="alert info">
      Read-only workspace. Admins can edit levers, experiments, and measurement
      bindings.
    </div>
  {/if}

  <div class="workspace">
    <aside class="rail">
      <section>
        <div class="section-title">Trees</div>
        {#if loading}
          <div class="muted">Loading…</div>
        {:else if summaries.length === 0}
          <div class="empty">No authored metric trees yet.</div>
        {:else}
          <div class="tree-list">
            {#each summaries as summary}
              <button
                type="button"
                class:active={summary.tree_id === tree?.tree_id}
                on:click={() => loadTree(summary.tree_id)}
              >
                <span>{summary.name}</span>
                <small>{summary.tree_id}</small>
              </button>
            {/each}
          </div>
        {/if}
      </section>

      <section>
        <div class="section-title">Validation</div>
        <div class="scoreline">
          <strong>{localValidation.errors.length}</strong> errors
          <strong>{localValidation.warnings.length}</strong> warnings
        </div>
        {#if localValidation.errors.length || localValidation.warnings.length}
          <div class="issues">
            {#each [...localValidation.errors, ...localValidation.warnings].slice(0, 8) as issue}
              <button
                type="button"
                on:click={() => issue.node_id && setSelectedId(issue.node_id)}
              >
                <span>{issue.code}</span>
                <small>{issue.message}</small>
              </button>
            {/each}
          </div>
        {:else}
          <div class="empty">Tree passes local validation.</div>
        {/if}
      </section>

      <section>
        <div class="section-title">Ownership</div>
        <div class="ownership-summary">{ownerCoverageLabel(ownershipRoster)}</div>
        {#if myOwnership}
          <button
            type="button"
            class="my-work-button"
            class:active={isMyOperatingView}
            on:click={selectMyOperatingView}
          >
            <span>My work</span>
            <strong>{myActionItemCount}</strong>
            <small>{myOwnership.leverCount} levers · {myOwnership.experimentCount} experiments</small>
          </button>
        {:else if currentUserId != null}
          <div class="my-work-empty">No levers or experiments assigned to you yet.</div>
        {/if}
        <div class="ownership-list">
          <button
            type="button"
            class:active={!activeOwnerFilter}
            on:click={() => selectOwnerFilter(null)}
          >
            <span>All owners</span>
            <small>{ownershipRoster.reduce((sum, owner) => sum + ownershipItemCount(owner), 0)} action items</small>
          </button>
          {#each ownershipRoster as owner}
            <button
              type="button"
              class:active={activeOwnerFilter === owner.ownerKey}
              class:warn={owner.unassigned}
              on:click={() => selectOwnerFilter(owner.ownerKey)}
            >
              <span>{owner.owner}</span>
              <small>{owner.leverCount} levers · {owner.experimentCount} experiments</small>
            </button>
            <div class="owner-preview">
              {#each owner.nodes.slice(0, 3) as item}
                <span>{item.label}<em>{ownerNodeText(item)}</em></span>
              {/each}
            </div>
          {/each}
        </div>
      </section>
    </aside>

    <main class="tree-panel">
      {#if !tree}
        <div class="empty large">
          Seed the D2C starter tree or select an existing authored tree.
        </div>
      {:else}
        <div class="panel-head">
          <div>
            <h2>{tree.name}</h2>
            <p>
              {nodes.length} nodes · {timeContext.label}{activeOwnerLabel ? ` · owner: ${activeOwnerLabel}` : ""}{liveLoading
                ? " · resolving live values"
                : ""}{liveError ? " · live fallback" : ""}{isMyOperatingView ? " · my operating view" : ""}
            </p>
          </div>
          {#if root}
            <div class="root-value">
              <span>{root.label}</span>
              <strong>{formatValue(rollup.values[root.id], root.unit)}</strong>
            </div>
          {/if}
        </div>

        {#if liveWarnings.length}
          <div class="live-warning">
            {liveWarnings[0]}
          </div>
        {/if}
        {#if workspaceMode === "review"}
          <div class="review-shell">
            <section class="review-summary">
              <div>
                <span>Root movement</span>
                <strong>{formatValue(root?.delta ?? 0, root?.unit)}</strong>
              </div>
              <div>
                <span>Review status</span>
                <strong>{reviewStopLabel(reviewWalk?.stopReason)}</strong>
              </div>
              <div>
                <span>Time context</span>
                <strong>{timeContext.label}</strong>
              </div>
              <div>
                <span>Ownership</span>
                <strong>{activeOwnerLabel ?? ownerCoverageLabel(ownershipRoster)}</strong>
              </div>
            </section>

            {#if isMyOperatingView && myOwnership}
              <section class="my-operating-band">
                <div>
                  <span>My operating view</span>
                  <strong>{myOwnership.owner}</strong>
                </div>
                <div>
                  <span>Action items</span>
                  <strong>{myActionItemCount}</strong>
                </div>
                <div>
                  <span>Levers</span>
                  <strong>{myOwnership.leverCount}</strong>
                </div>
                <div>
                  <span>Experiments</span>
                  <strong>{myOwnership.experimentCount}</strong>
                </div>
              </section>
            {/if}

            <div class="review-grid">
              <section class="review-block">
                <div class="section-title">Top-down review walk</div>
                <div class="review-walk">
                  {#each reviewWalk?.steps ?? [] as step}
                    <button type="button" on:click={() => setSelectedId(step.nodeId)}>
                      <span style={`padding-left: ${step.depth * 14}px`}>{step.label}</span>
                      <strong>{formatValue(step.delta ?? 0, step.node.unit)}</strong>
                    </button>
                  {/each}
                </div>
              </section>

              <section class="review-block">
                <div class="section-title">Highest root-impact levers</div>
                {#if filteredRankedLevers.length}
                  <div class="impact-list">
                    {#each filteredRankedLevers.slice(0, 8) as item}
                      <button type="button" class:active={item.nodeId === suggestedReviewLever?.id} on:click={() => setSelectedId(item.nodeId)}>
                        <span>{item.label}</span>
                        <strong>{item.rootImpact == null ? "unsized" : formatValue(item.rootImpact, root?.unit)}</strong>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <div class="empty">No levers match the selected owner.</div>
                {/if}
              </section>
            </div>

            <section class="review-block">
              <div class="section-title">Accountable lever</div>
              {#if suggestedReviewLever}
                <div class="review-lever-card">
                  <div>
                    <span>Lever</span>
                    <strong>{suggestedReviewLever.label}</strong>
                  </div>
                  <div>
                    <span>Owner</span>
                    <strong>{suggestedReviewLever.owner || "Unassigned"}</strong>
                  </div>
                  <div>
                    <span>Target</span>
                    <strong>{formatValue(suggestedReviewLever.target ?? null, suggestedReviewLever.unit)}</strong>
                  </div>
                  <div>
                    <span>Target date</span>
                    <strong>{suggestedReviewLever.targetDate || "No date"}</strong>
                  </div>
                </div>
              {:else}
                <div class="empty">Select a lever or resolve live movement so the review can stop at one.</div>
              {/if}
            </section>

            <section class="review-block">
              <div class="section-title">Experiments under lever</div>
              {#if !reviewLeverExperiments.length}
                <div class="empty">No experiments under the chosen lever.</div>
              {:else if !filteredReviewLeverExperiments.length}
                <div class="empty">No experiments under the chosen lever match the selected owner.</div>
              {:else}
                <div class="review-experiments">
                  {#each filteredReviewLeverExperiments as experiment}
                    {@const readiness = measurementReadiness(experiment.measurementBinding)}
                    <button type="button" class:active={selectedNode?.id === experiment.id} on:click={() => setSelectedId(experiment.id)}>
                      <span>{experiment.label}</span>
                      <small>{experiment.status || "backlog"} · {readiness.ready ? "measured" : `missing ${readiness.missing.join(", ")}`}</small>
                    </button>
                  {/each}
                </div>
              {/if}
            </section>

            <section class="review-block">
              <div class="section-title">Decision queue</div>
              {#if reviewEvidenceWarnings.length}
                <div class="readiness">
                  {reviewEvidenceWarnings.join(" ")}
                </div>
              {/if}
              <div class="review-composer">
                <label>
                  Action
                  <select bind:value={reviewAction}>
                    <option value="log_decision">Decision</option>
                    <option value="update_lever_plan">Update lever plan</option>
                    <option value="result_readout" disabled={selectedNode?.type !== "experiment"}>Result readout</option>
                    <option value="create_traffic_source_test" disabled={!suggestedReviewLever}>Create traffic-source test</option>
                  </select>
                </label>

                {#if reviewAction === "result_readout"}
                  <label>
                    Outcome
                    <select bind:value={reviewOutcome}>
                      <option value="won">Won</option>
                      <option value="lost">Lost</option>
                      <option value="shipped">Shipped</option>
                    </select>
                  </label>
                {:else if reviewAction === "update_lever_plan"}
                  <div class="field-grid three">
                    <label>Owner<input bind:value={reviewLeverOwner} placeholder={suggestedReviewLever?.owner || "Owner"} /></label>
                    <label>Target<input type="number" step="any" bind:value={reviewLeverTarget} placeholder={String(suggestedReviewLever?.target ?? "")} /></label>
                    <label>Target date<input type="date" bind:value={reviewLeverTargetDate} /></label>
                  </div>
                {:else if reviewAction === "create_traffic_source_test"}
                  <div class="field-grid">
                    <label>Label<input bind:value={reviewTrafficLabel} /></label>
                    <label>Source<input bind:value={reviewTrafficSource} placeholder="reddit" /></label>
                    <label>Campaign<input bind:value={reviewTrafficCampaign} placeholder="prospecting test" /></label>
                    <label>Channel<input bind:value={reviewTrafficChannel} placeholder="paid social" /></label>
                  </div>
                {/if}

                <label>
                  Review note
                  <textarea rows="3" bind:value={reviewDecisionNote} placeholder="What did we decide in this review?" />
                </label>
                <label>
                  Next action
                  <input bind:value={reviewNextAction} placeholder="Owner and next move" />
                </label>
                <button type="button" on:click={queueReviewDecision} disabled={!canEditWorkspace || reviewSaving}>Add to queue</button>
              </div>

              {#if reviewDecisionQueue.length}
                <div class="queued-decisions">
                  {#each reviewDecisionQueue as decision, index}
                    <div>
                      <span>{decision.actionType.replaceAll("_", " ")}</span>
                      <strong>{queuedDecisionLabel(decision)}</strong>
                      <button type="button" on:click={() => removeQueuedReviewDecision(index)}>Remove</button>
                    </div>
                  {/each}
                </div>
              {:else}
                <div class="empty">Queue one or more decisions, then save the review session.</div>
              {/if}

              <button
                type="button"
                class="primary"
                on:click={saveGuidedReviewSession}
                disabled={!canEditWorkspace || reviewSaving || !suggestedReviewLever || !reviewDecisionQueue.length}
              >Save review session</button>
            </section>
          </div>
        {:else}
          <div class="graph-shell">
            <MetricTreeGraph
              nodes={graphTree.nodes}
              edges={graphTree.edges}
              orientation="TB"
              selectedId={selectedNode?.id ?? null}
              onSelect={selectGraphNode}
            />
          </div>
        {/if}
      {/if}
    </main>

    <aside class="inspector">
      {#if selectedDraft}
        <fieldset class="inspector-fields" disabled={!canEditWorkspace}>
          <section>
            <div class="section-title">Node</div>
            <label>
              Label
              <input bind:value={selectedDraft.label} />
            </label>
            <div class="field-grid">
              <label>
                Type
                <select bind:value={selectedDraft.type}>
                  {#each EDITABLE_NODE_TYPES as type}
                    <option value={type}>{nodeTypeLabel(type)}</option>
                  {/each}
                </select>
              </label>
              <label>
                Unit
                <select bind:value={selectedDraft.unit}>
                  {#each UNITS as unit}
                    <option value={unit}>{unit}</option>
                  {/each}
                </select>
              </label>
            </div>
            <div class="field-grid">
              <label>
                Current value
                <input
                  type="number"
                  step="any"
                  bind:value={selectedDraft.value}
                />
              </label>
              <label>
                Good direction
                <select bind:value={selectedDraft.goodDirection}>
                  {#each GOOD_DIRECTIONS as direction}
                    <option value={direction}>{direction}</option>
                  {/each}
                </select>
              </label>
            </div>
          </section>

          {#if selectedDraft.type === "lever"}
            <section>
              <div class="section-title">Lever</div>
              <label>
                Owner
                <select value={ownerSelectionValue(selectedDraft)} on:change={(event) => applyOwnerSelection(event.currentTarget.value)}>
                  <option value="">Unassigned</option>
                  {#each teamMembers as member}
                    <option value={`user:${member.id}`}>{ownerOptionLabel(member)} · {member.role}</option>
                  {/each}
                  {#if selectedDraft.ownerLabel || selectedDraft.owner}
                    <option value={`label:${selectedDraft.ownerLabel || selectedDraft.owner}`}>Custom: {selectedDraft.ownerLabel || selectedDraft.owner}</option>
                  {/if}
                </select>
              </label>
              <label>
                Custom owner / persona
                <input value={selectedDraft.ownerLabel || selectedDraft.owner || ""} on:input={(event) => setCustomOwnerLabel(event.currentTarget.value)} placeholder="Lifecycle / retention" />
              </label>
              <div class="field-grid">
                <label>
                  Baseline
                  <input
                    type="number"
                    step="any"
                    bind:value={selectedDraft.baseline}
                  />
                </label>
                <label>
                  Target
                  <input
                    type="number"
                    step="any"
                    bind:value={selectedDraft.target}
                  />
                </label>
              </div>
              <label>
                Target date
                <input type="date" bind:value={selectedDraft.targetDate} />
              </label>
            </section>
          {/if}

          {#if selectedDraft.type === "experiment"}
            <section>
              <div class="section-title">Experiment</div>
              <label>
                Hypothesis
                <textarea rows="3" bind:value={selectedDraft.hypothesis}
                ></textarea>
              </label>
              <label>
                Status
                <select bind:value={selectedDraft.status}>
                  {#each EXPERIMENT_STATUSES as status}
                    <option value={status}>{status}</option>
                  {/each}
                </select>
              </label>
              <div class="field-grid three">
                <label
                  >Impact<input
                    type="number"
                    min="1"
                    max="10"
                    bind:value={selectedDraft.ice.impact}
                  /></label
                >
                <label
                  >Confidence<input
                    type="number"
                    min="1"
                    max="10"
                    bind:value={selectedDraft.ice.confidence}
                  /></label
                >
                <label
                  >Ease<input
                    type="number"
                    min="1"
                    max="10"
                    bind:value={selectedDraft.ice.ease}
                  /></label
                >
              </div>
            </section>

            <section>
              <div class="section-title">Measurement binding</div>
              {#if !selectedDraft.measurementBinding}
                <button type="button" on:click={ensureMeasurementBinding}
                  >Use traffic-source template</button
                >
              {:else}
                <label>
                  Type
                  <select
                    value={selectedDraft.measurementBinding.type}
                    on:change={(e) => setMeasurementType(e.currentTarget.value)}
                  >
                    <option value="traffic_source_test"
                      >traffic_source_test</option
                    >
                    <option value="metric_filter_test"
                      >metric_filter_test</option
                    >
                    <option value="manual">manual</option>
                  </select>
                </label>
                <div class="field-grid">
                  <label>
                    Metrics view
                    <input
                      value={selectedDraft.measurementBinding.metricsView}
                      on:input={(e) =>
                        setMeasurementField(
                          "metricsView",
                          e.currentTarget.value,
                        )}
                    />
                  </label>
                  <label>
                    Primary measure
                    <input
                      value={selectedDraft.measurementBinding.primaryMeasure}
                      on:input={(e) =>
                        setMeasurementField(
                          "primaryMeasure",
                          e.currentTarget.value,
                        )}
                    />
                  </label>
                </div>
                <label>
                  Success label
                  <input
                    value={selectedDraft.measurementBinding.successMetricLabel}
                    on:input={(e) =>
                      setMeasurementField(
                        "successMetricLabel",
                        e.currentTarget.value,
                      )}
                  />
                </label>
                <div class="field-grid">
                  <label>
                    Start date
                    <input
                      type="date"
                      value={selectedDraft.measurementBinding.startDate}
                      on:input={(e) =>
                        setMeasurementField("startDate", e.currentTarget.value)}
                    />
                  </label>
                  <label>
                    End date
                    <input
                      type="date"
                      value={selectedDraft.measurementBinding.endDate ?? ""}
                      on:input={(e) =>
                        setMeasurementField(
                          "endDate",
                          e.currentTarget.value || null,
                        )}
                    />
                  </label>
                </div>
                <label>
                  Attribution model
                  <select
                    value={selectedDraft.measurementBinding.attributionModel ??
                      "first_touch"}
                    on:change={(e) =>
                      setAttributionModel(e.currentTarget.value)}
                  >
                    {#each ATTRIBUTION_MODELS as model}
                      <option value={model}>{model}</option>
                    {/each}
                  </select>
                </label>
                <div class="field-grid">
                  {#each TRAFFIC_FILTER_DIMENSIONS as dimension}
                    <label>
                      {dimension} filter
                      <input
                        value={filterValues(dimension)}
                        on:input={(e) =>
                          setFilterValues(dimension, e.currentTarget.value)}
                        placeholder="comma-separated values"
                      />
                    </label>
                  {/each}
                </div>
                <label>
                  Guardrail node ids
                  <input
                    value={guardrailText()}
                    on:input={(e) => setGuardrails(e.currentTarget.value)}
                    placeholder="conversion, aov"
                  />
                </label>
                <div class="field-grid">
                  <label
                    >Spend measure<input
                      value={selectedDraft.measurementBinding.spendMeasure ??
                        ""}
                      on:input={(e) =>
                        setMeasurementField(
                          "spendMeasure",
                          e.currentTarget.value || null,
                        )}
                    /></label
                  >
                  <label
                    >Revenue measure<input
                      value={selectedDraft.measurementBinding.revenueMeasure ??
                        ""}
                      on:input={(e) =>
                        setMeasurementField(
                          "revenueMeasure",
                          e.currentTarget.value || null,
                        )}
                    /></label
                  >
                  <label
                    >Orders measure<input
                      value={selectedDraft.measurementBinding.ordersMeasure ??
                        ""}
                      on:input={(e) =>
                        setMeasurementField(
                          "ordersMeasure",
                          e.currentTarget.value || null,
                        )}
                    /></label
                  >
                  <label
                    >Traffic measure<input
                      value={selectedDraft.measurementBinding.trafficMeasure ??
                        ""}
                      on:input={(e) =>
                        setMeasurementField(
                          "trafficMeasure",
                          e.currentTarget.value || null,
                        )}
                    /></label
                  >
                </div>
                {#if selectedMeasurementReadiness}
                  <div class:ready={selectedMeasurementReadiness.ready} class="readiness">
                    {selectedMeasurementReadiness.ready
                      ? "Ready to launch"
                      : `Missing ${selectedMeasurementReadiness.missing.join(", ")}`}
                  </div>
                {/if}
                <div class="hint">
                  Attribution-backed experiment results are directional unless a
                  holdout method is added.
                </div>
                <button
                  type="button"
                  on:click={launchSelectedExperiment}
                  disabled={saving ||
                    !canEditWorkspace ||
                    !selectedMeasurementReadiness?.ready ||
                    selectedDraft.status === "running"}
                  >Launch experiment</button
                >
              {/if}
            </section>
          {/if}

          {#if selectedDraft.type !== "surface" && selectedDraft.type !== "experiment"}
            <section>
              <div class="section-title">Formula</div>
              <label class="inline">
                <input
                  type="checkbox"
                  checked={!!selectedDraft.formula}
                  on:change={(e) => setFormulaEnabled(e.currentTarget.checked)}
                />
                Roll up from children
              </label>
              {#if selectedDraft.formula}
                <div class="field-grid">
                  <label>
                    Operation
                    <select bind:value={selectedDraft.formula.op}>
                      {#each FORMULA_OPS as op}
                        <option value={op}>{op}</option>
                      {/each}
                    </select>
                  </label>
                  <label>
                    Operands
                    <input
                      value={formulaOperandsText}
                      on:input={(e) =>
                        setFormulaOperands(e.currentTarget.value)}
                    />
                  </label>
                </div>
                <div class="hint">Operands must be direct child node ids.</div>
              {/if}
            </section>
          {/if}

          <section>
            <div class="section-title">Metric binding</div>
            <label>
              Metrics view
              <input
                value={metricBindingView(selectedDraft)}
                on:input={(e) =>
                  setMetricBindingField("metricsView", e.currentTarget.value)}
              />
            </label>
            <div class="field-grid">
              <label>
                Measure
                <input
                  value={selectedDraft.metricBinding?.measure ?? ""}
                  on:input={(e) =>
                    setMetricBindingField("measure", e.currentTarget.value)}
                />
              </label>
              <label>
                Time range
                <input
                  value={selectedDraft.metricBinding?.time_range ??
                    selectedDraft.metricBinding?.timeMode ??
                    ""}
                  on:input={(e) =>
                    setMetricBindingField("time_range", e.currentTarget.value)}
                />
              </label>
            </div>
          </section>

          <section>
            <div class="section-title">Decision output</div>
            <div class="decision-lines">
              <div>
                <span>Current</span><strong
                  >{formatValue(
                    rollup.values[selectedDraft.id] ?? selectedDraft.value,
                    selectedDraft.unit,
                  )}</strong
                >
              </div>
              {#if selectedRankedLever}
                <div>
                  <span>Root impact</span><strong
                    >{selectedRankedLever.rootImpact == null
                      ? "unsized"
                      : formatValue(selectedRankedLever.rootImpact, root?.unit)}</strong
                  >
                </div>
              {/if}
              {#if selectedDraft.type === "lever" && leverProjection}
                {#each leverProjection.path as id}
                  <div>
                    <span>{nodes.find((n) => n.id === id)?.label ?? id}</span>
                    <strong
                      >{formatValue(
                        leverProjection.delta[id] ?? 0,
                        nodes.find((n) => n.id === id)?.unit,
                      )}</strong
                    >
                  </div>
                {/each}
              {/if}
              {#if selectedDraft.type === "experiment"}
                <div>
                  <span>ICE</span><strong
                    >{iceScore(selectedDraft) ?? "-"}</strong
                  >
                </div>
              {/if}
            </div>
          </section>

          {#if legalChildTypes.length}
            <section>
              <div class="section-title">Add child</div>
              <div class="field-grid add-child">
                <input placeholder="Node label" bind:value={newChildLabel} />
                <select bind:value={newChildType}>
                  {#each legalChildTypes as type}
                    <option value={type}>{nodeTypeLabel(type)}</option>
                  {/each}
                </select>
                <button
                  type="button"
                  on:click={addChild}
                  disabled={saving ||
                    !newChildLabel.trim() ||
                    !canEditWorkspace}>Add</button
                >
              </div>
            </section>
          {/if}
        </fieldset>

        {#if selectedNode?.type === "experiment"}
          <section class="readonly-section">
            <div class="section-title">Evidence</div>
            {#if evidenceLoading}
              <div class="muted">Loading evidence...</div>
            {:else if evidenceError}
              <div class="alert error compact">{evidenceError}</div>
            {:else if !evidence || evidence.status === "unbound"}
              <div class="empty">
                No measurement binding saved for this experiment.
              </div>
            {:else if evidence.status === "unsupported"}
              <div class="empty">{evidence.message}</div>
            {:else}
              <div class="evidence-label">{evidence.resultLabel}</div>
              <div class="evidence-grid">
                {#each [evidenceMetric("Spend", "spend", "currency"), evidenceMetric("Impressions", "impressions", "count"), evidenceMetric("Clicks", "clicks", "count"), evidenceMetric("NC purchases", "nc_purchases", "count"), evidenceMetric("NC CPA", "nc_cpa", "currency"), evidenceMetric("NC revenue", "nc_revenue", "currency"), evidenceMetric("Attributed revenue", "attributed_revenue", "currency"), evidenceMetric("NC ROAS", "nc_roas", "ratio")] as item}
                  <div>
                    <span>{item.label}</span>
                    <strong>{formatValue(item.value, item.unit)}</strong>
                  </div>
                {/each}
              </div>
              {#if evidence.guardrails?.length}
                <div class="guardrails">
                  <div class="section-title small">Guardrails</div>
                  {#each evidence.guardrails as guardrail}
                    <div>
                      <span>{guardrail.label}</span>
                      <strong
                        >{formatValue(
                          guardrailDisplayValue(guardrail),
                          guardrail.unit,
                        )}</strong
                      >
                      <small
                        >{liveGuardrailLabel(
                          guardrailLive[guardrail.nodeId]?.status,
                        )}</small
                      >
                    </div>
                  {/each}
                </div>
              {/if}
            {/if}
          </section>
        {/if}

        <div class="inspector-actions">
          <button
            type="button"
            class="primary"
            on:click={saveSelectedNode}
            disabled={saving || !canEditWorkspace}>Save node</button
          >
          <button
            type="button"
            on:click={removeSelectedNode}
            disabled={saving || !selectedDraft.parentId || !canEditWorkspace}
            >Delete</button
          >
        </div>
      {:else}
        <div class="empty">Select a node to inspect it.</div>
      {/if}

      {#if experiments.length}
        <section>
          <div class="section-title">Experiment backlog</div>
          {#if filteredExperiments.length}
            <div class="backlog">
              {#each filteredExperiments.slice(0, 8) as item}
                <button
                  type="button"
                  on:click={() => setSelectedId(item.node.id)}
                >
                  <span>{item.node.label}</span>
                  <strong
                    >{item.rootImpact == null
                      ? item.score ?? "-"
                      : formatValue(item.rootImpact, root?.unit)}</strong
                  >
                </button>
              {/each}
            </div>
          {:else}
            <div class="empty">No experiments match the selected owner.</div>
          {/if}
        </section>
      {/if}

      <section>
        <div class="section-title">Log decision</div>
        {#if selectedDraft && canEditWorkspace}
          <div class="decision-composer">
            <select bind:value={decisionEventType}>
              {#each DECISION_EVENT_TYPES as type}
                <option value={type}>{type.replaceAll("_", " ")}</option>
              {/each}
            </select>
            <textarea
              bind:value={decisionNote}
              rows="3"
              placeholder="What did we decide, based on what evidence?"
            />
            <button
              type="button"
              class="primary"
              on:click={logDecisionEvent}
              disabled={decisionSaving || !decisionNote.trim()}>Log</button
            >
          </div>
        {:else}
          <div class="empty">
            Select a node. Admins can log decisions against it.
          </div>
        {/if}
      </section>

      <section>
        <div class="section-title">Recent reviews</div>
        {#if reviewSessionsLoading}
          <div class="muted">Loading reviews...</div>
        {:else if !reviewSessions.length}
          <div class="empty">No saved reviews yet.</div>
        {:else}
          <div class="event-log">
            {#each reviewSessions.slice(0, 8) as session}
              <button
                type="button"
                class:active={selectedReviewSession?.session_id === session.session_id}
                on:click={() => openReviewSession(session)}
              >
                <span>{session.action_type.replaceAll("_", " ")}</span>
                <small>{session.outcome || session.note || session.next_action || session.created_at}</small>
              </button>
            {/each}
          </div>
          <MetricTreeReviewReadout
            session={selectedReviewSession ?? reviewSessions[0]}
            nodeLabel={nodeLabel}
            workspaceHref={reviewSessionWorkspaceHref(selectedReviewSession ?? reviewSessions[0])}
            reviewHref={reviewSessionPageHref(selectedReviewSession ?? reviewSessions[0])}
          />
        {/if}
      </section>

      <section>
        <div class="section-title">Decision log</div>
        {#if eventsLoading}
          <div class="muted">Loading events...</div>
        {:else if !events.length}
          <div class="empty">No tree events yet.</div>
        {:else}
          <div class="event-log">
            {#each events.slice(0, 12) as event}
              <button
                type="button"
                on:click={() => event.node_id && setSelectedId(event.node_id)}
              >
                <span>{event.event_type.replaceAll("_", " ")}</span>
                <small
                  >{shortPayload(event) ||
                    event.node_id ||
                    event.tree_id}</small
                >
              </button>
            {/each}
          </div>
        {/if}
      </section>
    </aside>
  </div>
</div>

<style>
  .metric-trees-page {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    height: calc(100vh - 64px);
    min-height: 0;
    overflow: hidden;
    padding: 20px 24px 28px;
    color: var(--color-text);
    background: var(--color-bg);
    --mt-warn: #d97706;
    --mt-good: #16a34a;
  }

  :global(.dark) .metric-trees-page {
    --mt-warn: #f59e0b;
    --mt-good: #22c55e;
  }

  .page-header,
  .workspace,
  .panel-head,
  .header-actions,
  .inspector-actions,
  .scoreline,
  .decision-lines div,
  .backlog button,
  .impact-list button,
  .review-walk button,
  .my-operating-band {
    display: flex;
  }

  .page-header {
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
    margin-bottom: 14px;
  }

  h1,
  h2,
  p {
    margin: 0;
  }

  h1 {
    font-family: "Outfit", sans-serif;
    font-size: 24px;
    font-weight: 900;
    color: var(--color-text);
  }

  h2 {
    font-family: "Outfit", sans-serif;
    font-size: 18px;
    font-weight: 800;
    color: var(--color-text);
  }

  p,
  .muted,
  .hint,
  small {
    color: var(--color-text-muted);
  }

  .header-actions,
  .inspector-actions {
    gap: 8px;
  }

  /* The Save/Delete row sits between sections (not inside one), so it needs
     the same 14px padding sections have to align with the panel content. */
  .inspector-actions {
    padding: 14px;
  }

  .mode-control,
  .time-control {
    align-items: center;
    display: flex;
    gap: 8px;
  }

  .mode-control {
    border: 0.5px solid var(--color-border-strong);
    border-radius: 6px;
    padding: 2px;
    background: var(--color-elevated);
  }

  .mode-control button {
    min-height: 28px;
    border: 0;
    padding: 4px 9px;
    background: transparent;
    color: var(--color-text-secondary);
  }

  .mode-control button.active {
    color: #0a0a0a;
    background: var(--color-acid);
  }

  .time-control select {
    min-width: 140px;
  }

  .live-warning {
    border: 1px solid color-mix(in srgb, var(--mt-warn) 45%, transparent);
    border-radius: 6px;
    padding: 8px 10px;
    color: var(--mt-warn);
    background: color-mix(in srgb, var(--mt-warn) 12%, transparent);
    font-size: 12px;
    font-weight: 650;
  }

  .my-operating-band {
    align-items: stretch;
    gap: 8px;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-acid-dim);
  }

  .my-operating-band div {
    min-width: 0;
    flex: 1;
    border: 0.5px solid var(--color-border);
    border-radius: 7px;
    padding: 9px 10px;
    background: var(--color-elevated);
  }

  .my-operating-band span {
    display: block;
    color: var(--color-acid-text);
    font-family: "Space Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
  }

  .my-operating-band strong {
    display: block;
    margin-top: 3px;
    overflow: hidden;
    color: var(--color-text);
    font-size: 14px;
    font-weight: 750;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .graph-shell,
  .review-shell {
    flex: 1;
    min-height: 0;
    border: 0.5px solid var(--color-border);
    border-radius: 8px;
    overflow: hidden;
    background: var(--color-elevated);
  }

  .review-shell {
    overflow: auto;
  }

  button,
  input,
  select,
  textarea {
    font: inherit;
  }

  button {
    min-height: 32px;
    border: 0.5px solid var(--color-border-strong);
    border-radius: 6px;
    padding: 6px 10px;
    color: var(--color-text);
    background: var(--color-elevated);
    cursor: pointer;
  }

  button:hover:not(:disabled) {
    border-color: var(--color-text-muted);
    background: var(--color-surface);
  }

  button:disabled {
    cursor: not-allowed;
    opacity: 0.58;
  }

  button.primary {
    border-color: var(--color-acid);
    color: #0a0a0a;
    background: var(--color-acid);
  }

  button.primary:hover:not(:disabled) {
    border-color: var(--color-acid);
    background: var(--color-acid);
    opacity: 0.88;
  }

  input,
  select,
  textarea {
    width: 100%;
    min-height: 32px;
    border: 0.5px solid var(--color-border-strong);
    border-radius: 6px;
    padding: 6px 8px;
    color: var(--color-text);
    background: var(--color-elevated);
  }

  textarea {
    resize: vertical;
  }

  label {
    display: grid;
    gap: 5px;
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary);
  }

  label.inline {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  label.inline input {
    width: auto;
    min-height: auto;
  }

  .workspace {
    align-items: stretch;
    flex: 1;
    gap: 14px;
    min-height: 0;
    overflow: hidden;
  }

  .rail,
  .inspector,
  .tree-panel {
    min-height: 0;
    border: 0.5px solid var(--color-border);
    border-top: 4px solid var(--color-acid);
    border-radius: 8px;
    background: var(--color-elevated);
  }

  .rail {
    width: 260px;
    flex: 0 0 260px;
    overflow: auto;
  }

  .inspector {
    width: 360px;
    flex: 0 0 360px;
    overflow: auto;
  }

  .tree-panel {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
    overflow: hidden;
  }

  section,
  .panel-head {
    padding: 14px;
    border-bottom: 0.5px solid var(--color-border);
  }

  section:last-child {
    border-bottom: 0;
  }

  .section-title {
    margin-bottom: 10px;
    color: var(--color-acid-text);
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1.5px;
  }

  .tree-list,
  .issues,
  .ownership-list,
  .backlog,
  .decision-lines,
  .impact-list,
  .review-walk,
  .review-experiments,
  .review-composer,
  .queued-decisions {
    display: grid;
    gap: 6px;
  }

  .tree-list button,
  .issues button,
  .ownership-list button,
  .backlog button,
  .impact-list button,
  .review-walk button,
  .review-experiments button {
    width: 100%;
    justify-content: space-between;
    text-align: left;
  }

  .tree-list button,
  .issues button,
  .ownership-list button {
    display: grid;
    gap: 3px;
  }

  .ownership-summary {
    border: 0.5px solid var(--color-border);
    border-radius: 6px;
    color: var(--color-text-secondary);
    font-size: 12px;
    font-weight: 700;
    margin-bottom: 8px;
    padding: 8px;
  }

  .ownership-list button.warn {
    border-color: color-mix(in srgb, var(--mt-warn) 50%, transparent);
    background: color-mix(in srgb, var(--mt-warn) 12%, transparent);
  }

  .owner-preview {
    display: grid;
    gap: 3px;
    margin: -2px 0 5px 8px;
  }

  .owner-preview span {
    color: var(--color-text-secondary);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .owner-preview em {
    color: var(--color-text-muted);
    display: block;
    font-style: normal;
  }

  .tree-list button.active,
  .ownership-list button.active,
  .impact-list button.active,
  .review-experiments button.active {
    border-color: var(--color-acid);
    background: var(--color-acid-dim);
  }

  .scoreline {
    align-items: center;
    gap: 6px;
    margin-bottom: 10px;
  }

  .empty,
  .alert {
    padding: 10px;
    border-radius: 6px;
    font-size: 13px;
  }

  .empty {
    color: var(--color-text-muted);
    background: var(--color-surface);
  }

  .empty.large {
    margin: 18px;
    padding: 18px;
  }

  .alert {
    margin-bottom: 10px;
  }

  .alert.error {
    border: 1px solid color-mix(in srgb, var(--bratrax-tomato) 45%, transparent);
    color: var(--bratrax-tomato);
    background: color-mix(in srgb, var(--bratrax-tomato) 12%, transparent);
  }

  .alert.success {
    border: 1px solid color-mix(in srgb, var(--mt-good) 45%, transparent);
    color: var(--mt-good);
    background: color-mix(in srgb, var(--mt-good) 12%, transparent);
  }

  .panel-head {
    align-items: center;
    justify-content: space-between;
  }

  .root-value {
    display: grid;
    gap: 3px;
    text-align: right;
  }

  .root-value strong {
    font-size: 20px;
  }

  .field-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-top: 10px;
  }

  .field-grid.three {
    grid-template-columns: 1fr 1fr 1fr;
  }

  .field-grid.add-child {
    grid-template-columns: 1fr 120px 64px;
  }

  .review-grid,
  .review-lever-card {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .review-summary {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .review-summary div,
  .review-lever-card div {
    border: 0.5px solid var(--color-border);
    border-radius: 6px;
    padding: 10px;
  }

  .review-summary span,
  .readiness {
    display: block;
    color: var(--color-text-muted);
    font-size: 12px;
    font-weight: 650;
  }

  .readiness {
    margin-top: 10px;
    border: 1px solid color-mix(in srgb, var(--mt-warn) 45%, transparent);
    border-radius: 6px;
    padding: 8px 10px;
    color: var(--mt-warn);
    background: color-mix(in srgb, var(--mt-warn) 12%, transparent);
  }

  .readiness.ready {
    border-color: color-mix(in srgb, var(--mt-good) 45%, transparent);
    color: var(--mt-good);
    background: color-mix(in srgb, var(--mt-good) 12%, transparent);
  }

  .decision-lines div,
  .backlog button {
    align-items: center;
    justify-content: space-between;
  }

  .decision-lines div {
    border-bottom: 0.5px solid var(--color-border);
    padding: 6px 0;
  }

  .decision-lines div:last-child {
    border-bottom: 0;
  }

  .decision-lines span {
    color: var(--color-text-muted);
  }

  @media (max-width: 1100px) {
    .workspace {
      display: grid;
    }

    .rail,
    .inspector {
      width: auto;
      flex-basis: auto;
    }
  }

  .inspector-fields {
    border: 0;
    margin: 0;
    min-inline-size: 0;
    padding: 0;
  }

  .alert.info {
    border: 0.5px solid var(--color-border);
    background: var(--color-surface);
    color: var(--color-text-secondary);
  }

  .readonly-section {
    border-top: 0.5px solid var(--color-border);
    margin-top: 12px;
    padding-top: 14px;
  }

  .alert.compact {
    margin: 0;
    padding: 8px 10px;
  }

  .evidence-label {
    border: 0.5px solid var(--color-border);
    background: var(--color-surface);
    color: var(--color-acid-text);
    font-family: "Space Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    margin-bottom: 10px;
    padding: 7px 8px;
    text-transform: uppercase;
  }

  .evidence-grid {
    display: grid;
    gap: 8px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .evidence-grid div,
  .guardrails div,
  .event-log button {
    border: 0.5px solid var(--color-border);
    background: var(--color-elevated);
  }

  .evidence-grid div,
  .guardrails div {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
    padding: 8px;
  }

  .evidence-grid span,
  .guardrails span {
    color: var(--color-text-muted);
    font-size: 11px;
  }

  .evidence-grid strong,
  .guardrails strong {
    color: var(--color-text);
    font-size: 13px;
  }

  .guardrails small {
    color: var(--color-text-muted);
    font-family: "Space Mono", monospace;
    font-size: 10px;
    text-transform: uppercase;
  }

  .guardrails {
    display: flex;
    flex-direction: column;
    gap: 7px;
    margin-top: 12px;
  }

  .section-title.small {
    margin-bottom: 0;
  }

  .decision-composer {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .decision-composer textarea {
    min-height: 74px;
    resize: vertical;
  }

  .event-log {
    display: flex;
    flex-direction: column;
    gap: 7px;
  }

  .event-log button {
    align-items: flex-start;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 8px;
    text-align: left;
  }

  .event-log button.active {
    border-color: var(--color-acid);
    background: var(--color-acid-dim);
  }

  .event-log span {
    color: var(--color-text);
    font-size: 12px;
    font-weight: 700;
  }

  .event-log small {
    color: var(--color-text-muted);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    width: 100%;
  }
</style>
