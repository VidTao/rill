<script lang="ts">
  import CanvasDashboardEmbed from "@rilldata/web-common/features/canvas/CanvasDashboardEmbed.svelte";
  import CanvasProvider from "@rilldata/web-common/features/canvas/CanvasProvider.svelte";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import EmbedHeader from "$lib/bratrax/EmbedHeader.svelte";
  import type { PageData } from "./$types";

  // Single-dashboard view for the Shopify admin iframe. Same two components
  // the full /canvas/[name] page uses — CanvasDashboardEmbed already renders
  // just the dashboard and its filter bar; everything else on that page comes
  // from the root layout, which suppresses itself on /embed/* (onEmbedPage).
  //
  // EmbedHeader stands in for the suppressed ApplicationHeader: logo, a line
  // explaining that the rest of the product lives in the full app, and the
  // "Open in Bratrax" escape hatch.
  //
  // Deliberately dropped versus /canvas/[name]: WelcomeBanner, the
  // "Edit dashboard" link, and the AI chat gate. None of them make sense in a
  // ~1150px-wide frame inside someone else's admin.
  //
  // Parametrised on [name] rather than hardcoding campaign_deep_dive so other
  // canvases can be embedded later without another route.

  export let data: PageData;

  $: ({ instanceId } = $runtime);
  $: ({ canvasName } = data);
</script>

{#key instanceId}
  <div class="flex h-full flex-col overflow-hidden">
    <EmbedHeader />
    <div class="canvas-area">
      <CanvasProvider {canvasName} {instanceId}>
        <CanvasDashboardEmbed {canvasName} />
      </CanvasProvider>
    </div>
  </div>
{/key}

<style lang="postcss">
  .canvas-area {
    @apply flex-1 overflow-hidden;
    position: relative;
  }
</style>
