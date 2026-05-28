<script lang="ts">
  export let open = false;
  export let loading = false;
  export let error = "";
  export let onSubmit: (creds: { clientId: string; clientSecret: string }) => void = () => {};
  export let onClose: () => void = () => {};

  let clientId = "";
  let clientSecret = "";
  let localError = "";

  function handleSubmit() {
    if (!clientId.trim() || !clientSecret.trim()) {
      localError = "Please enter both client_id and client_secret";
      return;
    }
    localError = "";
    onSubmit({ clientId: clientId.trim(), clientSecret: clientSecret.trim() });
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

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">Connect Taboola</h2>
      <p class="mb-4 font-mono text-[11px] text-bratrax-text-muted">
        Get these from your Taboola account manager — they are unique to your account.
      </p>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        {#if displayError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {displayError}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label for="tab-client-id" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Client ID
          </label>
          <input
            id="tab-client-id"
            type="text"
            bind:value={clientId}
            placeholder="Your Taboola client_id"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="tab-client-secret" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Client Secret
          </label>
          <input
            id="tab-client-secret"
            type="password"
            bind:value={clientSecret}
            placeholder="Your Taboola client_secret"
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
