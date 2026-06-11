<script lang="ts">
  import CanvasDashboardEmbed from "@rilldata/web-common/features/canvas/CanvasDashboardEmbed.svelte";
  import CanvasProvider from "@rilldata/web-common/features/canvas/CanvasProvider.svelte";
  import DashboardChat from "@rilldata/web-common/features/chat/DashboardChat.svelte";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors.ts";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import BratraxChatGate from "$lib/bratrax/BratraxChatGate.svelte";
  import WelcomeBanner from "$lib/bratrax/onboarding/WelcomeBanner.svelte";
  import { autoMarkOnce } from "$lib/bratrax/onboarding/checklist";
  import type { PageData } from "./$types";

  export let data: PageData;

  $: ({ instanceId } = $runtime);
  $: ({ canvasName } = data);
  $: isAdminOrSuper =
    $bratraxUser?.role === "admin" || $bratraxUser?.role === "super_admin";

  // Onboarding checklist (Section 4): mark "Open your Campaign Deep Dive
  // dashboard" the first time this canvas loads. Best-effort, fires once.
  $: if (canvasName === "campaign_deep_dive") {
    autoMarkOnce("visited_campaign_deep_dive");
  }
</script>

{#key instanceId}
  <div class="flex h-full flex-col overflow-hidden">
    <WelcomeBanner />
    <div class="flex flex-1 overflow-hidden">
      <!-- Position-context wrapper for the absolute Edit overlay below.
           `relative` + `overflow-hidden` keeps the overlay clipped to the
           canvas-area bounds; CanvasProvider/Embed render inside as usual. -->
      <div class="canvas-area">
        <CanvasProvider {canvasName} {instanceId} showBanner>
          <CanvasDashboardEmbed {canvasName} />
        </CanvasProvider>
        {#if isAdminOrSuper}
          <!-- Edit button overlay: sits in the top-right of the canvas area,
               above the filter bar's right edge. Inline CSS (not Tailwind +
               not Svelte-scoped) so nothing can override the absolute
               positioning regardless of how CanvasFilters lays itself out. -->
          <a
            href={`/files/dashboards/${canvasName}.yaml`}
            class="canvas-edit-link"
            style="position: absolute; top: 14px; right: 14px; z-index: 60;"
          >
            Edit
          </a>
        {/if}
      </div>
      <BratraxChatGate>
        <DashboardChat kind={ResourceKind.Canvas} />
      </BratraxChatGate>
    </div>
  </div>
{/key}

<style lang="postcss">
  .canvas-area {
    @apply flex-1 overflow-hidden;
    position: relative;
  }

  :global(.canvas-edit-link) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 32px;
    padding: 0 12px;
    background: var(--color-surface, transparent);
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 2px;
    text-decoration: none;
    cursor: pointer;
    transition:
      background-color 120ms ease,
      color 120ms ease,
      border-color 120ms ease;
  }
  :global(.canvas-edit-link:hover) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
    border-color: var(--color-acid, #d4ff00);
  }
  :global(.canvas-edit-link:focus-visible) {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
</style>
