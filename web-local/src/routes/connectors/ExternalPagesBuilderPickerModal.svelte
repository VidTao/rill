<script lang="ts">
  import { createEventDispatcher } from "svelte";

  // Sub-picker shown when the user picks "I have external landing pages".
  // Each entry maps to a builder-specific install pixel. Funnelish is the
  // only active option for v1; the rest are visible-but-disabled to set
  // expectations while we add their pixels.
  export type BuilderId = "funnelish" | "clickfunnels" | "ghl" | "other";

  interface Builder {
    id: BuilderId;
    name: string;
    note: string;
    available: boolean;
  }

  const builders: Builder[] = [
    {
      id: "funnelish",
      name: "Funnelish",
      note: "Pixel + link decoration. Tracks landing, checkout, OTOs, thank-you.",
      available: true,
    },
    {
      id: "clickfunnels",
      name: "ClickFunnels",
      note: "Coming soon.",
      available: false,
    },
    {
      id: "ghl",
      name: "GoHighLevel",
      note: "Coming soon.",
      available: false,
    },
    {
      id: "other",
      name: "Custom / Other",
      note: "Coming soon.",
      available: false,
    },
  ];

  const dispatch = createEventDispatcher<{
    select: { builder: BuilderId };
    close: void;
  }>();

  function pick(b: Builder) {
    if (!b.available) return;
    dispatch("select", { builder: b.id });
  }

  function close() {
    dispatch("close");
  }
</script>

<div
  class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
  on:click={close}
  on:keydown={(e) => e.key === "Escape" && close()}
  role="presentation"
>
  <div
    class="relative w-full max-w-xl border border-bratrax-border bg-bratrax-surface p-6"
    on:click|stopPropagation
    on:keydown|stopPropagation
    role="dialog"
    tabindex="-1"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-4 flex items-baseline justify-between">
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Where do your external pages live?
      </h2>
      <button
        on:click={close}
        class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
      >
        Close
      </button>
    </div>

    <p class="mb-4 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
      Pick the funnel builder your landing, checkout, and upsell pages run on.
      Each one ships with its own install snippet tuned to that platform's
      event model.
    </p>

    <div class="flex flex-col gap-2">
      {#each builders as b}
        <button
          on:click={() => pick(b)}
          disabled={!b.available}
          class="flex items-start justify-between gap-4 border border-bratrax-border bg-bratrax-bg p-4 text-left transition hover:border-bratrax-acid disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:border-bratrax-border"
        >
          <div class="flex flex-col">
            <span class="font-bold text-bratrax-text-headline">{b.name}</span>
            <span class="mt-1 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
              {b.note}
            </span>
          </div>
          {#if b.available}
            <span class="self-center font-mono text-[10px] uppercase tracking-[2px] text-bratrax-acid">
              Set up →
            </span>
          {:else}
            <span class="self-center font-mono text-[10px] uppercase tracking-[2px] text-bratrax-text-muted">
              Soon
            </span>
          {/if}
        </button>
      {/each}
    </div>
  </div>
</div>
