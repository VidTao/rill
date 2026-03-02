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
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click={onClose}>
    <div
      class="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl"
      on:click|stopPropagation
    >
      <h2 class="mb-4 text-lg font-semibold text-gray-800">Connect Shopify Store</h2>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <label for="shop-url" class="text-sm font-medium text-gray-700">Store URL</label>
          <input
            id="shop-url"
            type="text"
            bind:value={shopUrl}
            placeholder="my-store.myshopify.com"
            class="rounded border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            disabled={loading}
          />
          {#if error}
            <span class="text-xs text-red-600">{error}</span>
          {/if}
        </div>

        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="rounded bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
            on:click={onClose}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
            disabled={loading}
          >
            {loading ? "Connecting..." : "Connect"}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
