<script lang="ts">
  import HeaderToggleButton from "../../../../layout/HeaderToggleButton.svelte";
  import { chatOpen, sidebarActions } from "./sidebar-store";
</script>

<!-- "AI" header button — opens the AI chat sidebar. Rendered from three places
     (the header's own cluster on the developer and file routes, and the Explore
     and Canvas preview CTAs on the two dashboard routes), all of which land in
     the same header cluster, so it shares HeaderToggleButton with the theme
     switch and the support toggle.

     Distinct from ChatToggle, which wraps the same store in the shared Button.
     ChatToggle is what web-admin's nav uses; this exists so the Bratrax header
     cluster reads as one control group rather than three unrelated controls. -->
<HeaderToggleButton
  active={$chatOpen}
  onClick={sidebarActions.toggleChat}
  title="Ask AI"
  aria-label="Ask AI"
  aria-pressed={$chatOpen}
>
  AI
</HeaderToggleButton>
