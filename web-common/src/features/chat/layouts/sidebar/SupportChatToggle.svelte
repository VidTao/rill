<script lang="ts">
  import HeaderToggleButton from "../../../../layout/HeaderToggleButton.svelte";
  import { supportChatActions, supportChatOpen } from "./support-chat-store";
</script>

<!-- "?" header button — opens the Bratrax support-bot sidebar. Sits in the
     icon cluster next to the AI toggle and the theme switch, and shares their
     styling through HeaderToggleButton. -->
<HeaderToggleButton
  active={$supportChatOpen}
  onClick={supportChatActions.toggle}
  title="Ask support"
  aria-label="Ask support"
  aria-pressed={$supportChatOpen}
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
</HeaderToggleButton>
