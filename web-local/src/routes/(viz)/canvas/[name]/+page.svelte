<script lang="ts">
  import CanvasDashboardEmbed from "@rilldata/web-common/features/canvas/CanvasDashboardEmbed.svelte";
  import CanvasProvider from "@rilldata/web-common/features/canvas/CanvasProvider.svelte";
  import DashboardChat from "@rilldata/web-common/features/chat/DashboardChat.svelte";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors.ts";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import BratraxChatGate from "$lib/bratrax/BratraxChatGate.svelte";
  import DataFreshnessBadge from "$lib/bratrax/freshness/DataFreshnessBadge.svelte";
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
      <div class="canvas-area">
        <CanvasProvider {canvasName} {instanceId} showBanner>
          <CanvasDashboardEmbed {canvasName}>
            <!-- Freshness badge + Edit-dashboard button — rendered into
                 CanvasDashboardWrapper's filter-right slot, which sits INSIDE
                 the same max-width relative wrapper as CanvasFilters. The
                 wrapper positions it at that wrapper's top right edge and
                 reserves its width in the filter bar's first row, so the
                 pills wrap instead of overlapping it at any width (e.g. with
                 the chat or support sidebar open). svelte:fragment is required so
                 the {#if} lives INSIDE the slot (Svelte 4 requires slot="..."
                 on a direct child of the component). -->
            <svelte:fragment slot="filter-right">
              <div class="canvas-filter-right">
                <!-- Every role: "is this number current?" is a viewer's
                     question as much as an admin's. -->
                <DataFreshnessBadge />
                {#if isAdminOrSuper}
                  <a
                    href={`/files/dashboards/${canvasName}.yaml`}
                    class="canvas-edit-link"
                  >
                    Edit dashboard
                  </a>
                {/if}
              </div>
            </svelte:fragment>
          </CanvasDashboardEmbed>
        </CanvasProvider>
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

  /* Positioned (top right of the filter bar) and measured by
     CanvasDashboardWrapper, which keeps its width free in the filter bar's
     first row so the pills wrap instead of sliding underneath. */
  :global(.canvas-filter-right) {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  :global(.canvas-edit-link) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 26px;
    padding: 0 10px;
    background: var(--color-surface, transparent);
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    /* Inherit Inter from parent so the button reads as part of the filter
       row controls (matches the weight of "Comparing", "as of latest day",
       etc.) — not a separate uppercase Space-Mono pill. */
    font-size: 13px;
    font-weight: 500;
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
