<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { resetPassword } from "$lib/bratrax/superadmins/api";

  let password = "";
  let confirm = "";
  let error = "";
  let loading = false;

  // Token comes from the email link: /reset-password?t=<token>
  $: token = $page.url.searchParams.get("t") || "";

  async function handleSubmit() {
    error = "";
    if (password.length < 8 || password.length > 72) {
      error = "Password must be 8 to 72 characters.";
      return;
    }
    if (password !== confirm) {
      error = "Passwords don't match.";
      return;
    }
    if (!token) {
      error = "Missing reset token — open the link from your email.";
      return;
    }
    loading = true;
    try {
      await resetPassword(token, password);
      await goto("/login?reset=1");
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
        SET A NEW PASSWORD
      </div>
      <h1 class="text-2xl font-bold text-bratrax-text-primary">Reset password</h1>
      <p class="mt-3 font-mono text-xs text-bratrax-text-muted">
        Choose a strong password you'll remember.
      </p>
    </div>

    {#if !token}
      <div
        class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-3 font-mono text-xs text-bratrax-tomato"
      >
        This page expects a reset token in the URL. Open the link from your
        password-reset email.
      </div>
      <p class="mt-6 text-center font-mono text-[11px] text-bratrax-text-muted">
        <a href="/forgot-password" class="text-bratrax-acid hover:underline"
          >Request a new reset link</a
        >
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
            for="password"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            New password
          </label>
          <input
            id="password"
            type="password"
            bind:value={password}
            required
            autocomplete="new-password"
            minlength="8"
            maxlength="72"
            class="login-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            placeholder="At least 8 characters"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="confirm"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Confirm password
          </label>
          <input
            id="confirm"
            type="password"
            bind:value={confirm}
            required
            autocomplete="new-password"
            minlength="8"
            maxlength="72"
            class="login-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            placeholder="Re-enter the password"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {loading ? "UPDATING..." : "UPDATE PASSWORD →"}
        </button>
      </form>
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
