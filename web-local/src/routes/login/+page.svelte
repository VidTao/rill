<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { bratraxLogin } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let email = "";
  let password = "";
  let error = "";
  let loading = false;

  async function handleSubmit() {
    error = "";
    loading = true;
    try {
      const { user } = await bratraxLogin(email, password);
      bratraxUser.set(user);
      // Clear stale query cache from previous session/instance.
      // Without this, dashboards can show infinite spinner instead of rendering.
      queryClient.clear();
      const redirect = $page.url.searchParams.get("redirect") || "/developer";
      await goto(redirect);
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-page flex h-screen w-screen items-center justify-center">
  <!-- Halftone texture behind the card -->
  <div class="halftone-bg"></div>

  <div class="login-card relative w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-8">
    <!-- Acid green top accent bar -->
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-8 text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        SERVICE-AS-SOFTWARE ANALYTICS
      </div>
      <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
        BRAT<span class="font-serif italic text-bratrax-acid">R</span>AX<span class="text-bratrax-acid">.</span>
      </h1>
      <p class="mt-3 font-mono text-xs text-bratrax-text-muted">
        Sign in to continue
      </p>
    </div>

    <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
      {#if error}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {error}
        </div>
      {/if}

      <div class="flex flex-col gap-1">
        <label for="email" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
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

      <div class="flex flex-col gap-1">
        <label for="password" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
          Password
        </label>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          autocomplete="current-password"
          class="login-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          placeholder="Enter your password"
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
      >
        {loading ? "SIGNING IN..." : "SIGN IN →"}
      </button>
    </form>

    <p class="mt-6 text-center font-mono text-[11px] text-bratrax-text-muted">
      Don't have an account?
      <a href="/signup" class="text-bratrax-acid hover:underline">Create one</a>
    </p>
  </div>
</div>

<style>
  .login-page {
    background-color: #0A0A0A;
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
    border-color: #D4FF00 !important;
    box-shadow: 0 0 0 1px rgba(212, 255, 0, 0.25) !important;
  }
</style>
