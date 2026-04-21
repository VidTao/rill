<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { KnowledgeFile } from "$lib/bratrax/workshop/types";

  export let files: KnowledgeFile[] = [];
  export let selectedItem: { type: "prompt" } | { type: "file"; path: string } =
    { type: "prompt" };

  const dispatch = createEventDispatcher<{
    selectPrompt: void;
    selectFile: { path: string };
  }>();

  // Group files by category
  $: rootFiles = files.filter(
    (f) => !f.path.includes("/") && f.path !== "log.md",
  );
  $: discoveries = files.filter((f) => f.path.startsWith("discoveries/"));
  $: insights = files.filter((f) => f.path.startsWith("insights/"));
  $: patterns = files.filter((f) => f.path.startsWith("patterns/"));
  $: logFile = files.find((f) => f.path === "log.md");

  let expandedSections: Record<string, boolean> = {
    discoveries: true,
    insights: true,
    patterns: true,
  };

  function toggleSection(section: string) {
    expandedSections = {
      ...expandedSections,
      [section]: !expandedSections[section],
    };
  }

  function displayName(path: string): string {
    const name = path.split("/").pop() || path;
    return name
      .replace(/\.md$/, "")
      .replace(/_/g, " ")
      .replace(/\d{4}-\d{2}-\d{2}/, (d) => `(${d})`);
  }

  function isActive(path: string): boolean {
    return selectedItem.type === "file" && selectedItem.path === path;
  }

  function isPromptActive(): boolean {
    return selectedItem.type === "prompt";
  }
</script>

<nav
  class="flex h-full w-64 flex-col overflow-y-auto border-r"
  style="border-color: var(--bratrax-border); background: var(--bratrax-bg);"
>
  <!-- HOW I THINK -->
  <div class="px-4 pt-5 pb-2">
    <h3
      class="font-mono text-sm font-bold uppercase tracking-widest"
      style="color: var(--bratrax-text-muted);"
    >
      How I Think
    </h3>
  </div>
  <div class="px-2 pb-3">
    <button
      class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors"
      class:active-item={isPromptActive()}
      style="color: {isPromptActive()
        ? 'var(--bratrax-acid)'
        : 'var(--bratrax-text-body)'};"
      on:click={() => dispatch("selectPrompt")}
    >
      <span class="text-base">&#x2699;</span>
      <span class="font-medium">System Prompt</span>
    </button>
  </div>

  <!-- WHAT I KNOW -->
  <div class="px-4 pt-3 pb-2">
    <h3
      class="font-mono text-sm font-bold uppercase tracking-widest"
      style="color: var(--bratrax-text-muted);"
    >
      What I Know
    </h3>
  </div>
  <div class="px-2 pb-3">
    <!-- Root files (profile, index) -->
    {#each rootFiles as file}
      <button
        class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors"
        class:active-item={isActive(file.path)}
        style="color: {isActive(file.path)
          ? 'var(--bratrax-acid)'
          : 'var(--bratrax-text-body)'};"
        on:click={() => dispatch("selectFile", { path: file.path })}
      >
        <span class="text-base">&#x25CB;</span>
        <span>{displayName(file.path)}</span>
      </button>
    {/each}

    <!-- Insights -->
    {#if insights.length > 0 || true}
      <button
        class="mt-2 flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm"
        style="color: var(--bratrax-text-body);"
        on:click={() => toggleSection("insights")}
      >
        <span class="text-base"
          >{expandedSections.insights ? "\u25BE" : "\u25B8"}</span
        >
        <span class="font-medium">Insights</span>
        <span
          class="ml-auto font-mono text-sm"
          style="color: var(--bratrax-text-muted);">{insights.length}</span
        >
      </button>
      {#if expandedSections.insights}
        {#each insights as file}
          <button
            class="flex w-full items-center gap-2 py-1.5 pl-8 pr-3 text-left text-sm transition-colors"
            class:active-item={isActive(file.path)}
            style="color: {isActive(file.path)
              ? 'var(--bratrax-acid)'
              : 'var(--bratrax-text-body)'};"
            on:click={() => dispatch("selectFile", { path: file.path })}
          >
            <span class="truncate">{displayName(file.path)}</span>
          </button>
        {/each}
        {#if insights.length === 0}
          <p
            class="py-1.5 pl-8 pr-3 text-sm italic"
            style="color: var(--bratrax-text-muted);"
          >
            None yet
          </p>
        {/if}
      {/if}
    {/if}

    <!-- Discoveries -->
    <button
      class="mt-2 flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm"
      style="color: var(--bratrax-text-body);"
      on:click={() => toggleSection("discoveries")}
    >
      <span class="text-base"
        >{expandedSections.discoveries ? "\u25BE" : "\u25B8"}</span
      >
      <span class="font-medium">Discoveries</span>
      <span
        class="ml-auto font-mono text-sm"
        style="color: var(--bratrax-text-muted);">{discoveries.length}</span
      >
    </button>
    {#if expandedSections.discoveries}
      {#each discoveries as file}
        <button
          class="flex w-full items-center gap-2 py-1.5 pl-8 pr-3 text-left text-sm transition-colors"
          class:active-item={isActive(file.path)}
          style="color: {isActive(file.path)
            ? 'var(--bratrax-acid)'
            : 'var(--bratrax-text-body)'};"
          on:click={() => dispatch("selectFile", { path: file.path })}
        >
          <span class="truncate">{displayName(file.path)}</span>
        </button>
      {/each}
      {#if discoveries.length === 0}
        <p
          class="py-1.5 pl-8 pr-3 text-sm italic"
          style="color: var(--bratrax-text-muted);"
        >
          None yet
        </p>
      {/if}
    {/if}

    <!-- Patterns -->
    <button
      class="mt-2 flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm"
      style="color: var(--bratrax-text-body);"
      on:click={() => toggleSection("patterns")}
    >
      <span class="text-base"
        >{expandedSections.patterns ? "\u25BE" : "\u25B8"}</span
      >
      <span class="font-medium">Patterns</span>
      <span
        class="ml-auto font-mono text-sm"
        style="color: var(--bratrax-text-muted);">{patterns.length}</span
      >
    </button>
    {#if expandedSections.patterns}
      {#each patterns as file}
        <button
          class="flex w-full items-center gap-2 py-1.5 pl-8 pr-3 text-left text-sm transition-colors"
          class:active-item={isActive(file.path)}
          style="color: {isActive(file.path)
            ? 'var(--bratrax-acid)'
            : 'var(--bratrax-text-body)'};"
          on:click={() => dispatch("selectFile", { path: file.path })}
        >
          <span class="truncate">{displayName(file.path)}</span>
        </button>
      {/each}
      {#if patterns.length === 0}
        <p
          class="py-1.5 pl-8 pr-3 text-sm italic"
          style="color: var(--bratrax-text-muted);"
        >
          None yet
        </p>
      {/if}
    {/if}
  </div>

  <!-- ACTIVITY -->
  <div class="mt-auto border-t px-4 pt-3 pb-2" style="border-color: var(--bratrax-border);">
    <h3
      class="font-mono text-sm font-bold uppercase tracking-widest"
      style="color: var(--bratrax-text-muted);"
    >
      Activity
    </h3>
  </div>
  <div class="px-2 pb-4">
    {#if logFile}
      <button
        class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors"
        class:active-item={isActive("log.md")}
        style="color: {isActive('log.md')
          ? 'var(--bratrax-acid)'
          : 'var(--bratrax-text-body)'};"
        on:click={() => dispatch("selectFile", { path: "log.md" })}
      >
        <span class="text-base">&#x25CB;</span>
        <span>Timeline</span>
      </button>
    {/if}
  </div>
</nav>

<style>
  .active-item {
    border-left: 3px solid var(--bratrax-acid);
    background: var(--bratrax-acid-muted);
  }

  button:hover:not(.active-item) {
    background: var(--bratrax-hover);
  }
</style>
