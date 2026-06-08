<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { goto } from "$app/navigation";
  import { MoreVertical } from "lucide-svelte";
  import * as DropdownMenu from "@rilldata/web-common/components/dropdown-menu";
  import {
    markItem,
    skipItem,
    unskipItem,
    autoMark,
    type ChecklistItem,
  } from "./checklist";

  export let item: ChecklistItem;

  const dispatch = createEventDispatcher<{ changed: void }>();
  let busy = false;

  const GLYPH: Record<string, string> = {
    done: "✓",
    in_progress: "◐",
    todo: "○",
    skipped: "⊘",
    error: "✕",
  };

  $: glyph = GLYPH[item.status] ?? "○";
  $: showVisibleSkip =
    item.skippable_prominence === "prominent" &&
    item.status !== "skipped" &&
    item.user_override !== "done";

  async function run(fn: () => Promise<unknown>) {
    if (busy) return;
    busy = true;
    try {
      await fn();
      dispatch("changed");
    } catch (e) {
      // Surface nothing intrusive; the next refetch reflects reality.
      console.warn("checklist item action failed", item.key, e);
    } finally {
      busy = false;
    }
  }

  function primaryAction() {
    // Items with an explicit "first-use" semantics rather than a destination.
    if (item.key === "verify_tracking") {
      void run(() => markItem(item.key));
      return;
    }
    if (item.key === "asked_ai_a_question") {
      // The AI chat panel is baked into dashboard views; sending a message
      // there auto-marks this item.
      void autoMark; // referenced for clarity; auto-mark fires from ChatInput
      void goto("/canvas/campaign_deep_dive");
      return;
    }
    if (item.action_url) {
      void goto(item.action_url);
    }
  }
</script>

<div class="row" id={item.key} class:skipped={item.status === "skipped"}>
  <span class="glyph status-{item.status}" class:pulse={item.status === "in_progress"}>
    {glyph}
  </span>

  <div class="body">
    <div class="label">{item.label}</div>
    {#if item.description}
      <div class="desc">{item.description}</div>
    {/if}
  </div>

  <div class="actions">
    {#if item.user_override === "skipped"}
      <span class="meta">Skipped</span>
      <button class="link" disabled={busy} on:click={() => run(() => unskipItem(item.key))}>
        Un-skip
      </button>
    {:else if item.user_override === "done"}
      <span class="meta">Marked done</span>
      <button class="link" disabled={busy} on:click={() => run(() => unskipItem(item.key))}>
        Un-mark
      </button>
    {:else if item.status !== "done"}
      {#if item.action_label}
        <button class="btn-primary" disabled={busy} on:click={primaryAction}>
          {item.action_label}
        </button>
      {/if}
      {#if showVisibleSkip}
        <button class="btn-outline" disabled={busy} on:click={() => run(() => skipItem(item.key))}>
          Skip
        </button>
      {/if}
    {/if}

    <DropdownMenu.Root>
      <DropdownMenu.Trigger class="more" aria-label="More actions">
        <MoreVertical size={16} />
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="end">
        {#if item.status !== "done"}
          <DropdownMenu.Item on:click={() => run(() => markItem(item.key))}>
            Mark as done
          </DropdownMenu.Item>
        {/if}
        {#if item.status !== "skipped"}
          <DropdownMenu.Item on:click={() => run(() => skipItem(item.key))}>
            Skip
          </DropdownMenu.Item>
        {/if}
        {#if item.user_override !== null}
          <DropdownMenu.Item on:click={() => run(() => unskipItem(item.key))}>
            Reset to auto
          </DropdownMenu.Item>
        {/if}
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  </div>
</div>

<style>
  .row {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 10px 4px;
    scroll-margin-top: 80px;
  }
  .glyph {
    font-family: "Space Mono", monospace;
    font-size: 16px;
    line-height: 1.5;
    width: 18px;
    flex: 0 0 auto;
    text-align: center;
  }
  .status-done {
    color: var(--color-acid-text);
  }
  .status-in_progress {
    color: var(--color-acid-text);
  }
  .status-todo {
    color: var(--color-text-muted);
  }
  .status-skipped {
    color: var(--color-text-muted);
  }
  .status-error {
    color: var(--color-red);
  }
  .pulse {
    animation: pulse 1.2s ease-in-out infinite;
  }
  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.4;
    }
  }
  .body {
    flex: 1 1 auto;
    min-width: 0;
  }
  .label {
    font-family: "Outfit", sans-serif;
    font-weight: 300;
    font-size: 16px;
    color: var(--color-text);
  }
  .skipped .label {
    text-decoration: line-through;
    color: var(--color-text-muted);
  }
  .desc {
    font-family: "Outfit", sans-serif;
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-top: 2px;
  }
  .actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 0 0 auto;
  }
  .meta {
    font-family: "Outfit", sans-serif;
    font-size: 13px;
    color: var(--color-text-muted);
  }
  .btn-primary,
  .btn-outline,
  .link {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    cursor: pointer;
    border-radius: 0;
  }
  .btn-primary {
    background: var(--color-acid);
    color: #0a0a0a;
    border: 1px solid var(--color-acid);
    padding: 6px 12px;
  }
  .btn-primary:hover:not(:disabled) {
    opacity: 0.88;
  }
  .btn-outline {
    background: transparent;
    color: var(--color-text);
    border: 1px solid var(--color-border-strong);
    padding: 6px 12px;
  }
  .btn-outline:hover:not(:disabled) {
    border-color: var(--color-acid);
  }
  .link {
    background: transparent;
    border: none;
    color: var(--color-acid-text);
    text-transform: none;
    padding: 0;
    text-decoration: underline;
  }
  button:disabled {
    opacity: 0.5;
    cursor: default;
  }
  :global(.more) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 4px;
  }
  :global(.more:hover) {
    color: var(--color-text);
  }
</style>
