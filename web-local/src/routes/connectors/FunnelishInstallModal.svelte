<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { onboardConnect } from "$lib/bratrax/onboarding/api";

  // Snippet-install modal for the Funnelish pixel. Shared by /onboard/stack
  // and /connectors via the external-pages builder picker. Owns its own
  // snippet/copy/install state; parents react to the `installed` event by
  // refreshing connection state.
  export let clientId: string;
  export let shopifyShopDomain: string = "";
  export let connected: boolean = false;
  export let connectedAt: string = "";

  const dispatch = createEventDispatcher<{ close: void; installed: void }>();

  let snippetCopied = false;
  let webhookUrlCopied = false;
  let snippetMarking = false;
  let error = "";

  // The Svelte compiler scans this <script lang="ts"> block for a literal
  // closing tag, so the snippet's closing tag is built from concatenated
  // pieces — a raw closing tag anywhere in here (even in a comment or
  // string) would terminate the block early.
  function buildSnippet(): string {
    const id = clientId || "REPLACE_WITH_BRATRAX_CLIENT_ID";
    let domainsLiteral = `["brand.com", "brand.myshopify.com"]`;
    if (shopifyShopDomain) {
      const myshopify = shopifyShopDomain.endsWith(".myshopify.com")
        ? shopifyShopDomain
        : `${shopifyShopDomain.split(".")[0]}.myshopify.com`;
      const list = [shopifyShopDomain];
      if (myshopify !== shopifyShopDomain) list.push(myshopify);
      domainsLiteral = `[${list.map((d) => `"${d}"`).join(", ")}]`;
    }
    const closeTag = "</" + "script>";
    return (
      "<script>\n" +
      "  window.bratraxFunnelishPixel = {\n" +
      `    clientId: "${id}",\n` +
      `    endpoint: "https://api.bratrax.com/events/ingest",\n` +
      `    shopifyDomains: ${domainsLiteral},\n` +
      "    debug: false\n" +
      "  };\n" +
      closeTag + "\n" +
      `<script async src="https://api.bratrax.com/bratrax-funnelish-pixel.js">${closeTag}`
    );
  }

  $: snippetText = buildSnippet();

  function buildWebhookUrl(): string {
    const id = clientId || "REPLACE_WITH_BRATRAX_CLIENT_ID";
    return `https://api.bratrax.com/webhooks/funnelish?client_id=${id}`;
  }

  $: webhookUrl = buildWebhookUrl();

  async function copySnippet() {
    try {
      await navigator.clipboard.writeText(snippetText);
      snippetCopied = true;
      setTimeout(() => { snippetCopied = false; }, 2000);
    } catch {
      snippetCopied = false;
    }
  }

  async function copyWebhookUrl() {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      webhookUrlCopied = true;
      setTimeout(() => { webhookUrlCopied = false; }, 2000);
    } catch {
      webhookUrlCopied = false;
    }
  }

  async function markInstalled() {
    if (!clientId || snippetMarking) return;
    error = "";
    snippetMarking = true;
    try {
      await onboardConnect(clientId, "funnelish", { builder: "funnelish" });
      dispatch("installed");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      snippetMarking = false;
    }
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
    class="relative w-full max-w-2xl border border-bratrax-border bg-bratrax-surface p-6"
    on:click|stopPropagation
    on:keydown|stopPropagation
    role="dialog"
    tabindex="-1"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-4 flex items-baseline justify-between">
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Funnelish
      </h2>
      <button
        on:click={close}
        class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
      >
        Close
      </button>
    </div>

    <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
      Paste this snippet into Funnelish → Settings → Custom Codes (head). It
      captures page views, leads, checkout intent, purchases, OTO accepts,
      and decorates links pointing at your Shopify store so the visit
      stitches together end-to-end.
    </p>

    {#if connected && connectedAt}
      <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-acid/80">
        Marked installed: {new Date(connectedAt).toLocaleString()}
      </p>
    {/if}

    <pre
      class="mb-2 max-h-72 overflow-auto border border-bratrax-border bg-bratrax-bg p-3 font-mono text-[11px] leading-relaxed text-bratrax-text-body"
    >{snippetText}</pre>

    <div class="mb-3 flex items-center justify-end">
      <button
        on:click={copySnippet}
        class="btn-bratrax btn-neutral btn-compact"
      >
        {snippetCopied ? "Copied" : "Copy snippet"}
      </button>
    </div>

    <div class="mb-3 mt-5 border-t border-bratrax-border pt-4">
      <p class="mb-2 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
        Set up automation webhooks so server-side events (purchases, opt-ins,
        subscriptions, refunds) reach Bratrax. In Funnelish → Automations →
        New automation, pick a trigger, add a Webhook action, paste the URL
        below, and save. Repeat for each trigger you want to capture.
      </p>

      <pre
        class="mb-2 overflow-auto border border-bratrax-border bg-bratrax-bg p-3 font-mono text-[11px] leading-relaxed text-bratrax-text-body"
      >{webhookUrl}</pre>

      <div class="mb-2 flex items-center justify-end">
        <button
          on:click={copyWebhookUrl}
          class="btn-bratrax btn-neutral btn-compact"
        >
          {webhookUrlCopied ? "Copied" : "Copy webhook URL"}
        </button>
      </div>

      <p class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
        Available triggers: On Purchase · Purchase Attempt · Optin ·
        Recurring Payment · Recurring Payment Fails · On Refund ·
        New Subscription · Subscription Canceled
      </p>
    </div>

    {#if !shopifyShopDomain}
      <p class="mb-3 font-mono text-[10px] uppercase tracking-wider text-bratrax-tomato/80">
        Connect Shopify first to auto-fill the storefront domain, or replace
        the placeholders by hand.
      </p>
    {/if}

    {#if error}
      <div class="mb-3 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <div class="flex items-center justify-end gap-2">
      {#if !connected}
        <button
          on:click={markInstalled}
          disabled={snippetMarking || !clientId}
          class="btn-bratrax btn-primary btn-compact"
        >
          {snippetMarking ? "Saving…" : "I've installed it"}
        </button>
      {/if}
    </div>
  </div>
</div>
