<script lang="ts">
  export let open = false;
  export let loading = false;
  export let onSubmit: (shopUrl: string) => void = () => {};
  export let onClose: () => void = () => {};

  let shopUrl = "";
  let error = "";

  function handleSubmit() {
    const trimmed = shopUrl.trim();
    if (!trimmed) {
      error = "Please enter your Shopify store URL";
      return;
    }
    // Normalize: strip protocol and trailing slashes, extract just the subdomain
    const normalized = trimmed
      .replace(/^https?:\/\//, "")
      .replace(/\/$/, "");
    if (!normalized.includes(".myshopify.com")) {
      error = "URL must be a .myshopify.com domain (e.g., my-store.myshopify.com)";
      return;
    }
    error = "";
    onSubmit(normalized);
  }
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

      <h2 class="mb-4 text-lg font-bold text-bratrax-text-headline">Connect Shopify Store</h2>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <label for="shop-url" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Store URL
          </label>
          <input
            id="shop-url"
            type="text"
            bind:value={shopUrl}
            placeholder="my-store.myshopify.com"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
          />
          {#if error}
            <span class="font-mono text-[10px] text-bratrax-tomato">{error}</span>
          {/if}
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
