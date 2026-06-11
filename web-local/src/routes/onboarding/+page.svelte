<script lang="ts">
  import { onMount, tick } from "svelte";
  import { goto } from "$app/navigation";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import {
    getChecklist,
    dismissChecklist,
    coreSetupIncomplete,
    progressCounts,
    type ChecklistResult,
  } from "$lib/bratrax/onboarding/checklist";
  import Section from "$lib/bratrax/onboarding/Section.svelte";
  import ResourcesBlock from "$lib/bratrax/onboarding/ResourcesBlock.svelte";

  // Fixed section numbering (Section 3 is conditional, so we can't derive it
  // from array position — the numbers must stay stable per the spec mock).
  const SECTION_NUMBER: Record<string, number> = {
    connect_data: 1,
    configure_business: 2,
    paid_media_tracking: 3,
    first_look: 4,
    bring_your_team: 5,
  };
  const SECTION_NOTE: Record<string, string> = {
    paid_media_tracking: "Shown once at least one ad platform is connected.",
  };

  let result: ChecklistResult | null = null;
  let loading = true;
  let errorMessage = "";
  let dismissing = false;

  $: firstName = ($bratraxUser?.name || $bratraxUser?.email || "").split(" ")[0];
  $: isAdmin = result?.role === "admin" || result?.role === "super_admin";
  $: counts = result ? progressCounts(result) : { done: 0, total: 0 };
  $: canHide = result ? !coreSetupIncomplete(result) : false;
  $: pct = counts.total ? Math.round((counts.done / counts.total) * 100) : 0;

  async function load() {
    try {
      result = await getChecklist();
      errorMessage = "";
    } catch (e) {
      errorMessage = e instanceof Error ? e.message : "Failed to load checklist";
    } finally {
      loading = false;
    }
    await tick();
    scrollToHash();
  }

  function scrollToHash() {
    if (typeof location === "undefined" || !location.hash) return;
    const el = document.getElementById(location.hash.slice(1));
    el?.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  async function hideForNow() {
    if (dismissing) return;
    dismissing = true;
    try {
      await dismissChecklist();
    } catch (e) {
      console.warn("dismiss failed", e);
    }
    void goto("/developer");
  }

  onMount(load);
</script>

<div class="page">
  <div class="inner">
    {#if loading}
      <div class="state">Loading your setup checklist…</div>
    {:else if errorMessage}
      <div class="state error">{errorMessage}</div>
    {:else if result}
      <h1 class="headline">
        {#if isAdmin}
          Welcome back{firstName ? `, ${firstName}` : ""}. Let's finish setting up
          <span class="accent">{result.workspace_name}</span>.
        {:else}
          <span class="accent">{result.workspace_name}</span> — Getting started.
        {/if}
      </h1>

      {#if isAdmin}
        <div class="progress-card">
          <div class="track">
            <div class="fill" style={`width:${pct}%`}></div>
          </div>
          <div class="progress-meta">
            {counts.done} of {counts.total} steps complete
          </div>
          <button class="hide" disabled={!canHide || dismissing} on:click={hideForNow}>
            Hide for now
          </button>
        </div>
      {/if}

      {#each result.sections as section (section.key)}
        <Section
          {section}
          index={SECTION_NUMBER[section.key] ?? 0}
          numbered={isAdmin}
          note={SECTION_NOTE[section.key] ?? null}
          on:changed={load}
        />
      {/each}

      <ResourcesBlock />
    {/if}
  </div>
</div>

<style>
  .page {
    /* The root layout renders this inside a `h-screen overflow-hidden
       flex flex-col` container. Take the remaining vertical space and scroll
       internally (min-height:0 lets a flex child shrink below content size so
       overflow-y actually engages). */
    flex: 1 1 0;
    min-height: 0;
    overflow-y: auto;
    background: var(--color-bg);
    padding: 40px 24px 80px;
  }
  .inner {
    max-width: 820px;
    margin: 0 auto;
  }
  .state {
    font-family: "Outfit", sans-serif;
    color: var(--color-text-secondary);
    padding: 60px 0;
    text-align: center;
  }
  .state.error {
    color: var(--color-red);
  }
  .headline {
    font-family: "Outfit", sans-serif;
    font-weight: 900;
    font-size: 28px;
    line-height: 1.2;
    color: var(--color-text);
    margin: 0 0 28px;
  }
  .accent {
    font-style: italic;
    color: var(--color-acid-text);
  }
  .progress-card {
    display: flex;
    align-items: center;
    gap: 16px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    padding: 16px 18px;
    margin-bottom: 24px;
  }
  .track {
    flex: 1 1 auto;
    height: 10px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    overflow: hidden;
  }
  .fill {
    height: 100%;
    background: var(--color-acid);
    transition: width 300ms ease;
  }
  .progress-meta {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    color: var(--color-text-secondary);
    white-space: nowrap;
  }
  .hide {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    background: transparent;
    border: 1px solid var(--color-border-strong);
    color: var(--color-text);
    padding: 6px 12px;
    cursor: pointer;
    white-space: nowrap;
  }
  .hide:hover:not(:disabled) {
    border-color: var(--color-acid);
  }
  .hide:disabled {
    opacity: 0.4;
    cursor: default;
  }
</style>
