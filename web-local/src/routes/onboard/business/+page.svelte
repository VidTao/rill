<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { onboardMe, onboardBusinessProfile } from "$lib/bratrax/onboarding/api";

  let clientId = "";
  let submitting = false;
  let error = "";
  let stackSelections: Record<string, unknown> = {};

  // Questionnaire fields
  let aboutBusiness = "";
  let heroProduct = "";
  let priceRange = "";
  let revenueRange = "";
  let channels = "";
  let improving = "";
  let seasonality = "";

  const revenueOptions = [
    "Less than $10k/month",
    "$10k – $50k/month",
    "$50k – $250k/month",
    "$250k – $1M/month",
    "$1M+/month",
  ];

  onMount(async () => {
    const me = await onboardMe();
    if (!me?.client_id) {
      await goto("/onboard/shopify");
      return;
    }
    clientId = me.client_id;

    // Recover stack selections saved by the stack page
    try {
      const saved = sessionStorage.getItem("onboard_stack_selections");
      if (saved) stackSelections = JSON.parse(saved);
    } catch {
      // ignore
    }
  });

  async function handleSubmit() {
    if (!aboutBusiness.trim()) {
      error = "Please tell us about your business.";
      return;
    }

    error = "";
    submitting = true;

    try {
      await onboardBusinessProfile(clientId, {
        about_business: aboutBusiness.trim(),
        hero_product: heroProduct.trim(),
        price_range: priceRange.trim(),
        revenue_range: revenueRange,
        channels: channels.trim(),
        improving: improving.trim(),
        seasonality: seasonality.trim(),
      }, stackSelections);
      await goto("/onboard/loading");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      submitting = false;
    }
  }
</script>

<div class="flex min-h-screen items-start justify-center bg-bratrax-bg pt-16">
  <div class="w-full max-w-2xl px-6">
    <!-- Header -->
    <div class="mb-8">
      <p class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid">
        Almost there
      </p>
      <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
        Tell us about your business
      </h1>
      <p class="mt-2 text-sm text-bratrax-text-muted">
        This helps your AI analyst understand your business from day one.
        Your answers become permanent context — like briefing a new team member.
      </p>
    </div>

    <!-- Error -->
    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <!-- Form -->
    <form on:submit|preventDefault={handleSubmit} class="space-y-6">
      <!-- Q1: About -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          What does your business sell and who buys it? *
        </label>
        <textarea
          bind:value={aboutBusiness}
          rows="3"
          class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          placeholder="e.g., We sell premium skincare products direct to consumer, targeting women 25-45 who care about clean ingredients."
        ></textarea>
      </div>

      <!-- Q2: Hero product -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          What's your hero product?
        </label>
        <input
          type="text"
          bind:value={heroProduct}
          class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          placeholder="e.g., Vitamin C Brightening Serum ($48)"
        />
      </div>

      <!-- Q3: Price range -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          What's your typical price point?
        </label>
        <input
          type="text"
          bind:value={priceRange}
          class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          placeholder="e.g., $38-$68 per product, AOV around $65"
        />
      </div>

      <!-- Q4: Revenue range -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          Monthly revenue range
        </label>
        <div class="flex flex-wrap gap-2">
          {#each revenueOptions as option}
            <button
              type="button"
              class="border px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {revenueRange === option ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted'}"
              on:click={() => { revenueRange = revenueRange === option ? "" : option; }}
            >
              {option}
            </button>
          {/each}
        </div>
      </div>

      <!-- Q5: Channels -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          Where do your customers come from?
        </label>
        <textarea
          bind:value={channels}
          rows="2"
          class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          placeholder="e.g., Facebook Ads and Google Ads are the main drivers. Some organic from Instagram."
        ></textarea>
      </div>

      <!-- Q6: Improving -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          What are you trying to improve?
        </label>
        <textarea
          bind:value={improving}
          rows="3"
          class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          placeholder="e.g., We want to improve ROAS and understand which campaigns actually drive purchases vs just clicks. Also interested in LTV by channel."
        ></textarea>
      </div>

      <!-- Q7: Seasonality -->
      <div>
        <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          Any seasonality or upcoming events?
          <span class="ml-1 font-normal normal-case tracking-normal text-bratrax-text-muted/60">(optional)</span>
        </label>
        <input
          type="text"
          bind:value={seasonality}
          class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          placeholder="e.g., Q4 is our biggest season. Planning a product launch in June."
        />
      </div>

      <!-- Submit -->
      <div class="flex items-center justify-between pt-4 pb-12">
        <p class="text-xs text-bratrax-text-muted/60">
          You can update these answers anytime in Settings.
        </p>
        <button
          type="submit"
          disabled={submitting || !aboutBusiness.trim()}
          class="bg-bratrax-acid px-8 py-3 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90 disabled:opacity-40"
        >
          {submitting ? "Setting up..." : "Set up my analytics"}
        </button>
      </div>
    </form>
  </div>
</div>
