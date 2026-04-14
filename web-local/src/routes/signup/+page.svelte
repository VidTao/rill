<script lang="ts">
  import { goto } from "$app/navigation";
  import { bratraxSignup } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { onboardStart } from "$lib/bratrax/onboarding/api";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let email = "";
  let password = "";
  let companyName = "";
  let error = "";
  let loading = false;

  async function handleSubmit() {
    error = "";
    loading = true;
    try {
      // Step 1: Create user + get JWT cookie
      const { user } = await bratraxSignup(email, password, companyName);
      bratraxUser.set(user);
      queryClient.clear();

      // Step 2: Create client + CH database + template
      const result = await onboardStart(companyName);

      // Store client_id for subsequent onboarding steps
      sessionStorage.setItem("onboard_client_id", result.client_id);
      sessionStorage.setItem("onboard_client_name", result.client_name);

      // Redirect to Connect Shopify (Screen 2)
      await goto("/onboard/shopify");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="signup-page flex h-screen w-screen items-center justify-center">
  <!-- Halftone texture behind the card -->
  <div class="halftone-bg"></div>

  <div class="relative w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-8">
    <!-- Acid green top accent bar -->
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-8 text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        SERVICE-AS-SOFTWARE ANALYTICS
      </div>
      <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
        BRAT<span class="font-serif italic text-bratrax-acid">R</span>AX<span class="text-bratrax-acid">.</span>
      </h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Get your analytics live in under 60&nbsp;minutes
      </p>
    </div>

    <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
      {#if error}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {error}
        </div>
      {/if}

      <div class="flex flex-col gap-1">
        <label for="company" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
          Company name
        </label>
        <input
          id="company"
          type="text"
          bind:value={companyName}
          required
          class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          placeholder="Your brand name"
        />
      </div>

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
          class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
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
          minlength="8"
          autocomplete="new-password"
          class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          placeholder="8+ characters"
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
      >
        {loading ? "SETTING UP..." : "GET STARTED →"}
      </button>

      <p class="text-center font-mono text-[11px] text-bratrax-text-muted">
        Already have an account?
        <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
      </p>
    </form>
  </div>
</div>

<style>
  .signup-page {
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

  .signup-input {
    font-family: "Outfit", sans-serif;
  }

  .signup-input::placeholder {
    color: #858585;
  }

  .signup-input:focus {
    border-color: #D4FF00 !important;
    box-shadow: 0 0 0 1px rgba(212, 255, 0, 0.25) !important;
  }
</style>
