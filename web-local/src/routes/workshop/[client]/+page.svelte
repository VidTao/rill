<script lang="ts">
  import { onMount } from "svelte";
  import {
    getClientFiles,
    saveClientFile,
    validateClient,
    compileClient,
    deployClient,
  } from "$lib/bratrax/workshop/api";
  import {
    FILE_KEYS,
    FILE_LABELS,
    type FileKey,
    type ClientFiles,
    type WorkshopPhase,
    type WorkshopResults,
  } from "$lib/bratrax/workshop/types";
  import ResultsPanel from "./ResultsPanel.svelte";
  import SourcesEditor from "./editors/SourcesEditor.svelte";
  import OntologyEditor from "./editors/OntologyEditor.svelte";
  import TrackingPlanEditor from "./editors/TrackingPlanEditor.svelte";

  export let data: { clientName: string };

  // ── State ──
  let files: ClientFiles = { config: "", sources: "", ontology: "", tracking_plan: "" };
  let activeTab: FileKey = "sources";
  let loading = true;
  let viewMode: "visual" | "yaml" = "visual";
  let errorMessage = "";
  let saving = false;
  let dirty: Record<FileKey, boolean> = { config: false, sources: false, ontology: false, tracking_plan: false };

  let phase: WorkshopPhase = "idle";
  let results: WorkshopResults | null = null;

  // Deploy confirmation
  let showDeployConfirm = false;
  let deployApply = false;
  let deployCreatePr = false;

  onMount(async () => {
    try {
      files = await getClientFiles(data.clientName);
    } catch (e) {
      errorMessage = e instanceof Error ? e.message : "Failed to load client";
    } finally {
      loading = false;
    }
  });

  // ── Reactive dirty flag ──
  $: isDirty = Object.values(dirty).some(Boolean);

  // ── Helpers ──

  function showError(msg: string) {
    errorMessage = msg;
    setTimeout(() => { errorMessage = ""; }, 8000);
  }

  // ── Actions ──

  async function handleSave() {
    saving = true;
    errorMessage = "";
    try {
      const dirtyKeys = FILE_KEYS.filter((k) => dirty[k]);
      for (const key of dirtyKeys) {
        await saveClientFile(data.clientName, key, files[key]);
        dirty = { ...dirty, [key]: false };
      }
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      saving = false;
    }
  }

  async function handleValidate() {
    if (isDirty) await handleSave();
    phase = "validating";
    errorMessage = "";
    try {
      const validation = await validateClient(data.clientName);
      results = { phase: "validation", validation, compile: null, deploy: null };
    } catch (e) {
      showError(e instanceof Error ? e.message : "Validation failed");
    } finally {
      phase = "idle";
    }
  }

  async function handleCompile() {
    if (isDirty) await handleSave();
    phase = "compiling";
    errorMessage = "";
    try {
      const compile = await compileClient(data.clientName);
      results = { phase: "compile", validation: null, compile, deploy: null };
    } catch (e) {
      showError(e instanceof Error ? e.message : "Compilation failed");
    } finally {
      phase = "idle";
    }
  }

  function openDeployConfirm() {
    deployApply = false;
    deployCreatePr = false;
    showDeployConfirm = true;
  }

  async function handleDeploy() {
    showDeployConfirm = false;
    if (isDirty) await handleSave();
    phase = "deploying";
    errorMessage = "";
    try {
      const deploy = await deployClient(data.clientName, {
        apply: deployApply,
        create_pr: deployCreatePr,
      });
      results = { phase: "deploy", validation: null, compile: null, deploy };
    } catch (e) {
      showError(e instanceof Error ? e.message : "Deployment failed");
    } finally {
      phase = "idle";
    }
  }

  function handleEditorInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    files = { ...files, [activeTab]: target.value };
    dirty = { ...dirty, [activeTab]: true };
  }

  function handleVisualChange(newContent: string) {
    files = { ...files, [activeTab]: newContent };
    dirty = { ...dirty, [activeTab]: true };
  }

  // Visual editors are available for sources, ontology, tracking_plan (not config)
  $: hasVisualEditor = activeTab !== "config";
</script>

<div class="flex h-full flex-col overflow-hidden">
  <!-- Header -->
  <div class="flex items-center justify-between border-b border-bratrax-border px-4 py-2">
    <div class="flex items-center gap-3">
      <a href="/workshop" class="font-mono text-[11px] text-bratrax-text-muted hover:text-bratrax-acid">&larr; WORKSHOP</a>
      <h1 class="text-lg font-bold text-bratrax-text-headline">{data.clientName}</h1>
      {#if isDirty}
        <span class="bg-bratrax-acid/20 px-2 py-0.5 font-mono text-[10px] font-bold uppercase text-bratrax-acid">unsaved</span>
      {/if}
    </div>

    <div class="flex items-center gap-2">
      <button
        class="border border-bratrax-border px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-acid hover:text-bratrax-acid disabled:opacity-50"
        on:click={handleSave}
        disabled={!isDirty || saving || phase !== "idle"}
      >
        {saving ? "Saving..." : "Save"}
      </button>
      <button
        class="border border-bratrax-cyan/40 bg-bratrax-cyan/10 px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-cyan hover:bg-bratrax-cyan/20 disabled:opacity-50"
        on:click={handleValidate}
        disabled={phase !== "idle"}
      >
        {phase === "validating" ? "Validating..." : "Validate"}
      </button>
      <button
        class="bg-bratrax-acid px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
        on:click={handleCompile}
        disabled={phase !== "idle"}
      >
        {phase === "compiling" ? "Compiling..." : "Compile"}
      </button>
      <button
        class="bg-bratrax-acid px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
        on:click={openDeployConfirm}
        disabled={phase !== "idle"}
      >
        {phase === "deploying" ? "Deploying..." : "Deploy"}
      </button>
    </div>
  </div>

  {#if errorMessage}
    <div class="border-b border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-2 font-mono text-xs text-bratrax-tomato">
      {errorMessage}
      <button class="ml-2 text-bratrax-tomato/60 hover:text-bratrax-tomato" on:click={() => { errorMessage = ""; }}>&times;</button>
    </div>
  {/if}

  {#if loading}
    <div class="flex flex-1 items-center justify-center">
      <svg class="h-6 w-6 animate-spin text-bratrax-acid" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>
  {:else}
    <!-- Main content: editor + results -->
    <div class="flex flex-1 overflow-hidden">
      <!-- Left: YAML editor -->
      <div class="flex flex-1 flex-col overflow-hidden border-r border-bratrax-border">
        <!-- Tabs + view toggle -->
        <div class="flex items-center border-b border-bratrax-border bg-bratrax-bg">
          <div class="flex flex-1">
            {#each FILE_KEYS as key (key)}
              <button
                class="relative px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider transition
                  {activeTab === key
                    ? 'bg-bratrax-surface text-bratrax-acid border-b-2 border-bratrax-acid'
                    : 'text-bratrax-text-muted hover:text-bratrax-text-body hover:bg-bratrax-hover'}"
                on:click={() => { activeTab = key; }}
              >
                {FILE_LABELS[key]}
                {#if dirty[key]}
                  <span class="absolute right-1 top-1 h-1.5 w-1.5 bg-bratrax-acid" />
                {/if}
              </button>
            {/each}
          </div>
          {#if hasVisualEditor}
            <div class="mr-2 flex border border-bratrax-border text-xs">
              <button
                class="px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider transition {viewMode === 'visual' ? 'bg-bratrax-acid text-bratrax-bg' : 'text-bratrax-text-muted hover:bg-bratrax-hover'}"
                on:click={() => { viewMode = "visual"; }}
              >
                Visual
              </button>
              <button
                class="px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider transition {viewMode === 'yaml' ? 'bg-bratrax-acid text-bratrax-bg' : 'text-bratrax-text-muted hover:bg-bratrax-hover'}"
                on:click={() => { viewMode = "yaml"; }}
              >
                YAML
              </button>
            </div>
          {/if}
        </div>

        <!-- Editor area -->
        {#if viewMode === "yaml" || !hasVisualEditor}
          <textarea
            class="flex-1 resize-none bg-bratrax-bg p-4 font-mono text-sm text-bratrax-text-primary focus:outline-none"
            value={files[activeTab]}
            on:input={handleEditorInput}
            spellcheck="false"
          />
        {:else}
          <div class="flex-1 overflow-y-auto p-4">
            {#if activeTab === "sources"}
              <SourcesEditor content={files.sources} onChange={handleVisualChange} />
            {:else if activeTab === "ontology"}
              <OntologyEditor content={files.ontology} onChange={handleVisualChange} />
            {:else if activeTab === "tracking_plan"}
              <TrackingPlanEditor content={files.tracking_plan} onChange={handleVisualChange} />
            {/if}
          </div>
        {/if}
      </div>

      <!-- Right: Results panel -->
      <div class="w-96 flex-shrink-0 overflow-y-auto bg-bratrax-bg p-4">
        <h3 class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          Results
        </h3>
        <ResultsPanel {results} />
      </div>
    </div>
  {/if}
</div>

<!-- Deploy Confirmation Modal -->
{#if showDeployConfirm}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click|self={() => { showDeployConfirm = false; }}
  >
    <div class="relative w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-6">
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-4 text-lg font-bold text-bratrax-text-headline">Deploy {data.clientName}</h2>

      <label class="mb-3 flex items-center gap-2 text-sm text-bratrax-text-body">
        <input type="checkbox" bind:checked={deployApply} class="accent-bratrax-acid" />
        Apply changes (not just dry-run)
      </label>

      <label class="mb-4 flex items-center gap-2 text-sm text-bratrax-text-body">
        <input type="checkbox" bind:checked={deployCreatePr} disabled={!deployApply} class="accent-bratrax-acid" />
        Create pull request
      </label>

      <div class="flex justify-end gap-2">
        <button
          class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover"
          on:click={() => { showDeployConfirm = false; }}
        >
          Cancel
        </button>
        <button
          class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
          on:click={handleDeploy}
        >
          {deployApply ? "DEPLOY" : "DRY RUN"}
        </button>
      </div>
    </div>
  </div>
{/if}
