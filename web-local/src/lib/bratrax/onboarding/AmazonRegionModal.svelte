<script lang="ts">
  import { createEventDispatcher } from "svelte";

  // Region picker for Amazon Seller Central (SP-API). Unlike every other
  // connector, SP-API needs the region BEFORE the redirect: the Seller Central
  // consent host differs per region and the region cannot be recovered from the
  // callback or the token. Amazon Ads needs no picker — one consent there
  // reaches all three regions, so the backend discovers them.
  export type AmazonRegion = "na" | "eu" | "fe";

  interface Region {
    id: AmazonRegion;
    name: string;
    note: string;
  }

  const regions: Region[] = [
    {
      id: "na",
      name: "North America",
      note: "US, Canada, Mexico, Brazil.",
    },
    {
      id: "eu",
      name: "Europe",
      note: "UK, Germany, France, Italy, Spain, Netherlands, Sweden, Poland, Belgium, Turkey, UAE, Saudi Arabia, Egypt, India.",
    },
    {
      id: "fe",
      name: "Far East",
      note: "Japan, Australia, Singapore.",
    },
  ];

  const dispatch = createEventDispatcher<{
    select: { region: AmazonRegion };
    close: void;
  }>();

  function pick(r: Region) {
    dispatch("select", { region: r.id });
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
        Which Amazon region is your seller account in?
      </h2>
      <button
        on:click={close}
        class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
      >
        Close
      </button>
    </div>

    <p
      class="mb-4 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted"
    >
      Amazon runs a separate Seller Central per region, so we need to send you to
      the right one. Pick the region you sign in to. You'll choose which
      individual marketplaces to connect on the next screen.
    </p>

    <div class="flex flex-col gap-2">
      {#each regions as r}
        <button
          on:click={() => pick(r)}
          class="flex items-start justify-between gap-4 border border-bratrax-border bg-bratrax-bg p-4 text-left transition hover:border-bratrax-acid"
        >
          <div class="flex flex-col">
            <span class="font-bold text-bratrax-text-headline">{r.name}</span>
            <span
              class="mt-1 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted"
            >
              {r.note}
            </span>
          </div>
          <span
            class="self-center font-mono text-[10px] uppercase tracking-[2px] text-bratrax-acid"
          >
            Connect →
          </span>
        </button>
      {/each}
    </div>
  </div>
</div>
