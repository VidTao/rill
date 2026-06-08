<script lang="ts">
  import { goto } from "$app/navigation";

  // Generic empty-state CTA that links back to a specific checklist item.
  // The URL fragment (#<item_key>) scrolls the checklist to that item.
  export let relatedChecklistItem: string;
  export let message = "This panel needs data that isn't flowing yet.";
  export let cta = "Finish setting up tracking →";

  function go() {
    void goto(`/onboarding#${relatedChecklistItem}`);
  }
</script>

<div class="empty-link">
  <span class="msg">{message}</span>
  <button class="cta" on:click={go}>{cta}</button>
</div>

<style>
  .empty-link {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    text-align: center;
  }
  .msg {
    font-family: "Outfit", sans-serif;
    font-size: 14px;
    color: var(--color-text-secondary);
  }
  .cta {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    color: var(--color-acid-text);
    background: transparent;
    border: none;
    cursor: pointer;
  }
</style>
