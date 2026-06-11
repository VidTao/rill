<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { GeneratingMessage } from "@rilldata/web-common/components/generating-message";
  import { generatingCanvas } from "@rilldata/web-common/features/canvas/ai-generation/generateCanvas";
  import BratraxChatGate from "$lib/bratrax/BratraxChatGate.svelte";
  import { generatingSampleData } from "@rilldata/web-common/features/sample-data/generate-sample-data.ts";
  import ProjectCards from "@rilldata/web-common/features/welcome/ProjectCards.svelte";
  import TitleContent from "@rilldata/web-common/features/welcome/TitleContent.svelte";
  import { createRuntimeServiceListResources } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { fly } from "svelte/transition";
  import { get } from "svelte/store";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { isRillDemoCanvas } from "$lib/bratrax/dashboardPrefs";
  import type { LayoutData } from "../$types";

  export let data: LayoutData;

  // Bratrax 3-tier role enforcement on /developer landing:
  //   super_admin → current behavior (OnboardingWorkspace / welcome)
  //   admin + viewer → redirect to first dashboard; "no dashboards yet" placeholder if zero
  let redirectChecked = false;
  let noDashboardsForRole = false;

  $: instanceId = get(runtime).instanceId;

  // Lazy canvas list — only triggered when we need to find a target dashboard
  // (i.e. for admin/viewer roles after onMount).
  $: canvasQuery = createRuntimeServiceListResources(instanceId, {
    kind: "rill.runtime.v1.Canvas",
  });

  function firstCanvasName(): string | null {
    const resources = $canvasQuery.data?.resources ?? [];
    for (const r of resources) {
      const name = r.meta?.name?.name;
      // Skip Rill bundled-example canvases (see RILL_DEMO_CANVAS_NAMES).
      if (name && !isRillDemoCanvas(name)) return name;
    }
    return null;
  }

  onMount(async () => {
    if (!data.initialized) {
      redirectChecked = true;
      return;
    }
    // Wait one tick for the canvas query to settle, then redirect.
    // Every role (super_admin / admin / viewer) lands on the first canvas in
    // preview mode — same surface they'd reach after onboarding. Edit mode
    // is reachable from the "Edit" button on the preview header.
    let attempts = 0;
    while (attempts < 20 && $canvasQuery.isLoading) {
      await new Promise((r) => setTimeout(r, 100));
      attempts++;
    }
    const name = firstCanvasName();
    if (name) {
      await goto(`/canvas/${name}`);
    } else {
      noDashboardsForRole = true;
      redirectChecked = true;
    }
  });
</script>

<svelte:head>
  <title>Bratrax Developer</title>
</svelte:head>

{#if !redirectChecked && $bratraxUser && data.initialized}
  <!-- Brief hold while we resolve the first dashboard. Every role lands on
       /canvas/<name>; this loading state covers the canvas-query tick. -->
  <div class="grid h-full place-items-center text-bratrax-text-muted font-mono text-xs">
    Loading…
  </div>
{:else if noDashboardsForRole}
  <div class="grid h-full place-items-center px-8">
    <div class="max-w-md text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        DASHBOARDS
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">No dashboards yet</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Your team is still setting things up. Check back in a few minutes — we're loading
        your data and building your first dashboards now.
      </p>
    </div>
  </div>
{:else}
  <div class="flex size-full overflow-hidden">
    <div class="flex size-full overflow-hidden">
      {#if data.initialized}
        {#if $generatingSampleData}
          <GeneratingMessage title="Generating your sample data..." />
        {:else if $generatingCanvas}
          <GeneratingMessage title="Generating your Canvas dashboard..." />
        {/if}
      {:else}
        <div class="scroll" in:fly={{ duration: 1600, delay: 400, y: 8 }}>
          <div class="wrapper column p-10 2xl:py-16">
            <TitleContent />
            <div class="column" in:fly={{ duration: 1600, delay: 1200, y: 4 }}>
              <ProjectCards />
            </div>
          </div>
        </div>
      {/if}
    </div>
    <BratraxChatGate />
  </div>
{/if}

<style lang="postcss">
  .scroll {
    @apply size-full overflow-x-hidden overflow-y-auto;
  }

  .wrapper {
    @apply w-full h-fit min-h-screen bg-no-repeat bg-cover;
    background-image: url("/img/welcome-bg-art.jpg");
  }

  :global(.dark) .wrapper {
    background-image: url("/img/welcome-bg-art-dark.jpg");
  }

  .column {
    @apply flex flex-col items-center gap-y-6;
  }
</style>
