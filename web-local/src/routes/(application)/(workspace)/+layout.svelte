<script lang="ts">
  import Navigation from "@rilldata/web-common/layout/navigation/Navigation.svelte";
  import ViewerNavigation from "$lib/bratrax/ViewerNavigation.svelte";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import OrderDrilldownProvider from "$lib/bratrax/order-attribution/OrderDrilldownProvider.svelte";
  import type { LayoutData } from "../$types";

  export let data: LayoutData;

  // Bratrax 3-tier role enforcement:
  //   viewer      → ViewerNavigation (dashboards-only list)
  //   admin       → Navigation with filtered file tree (no rill.yaml/.env/connectors)
  //   super_admin → Navigation with full file tree (current behavior)
  $: role = $bratraxUser?.role ?? null;
  $: isViewer = role === "viewer";
  $: isAdmin = role === "admin";

  // Top-level paths an `admin` is allowed to see in the file tree. Folders
  // end with `/`; bare filenames (e.g. claude_system_prompt.txt) are exact-match.
  const ADMIN_ALLOWED_PATHS = [
    "/dashboards/",
    "/knowledge/",
    "/metrics/",
    "/models/",
    "/sources/",
    "/claude_system_prompt.txt",
  ];

  $: allowedPaths = isAdmin ? ADMIN_ALLOWED_PATHS : null;
</script>

<OrderDrilldownProvider>
  <div class="flex size-full overflow-hidden">
    {#if data.initialized && !isViewer}
      <Navigation allowedTopLevelPaths={allowedPaths} />
    {:else if isViewer}
      <ViewerNavigation />
    {/if}

    <section class="size-full overflow-hidden">
      <slot />
    </section>
  </div>
</OrderDrilldownProvider>
