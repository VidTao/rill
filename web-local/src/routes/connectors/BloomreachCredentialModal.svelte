<script lang="ts">
  export let open = false;
  export let loading = false;
  export let error = "";
  export let onSubmit: (creds: {
    projectToken: string;
    apiKey: string;
    apiSecret: string;
    baseUrl: string;
  }) => void = () => {};
  export let onClose: () => void = () => {};

  let projectToken = "";
  let apiKey = "";
  let apiSecret = "";
  let baseUrl = "";
  let localError = "";
  let showHelp = false;

  function handleSubmit() {
    if (!projectToken.trim() || !apiKey.trim() || !apiSecret.trim()) {
      localError = "Please enter the project token, API key, and API secret";
      return;
    }
    localError = "";
    onSubmit({
      projectToken: projectToken.trim(),
      apiKey: apiKey.trim(),
      apiSecret: apiSecret.trim(),
      baseUrl: baseUrl.trim(),
    });
  }

  $: displayError = error || localError;
  // Auto-expand the "where do I find these" help when the server rejects creds.
  $: if (error) showHelp = true;
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

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">Connect Bloomreach</h2>
      <p class="mb-4 font-mono text-[11px] text-bratrax-text-muted">
        Bloomreach Engagement API access, scoped to one project. These are unique to your account.
      </p>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        {#if displayError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {displayError}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label for="br-project-token" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Project Token
          </label>
          <input
            id="br-project-token"
            type="text"
            bind:value={projectToken}
            placeholder="Your Bloomreach project token"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="br-api-key" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            API Key
          </label>
          <input
            id="br-api-key"
            type="text"
            bind:value={apiKey}
            placeholder="API key ID (public)"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="br-api-secret" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            API Secret
          </label>
          <input
            id="br-api-secret"
            type="password"
            bind:value={apiSecret}
            placeholder="API secret (private)"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="br-base-url" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Region base URL <span class="font-normal normal-case tracking-normal text-bratrax-text-muted/60">(optional)</span>
          </label>
          <input
            id="br-base-url"
            type="text"
            bind:value={baseUrl}
            placeholder="https://api.exponea.com"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
            autocomplete="off"
          />
        </div>

        <button
          type="button"
          class="self-start font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-acid"
          on:click={() => (showHelp = !showHelp)}
        >
          {showHelp ? "− " : "+ "}Where do I find these?
        </button>
        {#if showHelp}
          <div class="border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-[11px] leading-relaxed text-bratrax-text-muted">
            In Bloomreach Engagement, open your project and go to
            <span class="text-bratrax-text-body">Project settings → Access management → API</span>.
            Create (or reuse) an API group, then copy its API key and secret. The
            project token is shown under Project settings → Overview. If your
            account is on a regional endpoint, set the base URL accordingly (e.g.
            <span class="text-bratrax-text-body">https://api.eu1.exponea.com</span>).
          </div>
        {/if}

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
