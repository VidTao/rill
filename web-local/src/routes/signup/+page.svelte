<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import {
    bratraxSignup,
    getAuthConfig,
    requestAccess,
  } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import {
    onboardStart,
    checkCompanyAvailable,
    type CompanyCheckResult,
  } from "$lib/bratrax/onboarding/api";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let email = "";
  let password = "";
  let companyName = "";
  let error = "";
  let loading = false;

  // Gate the form on the runtime ONLY_INVITATION_LINK env var. The Go proxy
  // exposes the value via /bratrax/auth/config (public). Default to gating
  // open while we wait for the response so we don't briefly flash the form
  // on slow networks then yank it.
  let configLoaded = false;
  let inviteOnly = true;

  // Company-name uniqueness check (run on blur of the company input).
  let companyCheckBusy = false;
  let companyCheckError = "";
  let companyChecked = false;

  function describeCompanyCheck(reason: CompanyCheckResult["reason"]): string {
    switch (reason) {
      case "ok":
      case "empty":
        return "";
      case "taken":
        return "That company name is already in use. Try another.";
      case "invalid":
        return "Use letters or numbers — that doesn't produce a valid name.";
      case "error":
      default:
        return "Couldn't check that name right now. Try again.";
    }
  }

  async function runCompanyCheck() {
    const value = companyName.trim();
    if (!value) return;
    if (value.length < 3) {
      companyCheckError = "Company name must be at least 3 characters.";
      companyChecked = false;
      return;
    }
    companyCheckBusy = true;
    try {
      const res = await checkCompanyAvailable(value);
      companyCheckError = describeCompanyCheck(res.reason);
      companyChecked = res.available;
    } catch {
      companyCheckError = "Couldn't check that name right now. Try again.";
      companyChecked = false;
    } finally {
      companyCheckBusy = false;
    }
  }

  function onCompanyInput() {
    // The user is editing — discard the previous check result so the
    // submit gate re-engages until they blur and we re-check.
    companyCheckError = "";
    companyChecked = false;
  }

  // Request Access modal state.
  let requestOpen = false;
  let requestEmail = "";
  let requestSubmitting = false;
  let requestSubmitted = false;
  let requestError = "";

  onMount(async () => {
    const cfg = await getAuthConfig();
    inviteOnly = cfg.invite_only;
    configLoaded = true;
  });

  function openRequestAccess() {
    requestEmail = "";
    requestError = "";
    requestSubmitted = false;
    requestOpen = true;
  }

  function closeRequestAccess() {
    requestOpen = false;
  }

  async function submitRequestAccess() {
    const value = requestEmail.trim().toLowerCase();
    if (!value || !value.includes("@")) {
      requestError = "Enter a valid email";
      return;
    }
    requestSubmitting = true;
    requestError = "";
    try {
      // The backend deliberately returns the same 2xx shape regardless of
      // whether the email is new, already pending, already approved, or
      // already a user. Treat them all as success in the UI to avoid leaking
      // who's already in the system.
      await requestAccess(value);
      requestSubmitted = true;
    } catch (e) {
      requestError = e instanceof Error ? e.message : String(e);
    } finally {
      requestSubmitting = false;
    }
  }

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
      <h1 class="flex justify-center">
        <img
          src="/img/bratrax/bratrax-logo-light.png"
          alt="Bratrax"
          class="brand-logo brand-logo-light h-8 w-auto"
        />
        <img
          src="/img/bratrax/bratrax-logo-dark.png"
          alt="Bratrax"
          class="brand-logo brand-logo-dark h-8 w-auto"
        />
      </h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Get your analytics live in under 60&nbsp;minutes
      </p>
    </div>

    {#if !configLoaded}
      <p class="text-center font-mono text-[11px] text-bratrax-text-muted">
        Loading…
      </p>
    {:else if inviteOnly}
      <div class="flex flex-col gap-4">
        <div class="border border-bratrax-acid/40 bg-bratrax-acid/10 px-4 py-3">
          <p class="mb-1 font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-acid">
            Invite-only
          </p>
          <p class="text-sm font-light text-bratrax-text-body">
            Bratrax is opening to merchants in waves. If you have a signup link, follow it to create your account. Otherwise, request access and we'll get back to you.
          </p>
        </div>

        <button
          type="button"
          on:click={openRequestAccess}
          class="block w-full bg-bratrax-acid px-4 py-3 text-center font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px"
        >
          Request access →
        </button>

        <p class="text-center font-mono text-[11px] text-bratrax-text-muted">
          Already have an account?
          <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
        </p>
      </div>
    {:else}
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
            on:input={onCompanyInput}
            on:blur={runCompanyCheck}
            required
            class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            placeholder="Your brand name"
          />
          {#if companyCheckBusy}
            <p class="font-mono text-[10px] text-bratrax-text-muted">Checking availability…</p>
          {:else if companyCheckError}
            <p class="font-mono text-[10px] text-bratrax-tomato">{companyCheckError}</p>
          {/if}
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
          disabled={loading || companyCheckBusy || !companyChecked || !!companyCheckError}
          class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {loading ? "SETTING UP..." : "GET STARTED →"}
        </button>

        <p class="text-center font-mono text-[11px] text-bratrax-text-muted">
          Already have an account?
          <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
        </p>
      </form>
    {/if}
  </div>
</div>

<!-- Request Access modal — opens from the invite-only CTA. Posts the email
     to /bratrax/access-requests; super_admin reviews on /superadmins. -->
{#if requestOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={closeRequestAccess}
    on:keydown={(e) => e.key === "Escape" && closeRequestAccess()}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      {#if !requestSubmitted}
        <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          REQUEST ACCESS
        </div>
        <h2 class="text-lg font-black text-bratrax-text-headline">
          Get on the early-access list
        </h2>
        <p class="mt-2 text-sm font-light text-bratrax-text-body">
          Drop your email and we'll send you a signup link as soon as we have
          a spot for you.
        </p>

        <div class="mt-4 flex flex-col gap-1">
          <label
            for="request-email"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Email
          </label>
          <input
            id="request-email"
            type="email"
            bind:value={requestEmail}
            placeholder="you@example.com"
            class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
        </div>

        {#if requestError}
          <div class="mt-3 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {requestError}
          </div>
        {/if}

        <div class="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            on:click={closeRequestAccess}
            class="btn-bratrax btn-neutral btn-compact"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={submitRequestAccess}
            disabled={requestSubmitting}
            class="btn-bratrax btn-primary btn-compact"
          >
            {requestSubmitting ? "Submitting…" : "Submit"}
          </button>
        </div>
      {:else}
        <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          THANKS
        </div>
        <h2 class="text-lg font-black text-bratrax-text-headline">
          You're on the list
        </h2>
        <p class="mt-2 text-sm font-light text-bratrax-text-body">
          We'll get back to you with a signup link soon. If you already have
          an invite, follow that link instead — it'll get you in faster.
        </p>

        <div class="mt-6 flex items-center justify-end">
          <button
            type="button"
            on:click={closeRequestAccess}
            class="btn-bratrax btn-primary btn-compact"
          >
            Done
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

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

  /* Filename describes the logo color, not the theme:
     dark-colored logo (brand-logo-dark) on light theme for contrast,
     light-colored logo (brand-logo-light) on dark theme. */
  .brand-logo {
    display: none;
  }
  :global(html:not(.dark)) .brand-logo-dark {
    display: block;
  }
  :global(html.dark) .brand-logo-light {
    display: block;
  }
</style>
