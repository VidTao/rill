<script lang="ts">
  export let open = false;
  export let loading = false;
  export let error = "";
  export let onSubmit: (creds: { encodedAuth: string }) => void = () => {};
  export let onClose: () => void = () => {};

  let username = "";
  let password = "";
  let localError = "";
  let showInstructions = false;

  function handleSubmit() {
    if (!username.trim() || !password.trim()) {
      localError = "Please enter both username and password";
      return;
    }
    localError = "";
    const encodedAuth = btoa(`${username}:${password}`);
    onSubmit({ encodedAuth });
  }

  $: displayError = error || localError;
  // The 401 from Outbrain when API access isn't enabled looks generic
  // ("Outbrain rejected the credentials"). Auto-expand the instructions
  // when that error shows so the user immediately sees what to do.
  $: if (error && /reject|401|credentials/i.test(error)) {
    showInstructions = true;
  }
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" on:click={onClose}>
    <div
      class="relative w-full max-w-md max-h-[90vh] overflow-y-auto border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">Connect Outbrain</h2>
      <p class="mb-4 font-mono text-[11px] text-bratrax-text-muted">
        Enter your Outbrain username and the dedicated API password Outbrain emailed you. This is <span class="text-bratrax-acid">NOT</span> your dashboard login password.
      </p>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        {#if displayError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {displayError}
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
            autocomplete="off"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="ob-pass" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            API Password
          </label>
          <input
            id="ob-pass"
            type="password"
            bind:value={password}
            placeholder="API password from Outbrain"
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

      <!-- Outbrain API access is gated — merchants can dashboard-login fine
           but the API rejects their credentials until Outbrain support
           explicitly enables API access on the account. This block walks
           them through requesting that enablement so we don't have to
           field the same support question for every onboarding. -->
      <div class="mt-5 border-t border-bratrax-border pt-4">
        <button
          type="button"
          on:click={() => (showInstructions = !showInstructions)}
          class="flex w-full items-center justify-between font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-acid hover:opacity-80"
        >
          <span>Don't have an API password yet?</span>
          <span class="text-base">{showInstructions ? "−" : "+"}</span>
        </button>

        {#if showInstructions}
          <div class="mt-3 flex flex-col gap-3 font-mono text-[11px] leading-relaxed text-bratrax-text-body">
            <p class="text-bratrax-text-muted">
              Outbrain's API is gated. Your dashboard email and password don't work against the API until Outbrain enables API access on your account and emails you a separate API password. Here's how to get it:
            </p>

            <ol class="flex flex-col gap-2 pl-1">
              <li class="flex gap-2">
                <span class="flex h-4 w-4 shrink-0 items-center justify-center bg-bratrax-acid text-[10px] font-bold text-bratrax-bg">1</span>
                <span>
                  Open
                  <a
                    href="https://www.outbrain.com/partner-api/"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-bratrax-acid underline hover:opacity-80"
                  >outbrain.com/partner-api</a>
                  and fill out the API access request form.
                </span>
              </li>
              <li class="flex gap-2">
                <span class="flex h-4 w-4 shrink-0 items-center justify-center bg-bratrax-acid text-[10px] font-bold text-bratrax-bg">2</span>
                <span>
                  In the "Additional context" field, paste:
                  <span class="mt-1 block border border-bratrax-border bg-bratrax-bg px-2 py-1.5 text-[10px] text-bratrax-text-primary">
                    Integrating with Bratrax (third-party analytics). Please enable Amplify API access on this account.
                  </span>
                </span>
              </li>
              <li class="flex gap-2">
                <span class="flex h-4 w-4 shrink-0 items-center justify-center bg-bratrax-acid text-[10px] font-bold text-bratrax-bg">3</span>
                <span>Wait 1–3 business days. Outbrain emails you a dedicated API password (different from your dashboard password).</span>
              </li>
              <li class="flex gap-2">
                <span class="flex h-4 w-4 shrink-0 items-center justify-center bg-bratrax-acid text-[10px] font-bold text-bratrax-bg">4</span>
                <span>Return here, enter your Outbrain <b>username</b> + the new <b>API password</b>, and click Connect.</span>
              </li>
            </ol>

            <p class="border-l-2 border-bratrax-acid/50 bg-bratrax-acid/5 px-2 py-1.5 text-[10px] text-bratrax-text-muted">
              <b class="text-bratrax-text-body">If you already have an Outbrain account manager</b>, emailing them directly is usually faster than the form — same ask: "Please enable Amplify API access on my account."
            </p>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
