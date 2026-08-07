<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import {
    onboardMe,
    getPaymentCheckoutUrl,
    getPaymentStatus,
    getOnboardResumeRoute,
  } from "$lib/bratrax/onboarding/api";
  import {
    isShopifyEmbedded,
    reserveTopLevelTab,
  } from "$lib/bratrax/shopify-embed";

  type Mode = "loading" | "intro" | "confirming" | "timeout" | "error";

  let mode: Mode = "loading";
  let error = "";
  let companyName = "";
  let busy = false;

  // Embedded (Shopify admin iframe) callers can't navigate to Lemon Squeezy in
  // place — LS won't render in a nested iframe. Checkout opens in a new tab and
  // this window stays put, polling, until the webhook flips is_paid.
  const embedded = isShopifyEmbedded();
  // Set when the popup was blocked, so we can offer a plain link instead of
  // stranding the merchant at the payment step.
  let manualCheckoutUrl = "";

  // Polling state for the post-checkout return.
  const POLL_INTERVAL_MS = 2000;
  // Non-embedded: LS has already bounced the browser back, so we're only
  // waiting on the webhook — ~30s is plenty. Embedded: this window starts
  // polling the moment checkout opens in the other tab, so the clock has to
  // cover the merchant actually paying — card entry, 3-D Secure, a bank-app
  // round trip. 30 minutes, because timing out mid-payment shows a "couldn't
  // confirm" screen to someone who is doing nothing wrong.
  $: POLL_MAX_ATTEMPTS = embedded ? 900 : 15; // ~30min vs ~30s
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
        // Route off the server's step, not a hardcoded path. Payment used to
        // always precede the store connection, so /onboard/store was right by
        // construction. A merchant who arrived from the Shopify App Store
        // already has their store attached and the webhook resolves them to
        // embed_pending (see onboarding._next_step_after_payment) — sending
        // them to /onboard/store would ask them to connect a store twice.
        await goto(getOnboardResumeRoute(s.step) ?? "/onboard/store");
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
    manualCheckoutUrl = "";

    // Reserve the tab NOW, while the click's popup grant is still live. It does
    // not survive the await below, so opening afterwards would be blocked
    // essentially every time — the fallback link would become the normal path
    // rather than the exception.
    const tab = embedded ? reserveTopLevelTab() : null;

    try {
      const result = await getPaymentCheckoutUrl(embedded);
      if (result.already_paid) {
        tab?.close();
        const me = await onboardMe().catch(() => null);
        await goto(getOnboardResumeRoute(me?.step) ?? "/onboard/store");
        return;
      }
      if (!result.checkout_url) {
        tab?.close();
        error = result.error || "Could not start checkout. Please try again.";
        return;
      }

      if (!embedded) {
        // Hard navigation to LS hosted checkout. They'll redirect us back to
        // /onboard/payment?return=1 on successful payment.
        window.location.href = result.checkout_url;
        return;
      }

      // Embedded: send the reserved tab to checkout and keep polling here. It
      // lands on /payment-complete; this window advances as soon as the LS
      // webhook flips is_paid, so the merchant never has to come back to it.
      if (tab?.ok) {
        tab.navigate(result.checkout_url);
      } else {
        manualCheckoutUrl = result.checkout_url;
      }
      startPolling();
    } catch (e) {
      tab?.close();
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
      await goto("/onboard/store");
      return;
    }

    companyName = me.company_name || "";

    // Already a paying subscriber → skip directly to wherever they actually
    // are. Not /onboard/store unconditionally: an App Store merchant arrives
    // with Shopify already attached and belongs on the embed step.
    if (me.is_paid_subscriber) {
      await goto(getOnboardResumeRoute(me.step) ?? "/onboard/store");
      return;
    }

    // ?return=1 = LS just bounced the user back after checkout. Webhook
    // may still be in flight on our side. Poll until is_paid flips, then
    // continue to /onboard/store.
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

<div
  class="flex h-full items-start justify-center overflow-y-auto bg-bratrax-bg pt-16 pb-16"
>
  <div class="w-full max-w-2xl px-6">
    <div class="mb-8">
      <p
        class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid"
      >
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
          Hang tight — we're checking with our payment provider. This usually
          takes a few seconds.
        {:else if mode === "timeout"}
          The payment may still come through. You can wait a bit and refresh, or
          try the checkout again. If you've already paid, your dashboard will
          unlock as soon as we hear back.
        {:else}
          {#if companyName}
            <span class="text-bratrax-text-body">{companyName}</span> is set up.
          {/if}
          Complete your subscription to start onboarding.
        {/if}
      </p>
    </div>

    {#if error}
      <div
        class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato"
      >
        {error}
      </div>
    {/if}

    <div class="border border-bratrax-border bg-bratrax-surface px-6 py-6">
      {#if mode === "loading"}
        <p
          class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted"
        >
          Loading…
        </p>
      {:else if mode === "confirming"}
        <div class="flex flex-col gap-3">
          <div class="flex items-center gap-3">
            <span
              class="inline-block h-2 w-2 animate-pulse rounded-full bg-bratrax-acid"
            ></span>
            <p
              class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted"
            >
              {#if embedded}
                <!-- No counter here. It reads as a deadline the merchant is
                     racing, when in fact nothing is lost if it runs out — the
                     webhook is the source of truth and "Check again" picks the
                     payment up whenever it lands. -->
                Complete payment in the new tab — we'll continue automatically
              {:else}
                Waiting for confirmation… ({pollAttempts}/{POLL_MAX_ATTEMPTS})
              {/if}
            </p>
          </div>
          {#if manualCheckoutUrl}
            <!-- Popup blocked. Never leave the merchant stuck at the payment
                 step because a blocker ate the window — give them the link. -->
            <div class="border border-bratrax-border bg-bratrax-bg px-4 py-3">
              <p class="mb-2 text-xs text-bratrax-text-muted">
                Your browser blocked the checkout window.
              </p>
              <a
                href={manualCheckoutUrl}
                target="_blank"
                rel="noopener noreferrer"
                class="inline-block bg-bratrax-acid px-5 py-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90"
              >
                Open checkout →
              </a>
            </div>
          {/if}
        </div>
      {:else if mode === "timeout"}
        <div class="flex flex-col gap-3">
          <button
            type="button"
            on:click={() => {
              mode = "confirming";
              startPolling();
            }}
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
            You'll complete payment on Lemon Squeezy's secure checkout page.
            After paying, we'll bring you back here automatically and unlock the
            rest of onboarding.
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
      <a
        href="mailto:support@bratrax.com"
        class="text-bratrax-acid hover:underline">support@bratrax.com</a
      >.
    </p>
  </div>
</div>
