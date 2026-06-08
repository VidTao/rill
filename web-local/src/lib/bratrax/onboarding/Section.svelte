<script lang="ts">
  import { ChevronDown } from "lucide-svelte";
  import ChecklistItem from "./ChecklistItem.svelte";
  import type { ChecklistSection } from "./checklist";

  export let section: ChecklistSection;
  export let index: number; // 1-based section number for the label
  export let note: string | null = null;
  export let numbered = true;

  function isResolved(status: string): boolean {
    return status === "done" || status === "skipped";
  }

  $: done = section.items.filter((i) => isResolved(i.status)).length;
  $: total = section.items.length;
  $: complete = total > 0 && done === total;

  // Completed sections collapse by default; incomplete sections open.
  let userToggled = false;
  let open = true;
  $: if (!userToggled) open = !complete;

  function toggle() {
    userToggled = true;
    open = !open;
  }
</script>

<section class="section">
  <button class="header" on:click={toggle} aria-expanded={open}>
    <ChevronDown size={16} class="chevron {open ? 'open' : ''}" />
    <span class="label">
      {#if numbered}Section {index} — {/if}{section.title}
    </span>
    <span class="count">({done}/{total})</span>
  </button>

  {#if open}
    {#if note}
      <div class="note">{note}</div>
    {/if}
    <div class="items">
      {#each section.items as item (item.key)}
        <ChecklistItem {item} on:changed />
      {/each}
    </div>
  {/if}
</section>

<style>
  .section {
    border-top: 1px solid var(--color-border);
    padding: 8px 0;
  }
  .header {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 8px 4px;
    text-align: left;
  }
  .label {
    font-family: "Space Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-acid-text);
    opacity: 0.85;
  }
  .count {
    font-family: "Space Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    color: var(--color-text-muted);
    margin-left: auto;
  }
  :global(.chevron) {
    color: var(--color-text-muted);
    transition: transform 150ms ease;
  }
  :global(.chevron.open) {
    transform: rotate(0deg);
  }
  :global(.chevron:not(.open)) {
    transform: rotate(-90deg);
  }
  .note {
    font-family: "Outfit", sans-serif;
    font-size: 13px;
    color: var(--color-text-secondary);
    padding: 0 4px 6px 30px;
  }
  .items {
    padding-left: 26px;
  }
</style>
