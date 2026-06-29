<script lang="ts">
  import MetricTreeReviewReadout from "./MetricTreeReviewReadout.svelte";
  import { nodeTypeTheme } from "./node-types";
  import { statusLabel, statusTheme } from "./status";
  import type {
    MetricTreeDetailData,
    MetricTreeDetailRow,
    MetricTreeNodeData,
  } from "./types";

  type AuthoredOperatingNode = {
    id: string;
    label: string;
    type: string;
    owner?: string | null;
    baseline?: number | null;
    target?: number | null;
    targetDate?: string | null;
    status?: string | null;
    measurementBinding?: Record<string, unknown>;
  };

  type ReviewDecision = {
    actionType: "log_decision" | "result_readout" | "create_traffic_source_test" | "update_lever_plan";
    nodeId?: string;
    leverNodeId?: string;
    experimentNodeId?: string;
    note?: string;
    outcome?: "won" | "lost" | "shipped";
    nextAction?: string;
    owner?: string | null;
    target?: number | null;
    targetDate?: string | null;
    evidenceSnapshot?: Record<string, unknown> | null;
    trafficSourceTest?: Record<string, unknown>;
  };

  type AuthoredEvent = {
    event_id: string;
    node_id: string;
    event_type: string;
    payload?: unknown;
    created_at: string;
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
    rootNodeId?: string;
    rootLabel?: string;
    rootDelta?: number | null;
    rootValue?: number | null;
    timeLabel?: string | null;
    stopReason: string;
    steps: Array<{ nodeId: string; label: string; delta: number | null; depth: number }>;
    topLevers: Array<{ nodeId: string; label: string; rootImpact: number | null }>;
  };

  type AuthoredReviewSession = {
    session_id: string;
    tree_id: string;
    root_node_id?: string;
    selected_node_id: string;
    chosen_lever_node_id?: string;
    experiment_node_id?: string;
    action_type: string;
    status?: string;
    note?: string;
    outcome?: string;
    next_action?: string;
    created_at: string;
    updated_at?: string;
    completed_at?: string;
    created_by: string;
    timeRange?: Record<string, unknown> | null;
    reviewWalk?: Array<{ nodeId?: string; label?: string; delta?: number | null; depth?: number | null }>;
    topLevers?: Array<{ nodeId?: string; label?: string; rootImpact?: number | null }>;
    evidenceSnapshot?: Record<string, unknown> | null;
    decisions?: ReviewDecision[];
    summary?: {
      rootNodeId?: string;
      rootLabel?: string;
      rootDelta?: number | null;
      rootValue?: number | null;
      chosenLeverId?: string;
      chosenLeverLabel?: string;
      chosenLeverOwner?: string;
      chosenLeverTarget?: number | null;
      timeLabel?: string;
      stopReason?: string;
      warnings?: string[];
    } | null;
  };

  type OperatingStatus = {
    liveCount: number;
    computedCount: number;
    fallbackCount: number;
    errorCount: number;
    warningCount: number;
    failedBindings: string[];
    timeLabel: string | null;
    filterLabel: string | null;
    summary: string;
  };

  type OwnershipPersona = {
    owner: string;
    leverCount: number;
    experimentCount: number;
    runningCount: number;
    backlogCount: number;
    unassigned: boolean;
    nodes: Array<{ id: string; label: string; type: string; status?: string | null }>;
  };

  export let node: MetricTreeNodeData | null;
  export let detailData: MetricTreeDetailData = {
    relationships: [],
    evidence: [],
    personas: [],
    logs: [],
    loading: false,
  };
  export let workspaceHref: string | null = null;
  export let authoredNode: AuthoredOperatingNode | null = null;
  export let authoredEvents: AuthoredEvent[] = [];
  export let authoredEvidence: AuthoredEvidence | null = null;
  export let authoredEvidenceLoading = false;
  export let authoredEventsLoading = false;
  export let authoredReviewSessions: AuthoredReviewSession[] = [];
  export let authoredReviewSessionsLoading = false;
  export let authoredOperatingError: string | null = null;
  export let authoredOperatingSaving = false;
  export let authoredReviewSummary: ReviewSummary | null = null;
  export let operatingStatus: OperatingStatus | null = null;
  export let authoredNodeLabels: Record<string, string> = {};
  export let authoredLeverExperiments: AuthoredOperatingNode[] = [];
  export let authoredOwnershipRoster: OwnershipPersona[] = [];
  export let onUpdateAuthoredNode: (nodeId: string, patch: Record<string, unknown>) => Promise<void> = async () => {};
  export let onCreateAuthoredEvent: (nodeId: string, payload: Record<string, unknown>) => Promise<void> = async () => {};
  export let onCreateReviewSession: (payload: Record<string, unknown>) => Promise<void> = async () => {};
  export let onCreateTrafficSourceTest: (nodeId: string, payload: Record<string, unknown>) => Promise<void> = async () => {};
  export let onClose: () => void = () => {};

  type TabId = "summary" | "operate" | "relationships" | "evidence" | "personas" | "logs";
  let activeTab: TabId = "summary";

  $: typeTheme = node ? nodeTypeTheme(node.nodeType) : nodeTypeTheme(null);
  $: healthTheme = node ? statusTheme(node.status) : statusTheme(null);
  $: healthLabel = node ? statusLabel(node.status) : null;
  $: if (!node) activeTab = "summary";

  let draftNodeId: string | null = null;
  let ownerDraft = "";
  let baselineDraft: number | null = null;
  let targetDraft: number | null = null;
  let targetDateDraft = "";
  let decisionNote = "";
  let reviewNote = "";
  let reviewNextAction = "";
  let reviewAction: ReviewDecision["actionType"] = "log_decision";
  let reviewOutcome: "won" | "lost" | "shipped" = "won";
  let reviewDecisionQueue: ReviewDecision[] = [];
  let readoutNote = "";
  let trafficLabel = "New traffic-source test";
  let trafficSource = "";
  let trafficCampaign = "";
  let trafficChannel = "";
  let trafficStartDate = new Date().toISOString().slice(0, 10);
  let selectedReadoutSessionId: string | null = null;

  $: if (authoredNode && authoredNode.id !== draftNodeId) {
    draftNodeId = authoredNode.id;
    ownerDraft = authoredNode.owner ?? "";
    baselineDraft = authoredNode.baseline ?? null;
    targetDraft = authoredNode.target ?? null;
    targetDateDraft = authoredNode.targetDate ?? "";
    decisionNote = "";
    reviewNote = "";
    reviewNextAction = "";
    reviewAction = authoredNode.type === "experiment" ? "result_readout" : "log_decision";
    reviewOutcome = "won";
    reviewDecisionQueue = [];
    readoutNote = "";
    trafficLabel = "New traffic-source test";
    trafficSource = "";
    trafficCampaign = "";
    trafficChannel = "";
    trafficStartDate = new Date().toISOString().slice(0, 10);
    selectedReadoutSessionId = null;
  }

  function fmtValue(
    v: number | string | null | undefined,
    unit: string | null | undefined,
  ): string {
    if (v == null || v === "") return "-";
    const n = typeof v === "number" ? v : Number(v);
    if (Number.isFinite(n)) {
      if (unit === "percent") return `${n.toLocaleString()}%`;
      if (unit === "usd" || unit === "currency") {
        return n.toLocaleString(undefined, {
          style: "currency",
          currency: "USD",
          maximumFractionDigits: 0,
        });
      }
      return `${n.toLocaleString()}${unit && unit !== "count" ? ` ${unit}` : ""}`;
    }
    return String(v);
  }

  function fmtCell(v: unknown): string {
    if (v == null || v === "") return "-";
    if (typeof v === "number") return v.toLocaleString();
    if (typeof v === "boolean") return v ? "true" : "false";
    if (typeof v === "string") return v;
    return JSON.stringify(v);
  }

  function rowTitle(row: MetricTreeDetailRow, fallback: string): string {
    return fmtCell(
      row.label ??
        row.relationship_label ??
        row.evidence_label ??
        row.segment ??
        row.title ??
        row.event_type ??
        fallback,
    );
  }

  function visibleEntries(row: MetricTreeDetailRow): [string, unknown][] {
    const hide = new Set(["tree_name", "node_id", "sort_order"]);
    return Object.entries(row)
      .filter(([k, v]) => !hide.has(k) && v != null && v !== "")
      .slice(0, 8);
  }

  function detailRows(tab: TabId): MetricTreeDetailRow[] {
    if (tab === "relationships") return detailData.relationships;
    if (tab === "evidence") return detailData.evidence;
    if (tab === "personas") return detailData.personas;
    if (tab === "logs") return detailData.logs;
    return [];
  }

  $: evidenceText = node
    ? [node.evidenceLabel, node.evidenceMetric].filter(Boolean).join(": ")
    : null;
  $: modelText = node
    ? [
        node.attributionModel,
        node.comparisonModel ? `vs ${node.comparisonModel}` : null,
      ]
        .filter(Boolean)
        .join(" ")
    : null;
  $: logText = node
    ? [node.logFilterKey, node.logFilterValue].filter(Boolean).join(" = ")
    : null;
  $: personaText = node
    ? [node.personaFilterKey, node.personaFilterValue]
        .filter(Boolean)
        .join(" = ")
    : null;
  $: sourceText = node
    ? [node.sourceStatus, node.sourceLabel, node.sourceContext]
        .filter(Boolean)
        .join(" · ")
    : null;
  $: computedText =
    node?.computedValue != null
      ? fmtValue(node.computedValue, node.unit)
      : null;
  $: measuredText =
    node?.measuredValue != null
      ? fmtValue(node.measuredValue, node.unit)
      : null;
  $: previousText =
    node?.previousValue != null
      ? fmtValue(node.previousValue, node.unit)
      : null;
  $: movementText =
    node?.delta != null
      ? `${node.delta > 0 ? "+" : ""}${fmtValue(node.delta, node.unit)}${node.deltaPercent != null ? ` (${(node.deltaPercent * 100).toFixed(1)}%)` : ""}`
      : null;
  $: driftText =
    node?.driftValue != null
      ? `${fmtValue(node.driftValue, node.unit)}${node.driftPercent != null ? ` (${(node.driftPercent * 100).toFixed(1)}%)` : ""}`
      : null;

  $: summarySections = node
    ? (
        [
          ["Source", sourceText],
          ["Formula", node.formulaText],
          ["Measured", measuredText],
          ["Previous", previousText],
          ["Movement", movementText],
          ["Computed", computedText],
          ["Drift", driftText],
          ["Source error", node.sourceError],
          ["Observation", node.observation],
          ["Decision", node.recommendedAction],
          ["Suggested test", node.suggestedTest],
          ["Success metric", node.successMetric],
          [
            "Legacy driver tag",
            node.driverType && node.driverType !== node.nodeType
              ? node.driverType
              : null,
          ],
          ["Attribution", modelText],
          ["Segment", node.segment],
          ["Evidence", evidenceText],
          ["Log filter", logText],
          ["Persona filter", personaText],
          ["Limitation", node.limitation],
        ] as const
      ).filter(([, v]) => !!v)
    : [];

  function numOrNull(value: unknown): number | null {
    if (value === "" || value == null) return null;
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }

  function evidenceSnapshot(): Record<string, unknown> | null {
    if (!authoredEvidence || authoredEvidence.status !== "ok") return null;
    return {
      status: authoredEvidence.status,
      resultLabel: authoredEvidence.resultLabel,
      queryBasis: authoredEvidence.queryBasis,
      metrics: authoredEvidence.metrics ?? {},
      dateRange: authoredEvidence.dateRange ?? {},
      warnings: authoredEvidence.warnings ?? [],
    };
  }

  function reviewLeverId(): string {
    if (authoredNode?.type === "lever") return authoredNode.id;
    return authoredReviewSummary?.topLevers?.[0]?.nodeId ?? "";
  }

  function labelFor(nodeId: string | null | undefined): string {
    if (!nodeId) return "";
    if (nodeId === authoredNode?.id) return authoredNode.label;
    return authoredNodeLabels[nodeId] ?? nodeId;
  }

  function evidenceCompletenessLabel(): string {
    if (authoredNode?.type === "experiment" && authoredNode.measurementBinding) {
      if (authoredEvidence?.status === "ok") return "Evidence attached";
      return "Evidence incomplete";
    }
    return "Movement-only snapshot";
  }

  function reviewWarnings(): string[] {
    const warnings: string[] = [];
    const evidenceLabel = evidenceCompletenessLabel();
    if (evidenceLabel !== "Evidence attached") warnings.push(evidenceLabel);
    if (!authoredReviewSummary?.steps?.length) warnings.push("Review walk is empty because live movement is unavailable.");
    if (!reviewLeverId()) warnings.push("No accountable lever selected.");
    if (authoredNode?.type === "experiment" && authoredNode.measurementBinding && authoredEvidenceLoading) warnings.push("Experiment evidence is still loading.");
    return warnings;
  }

  function reviewSummaryPayload(): Record<string, unknown> {
    return {
      rootNodeId: authoredReviewSummary?.rootNodeId,
      rootLabel: authoredReviewSummary?.rootLabel,
      rootDelta: authoredReviewSummary?.rootDelta,
      rootValue: authoredReviewSummary?.rootValue,
      chosenLeverId: reviewLeverId() || undefined,
      chosenLeverLabel: labelFor(reviewLeverId()) || undefined,
      timeLabel: authoredReviewSummary?.timeLabel,
      stopReason: authoredReviewSummary?.stopReason,
      warnings: reviewWarnings(),
    };
  }

  function reviewPayload(extra: Record<string, unknown> = {}): Record<string, unknown> {
    return {
      selectedNodeId: authoredNode?.id,
      chosenLeverNodeId: reviewLeverId() || undefined,
      experimentNodeId: authoredNode?.type === "experiment" ? authoredNode.id : undefined,
      timeRange: authoredEvidence?.dateRange ?? {},
      reviewWalk: authoredReviewSummary?.steps ?? [],
      topLevers: authoredReviewSummary?.topLevers ?? [],
      evidenceSnapshot: evidenceSnapshot() ?? undefined,
      summary: reviewSummaryPayload(),
      note: reviewNote.trim(),
      nextAction: reviewNextAction.trim(),
      ...extra,
    };
  }

  function sessionText(session: AuthoredReviewSession): string {
    const action = session.action_type.replaceAll("_", " ");
    const outcome = session.outcome ? ` · ${session.outcome}` : "";
    const note = session.note ? ` · ${session.note}` : "";
    return `${action}${outcome}${note}`;
  }

  function reviewPageHref(session: AuthoredReviewSession | null): string | null {
    if (!session?.tree_id || !session.session_id) return null;
    return `/metric-trees/reviews/${encodeURIComponent(session.tree_id)}/${encodeURIComponent(session.session_id)}`;
  }

  function ownerCoverageLabel(): string {
    const unassigned = authoredOwnershipRoster.find((owner) => owner.unassigned);
    if (!authoredOwnershipRoster.length) return "No owned levers or experiments";
    if (unassigned) {
      const count = unassigned.leverCount + unassigned.experimentCount;
      return `${count} unassigned action item${count === 1 ? "" : "s"}`;
    }
    return `${authoredOwnershipRoster.length} owner${authoredOwnershipRoster.length === 1 ? "" : "s"} accountable`;
  }

  function ownerNodeText(item: OwnershipPersona["nodes"][number]): string {
    return [item.type, item.status].filter(Boolean).join(" · ");
  }

  function eventText(event: AuthoredEvent): string {
    const payload = event.payload as Record<string, unknown> | undefined;
    if (payload?.note) return String(payload.note);
    if (payload?.nodeLabel) return String(payload.nodeLabel);
    return event.node_id;
  }

  async function saveOwnerTarget() {
    if (!authoredNode) return;
    await onUpdateAuthoredNode(authoredNode.id, {
      owner: ownerDraft.trim() || null,
      baseline: numOrNull(baselineDraft),
      target: numOrNull(targetDraft),
      targetDate: targetDateDraft || null,
    });
  }

  async function setExperimentStatus(status: string) {
    if (!authoredNode) return;
    await onUpdateAuthoredNode(authoredNode.id, { status });
  }

  function canQueueReviewDecision(): boolean {
    if (!authoredNode) return false;
    if ((reviewAction === "log_decision" || reviewAction === "result_readout") && !reviewNote.trim()) return false;
    if (reviewAction === "result_readout" && authoredNode.measurementBinding && (authoredEvidenceLoading || authoredEvidence?.status !== "ok")) return false;
    if (reviewAction === "create_traffic_source_test" && !trafficSource.trim() && !trafficCampaign.trim() && !trafficChannel.trim()) return false;
    if (reviewAction === "update_lever_plan" && !reviewLeverId()) return false;
    return true;
  }

  function queuedDecisionLabel(decision: ReviewDecision): string {
    if (decision.actionType === "log_decision") return decision.note || "Decision";
    if (decision.actionType === "result_readout") return `${labelFor(decision.experimentNodeId)}: ${decision.outcome}`;
    if (decision.actionType === "update_lever_plan") return `Update ${labelFor(decision.leverNodeId)}`;
    if (decision.actionType === "create_traffic_source_test") return String(decision.trafficSourceTest?.label ?? "Traffic-source test");
    return decision.actionType;
  }

  function queueReviewDecision() {
    if (!authoredNode || !canQueueReviewDecision()) return;
    let decision: ReviewDecision | null = null;
    const note = reviewNote.trim();
    const nextAction = reviewNextAction.trim() || undefined;
    if (reviewAction === "log_decision") {
      decision = { actionType: "log_decision", nodeId: authoredNode.id, note, nextAction };
    } else if (reviewAction === "result_readout") {
      decision = {
        actionType: "result_readout",
        experimentNodeId: authoredNode.id,
        note: note || `Marked ${reviewOutcome}`,
        outcome: reviewOutcome,
        nextAction,
        evidenceSnapshot: evidenceSnapshot(),
      };
    } else if (reviewAction === "update_lever_plan") {
      decision = {
        actionType: "update_lever_plan",
        leverNodeId: reviewLeverId(),
        note: note || "Updated lever plan",
        owner: ownerDraft.trim() || null,
        target: numOrNull(targetDraft),
        targetDate: targetDateDraft || null,
        nextAction,
      };
    } else if (reviewAction === "create_traffic_source_test") {
      decision = {
        actionType: "create_traffic_source_test",
        leverNodeId: reviewLeverId() || authoredNode.id,
        note: note || "Create traffic-source test",
        nextAction,
        trafficSourceTest: {
          label: trafficLabel.trim() || "New traffic-source test",
          source: trafficSource.trim() || undefined,
          campaign: trafficCampaign.trim() || undefined,
          channel_group: trafficChannel.trim() || undefined,
          startDate: trafficStartDate,
          attributionModel: "first_touch",
          primaryMeasure: "metric_nc_cpa",
        },
      };
    }
    if (!decision) return;
    reviewDecisionQueue = [...reviewDecisionQueue, decision];
    reviewNote = "";
    reviewNextAction = "";
  }

  function removeQueuedReviewDecision(index: number) {
    reviewDecisionQueue = reviewDecisionQueue.filter((_, i) => i !== index);
  }

  async function saveGuidedReview() {
    if (!authoredNode || !reviewDecisionQueue.length) return;
    await onCreateReviewSession(reviewPayload({
      actionType: "guided_review",
      status: "completed",
      decisions: reviewDecisionQueue,
      note: reviewDecisionQueue[0]?.note ?? "Guided review",
      nextAction: reviewDecisionQueue[0]?.nextAction,
    }));
    reviewDecisionQueue = [];
  }

  async function logDecision() {
    if (!authoredNode || !decisionNote.trim()) return;
    await onCreateAuthoredEvent(authoredNode.id, { eventType: "decision", note: decisionNote.trim() });
    decisionNote = "";
  }

  async function saveReadout(outcome: string) {
    if (!authoredNode) return;
    if (outcome) await onUpdateAuthoredNode(authoredNode.id, { status: outcome });
    await onCreateAuthoredEvent(authoredNode.id, {
      eventType: "result_readout",
      outcome,
      note: readoutNote.trim() || `Marked ${outcome}`,
      evidenceSnapshot: evidenceSnapshot(),
    });
    readoutNote = "";
  }

  async function createTrafficTest() {
    if (!authoredNode) return;
    await onCreateTrafficSourceTest(authoredNode.id, {
      label: trafficLabel.trim() || "New traffic-source test",
      source: trafficSource.trim() || undefined,
      campaign: trafficCampaign.trim() || undefined,
      channel_group: trafficChannel.trim() || undefined,
      startDate: trafficStartDate,
      attributionModel: "first_touch",
      primaryMeasure: "metric_nc_cpa",
    });
  }

  $: tabs = [
    { id: "summary" as const, label: "Summary", count: 1 },
    { id: "operate" as const, label: "Operate", count: authoredNode ? 1 : 0 },
    {
      id: "relationships" as const,
      label: "Relationships",
      count: detailData.relationships.length,
    },
    {
      id: "evidence" as const,
      label: "Evidence",
      count: detailData.evidence.length,
    },
    {
      id: "personas" as const,
      label: "Personas",
      count: detailData.personas.length,
    },
    { id: "logs" as const, label: "Logs", count: detailData.logs.length },
  ].filter((t) => t.id === "summary" || t.count > 0);

  $: visibleReviewSessions = authoredNode
    ? authoredReviewSessions
        .filter(
          (session) =>
            session.selected_node_id === authoredNode.id ||
            session.chosen_lever_node_id === authoredNode.id ||
            session.experiment_node_id === authoredNode.id,
        )
        .slice(0, 5)
    : [];
  $: selectedReadoutSession =
    visibleReviewSessions.find(
      (session) => session.session_id === selectedReadoutSessionId,
    ) ?? visibleReviewSessions[0] ?? null;
  $: selectedNodeEvents = authoredNode
    ? authoredEvents.filter((event) => event.node_id === authoredNode.id).slice(0, 6)
    : [];
  $: evidenceMetricEntries = Object.entries(authoredEvidence?.metrics ?? {}).slice(0, 6);
  $: reviewStopStep = authoredReviewSummary?.steps?.length
    ? authoredReviewSummary.steps[authoredReviewSummary.steps.length - 1]
    : null;
  $: primaryRootLever = authoredReviewSummary?.topLevers?.[0] ?? null;
  $: reviewWarningsList = reviewWarnings();
  $: evidenceStatusLabel = evidenceCompletenessLabel();
  $: if (
    selectedReadoutSessionId &&
    !visibleReviewSessions.some(
      (session) => session.session_id === selectedReadoutSessionId,
    )
  ) {
    selectedReadoutSessionId = null;
  }
  $: if (!tabs.some((t) => t.id === activeTab)) activeTab = "summary";
</script>

{#if node}
  <button class="scrim" aria-label="Close detail" on:click={onClose} />

  <aside
    class="panel"
    style:--type-color={typeTheme.color}
    style:--status-color={healthTheme.light}
    style:--status-color-dark={healthTheme.dark}
  >
    <header>
      <div>
        <div class="type-pill">{typeTheme.label}</div>
        <div class="title" title={node.label}>{node.label}</div>
        {#if workspaceHref}
          <a class="workspace-link" href={workspaceHref}>Edit tree structure</a>
        {/if}
      </div>
      <button class="close" aria-label="Close" on:click={onClose}>x</button>
    </header>

    <div class="status-row">
      {#if healthLabel}
        <span class="dot" />
        <span class="status">{healthLabel}</span>
      {/if}
      {#if node.confidence}
        <span class="conf">· {node.confidence} confidence</span>
      {/if}
    </div>

    <div class="value">
      {fmtValue(node.value, node.unit)}
      {#if node.delta != null}
        <span class="delta">
          {node.delta > 0 ? "+" : node.delta < 0 ? "-" : "="}
          {Math.abs(node.delta).toLocaleString()}
        </span>
      {/if}
    </div>

    <div class="tabs" role="tablist">
      {#each tabs as tab (tab.id)}
        <button
          class:active={activeTab === tab.id}
          role="tab"
          aria-selected={activeTab === tab.id}
          on:click={() => (activeTab = tab.id)}
        >
          {tab.label}{tab.id !== "summary" ? ` ${tab.count}` : ""}
        </button>
      {/each}
    </div>

    {#if detailData.loading && activeTab !== "summary"}
      <div class="muted">Loading detail...</div>
    {/if}

    {#if activeTab === "summary"}
      {#each summarySections as [label, value] (label)}
        <section>
          <div class="section-label">{label}</div>
          <div class="section-body">{value}</div>
        </section>
      {/each}
    {:else if activeTab === "operate" && authoredNode}
      <div class="cockpit">
        {#if authoredOperatingError}
          <section class="op-alert">{authoredOperatingError}</section>
        {/if}

        <section class:error={operatingStatus && operatingStatus.errorCount > 0} class="cockpit-hero">
          <div>
            <div class="section-label">Operating status</div>
            <strong>{operatingStatus?.errorCount ? "Live data degraded" : operatingStatus ? "Live data connected" : "Operating context"}</strong>
            <p>{operatingStatus?.summary ?? authoredReviewSummary?.stopReason ?? "Select a node to review the operating context."}</p>
          </div>
          <div class="cockpit-kpis">
            <div>
              <span>Root movement</span>
              <strong>{fmtValue(authoredReviewSummary?.rootDelta ?? node.delta ?? 0, node.unit)}</strong>
            </div>
            <div>
              <span>Selected value</span>
              <strong>{fmtValue(node.value, node.unit)}</strong>
            </div>
            <div>
              <span>Evidence</span>
              <strong>{evidenceStatusLabel}</strong>
            </div>
            <div>
              <span>Owners</span>
              <strong>{ownerCoverageLabel()}</strong>
            </div>
          </div>
          {#if operatingStatus?.timeLabel || operatingStatus?.filterLabel}
            <em>{[operatingStatus.timeLabel, operatingStatus.filterLabel].filter(Boolean).join(" · ")}</em>
          {/if}
          {#if operatingStatus}
            <div class="status-metrics compact-status">
              <span>Live {operatingStatus.liveCount}</span>
              <span>Computed {operatingStatus.computedCount}</span>
              <span>Fallback {operatingStatus.fallbackCount}</span>
              <span>Errors {operatingStatus.errorCount}</span>
            </div>
          {/if}
          {#if operatingStatus?.failedBindings.length}
            <div class="failed-bindings">Failed bindings: {operatingStatus.failedBindings.slice(0, 6).join(", ")}</div>
          {/if}
        </section>

        <div class="cockpit-grid">
          <section class="cockpit-card review-card">
            <div class="section-label">Weekly review path</div>
            <div class="review-headline">
              <strong>{reviewStopStep?.label ?? authoredNode.label}</strong>
              <span>{authoredReviewSummary?.stopReason ?? "No review path yet"}</span>
            </div>
            {#if authoredReviewSummary?.steps?.length}
              <div class="review-walk-compact">
                {#each authoredReviewSummary.steps as step}
                  <div style={`padding-left: ${step.depth * 12}px`}>
                    <span>{step.label}</span>
                    <strong>{fmtValue(step.delta ?? 0, node.unit)}</strong>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="op-muted">Live movement has not produced a review walk for this node.</div>
            {/if}
          </section>

          <section class="cockpit-card lever-card">
            <div class="section-label">Accountable lever</div>
            <div class="lever-focus">
              <strong>{primaryRootLever?.label ?? reviewStopStep?.label ?? authoredNode.label}</strong>
              <span>{primaryRootLever?.rootImpact == null ? "Root impact unsized" : `Root impact ${fmtValue(primaryRootLever.rootImpact, node.unit)}`}</span>
            </div>
            {#if authoredReviewSummary?.topLevers?.length}
              <div class="impact-list-compact">
                {#each authoredReviewSummary.topLevers.slice(0, 4) as lever}
                  <div>
                    <span>{lever.label}</span>
                    <strong>{lever.rootImpact == null ? "unsized" : fmtValue(lever.rootImpact, node.unit)}</strong>
                  </div>
                {/each}
              </div>
            {/if}
            {#if authoredNode.type === "lever"}
              <div class="inline-plan">
                <label>Owner<input bind:value={ownerDraft} /></label>
                <label>Target<input type="number" step="any" bind:value={targetDraft} /></label>
                <button type="button" on:click={saveOwnerTarget} disabled={authoredOperatingSaving}>Save plan</button>
              </div>
            {/if}
          </section>
        </div>

        <section class="cockpit-card action-card">
          <div class="section-label">Decision queue</div>
          <div class:warn={evidenceStatusLabel !== "Evidence attached"} class="evidence-status">{evidenceStatusLabel}</div>
          {#each reviewWarningsList as warning}
            <div class="op-warning">{warning}</div>
          {/each}
          <div class="review-composer-compact">
            <label>
              Action
              <select bind:value={reviewAction}>
                <option value="log_decision">Decision</option>
                <option value="update_lever_plan">Update lever plan</option>
                {#if authoredNode.type === "experiment"}
                  <option value="result_readout">Result readout</option>
                {/if}
                {#if authoredNode.type === "lever" || authoredNode.type === "surface"}
                  <option value="create_traffic_source_test">Create traffic-source test</option>
                {/if}
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
            {:else if reviewAction === "create_traffic_source_test"}
              <div class="field-grid-compact">
                <label>Label<input bind:value={trafficLabel} /></label>
                <label>Channel<input bind:value={trafficChannel} placeholder="Meta" /></label>
                <label>Source<input bind:value={trafficSource} placeholder="Conversion" /></label>
                <label>Campaign<input bind:value={trafficCampaign} placeholder="Campaign name" /></label>
              </div>
            {:else if reviewAction === "update_lever_plan"}
              <div class="field-grid-compact three">
                <label>Owner<input bind:value={ownerDraft} /></label>
                <label>Target<input type="number" step="any" bind:value={targetDraft} /></label>
                <label>Target date<input type="date" bind:value={targetDateDraft} /></label>
              </div>
            {/if}
            <label class="wide-field">Review note<textarea rows="3" bind:value={reviewNote} placeholder="What did we decide in this review?" /></label>
            <label class="wide-field">Next action<input bind:value={reviewNextAction} placeholder="Owner and next move" /></label>
            <div class="action-buttons">
              <button type="button" on:click={queueReviewDecision} disabled={authoredOperatingSaving || !canQueueReviewDecision()}>Add to queue</button>
              <button class="op-primary" type="button" on:click={saveGuidedReview} disabled={authoredOperatingSaving || !reviewDecisionQueue.length}>Save review session</button>
            </div>
          </div>
          {#if reviewDecisionQueue.length}
            <div class="queued-decisions compact-queue">
              {#each reviewDecisionQueue as decision, index}
                <div>
                  <span>{decision.actionType.replaceAll("_", " ")}</span>
                  <strong>{queuedDecisionLabel(decision)}</strong>
                  <button type="button" on:click={() => removeQueuedReviewDecision(index)}>Remove</button>
                </div>
              {/each}
            </div>
          {:else}
            <div class="op-muted">Queue one or more decisions, then save the review session.</div>
          {/if}
        </section>

        <div class="cockpit-grid side-grid">
          <section class="cockpit-card">
            <div class="section-label">Experiments</div>
            {#if authoredLeverExperiments.length}
              <div class="experiment-list-compact">
                {#each authoredLeverExperiments.slice(0, 6) as experiment}
                  <div>
                    <span>{experiment.label}</span>
                    <strong>{experiment.status ?? "backlog"}</strong>
                  </div>
                {/each}
              </div>
            {:else if authoredNode.type === "experiment"}
              <div class="experiment-list-compact">
                <div><span>{authoredNode.label}</span><strong>{authoredNode.status ?? "backlog"}</strong></div>
              </div>
              <div class="op-actions tight">
                <button type="button" on:click={() => setExperimentStatus("running")} disabled={authoredOperatingSaving}>Launch</button>
                <button type="button" on:click={() => saveReadout("won")} disabled={authoredOperatingSaving}>Won</button>
                <button type="button" on:click={() => saveReadout("lost")} disabled={authoredOperatingSaving}>Lost</button>
                <button type="button" on:click={() => saveReadout("shipped")} disabled={authoredOperatingSaving}>Shipped</button>
              </div>
              <textarea rows="2" bind:value={readoutNote} placeholder="Readout note" />
            {:else}
              <div class="op-muted">No experiments under this lever yet.</div>
            {/if}
          </section>

          <section class="cockpit-card ownership-card">
            <div class="section-label">Ownership map</div>
            {#if authoredOwnershipRoster.length}
              <div class="owner-list-compact">
                {#each authoredOwnershipRoster.slice(0, 5) as owner}
                  <div class:warn={owner.unassigned} class="owner-group">
                    <div class="owner-head">
                      <strong>{owner.owner}</strong>
                      <span>{owner.leverCount} levers · {owner.experimentCount} experiments</span>
                    </div>
                    <div class="owner-subline">
                      {owner.runningCount} running · {owner.backlogCount} queued
                    </div>
                    {#each owner.nodes.slice(0, 3) as item}
                      <div class="owner-node">
                        <span>{item.label}</span>
                        <em>{ownerNodeText(item)}</em>
                      </div>
                    {/each}
                  </div>
                {/each}
              </div>
            {:else}
              <div class="op-muted">No owners assigned to levers or experiments yet.</div>
            {/if}
          </section>

          <section class="cockpit-card">
            <div class="section-label">Evidence snapshot</div>
            {#if authoredEvidenceLoading}
              <div class="op-muted">Loading evidence...</div>
            {:else if !authoredEvidence}
              <div class="op-muted">No evidence loaded.</div>
            {:else if authoredEvidence.status !== "ok"}
              <div class="op-muted">{authoredEvidence.message ?? authoredEvidence.status}</div>
            {:else}
              <div class="op-muted">{authoredEvidence.resultLabel} · {authoredEvidence.queryBasis}</div>
              {#each evidenceMetricEntries as [key, value]}
                <div class="kv compact"><span>{key.replaceAll("_", " ")}</span><strong>{fmtValue(value, key.includes("revenue") || key.includes("spend") || key.includes("cpa") ? "currency" : "count")}</strong></div>
              {/each}
              {#each authoredEvidence.warnings ?? [] as warning}
                <div class="op-warning">{warning}</div>
              {/each}
            {/if}
          </section>
        </div>

        <div class="cockpit-grid history-grid">
          <section class="cockpit-card">
            <div class="section-label">Recent reviews</div>
            {#if authoredReviewSessionsLoading}
              <div class="op-muted">Loading reviews...</div>
            {:else if !visibleReviewSessions.length}
              <div class="op-muted">No saved reviews yet.</div>
            {:else}
              <div class="review-pickers compact-pickers">
                {#each visibleReviewSessions as session}
                  <button
                    type="button"
                    class:active={selectedReadoutSession?.session_id === session.session_id}
                    on:click={() => (selectedReadoutSessionId = session.session_id ?? null)}
                  >
                    <strong>{new Date(session.created_at ?? "").toLocaleDateString()}</strong>
                    <span>{sessionText(session)}</span>
                  </button>
                {/each}
              </div>
              <MetricTreeReviewReadout
                session={selectedReadoutSession}
                nodeLabel={labelFor}
                reviewHref={reviewPageHref(selectedReadoutSession)}
                compact
              />
            {/if}
          </section>

          <section class="cockpit-card">
            <div class="section-label">Decision log</div>
            {#if authoredEventsLoading}
              <div class="op-muted">Loading events...</div>
            {:else if !selectedNodeEvents.length}
              <div class="op-muted">No decisions logged for this node.</div>
            {:else}
              {#each selectedNodeEvents as event}
                <div class="event-row compact-event">
                  <strong>{event.event_type.replaceAll("_", " ")}</strong>
                  <span>{eventText(event)}</span>
                </div>
              {/each}
            {/if}
          </section>
        </div>
      </div>
    {:else}
      {#each detailRows(activeTab) as row, i (`${activeTab}-${i}`)}
        <section class="row-card">
          <div class="row-title">{rowTitle(row, `${activeTab} ${i + 1}`)}</div>
          {#each visibleEntries(row) as [key, value] (key)}
            <div class="kv">
              <span>{key.replaceAll("_", " ")}</span>
              <strong>{fmtCell(value)}</strong>
            </div>
          {/each}
        </section>
      {/each}
    {/if}
  </aside>
{/if}

<style lang="postcss">
  .scrim {
    @apply absolute inset-0 z-10 cursor-default border-0 bg-black/10;
  }
  .panel {
    @apply absolute inset-y-0 right-0 z-20 flex flex-col gap-3 overflow-y-auto p-4;
    width: min(620px, 96%);
    background: var(--color-surface-card, #ffffff);
    border-left: 4px solid var(--type-color);
    box-shadow: -8px 0 24px rgba(15, 23, 42, 0.12);
  }
  header {
    @apply flex items-start justify-between gap-2;
  }
  .type-pill {
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--type-color);
    margin-bottom: 3px;
  }
  .title {
    font-size: 15px;
    font-weight: 700;
    line-height: 1.25;
    color: var(--color-text, #1a1a18);
  }
  .workspace-link {
    display: inline-block;
    margin-top: 5px;
    font-size: 11px;
    font-weight: 700;
    color: var(--type-color);
  }
  .close {
    @apply shrink-0 rounded px-1 text-sm;
    color: var(--color-text-muted, #6b7280);
  }
  .status-row {
    @apply flex items-center gap-1.5;
    font-size: 11px;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 9999px;
    background: var(--status-color);
  }
  .status {
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--status-color);
  }
  .conf,
  .delta,
  .muted {
    color: var(--color-text-muted, #6b7280);
  }
  .value {
    font-size: 22px;
    font-weight: 700;
    color: var(--color-text, #1a1a18);
  }
  .delta {
    font-size: 12px;
    font-weight: 500;
    margin-left: 6px;
  }
  .tabs {
    @apply flex flex-wrap gap-1 border-b border-gray-200 pb-2;
  }
  .tabs button {
    @apply rounded px-2 py-1 text-xs font-semibold;
    color: var(--color-text-muted, #6b7280);
  }
  .tabs button.active {
    background: color-mix(in srgb, var(--type-color) 14%, transparent);
    color: var(--type-color);
  }
  .section-label {
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.6px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    margin-bottom: 2px;
  }
  .section-body,
  .row-title,
  .kv strong {
    font-size: 13px;
    line-height: 1.4;
    color: var(--color-text, #1a1a18);
  }
  .row-card {
    @apply rounded border border-gray-200 p-2;
  }
  .row-title {
    font-weight: 700;
    margin-bottom: 4px;
  }
  .kv {
    @apply grid gap-2 py-0.5;
    grid-template-columns: minmax(80px, 0.42fr) minmax(0, 1fr);
    font-size: 11px;
  }
  .kv span {
    color: var(--color-text-muted, #6b7280);
    text-transform: capitalize;
  }
  .kv strong {
    font-weight: 500;
    overflow-wrap: anywhere;
  }

  .cockpit {
    display: grid;
    gap: 10px;
  }
  .cockpit-hero,
  .cockpit-card {
    border: 1px solid #dfe6ee;
    border-radius: 8px;
    background: #ffffff;
    padding: 10px;
  }
  .cockpit-hero {
    display: grid;
    gap: 9px;
    border-color: #badfc9;
    background: #f5fbf7;
    color: #173b25;
  }
  .cockpit-hero.error {
    border-color: #efc466;
    background: #fff8df;
    color: #684600;
  }
  .cockpit-hero strong,
  .review-headline strong,
  .lever-focus strong {
    display: block;
    font-size: 14px;
    line-height: 1.25;
  }
  .cockpit-hero p {
    margin: 2px 0 0;
    font-size: 12px;
  }
  .cockpit-hero em {
    color: currentColor;
    font-size: 11px;
    font-style: normal;
    opacity: 0.72;
  }
  .cockpit-kpis {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }
  .cockpit-kpis div {
    border: 1px solid rgba(23, 59, 37, 0.18);
    border-radius: 7px;
    background: rgba(255, 255, 255, 0.68);
    padding: 7px;
  }
  .cockpit-kpis span,
  .review-headline span,
  .lever-focus span {
    display: block;
    color: #64748b;
    font-size: 10px;
    font-weight: 750;
    letter-spacing: 0.4px;
    text-transform: uppercase;
  }
  .cockpit-kpis strong {
    display: block;
    margin-top: 2px;
    color: inherit;
    font-size: 13px;
  }
  .cockpit-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.15fr) minmax(0, 0.85fr);
    gap: 10px;
  }
  .review-card {
    min-width: 0;
  }
  .review-headline,
  .lever-focus {
    margin-bottom: 8px;
  }
  .review-walk-compact,
  .impact-list-compact,
  .experiment-list-compact,
  .owner-list-compact {
    display: grid;
    gap: 4px;
  }
  .review-walk-compact div,
  .impact-list-compact div,
  .experiment-list-compact div {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    border-bottom: 1px solid #edf0f4;
    padding: 5px 0;
    font-size: 12px;
  }
  .review-walk-compact span,
  .impact-list-compact span,
  .experiment-list-compact span {
    min-width: 0;
    overflow: hidden;
    color: #17202a;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .review-walk-compact strong,
  .impact-list-compact strong,
  .experiment-list-compact strong {
    color: #475569;
    font-size: 11px;
    white-space: nowrap;
  }
  .owner-group {
    border: 1px solid #edf0f4;
    border-radius: 7px;
    display: grid;
    gap: 4px;
    padding: 7px;
  }
  .owner-group.warn {
    border-color: #f0cc78;
    background: #fff9e8;
  }
  .owner-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
  }
  .owner-head strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .owner-head span,
  .owner-subline,
  .owner-node em {
    color: #64748b;
    font-size: 11px;
    font-style: normal;
    white-space: nowrap;
  }
  .owner-node {
    display: flex;
    justify-content: space-between;
    gap: 8px;
    font-size: 12px;
  }
  .owner-node span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .inline-plan {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
    align-items: end;
    gap: 6px;
    margin-top: 10px;
  }
  .inline-plan label,
  .review-composer-compact label,
  .field-grid-compact label {
    margin: 0;
  }
  .action-card {
    display: grid;
    gap: 8px;
  }
  .review-composer-compact {
    display: grid;
    grid-template-columns: minmax(120px, 0.35fr) minmax(0, 1fr);
    gap: 8px;
  }
  .wide-field,
  .field-grid-compact,
  .action-buttons,
  .compact-queue {
    grid-column: 1 / -1;
  }
  .field-grid-compact {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .field-grid-compact.three {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .action-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
  }
  .side-grid,
  .history-grid {
    align-items: start;
  }
  .tight {
    margin-top: 8px;
  }
  .compact-pickers {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .compact-event {
    padding: 6px 0;
  }
  @media (max-width: 720px) {
    .panel {
      width: min(100%, 100%);
    }
    .cockpit-grid,
    .cockpit-kpis,
    .review-composer-compact,
    .field-grid-compact,
    .field-grid-compact.three,
    .inline-plan,
    .compact-pickers {
      grid-template-columns: 1fr;
    }
  }

  :global(.dark) .panel {
    background: #161616;
    border-left-color: var(--type-color);
  }
  :global(.dark) .title,
  :global(.dark) .value,
  :global(.dark) .section-body,
  :global(.dark) .row-title,
  :global(.dark) .kv strong {
    color: #ece7dd;
  }
  :global(.dark) .tabs,
  :global(.dark) .row-card {
    border-color: #2a2a2a;
  }
  :global(.dark) .status {
    color: var(--status-color-dark);
  }
  :global(.dark) .dot {
    background: var(--status-color-dark);
  }
  :global(.dark) .tabs button.active,
  :global(.dark) .type-pill,
  :global(.dark) .workspace-link {
    color: var(--type-color);
  }
  :global(.dark) .conf,
  :global(.dark) .delta,
  :global(.dark) .section-label,
  :global(.dark) .tabs button,
  :global(.dark) .kv span,
  :global(.dark) .muted,
  :global(.dark) .close {
    color: #a39d90;
  }
  .op-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .op-primary {
    margin-top: 8px;
    border-color: #245bdb;
    color: #fff;
    background: #245bdb;
  }
  .op-muted {
    color: #64748b;
    font-size: 12px;
    margin-bottom: 8px;
  }
  .status-metrics {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px;
    margin: 8px 0;
    font-size: 11px;
    font-weight: 700;
  }
  .status-metrics span {
    border: 1px solid currentColor;
    border-radius: 999px;
    opacity: 0.8;
    padding: 3px 7px;
  }
  .failed-bindings {
    overflow-wrap: anywhere;
    opacity: 0.85;
  }
  .op-alert,
  .op-warning {
    border: 1px solid #f3c969;
    border-radius: 6px;
    padding: 8px;
    color: #6f4b00;
    background: #fff8df;
    font-size: 12px;
  }
  .kv.compact {
    padding: 4px 0;
  }
  label {
    display: grid;
    gap: 4px;
    margin-bottom: 8px;
    font-size: 12px;
    font-weight: 650;
    color: #475569;
  }
  input,
  select,
  textarea {
    min-height: 30px;
    border: 1px solid #cbd5df;
    border-radius: 6px;
    padding: 6px 8px;
    font: inherit;
  }
  textarea {
    resize: vertical;
  }
  .evidence-status {
    border: 1px solid #9ed7b5;
    border-radius: 999px;
    color: #17663a;
    background: #eefaf2;
    display: inline-flex;
    font-size: 11px;
    font-weight: 750;
    margin-bottom: 8px;
    padding: 3px 8px;
  }
  .evidence-status.warn {
    border-color: #f0cc78;
    color: #7a5308;
    background: #fff7df;
  }
  .queued-decisions {
    display: grid;
    gap: 6px;
    margin-top: 8px;
  }
  .queued-decisions > div {
    border: 1px solid #edf0f4;
    border-radius: 6px;
    display: grid;
    gap: 2px;
    padding: 7px;
  }
  .queued-decisions span {
    color: #64748b;
    font-size: 11px;
    text-transform: capitalize;
  }
  .queued-decisions button {
    justify-self: start;
    margin-top: 2px;
  }
  .review-pickers {
    display: grid;
    gap: 6px;
    margin-bottom: 8px;
  }
  .review-pickers button {
    display: grid;
    gap: 2px;
    border: 1px solid #edf0f4;
    border-radius: 6px;
    padding: 7px;
    text-align: left;
    background: #fff;
    font-size: 12px;
  }
  .review-pickers button.active {
    border-color: #245bdb;
    background: #f4f7ff;
  }
  .review-pickers span {
    color: #64748b;
  }
  :global(.dark) .review-pickers button {
    border-color: #2a2a2a;
    background: #161616;
  }
  :global(.dark) .cockpit-card {
    border-color: #2a2a2a;
    background: #181818;
  }
  :global(.dark) .cockpit-hero {
    border-color: #235b38;
    background: rgba(15, 44, 27, 0.94);
    color: #bdf0ca;
  }
  :global(.dark) .cockpit-hero.error {
    border-color: #7a5a1d;
    background: rgba(55, 39, 12, 0.94);
    color: #f8dda2;
  }
  :global(.dark) .cockpit-kpis div {
    border-color: rgba(189, 240, 202, 0.18);
    background: rgba(255, 255, 255, 0.04);
  }
  :global(.dark) .review-walk-compact div,
  :global(.dark) .impact-list-compact div,
  :global(.dark) .experiment-list-compact div,
  :global(.dark) .owner-group,
  :global(.dark) .event-row {
    border-color: #2a2a2a;
  }
  :global(.dark) .owner-group.warn {
    border-color: #7a5a1d;
    background: rgba(55, 39, 12, 0.94);
  }
  :global(.dark) .review-walk-compact span,
  :global(.dark) .impact-list-compact span,
  :global(.dark) .experiment-list-compact span,
  :global(.dark) .owner-node span,
  :global(.dark) .owner-head strong {
    color: #ece7dd;
  }
  :global(.dark) input,
  :global(.dark) select,
  :global(.dark) textarea {
    border-color: #343434;
    background: #111;
    color: #ece7dd;
  }
  .event-row {
    display: grid;
    gap: 2px;
    border-bottom: 1px solid #edf0f4;
    padding: 7px 0;
    font-size: 12px;
  }
  .event-row span {
    color: #64748b;
  }
</style>
