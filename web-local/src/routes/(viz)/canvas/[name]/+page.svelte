<script lang="ts">
  import CanvasDashboardEmbed from "@rilldata/web-common/features/canvas/CanvasDashboardEmbed.svelte";
  import CanvasProvider from "@rilldata/web-common/features/canvas/CanvasProvider.svelte";
  import DashboardChat from "@rilldata/web-common/features/chat/DashboardChat.svelte";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors.ts";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import WelcomeBanner from "$lib/bratrax/onboarding/WelcomeBanner.svelte";
  import { autoMarkOnce } from "$lib/bratrax/onboarding/checklist";
  import type { PageData } from "./$types";

  export let data: PageData;

  $: ({ instanceId } = $runtime);
  $: ({ canvasName } = data);

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
      <div class="flex-1 overflow-hidden">
        <CanvasProvider {canvasName} {instanceId} showBanner>
          <CanvasDashboardEmbed {canvasName} />
        </CanvasProvider>
      </div>
      <DashboardChat kind={ResourceKind.Canvas} />
    </div>
  </div>
{/key}
