<script lang="ts">
  import { requestDemo } from "$lib/bratrax/auth";

  export let open = false;
  export let onClose: () => void = () => {};

  let email = "";
  let submitting = false;
  let error = "";
  // idle → the email form; already_user → point at login. The success case
  // never renders a phase: it navigates away to the hand-off URL.
  let phase: "idle" | "already_user" = "idle";

  function close() {
    open = false;
    onClose();
    // Reset so a re-open starts fresh (invisible — the modal is already hidden).
    email = "";
    error = "";
    submitting = false;
    phase = "idle";
  }

  async function submit() {
    const value = email.trim().toLowerCase();
    if (!value || !value.includes("@")) {
      error = "Enter a valid email";
      return;
    }
    submitting = true;
    error = "";
    try {
      const res = await requestDemo(value);
      if (res.status === "already_user") {
        phase = "already_user";
      } else if (res.handoff_url) {
        // Full navigation, not goto(): the hand-off is a server redirect on the
        // Go proxy that sets the bratrax_auth cookie before landing on the demo
        // dashboards — a client-side route change would never reach it.
        //
        // Returning here deliberately leaves `submitting` true, so the button
        // stays disabled while the browser unloads. The token is single-use; a
        // second submit would mint a new one and waste this.
        window.location.href = res.handoff_url;
        return;
      } else {
        error = "Something went wrong. Try again.";
      }
    } catch (e: unknown) {
      error =
        e instanceof Error ? e.message : "Something went wrong. Try again.";
    }
    submitting = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") close();
  }

  const acidButton =
    "block w-full bg-bratrax-acid px-4 py-3 text-center font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50 disabled:hover:translate-y-0";
  const eyebrow =
    "mb-2 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid";
  const heading = "mb-3 text-2xl font-black text-bratrax-text-headline";
</script>

{#if open}
  <!-- Backdrop. Click-away + Escape close; keydown makes the overlay a11y-focusable. -->
  <div
    class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/70 p-4"
    on:click|self={close}
    on:keydown={onKeydown}
    role="presentation"
    tabindex="-1"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8"
      role="dialog"
      aria-modal="true"
      aria-label="Try the Bratrax demo"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <button
        type="button"
        on:click={close}
        aria-label="Close"
        class="absolute right-3 top-2 font-mono text-xl leading-none text-bratrax-text-muted transition-colors hover:text-bratrax-text-primary"
        >&times;</button
      >

      {#if phase === "already_user"}
        <p class={eyebrow}>You're already in</p>
        <h2 class={heading}>You already have an account</h2>
        <p class="text-sm font-light text-bratrax-text-body">
          <strong class="text-bratrax-text-primary">{email}</strong> already has
          a Bratrax account — just log in.
        </p>
        <a href="/login" class="mt-6 {acidButton}">Log in &rarr;</a>
      {:else}
        <p class={eyebrow}>Try Bratrax</p>
        <h2 class={heading}>See honest attribution on a live demo</h2>
        <p class="mb-5 text-sm font-light text-bratrax-text-body">
          Enter your email and we'll drop you straight into the Bratrax demo
          account — real dashboards, synthetic data, no password, no card.
        </p>

        {#if error}
          <div
            class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
          >
            {error}
          </div>
        {/if}

        <form on:submit|preventDefault={submit} class="flex flex-col gap-4">
          <div class="flex flex-col gap-1">
            <label
              for="demo-email"
              class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
            >
              Email
            </label>
            <!-- svelte-ignore a11y-autofocus -->
            <input
              id="demo-email"
              type="email"
              bind:value={email}
              required
              autofocus
              autocomplete="email"
              placeholder="you@example.com"
              class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            />
          </div>

          <button type="submit" disabled={submitting} class={acidButton}>
            {submitting ? "Opening…" : "Try demo →"}
          </button>
        </form>

        <p
          class="mt-4 text-center font-mono text-[11px] text-bratrax-text-muted"
        >
          Already have an account?
          <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
        </p>
      {/if}
    </div>
  </div>
{/if}
