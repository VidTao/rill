<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import {
    listClients,
    listTemplates,
    createClient,
  } from "$lib/bratrax/workshop/api";
  import type {
    ClientSummary,
    TemplateSummary,
  } from "$lib/bratrax/workshop/types";

  let clients: ClientSummary[] = [];
  let templates: TemplateSummary[] = [];
  let loading = true;
  let creating = false;
  let errorMessage = "";

  // New client modal
  let showNewModal = false;
  let selectedTemplate = "";

  onMount(async () => {
    try {
      const [c, t] = await Promise.all([listClients(), listTemplates()]);
      clients = c;
      templates = t;
    } catch (e) {
      errorMessage =
        e instanceof Error ? e.message : "Failed to load workshop data";
    } finally {
      loading = false;
    }
  });

  async function handleCreate() {
    if (!selectedTemplate) return;
    creating = true;
    errorMessage = "";
    try {
      const result = await createClient(selectedTemplate);
      showNewModal = false;
      selectedTemplate = "";
      goto(`/workshop/${encodeURIComponent(result.name)}`);
    } catch (e) {
      errorMessage =
        e instanceof Error ? e.message : "Failed to create client";
    } finally {
      creating = false;
    }
  }
</script>

<div class="mx-auto w-full max-w-4xl px-6 py-12">
  <div class="mb-10 flex items-center justify-between">
    <div>
      <div class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        02 — WORKSHOP
      </div>
      <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
        Client Models
      </h1>
      <p class="mt-2 text-[15px] font-light text-bratrax-text-body">
        Manage client data models — edit, compile, and&nbsp;deploy
      </p>
    </div>
    <div class="flex items-center gap-2">
      <a
        href="/workshop/catalogs"
        class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-acid hover:text-bratrax-acid"
      >
        Source Catalogs
      </a>
      <button
        class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
        on:click={() => {
          showNewModal = true;
        }}
        disabled={loading}
      >
        New Client
      </button>
    </div>
  </div>

  {#if errorMessage}
    <div class="mb-6 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato">
      {errorMessage}
      <button
        class="ml-2 text-bratrax-tomato/60 hover:text-bratrax-tomato"
        on:click={() => {
          errorMessage = "";
        }}>&times;</button
      >
    </div>
  {/if}

  {#if loading}
    <div class="flex justify-center py-12">
      <svg
        class="h-6 w-6 animate-spin text-bratrax-acid"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>
  {:else if clients.length === 0}
    <div class="border border-dashed border-bratrax-border px-8 py-16 text-center">
      <p class="text-bratrax-text-muted">No clients yet</p>
      <p class="mt-1 text-sm text-bratrax-text-muted">
        Create one from a template to get started
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {#each clients as client (client.name)}
        {@const fileCount = [client.has_config, client.has_sources, client.has_ontology, client.has_tracking_plan].filter(Boolean).length}
        <a
          href="/workshop/{encodeURIComponent(client.name)}"
          class="group relative border border-bratrax-border border-t-4 border-t-bratrax-acid bg-bratrax-surface px-5 py-4 transition hover:border-bratrax-acid/50 hover:bg-bratrax-hover"
        >
          <h3 class="font-semibold text-bratrax-text-headline group-hover:text-bratrax-acid">
            {client.name}
          </h3>
          <p class="mt-1 font-mono text-[10px] text-bratrax-text-muted">
            {fileCount} / 4 files
          </p>
        </a>
      {/each}
    </div>
  {/if}
</div>

<!-- New Client Modal -->
{#if showNewModal}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click|self={() => {
      showNewModal = false;
    }}
  >
    <div class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6">
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-4 text-lg font-bold text-bratrax-text-headline">New Client</h2>

      {#if templates.length === 0}
        <p class="text-sm text-bratrax-text-muted">No templates available.</p>
      {:else}
        <label class="mb-1 block font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
          Template
        </label>
        <select
          bind:value={selectedTemplate}
          class="mb-4 w-full border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none focus:border-bratrax-acid"
        >
          <option value="" disabled>Select a template...</option>
          {#each templates as t (t.name)}
            <option value={t.name}>{t.display_name || t.name}</option>
          {/each}
        </select>
      {/if}

      <div class="flex justify-end gap-2">
        <button
          class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover"
          on:click={() => {
            showNewModal = false;
          }}
        >
          Cancel
        </button>
        <button
          class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
          on:click={handleCreate}
          disabled={!selectedTemplate || creating}
        >
          {creating ? "CREATING..." : "CREATE"}
        </button>
      </div>
    </div>
  </div>
{/if}
