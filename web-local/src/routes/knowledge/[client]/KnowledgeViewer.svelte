<script lang="ts">
  import DOMPurify from "dompurify";
  import { marked } from "marked";

  export let content: string;
  export let title: string;
  export let modified: string | null = null;
  export let loading = false;

  function formatDate(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }
</script>

<div class="flex h-full flex-col overflow-hidden">
  <!-- Header -->
  <div
    class="flex items-center justify-between border-b px-6 py-3"
    style="border-color: var(--bratrax-border); background: var(--bratrax-surface);"
  >
    <h2
      class="font-mono text-sm font-bold uppercase tracking-wider"
      style="color: var(--bratrax-text-headline);"
    >
      {title}
    </h2>
    {#if modified}
      <span
        class="font-mono text-sm"
        style="color: var(--bratrax-text-muted);"
      >
        {formatDate(modified)}
      </span>
    {/if}
  </div>

  <!-- Content -->
  <div class="flex-1 overflow-y-auto px-6 py-6">
    {#if loading}
      <div class="space-y-3">
        {#each Array(8) as _}
          <div
            class="h-4 animate-pulse"
            style="background: var(--bratrax-hover); width: {60 + Math.random() * 40}%;"
          />
        {/each}
      </div>
    {:else if !content}
      <div class="flex h-full items-center justify-center">
        <p
          class="text-center font-mono text-sm"
          style="color: var(--bratrax-text-muted);"
        >
          No content yet — knowledge builds as your analyst works.
        </p>
      </div>
    {:else}
      <div class="bratrax-markdown">
        {#await marked(content) then html}
          {@html DOMPurify.sanitize(html)}
        {/await}
      </div>
    {/if}
  </div>
</div>

<style>
  :global(.bratrax-markdown) {
    color: var(--bratrax-text-body);
    line-height: 1.7;
  }

  :global(.bratrax-markdown h1) {
    color: var(--bratrax-text-headline);
    font-size: 1.25rem;
    font-weight: 700;
    margin-top: 1.5rem;
    margin-bottom: 0.75rem;
  }

  :global(.bratrax-markdown h1:first-child) {
    margin-top: 0;
  }

  :global(.bratrax-markdown h2) {
    color: var(--bratrax-text-headline);
    font-size: 1.1rem;
    font-weight: 600;
    margin-top: 1.25rem;
    margin-bottom: 0.5rem;
  }

  :global(.bratrax-markdown h2:first-child) {
    margin-top: 0;
  }

  :global(.bratrax-markdown h3) {
    color: var(--bratrax-text-headline);
    font-size: 1rem;
    font-weight: 600;
    margin-top: 1rem;
    margin-bottom: 0.4rem;
  }

  :global(.bratrax-markdown h4),
  :global(.bratrax-markdown h5),
  :global(.bratrax-markdown h6) {
    color: var(--bratrax-text-headline);
    font-size: 0.875rem;
    font-weight: 600;
    margin-top: 0.75rem;
    margin-bottom: 0.25rem;
  }

  :global(.bratrax-markdown p) {
    font-size: 0.875rem;
    margin-bottom: 0.75rem;
  }

  :global(.bratrax-markdown p:last-child) {
    margin-bottom: 0;
  }

  :global(.bratrax-markdown ul) {
    font-size: 0.875rem;
    list-style-type: disc;
    padding-left: 1.5rem;
    margin-bottom: 0.75rem;
  }

  :global(.bratrax-markdown ol) {
    font-size: 0.875rem;
    list-style-type: decimal;
    padding-left: 1.5rem;
    margin-bottom: 0.75rem;
  }

  :global(.bratrax-markdown li) {
    margin-bottom: 0.3rem;
  }

  :global(.bratrax-markdown blockquote) {
    border-left: 3px solid var(--bratrax-acid);
    padding-left: 1rem;
    padding-top: 0.25rem;
    padding-bottom: 0.25rem;
    margin-bottom: 0.75rem;
    color: var(--bratrax-text-muted);
    font-style: italic;
    font-size: 0.875rem;
  }

  :global(.bratrax-markdown code) {
    font-family: var(--monospace);
    font-size: 0.8125rem;
    background: var(--bratrax-surface);
    border: 1px solid var(--bratrax-border);
    padding: 0.125rem 0.375rem;
  }

  :global(.bratrax-markdown pre) {
    background: var(--bratrax-surface);
    border: 1px solid var(--bratrax-border);
    padding: 1rem;
    margin-bottom: 0.75rem;
    overflow-x: auto;
    font-size: 0.8125rem;
  }

  :global(.bratrax-markdown pre code) {
    background: transparent;
    border: none;
    padding: 0;
  }

  :global(.bratrax-markdown a) {
    color: var(--bratrax-acid);
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  :global(.bratrax-markdown a:hover) {
    opacity: 0.8;
  }

  :global(.bratrax-markdown strong) {
    font-weight: 600;
    color: var(--bratrax-text-headline);
  }

  :global(.bratrax-markdown em) {
    font-style: italic;
  }

  :global(.bratrax-markdown hr) {
    border: none;
    border-top: 1px solid var(--bratrax-border);
    margin: 1.5rem 0;
  }

  :global(.bratrax-markdown table) {
    width: 100%;
    font-size: 0.8125rem;
    border-collapse: collapse;
    margin-bottom: 0.75rem;
  }

  :global(.bratrax-markdown th) {
    background: var(--bratrax-hover);
    border: 1px solid var(--bratrax-border);
    font-weight: 600;
    padding: 0.4rem 0.75rem;
    text-align: left;
    color: var(--bratrax-text-headline);
  }

  :global(.bratrax-markdown td) {
    border: 1px solid var(--bratrax-border);
    padding: 0.4rem 0.75rem;
  }

  :global(.bratrax-markdown img) {
    max-width: 100%;
    height: auto;
  }
</style>
