<script lang="ts">
  import Navigation from "@rilldata/web-common/layout/navigation/Navigation.svelte";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import type { LayoutData } from "../$types";

  export let data: LayoutData;

  // Lite viewers don't see the developer navigation sidebar
  $: isViewer = $bratraxUser?.role === "viewer";
</script>

<div class="flex size-full overflow-hidden">
  {#if data.initialized && !isViewer}
    <Navigation />
  {/if}

  <section class="size-full overflow-hidden">
    <slot />
  </section>
</div>
