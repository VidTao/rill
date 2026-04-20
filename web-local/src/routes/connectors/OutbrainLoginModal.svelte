<script lang="ts">
  export let open = false;
  export let loading = false;
  export let onSubmit: (encodedAuth: string) => void = () => {};
  export let onClose: () => void = () => {};

  let username = "";
  let password = "";
  let error = "";

  function handleSubmit() {
    if (!username.trim() || !password.trim()) {
      error = "Please enter both username and password";
      return;
    }
    error = "";
    const encodedAuth = btoa(`${username}:${password}`);
    onSubmit(encodedAuth);
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

      <h2 class="mb-4 text-lg font-bold text-bratrax-text-headline">Connect Outbrain</h2>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        {#if error}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {error}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label for="ob-user" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Username
          </label>
          <input
            id="ob-user"
            type="text"
            bind:value={username}
            placeholder="your@email.com"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="ob-pass" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Password
          </label>
          <input
            id="ob-pass"
            type="password"
            bind:value={password}
            placeholder="Enter your password"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            disabled={loading}
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
