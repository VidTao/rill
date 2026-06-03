<script lang="ts">
  import { requestPasswordReset } from "$lib/bratrax/superadmins/api";

  let email = "";
  let submitted = false;
  let loading = false;
  let error = "";

  async function handleSubmit() {
    error = "";
    loading = true;
    try {
      // Server always returns 200 — see auth.py password_reset_request.
      // We never know whether the email matched a real account; that's
      // the point (account non-enumeration).
      await requestPasswordReset(email);
      submitted = true;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-page flex h-screen w-screen items-center justify-center">
  <div class="halftone-bg"></div>

  <div
    class="login-card relative w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-8"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-8 text-center">
      <div
        class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        PASSWORD RESET
      </div>
      <h1 class="text-2xl font-bold text-bratrax-text-primary">Forgot your password?</h1>
      <p class="mt-3 font-mono text-xs text-bratrax-text-muted">
        Enter your email — we'll send a reset link.
      </p>
    </div>

    {#if submitted}
      <div
        class="border border-bratrax-acid/30 bg-bratrax-acid/10 px-3 py-3 font-mono text-xs text-bratrax-text-primary"
      >
        If <span class="font-semibold">{email}</span> is registered, you'll get
        a reset link shortly. The link expires in 1 hour.
      </div>
      <p class="mt-6 text-center font-mono text-[11px] text-bratrax-text-muted">
        <a href="/login" class="text-bratrax-acid hover:underline">Back to sign-in</a>
      </p>
    {:else}
      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
        {#if error}
          <div
            class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
          >
            {error}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label
            for="email"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Email
          </label>
          <input
            id="email"
            type="email"
            bind:value={email}
            required
            autocomplete="email"
            class="login-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            placeholder="you@example.com"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {loading ? "SENDING..." : "SEND RESET LINK →"}
        </button>
      </form>

      <p class="mt-6 text-center font-mono text-[11px] text-bratrax-text-muted">
        Remembered it?
        <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
      </p>
    {/if}
  </div>
</div>

<style>
  .login-page {
    background-color: #0a0a0a;
    position: relative;
    overflow: hidden;
  }

  .halftone-bg {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background-image: radial-gradient(
      rgba(212, 255, 0, 0.06) 1.5px,
      transparent 1.5px
    );
    background-size: 8px 8px;
  }

  .login-input {
    font-family: "Outfit", sans-serif;
  }

  .login-input::placeholder {
    color: #858585;
  }

  .login-input:focus {
    border-color: #d4ff00 !important;
    box-shadow: 0 0 0 1px rgba(212, 255, 0, 0.25) !important;
  }
</style>
