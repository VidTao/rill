<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { onboardConnect } from "$lib/bratrax/onboarding/api";
  import BrandDomainsEditor from "$lib/bratrax/BrandDomainsEditor.svelte";

  // Snippet-install modal for the generic third-party page pixel. Shared by
  // /onboard/stack and /connectors via the external-pages builder picker,
  // mirroring FunnelishInstallModal's shape.
  export let clientId: string;
  export let shopifyShopDomain: string = "";
  export let connected: boolean = false;
  export let connectedAt: string = "";

  const dispatch = createEventDispatcher<{ close: void; installed: void }>();

  let snippetCopied = false;
  let snippetMarking = false;
  let error = "";

  // The Svelte compiler scans this <script lang="ts"> block for a literal
  // closing tag, so the snippet's closing tag is built from concatenated
  // pieces — a raw closing tag anywhere in here (even in a comment or
  // string) would terminate the block early.
  function buildSnippet(): string {
    const id = clientId || "REPLACE_WITH_BRATRAX_CLIENT_ID";

    // handoffDomains is load-bearing, not cosmetic. Decorating the outbound
    // link with bt_aid is what lets the Shopify Web Pixel record
    // features.bratrax_aid, which is the only key the warehouse can use to tie
    // a page view on this domain to the eventual order. Get it wrong and the
    // events still arrive but can never be credited to a sale.
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
      "  window.bratraxThirdPartyPixel = {\n" +
      `    clientId: "${id}",\n` +
      `    endpoint: "https://api.bratrax.com/events/ingest",\n` +
      `    handoffDomains: ${domainsLiteral},\n` +
      "    debug: false\n" +
      "  };\n" +
      closeTag +
      "\n" +
      `<script async src="https://api.bratrax.com/bratrax-third-party-pixel.js">${closeTag}`
    );
  }

  $: snippetText = buildSnippet();

  async function copySnippet() {
    try {
      await navigator.clipboard.writeText(snippetText);
      snippetCopied = true;
      setTimeout(() => {
        snippetCopied = false;
      }, 2000);
    } catch {
      snippetCopied = false;
    }
  }

  async function markInstalled() {
    if (!clientId || snippetMarking) return;
    error = "";
    snippetMarking = true;
    try {
      // Platform id is "custom_pages", NOT "external_pages"/"funnelish":
      // _determine_stack treats those two as "Funnelish selected" and would
      // scaffold this client onto the Funnelish stack.
      await onboardConnect(clientId, "custom_pages", { builder: "other" });
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
    class="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto border border-bratrax-border bg-bratrax-surface p-6"
    on:click|stopPropagation
    on:keydown|stopPropagation
    role="dialog"
    tabindex="-1"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-4 flex items-baseline justify-between">
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Custom / Other pages
      </h2>
      <button
        on:click={close}
        class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
      >
        Close
      </button>
    </div>

    <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
      Paste this into the &lt;head&gt; of every page you want tracked —
      advertorials, pre-sell and bridge pages, quizzes, custom checkouts. Works
      from a plain HTML template or a Google Tag Manager Custom HTML tag. It
      records page views with their UTM and click IDs, and tags every link
      pointing at your store so the visit stitches to the order.
    </p>

    {#if connected && connectedAt}
      <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-acid/80">
        Marked installed: {new Date(connectedAt).toLocaleString()}
      </p>
    {/if}

    <pre
      class="mb-2 max-h-72 overflow-auto border border-bratrax-border bg-bratrax-bg p-3 font-mono text-[11px] leading-relaxed text-bratrax-text-body">{snippetText}</pre>

    <div class="mb-3 flex items-center justify-end">
      <button on:click={copySnippet} class="btn-bratrax btn-neutral btn-compact">
        {snippetCopied ? "Copied" : "Copy snippet"}
      </button>
    </div>

    {#if !shopifyShopDomain}
      <p class="mb-3 font-mono text-[10px] uppercase tracking-wider text-bratrax-tomato/80">
        Connect your store first so we can fill in your storefront domain, or
        replace the placeholders by hand. Without it, links to your store won't
        be tagged and these pages can't be matched to orders.
      </p>
    {/if}

    <div class="mb-3 mt-5 border-t border-bratrax-border pt-4">
      <BrandDomainsEditor embedded={true} />
    </div>

    {#if error}
      <div
        class="mb-3 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
      >
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
