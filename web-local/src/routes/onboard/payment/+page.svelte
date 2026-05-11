<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import {
    onboardMe,
    getPaymentCheckoutUrl,
    getPaymentStatus,
  } from "$lib/bratrax/onboarding/api";

  type Mode = "loading" | "intro" | "confirming" | "timeout" | "error";

  let mode: Mode = "loading";
  let error = "";
  let companyName = "";
  let busy = false;

  // Polling state for the post-checkout return.
  const POLL_INTERVAL_MS = 2000;
  const POLL_MAX_ATTEMPTS = 15; // ~30s
  let pollHandle: ReturnType<typeof setInterval> | null = null;
  let pollAttempts = 0;

  function clearPoll() {
    if (pollHandle) {
      clearInterval(pollHandle);
      pollHandle = null;
    }
  }

  async function tickStatus(): Promise<boolean> {
    try {
      const s = await getPaymentStatus();
      if (s.is_paid) {
        clearPoll();
        await goto("/onboard/shopify");
        return true;
      }
    } catch (e) {
      // Transient — keep polling. Real failures fall through to timeout.
      // eslint-disable-next-line no-console
      console.warn("[bratrax payment] status poll failed", e);
    }
    return false;
  }

  function startPolling() {
    clearPoll();
    mode = "confirming";
    pollAttempts = 0;
    pollHandle = setInterval(async () => {
      pollAttempts++;
      const done = await tickStatus();
      if (done) return;
      if (pollAttempts >= POLL_MAX_ATTEMPTS) {
        clearPoll();
        mode = "timeout";
      }
    }, POLL_INTERVAL_MS);
  }

  async function startCheckout() {
    busy = true;
    error = "";
    try {
      const result = await getPaymentCheckoutUrl();
      if (result.already_paid) {
        await goto("/onboard/shopify");
        return;
      }
      if (!result.checkout_url) {
        error = result.error || "Could not start checkout. Please try again.";
        return;
      }
      // Hard navigation to LS hosted checkout. They'll redirect us back to
      // /onboard/payment?return=1 on successful payment.
      window.location.href = result.checkout_url;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy = false;
    }
  }

  onMount(async () => {
    let me;
    try {
      me = await onboardMe();
    } catch {
      me = null;
    }
    if (!me?.client_id) {
      // No client yet — fall back to the start of onboarding. Layout will
      // sort it out.
      await goto("/onboard/shopify");
      return;
    }

    companyName = me.company_name || "";

    // Already a paying subscriber → skip directly to onboarding.
    if (me.is_paid_subscriber) {
      await goto("/onboard/shopify");
      return;
    }

    // ?return=1 = LS just bounced the user back after checkout. Webhook
    // may still be in flight on our side. Poll until is_paid flips, then
    // continue to /onboard/shopify.
    if ($page.url.searchParams.get("return") === "1") {
      // Run one immediate check before kicking off the interval so
      // already-paid (fast webhook) cases don't wait the first 2s.
      const done = await tickStatus();
      if (!done) startPolling();
      return;
    }

    // Initial state — show the "Continue to payment" CTA.
    mode = "intro";
  });

  onDestroy(() => clearPoll());
</script>

<div class="flex h-full items-start justify-center overflow-y-auto bg-bratrax-bg pt-16 pb-16">
  <div class="w-full max-w-2xl px-6">
    <div class="mb-8">
      <p class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid">
        Subscription
      </p>
      <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
        {#if mode === "confirming"}
          Confirming your payment…
        {:else if mode === "timeout"}
          We couldn't confirm your payment yet
        {:else}
          Activate your Bratrax subscription
        {/if}
      </h1>
      <p class="mt-2 text-sm text-bratrax-text-muted">
        {#if mode === "confirming"}
          Hang tight — we're checking with our payment provider. This usually takes a few seconds.
        {:else if mode === "timeout"}
          The payment may still come through. You can wait a bit and refresh, or try the checkout again. If you've already paid, your dashboard will unlock as soon as we hear back.
        {:else}
          {#if companyName}
            <span class="text-bratrax-text-body">{companyName}</span> is set up.
          {/if}
          Complete your subscription to start onboarding.
        {/if}
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <div class="border border-bratrax-border bg-bratrax-surface px-6 py-6">
      {#if mode === "loading"}
        <p class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted">
          Loading…
        </p>
      {:else if mode === "confirming"}
        <div class="flex items-center gap-3">
          <span class="inline-block h-2 w-2 animate-pulse rounded-full bg-bratrax-acid"></span>
          <p class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted">
            Waiting for confirmation… ({pollAttempts}/{POLL_MAX_ATTEMPTS})
          </p>
        </div>
      {:else if mode === "timeout"}
        <div class="flex flex-col gap-3">
          <button
            type="button"
            on:click={() => { mode = "confirming"; startPolling(); }}
            class="bg-bratrax-acid px-6 py-3 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90"
          >
            Check again
          </button>
          <button
            type="button"
            on:click={startCheckout}
            disabled={busy}
            class="border border-bratrax-border px-6 py-3 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted transition-colors hover:border-bratrax-acid hover:text-bratrax-acid disabled:opacity-40"
          >
            {busy ? "Opening checkout…" : "Start checkout again"}
          </button>
        </div>
      {:else if mode === "intro"}
        <div class="flex flex-col gap-3">
          <p class="text-sm text-bratrax-text-body">
            You'll complete payment on Lemon Squeezy's secure checkout page. After paying, we'll bring you back here automatically and unlock the rest of onboarding.
          </p>
          <button
            type="button"
            on:click={startCheckout}
            disabled={busy}
            class="mt-3 bg-bratrax-acid px-6 py-3 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90 disabled:opacity-40"
          >
            {busy ? "Opening checkout…" : "Continue to payment →"}
          </button>
        </div>
      {/if}
    </div>

    <p class="mt-6 text-xs text-bratrax-text-muted/60">
      Questions about pricing or your subscription? Email
      <a href="mailto:hello@bratrax.com" class="text-bratrax-acid hover:underline">hello@bratrax.com</a>.
    </p>
  </div>
</div>
