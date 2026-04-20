<script lang="ts">
  import Navigation from "@rilldata/web-common/layout/navigation/Navigation.svelte";
  import ViewerNavigation from "$lib/bratrax/ViewerNavigation.svelte";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import type { LayoutData } from "../$types";

  export let data: LayoutData;

  // Lite viewers see a simplified dashboard-only navigation
  $: isViewer = $bratraxUser?.role === "viewer";
</script>

<div class="flex size-full overflow-hidden">
  {#if data.initialized && !isViewer}
    <Navigation />
  {:else if isViewer}
    <ViewerNavigation />
  {/if}

  <section class="size-full overflow-hidden">
    <slot />
  </section>
</div>
