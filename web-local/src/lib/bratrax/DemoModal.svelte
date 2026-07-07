<script lang="ts">
  import { requestDemo } from "$lib/bratrax/auth";

  export let open = false;
  export let onClose: () => void = () => {};

  let email = "";
  let submitting = false;
  let error = "";
  // idle → the email form; sent → confirmation; already_user → point at login.
  let phase: "idle" | "sent" | "already_user" = "idle";

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
      phase = res.status === "already_user" ? "already_user" : "sent";
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : "Something went wrong. Try again.";
    } finally {
      submitting = false;
    }
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
      >&times;</button>

      {#if phase === "sent"}
        <p class={eyebrow}>Check your inbox</p>
        <h2 class={heading}>Your demo is on the way</h2>
        <p class="text-sm font-light text-bratrax-text-body">
          We sent an access link to
          <strong class="text-bratrax-text-primary">{email}</strong>. Click it,
          set a password, and you'll drop straight into a live Bratrax demo
          account — real dashboards, synthetic data.
        </p>
        <button type="button" on:click={close} class="mt-6 {acidButton}">Done</button>
      {:else if phase === "already_user"}
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
          Enter your email and we'll send instant viewer access to the Bratrax
          demo account — real dashboards, synthetic data, no card.
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
            {submitting ? "Sending…" : "Try demo →"}
          </button>
        </form>

        <p class="mt-4 text-center font-mono text-[11px] text-bratrax-text-muted">
          Already have an account?
          <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
        </p>
      {/if}
    </div>
  </div>
{/if}
