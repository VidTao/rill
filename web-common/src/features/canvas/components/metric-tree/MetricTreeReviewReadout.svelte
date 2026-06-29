<script lang="ts">
  type ReviewStep = {
    nodeId?: string;
    label?: string;
    delta?: number | null;
    depth?: number | null;
    rootImpact?: number | null;
  };

  type ReviewDecision = {
    actionType?: string;
    nodeId?: string;
    leverNodeId?: string;
    experimentNodeId?: string;
    note?: string;
    outcome?: string;
    nextAction?: string;
    owner?: string;
    target?: number | null;
    targetDate?: string;
    evidenceSnapshot?: Record<string, unknown> | null;
    trafficSourceTest?: Record<string, unknown> | null;
  };

  type ReviewSummary = {
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
  };

  type MetricTreeReviewReadoutSession = {
    session_id?: string;
    tree_id?: string;
    root_node_id?: string;
    selected_node_id?: string;
    chosen_lever_node_id?: string;
    experiment_node_id?: string;
    action_type?: string;
    status?: string;
    note?: string;
    outcome?: string;
    next_action?: string;
    created_at?: string;
    updated_at?: string;
    completed_at?: string;
    created_by?: string;
    timeRange?: { start?: string; end?: string; timeZone?: string } | Record<string, unknown> | null;
    reviewWalk?: ReviewStep[];
    topLevers?: ReviewStep[];
    evidenceSnapshot?: Record<string, unknown> | null;
    decisions?: ReviewDecision[];
    summary?: ReviewSummary | null;
  };

  export let session: MetricTreeReviewReadoutSession | null = null;
  export let nodeLabel: (nodeId: string) => string = (nodeId) => nodeId;
  export let workspaceHref: string | null = null;
  export let reviewHref: string | null = null;
  export let compact = false;

  function actionLabel(action: string | null | undefined): string {
    return action ? action.replaceAll("_", " ") : "review";
  }

  function dateLabel(value: string | null | undefined): string {
    if (!value) return "Unscheduled";
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
  }

  function fmtValue(value: number | null | undefined): string {
    if (value == null || !Number.isFinite(value)) return "-";
    return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
  }

  function timeLabel(s: MetricTreeReviewReadoutSession | null): string {
    if (!s) return "No time context";
    if (s.summary?.timeLabel) return s.summary.timeLabel;
    const range = s.timeRange as { start?: string; end?: string } | null | undefined;
    if (range?.start && range?.end) {
      return `${String(range.start).slice(0, 10)} to ${String(range.end).slice(0, 10)}`;
    }
    return "No time context";
  }

  function selectedNodeId(s: MetricTreeReviewReadoutSession | null): string | null {
    return s?.selected_node_id || s?.chosen_lever_node_id || s?.experiment_node_id || s?.root_node_id || null;
  }

  function evidenceObject(snapshot: Record<string, unknown> | null | undefined): Record<string, unknown> | null {
    if (!snapshot) return null;
    const nested = snapshot.evidence;
    return nested && typeof nested === "object" ? nested as Record<string, unknown> : snapshot;
  }

  function evidenceStatus(s: MetricTreeReviewReadoutSession | null): { label: string; tone: "ok" | "warn" | "muted" } {
    if (!s) return { label: "No review selected", tone: "muted" };
    const snapshots = [s.evidenceSnapshot, ...(s.decisions ?? []).map((d) => d.evidenceSnapshot)].map(evidenceObject).filter(Boolean) as Record<string, unknown>[];
    if (!snapshots.length) return { label: "Movement-only snapshot", tone: "muted" };
    if (snapshots.some((snapshot) => snapshot.status === "ok" || snapshot.resultLabel || snapshot.metrics)) {
      return { label: "Evidence attached", tone: "ok" };
    }
    if (snapshots.some((snapshot) => ["unsupported", "unbound", "error"].includes(String(snapshot.status)))) {
      return { label: "Evidence incomplete", tone: "warn" };
    }
    return { label: "Movement-only snapshot", tone: "muted" };
  }

  function evidenceWarnings(s: MetricTreeReviewReadoutSession | null): string[] {
    if (!s) return [];
    const warnings = new Set<string>();
    for (const warning of s.summary?.warnings ?? []) warnings.add(warning);
    const snapshots = [s.evidenceSnapshot, ...(s.decisions ?? []).map((d) => d.evidenceSnapshot)].map(evidenceObject).filter(Boolean) as Record<string, unknown>[];
    for (const snapshot of snapshots) {
      const next = snapshot.warnings;
      if (Array.isArray(next)) {
        for (const warning of next) warnings.add(String(warning));
      }
      if (snapshot.message) warnings.add(String(snapshot.message));
    }
    return [...warnings];
  }

  function decisionsFor(s: MetricTreeReviewReadoutSession | null): ReviewDecision[] {
    if (!s) return [];
    if (s.decisions?.length) return s.decisions;
    return [{
      actionType: s.action_type,
      nodeId: s.selected_node_id,
      leverNodeId: s.chosen_lever_node_id,
      experimentNodeId: s.experiment_node_id,
      note: s.note,
      outcome: s.outcome,
      nextAction: s.next_action,
    }];
  }

  function decisionTitle(decision: ReviewDecision): string {
    if (decision.actionType === "create_traffic_source_test" && decision.trafficSourceTest?.label) {
      return String(decision.trafficSourceTest.label);
    }
    if (decision.actionType === "update_lever_plan" && decision.leverNodeId) {
      return `Update ${nodeLabel(decision.leverNodeId)}`;
    }
    if (decision.actionType === "result_readout" && decision.experimentNodeId) {
      return `${nodeLabel(decision.experimentNodeId)}${decision.outcome ? `: ${decision.outcome}` : ""}`;
    }
    return decision.note || actionLabel(decision.actionType);
  }

  function evidenceMetricEntries(s: MetricTreeReviewReadoutSession | null): [string, unknown][] {
    if (!s) return [];
    const snapshots = [s.evidenceSnapshot, ...(s.decisions ?? []).map((d) => d.evidenceSnapshot)].map(evidenceObject).filter(Boolean) as Record<string, unknown>[];
    const metrics = snapshots.find((snapshot) => snapshot.metrics && typeof snapshot.metrics === "object")?.metrics;
    return metrics && typeof metrics === "object" ? Object.entries(metrics as Record<string, unknown>).slice(0, 6) : [];
  }

  $: status = evidenceStatus(session);
  $: warnings = evidenceWarnings(session);
  $: decisions = decisionsFor(session);
  $: selectedId = selectedNodeId(session);
  $: metrics = evidenceMetricEntries(session);
</script>

{#if session}
  <section class:compact class="review-readout">
    <header>
      <div>
        <div class="eyebrow">Review readout</div>
        <h3>{session.summary?.rootLabel ?? (session.root_node_id ? nodeLabel(session.root_node_id) : "Metric tree review")}</h3>
        <p>{timeLabel(session)} · {dateLabel(session.completed_at ?? session.created_at)}</p>
      </div>
      <span class:toneOk={status.tone === "ok"} class:toneWarn={status.tone === "warn"} class="status-pill">{status.label}</span>
    </header>

    <div class="summary-grid">
      <div>
        <span>Root movement</span>
        <strong>{fmtValue(session.summary?.rootDelta)}</strong>
      </div>
      <div>
        <span>Root value</span>
        <strong>{fmtValue(session.summary?.rootValue)}</strong>
      </div>
      <div>
        <span>Chosen lever</span>
        <strong>{session.summary?.chosenLeverLabel ?? (session.chosen_lever_node_id ? nodeLabel(session.chosen_lever_node_id) : "Unassigned")}</strong>
      </div>
      <div>
        <span>Stop reason</span>
        <strong>{session.summary?.stopReason ?? actionLabel(session.action_type)}</strong>
      </div>
    </div>

    {#if session.reviewWalk?.length}
      <div class="readout-block">
        <div class="block-label">Review walk</div>
        {#each session.reviewWalk as step}
          <div class="walk-row" style={`padding-left: ${(step.depth ?? 0) * 12}px`}>
            <span>{step.label ?? (step.nodeId ? nodeLabel(step.nodeId) : "Step")}</span>
            <strong>{fmtValue(step.delta)}</strong>
          </div>
        {/each}
      </div>
    {/if}

    {#if decisions.length}
      <div class="readout-block">
        <div class="block-label">Decisions and next actions</div>
        {#each decisions as decision}
          <div class="decision-row">
            <span>{actionLabel(decision.actionType)}</span>
            <strong>{decisionTitle(decision)}</strong>
            {#if decision.nextAction}<em>{decision.nextAction}</em>{/if}
            {#if decision.owner || decision.target != null || decision.targetDate}
              <em>{[decision.owner, decision.target != null ? `target ${fmtValue(decision.target)}` : null, decision.targetDate].filter(Boolean).join(" · ")}</em>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    {#if metrics.length || warnings.length}
      <div class="readout-block">
        <div class="block-label">Evidence snapshot</div>
        {#each metrics as [key, value]}
          <div class="metric-row"><span>{key.replaceAll("_", " ")}</span><strong>{String(value ?? "-")}</strong></div>
        {/each}
        {#each warnings as warning}
          <div class="warning">{warning}</div>
        {/each}
      </div>
    {/if}

    {#if session.topLevers?.length}
      <div class="readout-block">
        <div class="block-label">Other high-impact levers</div>
        {#each session.topLevers.slice(0, 4) as lever}
          <div class="metric-row"><span>{lever.label ?? (lever.nodeId ? nodeLabel(lever.nodeId) : "Lever")}</span><strong>{fmtValue(lever.rootImpact)}</strong></div>
        {/each}
      </div>
    {/if}

    <div class="readout-links">
      {#if reviewHref}
        <a class="workspace-link" href={reviewHref}>Open review page</a>
      {/if}
      {#if workspaceHref}
        <a class="workspace-link" href={workspaceHref}>Open selected node in workspace</a>
      {:else if selectedId}
        <div class="selected-node">Selected node: {nodeLabel(selectedId)}</div>
      {/if}
    </div>
  </section>
{/if}

<style lang="postcss">
  .review-readout {
    display: grid;
    gap: 12px;
    border: 1px solid #d9e0ea;
    border-radius: 8px;
    padding: 14px;
    background: #ffffff;
    color: #17202a;
  }
  .review-readout.compact {
    padding: 10px;
    gap: 10px;
  }
  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
  }
  .eyebrow,
  .block-label,
  .summary-grid span,
  .decision-row span,
  .metric-row span,
  .selected-node {
    color: #64748b;
    font-size: 11px;
  }
  .eyebrow,
  .block-label {
    font-weight: 800;
    letter-spacing: 0.5px;
    text-transform: uppercase;
  }
  h3 {
    margin: 1px 0;
    font-size: 15px;
    font-weight: 750;
    line-height: 1.25;
  }
  p {
    margin: 0;
    color: #64748b;
    font-size: 12px;
  }
  .status-pill {
    flex: none;
    border: 1px solid #d9e0ea;
    border-radius: 999px;
    padding: 3px 8px;
    color: #64748b;
    background: #f8fafc;
    font-size: 11px;
    font-weight: 700;
  }
  .status-pill.toneOk {
    border-color: #9ed7b5;
    color: #17663a;
    background: #eefaf2;
  }
  .status-pill.toneWarn {
    border-color: #f0cc78;
    color: #7a5308;
    background: #fff7df;
  }
  .summary-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .summary-grid div,
  .decision-row {
    display: grid;
    gap: 2px;
    border: 1px solid #edf0f4;
    border-radius: 6px;
    padding: 8px;
    min-width: 0;
  }
  strong {
    min-width: 0;
    overflow-wrap: anywhere;
    font-size: 12px;
    color: #17202a;
  }
  .readout-block {
    display: grid;
    gap: 6px;
  }
  .walk-row,
  .metric-row {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    border-bottom: 1px solid #edf0f4;
    padding: 5px 0;
    font-size: 12px;
  }
  .walk-row span,
  .metric-row span {
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .decision-row em {
    color: #64748b;
    font-size: 11px;
    font-style: normal;
  }
  .warning {
    border: 1px solid #f0cc78;
    border-radius: 6px;
    padding: 7px;
    color: #7a5308;
    background: #fff7df;
    font-size: 12px;
  }
  .readout-links {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }
  .workspace-link {
    color: #245bdb;
    font-size: 12px;
    font-weight: 750;
  }
  :global(.dark) .review-readout {
    border-color: #2a2a2a;
    background: #161616;
    color: #ece7dd;
  }
  :global(.dark) h3,
  :global(.dark) strong {
    color: #ece7dd;
  }
  :global(.dark) .summary-grid div,
  :global(.dark) .decision-row,
  :global(.dark) .walk-row,
  :global(.dark) .metric-row {
    border-color: #2a2a2a;
  }
</style>
