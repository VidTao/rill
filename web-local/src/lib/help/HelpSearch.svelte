<script lang="ts">
  import { goto } from "$app/navigation";
  import Search from "@rilldata/web-common/components/search/Search.svelte";
  import {
    HELP_SNIPPET_MARK_END,
    HELP_SNIPPET_MARK_START,
    searchHelp,
    type HelpPage,
    type HelpSearchResult,
  } from "$lib/help";

  export let pages: HelpPage[];
  export let variant: "sidebar" | "landing" = "sidebar";
  export let placeholder = "Search help…";
  export let query = "";

  let highlighted = 0;

  $: results = query.trim() ? searchHelp(query, pages) : [];
  $: if (results.length > 0 && highlighted >= results.length) highlighted = 0;

  function splitSnippet(
    snippet: string,
  ): Array<{ text: string; highlight: boolean }> {
    const parts: Array<{ text: string; highlight: boolean }> = [];
    let i = 0;
    while (i < snippet.length) {
      const start = snippet.indexOf(HELP_SNIPPET_MARK_START, i);
      if (start === -1) {
        if (i < snippet.length) {
          parts.push({ text: snippet.slice(i), highlight: false });
        }
        break;
      }
      if (start > i) {
        parts.push({ text: snippet.slice(i, start), highlight: false });
      }
      const end = snippet.indexOf(HELP_SNIPPET_MARK_END, start + 1);
      if (end === -1) {
        parts.push({ text: snippet.slice(start + 1), highlight: true });
        break;
      }
      parts.push({ text: snippet.slice(start + 1, end), highlight: true });
      i = end + 1;
    }
    return parts;
  }

  function audiencePill(result: HelpSearchResult): string | null {
    if (result.page.audience === "admin") return "Admin";
    if (result.page.audience === "shared") return "Reference";
    return null;
  }

  function open(result: HelpSearchResult) {
    query = "";
    highlighted = 0;
    void goto(`/help/${result.page.slug}`);
  }

  function handleSubmit() {
    if (results.length === 0) return;
    open(results[highlighted] ?? results[0]);
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && query) {
      event.preventDefault();
      query = "";
      highlighted = 0;
      return;
    }
    if (results.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      highlighted = (highlighted + 1) % results.length;
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      highlighted = (highlighted - 1 + results.length) % results.length;
    }
  }
</script>

<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
<div class="help-search" class:landing={variant === "landing"} on:keydown={handleKeydown} role="search">
  <Search
    bind:value={query}
    {placeholder}
    label="Search help"
    autofocus={false}
    retainValueOnMount={true}
    large={variant === "landing"}
    onSubmit={handleSubmit}
  />

  {#if query.trim()}
    <div class="results-panel" class:landing={variant === "landing"}>
      {#if results.length === 0}
        <div class="empty">No pages match "{query.trim()}".</div>
      {:else}
        <ul class="results">
          {#each results as result, i (result.page.slug)}
            {@const pill = audiencePill(result)}
            <li>
              <button
                type="button"
                class="result"
                class:active={i === highlighted}
                on:mouseenter={() => (highlighted = i)}
                on:click={() => open(result)}
              >
                <div class="result-head">
                  <span class="result-title">{result.page.title}</span>
                  {#if pill}
                    <span class="result-pill">{pill}</span>
                  {/if}
                </div>
                <div class="result-snippet">
                  {#each splitSnippet(result.snippet) as part}
                    {#if part.highlight}<mark>{part.text}</mark>{:else}{part.text}{/if}
                  {/each}
                </div>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>

<style lang="postcss">
  .help-search {
    @apply flex flex-col gap-2 w-full;
  }
  .help-search.landing {
    @apply gap-3;
  }

  .results-panel {
    @apply flex flex-col rounded-md border bg-bratrax-surface overflow-hidden;
    border-color: var(--color-bratrax-border, #e5e7eb);
  }
  .results-panel.landing {
    @apply rounded-lg;
  }

  .results {
    list-style: none;
    padding: 0;
    margin: 0;
    max-height: 480px;
    overflow-y: auto;
  }

  .result {
    @apply w-full text-left px-3 py-2 flex flex-col gap-1;
    background: transparent;
    border: 0;
    border-left: 2px solid transparent;
    cursor: pointer;
    color: inherit;
  }
  .results li + li .result {
    border-top: 1px solid var(--color-bratrax-border, #e5e7eb);
  }
  .result:hover,
  .result.active {
    background-color: var(--color-acid-dim);
    border-left-color: var(--color-acid);
  }

  .result-head {
    @apply flex items-center gap-2;
  }
  .result-title {
    font-size: 0.875rem;
    font-weight: 600;
    line-height: 1.3;
  }
  .result-pill {
    font-family: "Space Mono", monospace;
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--color-text-muted);
    background: var(--color-acid-dim);
    padding: 1px 6px;
    border-radius: 3px;
  }

  .result-snippet {
    font-size: 0.8125rem;
    line-height: 1.45;
    color: var(--color-text-muted);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .result-snippet :global(mark) {
    background-color: var(--color-acid-mid);
    color: inherit;
    padding: 0 1px;
    border-radius: 2px;
  }

  .empty {
    @apply px-3 py-3 text-sm;
    color: var(--color-text-muted);
  }
</style>
