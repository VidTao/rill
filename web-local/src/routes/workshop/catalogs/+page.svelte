<script lang="ts">
  import { onMount } from "svelte";
  import {
    listCatalogs,
    searchCatalog,
  } from "$lib/bratrax/workshop/api";
  import type {
    CatalogSource,
    CatalogStream,
    CatalogSearchResult,
  } from "$lib/bratrax/workshop/types";

  let sources: CatalogSource[] = [];
  let loading = true;
  let errorMessage = "";

  // Expansion state
  let expandedSource: string | null = null;
  let expandedStream: string | null = null;

  // Search
  let searchQuery = "";
  let searchResults: CatalogSearchResult[] = [];
  let searchCount = 0;
  let searching = false;
  let searchTimer: ReturnType<typeof setTimeout> | null = null;

  onMount(async () => {
    try {
      sources = await listCatalogs();
    } catch (e) {
      errorMessage = e instanceof Error ? e.message : "Failed to load catalogs";
    } finally {
      loading = false;
    }
  });

  function toggleSource(name: string) {
    expandedSource = expandedSource === name ? null : name;
    expandedStream = null;
  }

  function toggleStream(key: string) {
    expandedStream = expandedStream === key ? null : key;
  }

  function streamFieldCount(stream: CatalogStream): number {
    return stream.fields.length;
  }

  function totalFields(src: CatalogSource): number {
    return src.streams.reduce((sum, s) => sum + s.fields.length, 0);
  }

  function handleSearchInput(e: Event) {
    const value = (e.target as HTMLInputElement).value;
    searchQuery = value;
    if (searchTimer) clearTimeout(searchTimer);
    if (!value.trim()) {
      searchResults = [];
      searchCount = 0;
      return;
    }
    searchTimer = setTimeout(() => doSearch(value), 300);
  }

  async function doSearch(query: string) {
    searching = true;
    try {
      const res = await searchCatalog(query);
      searchResults = res.results;
      searchCount = res.count;
    } catch (e) {
      errorMessage = e instanceof Error ? e.message : "Search failed";
    } finally {
      searching = false;
    }
  }

  function copyRef(ref: string) {
    navigator.clipboard.writeText(ref);
  }

  const CATEGORY_COLORS: Record<string, string> = {
    ads: "bg-bratrax-lavender/15 text-bratrax-lavender",
    commerce: "bg-bratrax-cyan/15 text-bratrax-cyan",
    crm: "bg-bratrax-acid/15 text-bratrax-acid",
    payments: "bg-bratrax-tomato/15 text-bratrax-tomato",
  };

  function categoryStyle(cat: string): string {
    return CATEGORY_COLORS[cat] ?? "bg-bratrax-hover text-bratrax-text-muted";
  }

  const TYPE_COLORS: Record<string, string> = {
    STRING: "text-bratrax-acid",
    INTEGER: "text-bratrax-cyan",
    INT64: "text-bratrax-cyan",
    FLOAT64: "text-bratrax-lavender",
    FLOAT: "text-bratrax-lavender",
    BOOLEAN: "text-bratrax-acid",
    TIMESTAMP: "text-bratrax-tomato",
    DATE: "text-bratrax-tomato",
    RECORD: "text-bratrax-text-muted",
    ARRAY: "text-bratrax-text-muted",
  };
</script>

<div class="flex h-full flex-col overflow-hidden">
  <!-- Header -->
  <div class="flex items-center justify-between border-b border-bratrax-border px-4 py-2">
    <div class="flex items-center gap-3">
      <a href="/workshop" class="font-mono text-[11px] text-bratrax-text-muted hover:text-bratrax-acid">&larr; WORKSHOP</a>
      <h1 class="text-lg font-bold text-bratrax-text-headline">Source Catalogs</h1>
      {#if !loading}
        <span class="font-mono text-[10px] text-bratrax-text-muted">
          {sources.length} source{sources.length !== 1 ? "s" : ""}
        </span>
      {/if}
    </div>
  </div>

  {#if errorMessage}
    <div class="border-b border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-2 font-mono text-xs text-bratrax-tomato">
      {errorMessage}
      <button class="ml-2 text-bratrax-tomato/60 hover:text-bratrax-tomato" on:click={() => { errorMessage = ""; }}>&times;</button>
    </div>
  {/if}

  <div class="flex flex-1 overflow-hidden">
    <!-- Left: source tree -->
    <div class="flex-1 overflow-y-auto p-4">
      <!-- Search bar -->
      <div class="mb-4">
        <div class="relative">
          <input
            type="text"
            placeholder="Search fields across all sources..."
            value={searchQuery}
            on:input={handleSearchInput}
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 pl-8 text-sm text-bratrax-text-primary focus:border-bratrax-acid focus:outline-none"
          />
          <svg class="absolute left-2.5 top-2.5 h-4 w-4 text-bratrax-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          {#if searching}
            <svg class="absolute right-2.5 top-2.5 h-4 w-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          {/if}
        </div>
      </div>

      <!-- Search results -->
      {#if searchQuery.trim() && searchResults.length > 0}
        <div class="mb-6">
          <h3 class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
            {searchCount} result{searchCount !== 1 ? "s" : ""} for "{searchQuery}"
          </h3>
          <div class="space-y-1">
            {#each searchResults as result}
              <div class="flex items-center justify-between bg-bratrax-surface px-3 py-2 text-xs border border-bratrax-border">
                <div class="flex items-center gap-2">
                  <span class="font-mono font-medium text-bratrax-text-primary">{result.field}</span>
                  <span class="font-mono {TYPE_COLORS[result.type] ?? 'text-bratrax-text-muted'}">{result.type}</span>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-bratrax-text-muted">{result.label} / {result.stream}</span>
                  <button
                    class="bg-bratrax-hover px-1.5 py-0.5 font-mono text-[10px] text-bratrax-text-muted hover:bg-bratrax-acid/20 hover:text-bratrax-acid"
                    on:click={() => copyRef(result.source_ref)}
                    title="Copy source_ref"
                  >
                    copy ref
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else if searchQuery.trim() && !searching && searchResults.length === 0}
        <p class="mb-6 text-sm text-gray-400">No fields match "{searchQuery}"</p>
      {/if}

      <!-- Source tree -->
      {#if loading}
        <div class="flex justify-center py-12">
          <svg class="h-6 w-6 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        </div>
      {:else}
        <div class="space-y-1">
          {#each sources as src (src.source_name)}
            <!-- Source row -->
            <button
              class="flex w-full items-center gap-2 rounded px-3 py-2 text-left text-sm transition hover:bg-gray-50
                {expandedSource === src.source_name ? 'bg-gray-50' : ''}"
              on:click={() => toggleSource(src.source_name)}
            >
              <svg
                class="h-3 w-3 flex-shrink-0 text-gray-400 transition-transform {expandedSource === src.source_name ? 'rotate-90' : ''}"
                fill="currentColor" viewBox="0 0 20 20"
              >
                <path d="M6 4l8 6-8 6V4z" />
              </svg>
              <span class="font-medium text-gray-800">{src.label}</span>
              <span class="rounded px-1.5 py-0.5 text-[10px] font-medium uppercase {categoryStyle(src.category)}">
                {src.category}
              </span>
              <span class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-500">
                {src.source_type}
              </span>
              <span class="ml-auto text-xs text-gray-400">
                {src.streams.length} stream{src.streams.length !== 1 ? "s" : ""} &middot; {totalFields(src)} fields
              </span>
            </button>

            <!-- Streams -->
            {#if expandedSource === src.source_name}
              <div class="ml-5 space-y-0.5">
                {#each src.streams as stream (stream.name)}
                  {@const streamKey = `${src.source_name}/${stream.name}`}
                  <!-- Stream row -->
                  <button
                    class="flex w-full items-center gap-2 rounded px-3 py-1.5 text-left text-sm transition hover:bg-blue-50
                      {expandedStream === streamKey ? 'bg-blue-50' : ''}"
                    on:click={() => toggleStream(streamKey)}
                  >
                    <svg
                      class="h-2.5 w-2.5 flex-shrink-0 text-gray-300 transition-transform {expandedStream === streamKey ? 'rotate-90' : ''}"
                      fill="currentColor" viewBox="0 0 20 20"
                    >
                      <path d="M6 4l8 6-8 6V4z" />
                    </svg>
                    <span class="font-mono text-xs text-gray-700">{stream.name}</span>
                    <span class="ml-auto text-xs text-gray-400">
                      {streamFieldCount(stream)} field{streamFieldCount(stream) !== 1 ? "s" : ""}
                    </span>
                  </button>

                  <!-- Fields -->
                  {#if expandedStream === streamKey}
                    <div class="ml-5 rounded border border-gray-100 bg-white">
                      <table class="w-full text-xs">
                        <thead>
                          <tr class="border-b border-gray-100 bg-gray-50">
                            <th class="px-3 py-1.5 text-left font-medium text-gray-500">Field</th>
                            <th class="px-3 py-1.5 text-left font-medium text-gray-500">Type</th>
                            <th class="px-3 py-1.5 text-left font-medium text-gray-500">Nullable</th>
                            <th class="px-3 py-1.5 text-right font-medium text-gray-500">Ref</th>
                          </tr>
                        </thead>
                        <tbody>
                          {#each stream.fields as field (field.name)}
                            <tr class="border-b border-gray-50 hover:bg-gray-50">
                              <td class="px-3 py-1 font-mono text-gray-800">{field.name}</td>
                              <td class="px-3 py-1 font-mono {TYPE_COLORS[field.type] ?? 'text-gray-500'}">{field.type}</td>
                              <td class="px-3 py-1 text-gray-400">{field.nullable ? "yes" : "no"}</td>
                              <td class="px-3 py-1 text-right">
                                <button
                                  class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-500 hover:bg-gray-200 hover:text-gray-700"
                                  on:click|stopPropagation={() => copyRef(`$sources.${src.source_name}.${stream.name}.${field.name}`)}
                                  title="Copy $sources ref to clipboard"
                                >
                                  copy
                                </button>
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if}
                {/each}
              </div>
            {/if}
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
