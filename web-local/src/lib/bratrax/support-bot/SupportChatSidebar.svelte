<script lang="ts">
  import { chatOpen } from "@rilldata/web-common/features/chat/layouts/sidebar/sidebar-store";
  import {
    supportChatActions,
    supportChatOpen,
  } from "@rilldata/web-common/features/chat/layouts/sidebar/support-chat-store";
  import SupportBotChat from "./SupportBotChat.svelte";

  /**
   * Float over the page instead of taking a column from it. Set on /embed/*,
   * where the whole viewport is Shopify's ~1150px content column and a 420px
   * in-flow panel would leave the canvas unusable. Anchors to the `relative`
   * flex row in web-local/src/routes/+layout.svelte.
   */
  export let overlay = false;

  // Only one right-hand sidebar at a time: the AI chat opening closes this
  // panel (the reverse lives in supportChatActions). The panel stays mounted
  // while hidden so the conversation survives open/close toggles.
  $: if ($chatOpen && $supportChatOpen) supportChatActions.close();
</script>

<aside
  class="support-sidebar"
  class:open={$supportChatOpen}
  class:overlay
  aria-label="Ask support"
>
  <div class="panel-header">
    <span class="panel-title">Ask support</span>
    <button
      type="button"
      class="panel-close"
      on:click={supportChatActions.close}
      aria-label="Close support chat"
    >
      <svg
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        aria-hidden="true"
      >
        <line x1="18" y1="6" x2="6" y2="18" />
        <line x1="6" y1="6" x2="18" y2="18" />
      </svg>
    </button>
  </div>
  <div class="panel-body">
    <SupportBotChat />
  </div>
</aside>

<style lang="postcss">
  /* In-flow flex sibling of the page content (see web-local/src/routes/
     +layout.svelte), so opening the panel narrows the page instead of
     covering it — matching how the per-page AI sidebar behaves. Mounted once
     in the root layout, so it works on every route without touching page
     layouts. `display: none` while closed keeps it out of the flex row and
     preserves the conversation across toggles. */
  .support-sidebar {
    flex: none;
    width: 420px;
    max-width: 100%;
    height: 100%;
    min-height: 0;
    /* Above any page content that overflows its own (now narrower) column. */
    position: relative;
    z-index: 20;
    display: none;
    flex-direction: column;
    background: var(--color-bg, var(--surface));
    border-left: 1px solid var(--color-border-strong, var(--border));
  }
  .support-sidebar.open {
    display: flex;
  }

  /* Overlay variant (embed). Out of flow, so opening it never reflows the
     canvas — which is the point: a dashboard that re-lays-out every time you
     ask a question is worse than no help button. The shadow is what sells it
     as floating rather than as a column that failed to push. */
  .support-sidebar.overlay {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    height: auto;
    box-shadow: -8px 0 24px rgb(0 0 0 / 28%);
  }

  /* Below the widest phone the app is usable on, a 420px column would leave
     nothing for the page; take the full width instead. */
  @media (max-width: 720px) {
    .support-sidebar {
      width: 100%;
    }
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 16px;
    border-bottom: 1px solid var(--color-border, var(--border));
    flex: none;
  }

  .panel-title {
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-acid-text);
  }

  .panel-close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: transparent;
    border: none;
    color: var(--color-text-muted);
    cursor: pointer;
  }
  .panel-close:hover {
    color: var(--color-text);
  }

  .panel-body {
    flex: 1;
    min-height: 0;
  }
</style>
