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

  $: treeId = $page.params.tree_id;
  $: sessionId = $page.params.session_id;
  $: selectedNodeId =
    session?.selected_node_id ||
    session?.chosen_lever_node_id ||
    session?.experiment_node_id ||
    session?.root_node_id ||
    null;
  $: workspaceHref = session
    ? `/metric-trees?tree_id=${encodeURIComponent(session.tree_id)}${selectedNodeId ? `&node_id=${encodeURIComponent(selectedNodeId)}` : ""}`
    : null;

  function nodeLabel(nodeId: string): string {
    return tree?.nodes.find((node) => node.id === nodeId)?.label ?? nodeId;
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
    {#if workspaceHref}
      <a href={workspaceHref}>Open workspace</a>
    {/if}
  </header>

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
  a {
    color: #245bdb;
    font-size: 13px;
    font-weight: 750;
    white-space: nowrap;
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
</style>
