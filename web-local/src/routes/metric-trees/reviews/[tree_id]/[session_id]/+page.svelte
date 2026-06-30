<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import MetricTreeReviewReadout from "@rilldata/web-common/features/canvas/components/metric-tree/MetricTreeReviewReadout.svelte";
  import {
    getMetricTree,
    getMetricTreeReviewSession,
  } from "$lib/bratrax/metric-trees/api";
  import type {
    AuthoredMetricTree,
    MetricTreeReviewSession,
  } from "$lib/bratrax/metric-trees/types";

  let tree: AuthoredMetricTree | null = null;
  let session: MetricTreeReviewSession | null = null;
  let loading = true;
  let error = "";
  let shareMessage = "";

  $: treeId = $page.params.tree_id;
  $: sessionId = $page.params.session_id;
  $: selectedNodeId =
    session?.selected_node_id ||
    session?.chosen_lever_node_id ||
    session?.experiment_node_id ||
    session?.root_node_id ||
    null;
  function workspaceHrefForSession(nextSession: MetricTreeReviewSession, nodeId: string | null): string {
    const params = new URLSearchParams({
      tree_id: nextSession.tree_id,
      mode: "review",
      review_session_id: nextSession.session_id,
    });
    if (nodeId) params.set("node_id", nodeId);
    const preset = (nextSession.timeRange as { preset?: string } | undefined)?.preset;
    if (preset) params.set("time_preset", preset);
    return `/metric-trees?${params.toString()}`;
  }

  $: workspaceHref = session ? workspaceHrefForSession(session, selectedNodeId) : null;

  function nodeLabel(nodeId: string): string {
    return tree?.nodes.find((node) => node.id === nodeId)?.label ?? nodeId;
  }

  function actionLabel(action: string | null | undefined): string {
    return action ? action.replaceAll("_", " ") : "review";
  }

  function formatValue(value: number | null | undefined): string {
    if (value == null || !Number.isFinite(value)) return "-";
    return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
  }

  function reviewUrl(): string {
    if (typeof window === "undefined") return `/metric-trees/reviews/${treeId}/${sessionId}`;
    return window.location.href;
  }

  function reviewSummaryText(): string {
    if (!session) return "";
    const decisions = session.decisions?.length
      ? session.decisions
      : [{
          actionType: session.action_type,
          note: session.note,
          outcome: session.outcome,
          nextAction: session.next_action,
        }];
    const lines = [
      `Metric tree review: ${session.summary?.rootLabel ?? tree?.name ?? session.tree_id}`,
      `Time: ${session.summary?.timeLabel ?? "No time context"}`,
      `Root movement: ${formatValue(session.summary?.rootDelta)}`,
      `Root value: ${formatValue(session.summary?.rootValue)}`,
      `Chosen lever: ${session.summary?.chosenLeverLabel ?? (session.chosen_lever_node_id ? nodeLabel(session.chosen_lever_node_id) : "Unassigned")}`,
      `Stop reason: ${session.summary?.stopReason ?? actionLabel(session.action_type)}`,
      "",
      "Decisions:",
      ...decisions.map((decision, index) => {
        const title = decision.note || actionLabel(decision.actionType);
        const outcome = decision.outcome ? ` (${decision.outcome})` : "";
        const next = decision.nextAction ? ` -> ${decision.nextAction}` : "";
        return `${index + 1}. ${actionLabel(decision.actionType)}: ${title}${outcome}${next}`;
      }),
      "",
      `Readout: ${reviewUrl()}`,
    ];
    return lines.join("\n");
  }

  async function copyText(text: string, success: string) {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      shareMessage = success;
    } catch {
      shareMessage = "Copy failed. Select the page URL manually.";
    }
  }

  function printReadout() {
    if (typeof window !== "undefined") window.print();
  }

  async function loadReadout() {
    loading = true;
    error = "";
    try {
      const [nextTree, nextSession] = await Promise.all([
        getMetricTree(treeId),
        getMetricTreeReviewSession(treeId, sessionId),
      ]);
      tree = nextTree;
      session = nextSession;
    } catch (err) {
      tree = null;
      session = null;
      error = err instanceof Error ? err.message : "Could not load review readout.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadReadout();
  });
</script>

<svelte:head>
  <title>Metric tree review readout</title>
</svelte:head>

<div class="readout-page">
  <header>
    <div>
      <div class="eyebrow">Metric tree review</div>
      <h1>{session?.summary?.rootLabel ?? tree?.name ?? "Review readout"}</h1>
    </div>
    <div class="actions">
      {#if session}
        <button type="button" on:click={() => copyText(reviewUrl(), "Review link copied.")}>Copy link</button>
        <button type="button" on:click={() => copyText(reviewSummaryText(), "Review summary copied.")}>Copy summary</button>
        <button type="button" on:click={printReadout}>Print / save PDF</button>
      {/if}
      {#if workspaceHref}
        <a href={workspaceHref}>Open workspace</a>
      {/if}
    </div>
  </header>

  {#if shareMessage}
    <div class="share-message">{shareMessage}</div>
  {/if}

  {#if loading}
    <div class="state">Loading review readout...</div>
  {:else if error}
    <div class="error">{error}</div>
  {:else if session}
    <MetricTreeReviewReadout
      {session}
      nodeLabel={nodeLabel}
      {workspaceHref}
    />
  {:else}
    <div class="state">Review session not found.</div>
  {/if}
</div>

<style>
  :global(body) {
    background: #f7f8fa;
  }
  .readout-page {
    display: grid;
    gap: 18px;
    margin: 0 auto;
    max-width: 980px;
    padding: 32px 20px 56px;
  }
  header {
    align-items: flex-start;
    display: flex;
    gap: 16px;
    justify-content: space-between;
  }
  .eyebrow {
    color: #64748b;
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.5px;
    text-transform: uppercase;
  }
  h1 {
    color: #101828;
    font-size: 28px;
    line-height: 1.1;
    margin: 3px 0 0;
  }
  .actions {
    align-items: center;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
  }
  a,
  button {
    border: 1px solid #d9e0ea;
    border-radius: 999px;
    background: #fff;
    color: #245bdb;
    cursor: pointer;
    font: inherit;
    font-size: 13px;
    font-weight: 750;
    padding: 7px 10px;
    text-decoration: none;
    white-space: nowrap;
  }
  button:hover,
  a:hover {
    border-color: #245bdb;
    background: #f4f7ff;
  }
  .share-message {
    border: 1px solid #b8dcc7;
    border-radius: 8px;
    background: #f0f9f3;
    color: #17663a;
    font-size: 13px;
    padding: 10px 12px;
  }
  .state,
  .error {
    border: 1px solid #d9e0ea;
    border-radius: 8px;
    background: #fff;
    color: #64748b;
    padding: 16px;
  }
  .error {
    border-color: #f0b4a8;
    color: #9b2c1f;
    background: #fff5f3;
  }

  @media print {
    :global(body) {
      background: #fff;
    }
    .readout-page {
      max-width: none;
      padding: 0;
    }
    .actions,
    .share-message {
      display: none;
    }
  }
</style>
