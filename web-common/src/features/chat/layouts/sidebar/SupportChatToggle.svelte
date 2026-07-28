<script lang="ts">
  import { supportChatActions, supportChatOpen } from "./support-chat-store";
</script>

<!-- "?" header button — opens the Bratrax support-bot sidebar. Sits in the
     icon cluster next to the AI ChatToggle; styled like the theme toggle. -->
<button
  type="button"
  on:click={supportChatActions.toggle}
  class="support-chat-toggle"
  class:active={$supportChatOpen}
  title="Ask support"
  aria-label="Ask support"
>
  <svg
    width="16"
    height="16"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="10" />
    <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
    <line x1="12" y1="17" x2="12.01" y2="17" />
  </svg>
</button>

<style lang="postcss">
  .support-chat-toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    cursor: pointer;
    transition:
      background-color 120ms ease,
      color 120ms ease,
      border-color 120ms ease;
  }
  .support-chat-toggle:hover {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
  }
  .support-chat-toggle.active {
    border-color: var(--color-acid, #d4ff00);
    color: var(--color-acid-text, var(--color-acid, #d4ff00));
  }
  .support-chat-toggle:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
</style>
