<script lang="ts">
  import { chatOpen } from "@rilldata/web-common/features/chat/layouts/sidebar/sidebar-store";
  import {
    supportChatActions,
    supportChatOpen,
  } from "@rilldata/web-common/features/chat/layouts/sidebar/support-chat-store";
  import SupportBotChat from "./SupportBotChat.svelte";

  // Only one right-hand sidebar at a time: the AI chat opening closes this
  // panel (the reverse lives in supportChatActions). The panel stays mounted
  // while hidden so the conversation survives open/close toggles.
  $: if ($chatOpen && $supportChatOpen) supportChatActions.close();
</script>

<aside
  class="support-sidebar"
  class:open={$supportChatOpen}
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
  /* Fixed panel below the 3.25rem application header. Overlays page content
     rather than squeezing it (unlike the per-page AI sidebar) so it works on
     every route without touching page layouts. */
  .support-sidebar {
    position: fixed;
    top: 3.25rem;
    right: 0;
    bottom: 0;
    width: 420px;
    max-width: 100vw;
    z-index: 40;
    display: none;
    flex-direction: column;
    background: var(--color-bg, var(--surface));
    border-left: 1px solid var(--color-border-strong, var(--border));
    box-shadow: -8px 0 24px rgba(0, 0, 0, 0.25);
  }
  .support-sidebar.open {
    display: flex;
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
