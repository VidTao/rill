<script lang="ts">
  export let open = false;
  export let loading = false;
  export let error = "";
  export let onSubmit: (creds: {
    storeUrl: string;
    consumerKey: string;
    consumerSecret: string;
  }) => void = () => {};
  export let onClose: () => void = () => {};

  let storeUrl = "";
  let consumerKey = "";
  let consumerSecret = "";
  let localError = "";
  let showInstructions = false;

  function handleSubmit() {
    if (!storeUrl.trim() || !consumerKey.trim() || !consumerSecret.trim()) {
      localError = "Please enter the store URL, consumer key and consumer secret";
      return;
    }
    localError = "";
    onSubmit({
      storeUrl: storeUrl.trim(),
      consumerKey: consumerKey.trim(),
      consumerSecret: consumerSecret.trim(),
    });
  }

  $: displayError = error || localError;
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" on:click={onClose}>
    <div
      class="relative w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">Connect WooCommerce</h2>
      <p class="mb-3 font-mono text-[11px] text-bratrax-text-muted">
        Paste a REST API key generated in your store admin. We only need Read access.
      </p>

      <button
        type="button"
        class="mb-3 font-mono text-[11px] text-bratrax-acid hover:underline"
        on:click={() => (showInstructions = !showInstructions)}
      >
        {showInstructions ? "▾" : "▸"} How do I get these?
      </button>
      {#if showInstructions}
        <ol class="mb-4 list-decimal space-y-1 pl-4 font-mono text-[11px] text-bratrax-text-muted">
          <li>In WordPress admin, go to WooCommerce → Settings → Advanced → REST API.</li>
          <li>Click "Add key", set Permissions to <span class="text-bratrax-text-body">Read</span>, then Generate.</li>
          <li>Copy the Consumer key (ck_…) and Consumer secret (cs_…) into the fields below.</li>
        </ol>
      {/if}

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        {#if displayError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {displayError}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label for="woo-store-url" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Store URL
          </label>
          <input
            id="woo-store-url"
            type="text"
            bind:value={storeUrl}
            placeholder="https://yourstore.com"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="woo-consumer-key" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Consumer Key
          </label>
          <input
            id="woo-consumer-key"
            type="text"
            bind:value={consumerKey}
            placeholder="ck_..."
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="woo-consumer-secret" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Consumer Secret
          </label>
          <input
            id="woo-consumer-secret"
            type="password"
            bind:value={consumerSecret}
            placeholder="cs_..."
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover"
            on:click={onClose}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="submit"
            class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
            disabled={loading}
          >
            {loading ? "CONNECTING..." : "CONNECT"}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
