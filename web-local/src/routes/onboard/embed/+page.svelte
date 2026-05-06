<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount, onDestroy } from "svelte";
  import { onboardMe, verifyEmbedStatus } from "$lib/bratrax/onboarding/api";

  let clientId = "";
  let shopDomain = "";
  let themeEditorUrl = "";
  let status: "idle" | "polling" | "verified" | "timeout" = "idle";
  let error = "";
  let manualChecking = false;

  let pollHandle: ReturnType<typeof setInterval> | null = null;
  let pollAttempts = 0;
  const POLL_INTERVAL_MS = 3000;
  const POLL_MAX_ATTEMPTS = 100; // ~5 minutes

  function buildThemeEditorUrl(shop: string): string {
    // Shopify Theme Editor deep-link with the App Embeds panel pre-opened.
    // The merchant still has to flip the toggle and Save — Shopify policy
    // forbids auto-enable for theme app extensions.
    return `https://${shop}/admin/themes/current/editor?context=apps`;
  }

  async function check(): Promise<boolean> {
    if (!clientId) return false;
    try {
      const res = await verifyEmbedStatus(clientId);
      if (res.enabled) {
        status = "verified";
        if (pollHandle) {
          clearInterval(pollHandle);
          pollHandle = null;
        }
        // Backend has already advanced step → platforms_connected.
        // From here, return user to /onboard so they can connect ad platforms.
        await goto("/onboard");
        return true;
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
    return false;
  }

  async function manualRecheck() {
    manualChecking = true;
    error = "";
    try {
      await check();
    } finally {
      manualChecking = false;
    }
  }

  function startPolling() {
    if (pollHandle) return;
    status = "polling";
    pollHandle = setInterval(async () => {
      pollAttempts++;
      if (pollAttempts > POLL_MAX_ATTEMPTS) {
        if (pollHandle) clearInterval(pollHandle);
        pollHandle = null;
        status = "timeout";
        return;
      }
      await check();
    }, POLL_INTERVAL_MS);
  }

  onMount(async () => {
    const me = await onboardMe();
    if (!me?.client_id) {
      await goto("/onboard/shopify");
      return;
    }
    if (me.shopify_embed_enabled) {
      // Already verified on a prior session — skip straight ahead.
      await goto("/onboard");
      return;
    }
    clientId = me.client_id;

    const creds = (me.stack_selections as Record<string, unknown>)
      ?.shopify_credentials as { shop?: string } | undefined;
    shopDomain = creds?.shop ?? "";
    if (!shopDomain) {
      error = "Shopify isn't connected yet. Please connect Shopify first.";
      return;
    }
    themeEditorUrl = buildThemeEditorUrl(shopDomain);

    // Run an immediate check (handles re-entry after the user already toggled
    // it on in another tab) before kicking off the polling loop.
    const alreadyEnabled = await check();
    if (!alreadyEnabled) startPolling();
  });

  onDestroy(() => {
    if (pollHandle) clearInterval(pollHandle);
  });
</script>

<div class="flex h-full items-start justify-center overflow-y-auto bg-bratrax-bg pt-16 pb-16">
  <div class="w-full max-w-2xl px-6">
    <div class="mb-8">
      <p class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid">
        One more step
      </p>
      <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
        Enable the Bratrax tracker on your store
      </h1>
      <p class="mt-2 text-sm text-bratrax-text-muted">
        We need to install a small tracker in your Shopify theme so we can
        see email signups, Klaviyo events, and on-site activity. Shopify
        requires you — not us — to flip the switch. Takes about 30 seconds.
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <ol class="mb-8 space-y-4 border border-bratrax-border bg-bratrax-surface p-6">
      <li class="flex gap-4">
        <span class="font-mono text-xs font-bold text-bratrax-acid">01</span>
        <div class="flex-1">
          <p class="text-sm text-bratrax-text-body">
            Open your Shopify Theme Editor.
          </p>
          {#if themeEditorUrl}
            <a
              href={themeEditorUrl}
              target="_blank"
              rel="noopener noreferrer"
              class="mt-2 inline-block bg-bratrax-acid px-6 py-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90"
            >
              Open Theme Editor →
            </a>
          {/if}
        </div>
      </li>
      <li class="flex gap-4">
        <span class="font-mono text-xs font-bold text-bratrax-acid">02</span>
        <p class="flex-1 text-sm text-bratrax-text-body">
          In the left sidebar, click <span class="font-mono text-bratrax-text-headline">App embeds</span>
          (the puzzle-piece icon).
        </p>
      </li>
      <li class="flex gap-4">
        <span class="font-mono text-xs font-bold text-bratrax-acid">03</span>
        <p class="flex-1 text-sm text-bratrax-text-body">
          Find <span class="font-mono text-bratrax-text-headline">Bratrax</span>
          in the list and toggle it <span class="font-mono text-bratrax-text-headline">on</span>.
        </p>
      </li>
      <li class="flex gap-4">
        <span class="font-mono text-xs font-bold text-bratrax-acid">04</span>
        <p class="flex-1 text-sm text-bratrax-text-body">
          Click <span class="font-mono text-bratrax-text-headline">Save</span>
          in the top-right of the Theme Editor. We'll detect it automatically.
        </p>
      </li>
    </ol>

    <div class="flex items-center justify-between border border-bratrax-border bg-bratrax-surface px-6 py-4">
      <div class="flex items-center gap-3">
        {#if status === "polling"}
          <span class="inline-block h-2 w-2 animate-pulse rounded-full bg-bratrax-acid"></span>
          <p class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted">
            Watching for the toggle…
          </p>
        {:else if status === "verified"}
          <span class="inline-block h-2 w-2 rounded-full bg-bratrax-acid"></span>
          <p class="font-mono text-xs uppercase tracking-wider text-bratrax-acid">
            Verified — moving on
          </p>
        {:else if status === "timeout"}
          <p class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted">
            Couldn't detect the embed yet
          </p>
        {:else}
          <p class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted">
            Ready when you are
          </p>
        {/if}
      </div>
      <button
        type="button"
        on:click={manualRecheck}
        disabled={manualChecking || status === "verified"}
        class="border border-bratrax-border px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted transition-colors hover:border-bratrax-acid hover:text-bratrax-acid disabled:opacity-40"
      >
        {manualChecking ? "Checking…" : "I've enabled it — re-check"}
      </button>
    </div>

    <p class="mt-6 text-xs text-bratrax-text-muted/60">
      Why this step? Shopify doesn't let apps install themselves into themes,
      so we ask you to flip the switch once. We'll never ask again.
    </p>
  </div>
</div>
