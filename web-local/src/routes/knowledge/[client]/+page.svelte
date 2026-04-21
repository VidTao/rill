<script lang="ts">
  import { onMount } from "svelte";
  import {
    listKnowledgeFiles,
    readKnowledgeFile,
    getSystemPrompt,
  } from "$lib/bratrax/workshop/api";
  import type { KnowledgeFile } from "$lib/bratrax/workshop/types";
  import KnowledgeSidebar from "./KnowledgeSidebar.svelte";
  import KnowledgeViewer from "./KnowledgeViewer.svelte";

  export let data: { clientName: string };

  let files: KnowledgeFile[] = [];
  let systemPrompt = "";
  let selectedItem: { type: "prompt" } | { type: "file"; path: string } = {
    type: "prompt",
  };
  let content = "";
  let contentModified: string | null = null;
  let loading = true;
  let loadingContent = false;
  let error = "";

  $: viewerTitle =
    selectedItem.type === "prompt"
      ? "System Prompt"
      : selectedItem.path.split("/").pop()?.replace(/\.md$/, "") || "";

  onMount(async () => {
    try {
      const [fileList, prompt] = await Promise.all([
        listKnowledgeFiles(data.clientName),
        getSystemPrompt(data.clientName),
      ]);
      files = fileList;
      systemPrompt = prompt;
      content = prompt;
      contentModified = null;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load knowledge base";
    } finally {
      loading = false;
    }
  });

  async function handleSelectPrompt() {
    selectedItem = { type: "prompt" };
    content = systemPrompt;
    contentModified = null;
  }

  async function handleSelectFile(event: CustomEvent<{ path: string }>) {
    const { path } = event.detail;
    selectedItem = { type: "file", path };
    loadingContent = true;
    try {
      const result = await readKnowledgeFile(data.clientName, path);
      content = result.content;
      const fileMeta = files.find((f) => f.path === path);
      contentModified = fileMeta?.modified || null;
    } catch (e) {
      content = "";
      error = e instanceof Error ? e.message : "Failed to load file";
    } finally {
      loadingContent = false;
    }
  }
</script>

<div
  class="flex h-screen flex-col"
  style="background: var(--bratrax-bg);"
>
  <!-- Header -->
  <header
    class="flex items-center justify-between border-b px-6 py-4"
    style="border-color: var(--bratrax-border); background: var(--bratrax-surface);"
  >
    <div class="flex items-center gap-3">
      <a
        href="/workshop/{data.clientName}"
        class="font-mono text-sm uppercase tracking-wider"
        style="color: var(--bratrax-text-muted); text-decoration: none;"
      >
        &larr; Workshop
      </a>
      <span style="color: var(--bratrax-border);">/</span>
      <h1
        class="font-mono text-sm font-bold uppercase tracking-wider"
        style="color: var(--bratrax-text-headline);"
      >
        {data.clientName}
      </h1>
      <span style="color: var(--bratrax-border);">/</span>
      <span
        class="font-mono text-sm font-bold uppercase tracking-wider"
        style="color: var(--bratrax-acid);"
      >
        Knowledge
      </span>
    </div>

    <div class="flex items-center gap-2">
      <span
        class="font-mono text-sm"
        style="color: var(--bratrax-text-muted);"
      >
        {files.length} files
      </span>
    </div>
  </header>

  <!-- Error banner -->
  {#if error}
    <div
      class="border-b px-6 py-2 text-sm"
      style="background: rgba(255, 59, 48, 0.08); border-color: var(--bratrax-border); color: var(--bratrax-tomato);"
    >
      {error}
      <button
        class="ml-3 underline"
        on:click={() => (error = "")}
      >
        dismiss
      </button>
    </div>
  {/if}

  <!-- Main content -->
  {#if loading}
    <div class="flex flex-1 items-center justify-center">
      <p
        class="font-mono text-sm"
        style="color: var(--bratrax-text-muted);"
      >
        Loading knowledge base...
      </p>
    </div>
  {:else}
    <div class="flex flex-1 overflow-hidden">
      <KnowledgeSidebar
        {files}
        {selectedItem}
        on:selectPrompt={handleSelectPrompt}
        on:selectFile={handleSelectFile}
      />
      <div class="flex-1 overflow-hidden">
        <KnowledgeViewer
          {content}
          title={viewerTitle}
          modified={contentModified}
          loading={loadingContent}
        />
      </div>
    </div>
  {/if}
</div>
