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
      <h1 class="text-2xl font-semibold text-gray-800">Workshop</h1>
      <p class="mt-1 text-sm text-gray-500">
        Manage client data models — edit, compile, and deploy
      </p>
    </div>
    <button
      class="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      on:click={() => {
        showNewModal = true;
      }}
      disabled={loading}
    >
      New Client
    </button>
  </div>

  {#if errorMessage}
    <div
      class="mb-6 rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
    >
      {errorMessage}
      <button
        class="ml-2 text-red-400 hover:text-red-600"
        on:click={() => {
          errorMessage = "";
        }}>&times;</button
      >
    </div>
  {/if}

  {#if loading}
    <div class="flex justify-center py-12">
      <svg
        class="h-6 w-6 animate-spin text-gray-400"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          class="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="4"
        />
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
      </svg>
    </div>
  {:else if clients.length === 0}
    <div class="rounded border border-dashed border-gray-300 px-8 py-16 text-center">
      <p class="text-gray-500">No clients yet</p>
      <p class="mt-1 text-sm text-gray-400">
        Create one from a template to get started
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {#each clients as client (client.name)}
        {@const fileCount = [client.has_config, client.has_sources, client.has_ontology, client.has_tracking_plan].filter(Boolean).length}
        <a
          href="/workshop/{encodeURIComponent(client.name)}"
          class="group rounded border border-gray-200 bg-white px-5 py-4 transition hover:border-blue-300 hover:shadow-sm"
        >
          <h3 class="font-medium text-gray-800 group-hover:text-blue-600">
            {client.name}
          </h3>
          <p class="mt-1 text-xs text-gray-400">
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
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
    on:click|self={() => {
      showNewModal = false;
    }}
  >
    <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
      <h2 class="mb-4 text-lg font-semibold text-gray-800">New Client</h2>

      {#if templates.length === 0}
        <p class="text-sm text-gray-500">No templates available.</p>
      {:else}
        <label class="mb-1 block text-sm font-medium text-gray-700">
          Template
        </label>
        <select
          bind:value={selectedTemplate}
          class="mb-4 w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        >
          <option value="" disabled>Select a template...</option>
          {#each templates as t (t.name)}
            <option value={t.name}>{t.display_name || t.name}</option>
          {/each}
        </select>
      {/if}

      <div class="flex justify-end gap-2">
        <button
          class="rounded border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
          on:click={() => {
            showNewModal = false;
          }}
        >
          Cancel
        </button>
        <button
          class="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          on:click={handleCreate}
          disabled={!selectedTemplate || creating}
        >
          {creating ? "Creating..." : "Create"}
        </button>
      </div>
    </div>
  </div>
{/if}
