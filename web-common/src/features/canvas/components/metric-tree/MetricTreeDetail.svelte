<script lang="ts">
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
  let reviewAction = "log_decision";
  let readoutNote = "";
  let trafficLabel = "New traffic-source test";
  let trafficSource = "";
  let trafficCampaign = "";
  let trafficChannel = "";
  let trafficStartDate = new Date().toISOString().slice(0, 10);

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
    readoutNote = "";
    trafficLabel = "New traffic-source test";
    trafficSource = "";
    trafficCampaign = "";
    trafficChannel = "";
    trafficStartDate = new Date().toISOString().slice(0, 10);
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

  function reviewPayload(extra: Record<string, unknown> = {}): Record<string, unknown> {
    return {
      selectedNodeId: authoredNode?.id,
      chosenLeverNodeId: reviewLeverId() || undefined,
      experimentNodeId: authoredNode?.type === "experiment" ? authoredNode.id : undefined,
      timeRange: authoredEvidence?.dateRange ?? {},
      reviewWalk: authoredReviewSummary?.steps ?? [],
      topLevers: authoredReviewSummary?.topLevers ?? [],
      evidenceSnapshot: evidenceSnapshot() ?? undefined,
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

  async function saveReview() {
    if (!authoredNode) return;
    if ((reviewAction === "log_decision" || reviewAction === "result_readout") && !reviewNote.trim()) return;
    if (reviewAction === "create_traffic_source_test" && !trafficSource.trim() && !trafficCampaign.trim() && !trafficChannel.trim()) return;
    const actionPayload = reviewAction === "create_traffic_source_test"
      ? {
          actionType: reviewAction,
          trafficSourceTest: {
            label: trafficLabel.trim() || "New traffic-source test",
            source: trafficSource.trim() || undefined,
            campaign: trafficCampaign.trim() || undefined,
            channel_group: trafficChannel.trim() || undefined,
            startDate: trafficStartDate,
            attributionModel: "first_touch",
            primaryMeasure: "metric_nc_cpa",
          },
        }
      : { actionType: reviewAction, outcome: authoredNode.type === "experiment" ? authoredNode.status || undefined : undefined };
    await onCreateReviewSession(reviewPayload(actionPayload));
    reviewNote = "";
    reviewNextAction = "";
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
          <a class="workspace-link" href={workspaceHref}>Open in workspace</a>
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
      {#if authoredOperatingError}
        <section class="op-alert">{authoredOperatingError}</section>
      {/if}

      <section>
        <div class="section-label">Review</div>
        <div class="op-muted">{authoredReviewSummary?.stopReason ?? "No review context"}</div>
        {#each authoredReviewSummary?.steps ?? [] as step}
          <button class="op-row" type="button" disabled>
            <span style={`padding-left: ${step.depth * 12}px`}>{step.label}</span>
            <strong>{fmtValue(step.delta ?? 0, node.unit)}</strong>
          </button>
        {/each}
        {#if authoredReviewSummary?.topLevers?.length}
          <div class="section-label nested">Root-impact levers</div>
          {#each authoredReviewSummary.topLevers as lever}
            <div class="kv compact">
              <span>{lever.label}</span>
              <strong>{lever.rootImpact == null ? "unsized" : fmtValue(lever.rootImpact, node.unit)}</strong>
            </div>
          {/each}
        {/if}
      </section>

      <section>
        <div class="section-label">Save review</div>
        <select bind:value={reviewAction}>
          <option value="log_decision">Decision</option>
          {#if authoredNode.type === "experiment"}
            <option value="result_readout">Result readout</option>
          {/if}
          {#if authoredNode.type === "lever" || authoredNode.type === "surface"}
            <option value="create_traffic_source_test">Create follow-up test</option>
          {/if}
        </select>
        <textarea rows="3" bind:value={reviewNote} placeholder="What did we decide in this review?" />
        <input bind:value={reviewNextAction} placeholder="Next action" />
        <button class="op-primary" type="button" on:click={saveReview} disabled={authoredOperatingSaving || ((reviewAction === "log_decision" || reviewAction === "result_readout") && !reviewNote.trim())}>Save review</button>
      </section>

      {#if authoredNode.type === "lever"}
        <section>
          <div class="section-label">Owner and target</div>
          <label>Owner<input bind:value={ownerDraft} /></label>
          <div class="op-grid">
            <label>Baseline<input type="number" step="any" bind:value={baselineDraft} /></label>
            <label>Target<input type="number" step="any" bind:value={targetDraft} /></label>
          </div>
          <label>Target date<input type="date" bind:value={targetDateDraft} /></label>
          <button class="op-primary" type="button" on:click={saveOwnerTarget} disabled={authoredOperatingSaving}>Save owner/target</button>
        </section>
      {/if}

      {#if authoredNode.type === "lever" || authoredNode.type === "surface"}
        <section>
          <div class="section-label">New traffic-source test</div>
          <label>Label<input bind:value={trafficLabel} /></label>
          <div class="op-grid">
            <label>Channel<input bind:value={trafficChannel} placeholder="Meta" /></label>
            <label>Source<input bind:value={trafficSource} placeholder="Conversion" /></label>
          </div>
          <label>Campaign<input bind:value={trafficCampaign} placeholder="Campaign name" /></label>
          <label>Start date<input type="date" bind:value={trafficStartDate} /></label>
          <button class="op-primary" type="button" on:click={createTrafficTest} disabled={authoredOperatingSaving || (!trafficSource.trim() && !trafficCampaign.trim() && !trafficChannel.trim())}>Create test</button>
        </section>
      {/if}

      {#if authoredNode.type === "experiment"}
        <section>
          <div class="section-label">Experiment status</div>
          <div class="op-actions">
            <button type="button" on:click={() => setExperimentStatus("running")} disabled={authoredOperatingSaving}>Launch</button>
            <button type="button" on:click={() => saveReadout("won")} disabled={authoredOperatingSaving}>Won</button>
            <button type="button" on:click={() => saveReadout("lost")} disabled={authoredOperatingSaving}>Lost</button>
            <button type="button" on:click={() => saveReadout("shipped")} disabled={authoredOperatingSaving}>Shipped</button>
          </div>
          <textarea rows="3" bind:value={readoutNote} placeholder="Readout note" />
        </section>

        <section>
          <div class="section-label">Evidence snapshot</div>
          {#if authoredEvidenceLoading}
            <div class="op-muted">Loading evidence...</div>
          {:else if !authoredEvidence}
            <div class="op-muted">No evidence loaded.</div>
          {:else if authoredEvidence.status !== "ok"}
            <div class="op-muted">{authoredEvidence.message ?? authoredEvidence.status}</div>
          {:else}
            <div class="op-muted">{authoredEvidence.resultLabel} · {authoredEvidence.queryBasis}</div>
            {#each Object.entries(authoredEvidence.metrics ?? {}).slice(0, 8) as [key, value]}
              <div class="kv compact"><span>{key.replaceAll("_", " ")}</span><strong>{fmtValue(value, key.includes("revenue") || key.includes("spend") || key.includes("cpa") ? "currency" : "count")}</strong></div>
            {/each}
            {#each authoredEvidence.warnings ?? [] as warning}
              <div class="op-warning">{warning}</div>
            {/each}
          {/if}
        </section>
      {/if}

      <section>
        <div class="section-label">Decision</div>
        <textarea rows="3" bind:value={decisionNote} placeholder="What did we decide?" />
        <button class="op-primary" type="button" on:click={logDecision} disabled={authoredOperatingSaving || !decisionNote.trim()}>Log decision</button>
      </section>

      <section>
        <div class="section-label">Recent reviews</div>
        {#if authoredReviewSessionsLoading}
          <div class="op-muted">Loading reviews...</div>
        {:else if !authoredReviewSessions.length}
          <div class="op-muted">No saved reviews yet.</div>
        {:else}
          {#each authoredReviewSessions.filter((session) => session.selected_node_id === authoredNode?.id || session.chosen_lever_node_id === authoredNode?.id || session.experiment_node_id === authoredNode?.id).slice(0, 5) as session}
            <div class="event-row">
              <strong>{new Date(session.created_at).toLocaleDateString()}</strong>
              <span>{sessionText(session)}</span>
            </div>
          {/each}
        {/if}
      </section>

      <section>
        <div class="section-label">Decision log</div>
        {#if authoredEventsLoading}
          <div class="op-muted">Loading events...</div>
        {:else}
          {#each authoredEvents.filter((event) => event.node_id === authoredNode?.id).slice(0, 8) as event}
            <div class="event-row">
              <strong>{event.event_type.replaceAll("_", " ")}</strong>
              <span>{eventText(event)}</span>
            </div>
          {/each}
        {/if}
      </section>
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
    width: min(420px, 90%);
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
  .nested {
    margin-top: 12px;
  }
  .op-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
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
  .op-alert,
  .op-warning {
    border: 1px solid #f3c969;
    border-radius: 6px;
    padding: 8px;
    color: #6f4b00;
    background: #fff8df;
    font-size: 12px;
  }
  .op-row {
    display: flex;
    width: 100%;
    justify-content: space-between;
    border: 0;
    border-bottom: 1px solid #edf0f4;
    padding: 6px 0;
    background: transparent;
    color: #17202a;
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
