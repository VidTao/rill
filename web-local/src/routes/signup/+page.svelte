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
  import {
    TERMS_OF_SERVICE_URL,
    PRIVACY_POLICY_URL,
    WAITLIST_TYPEFORM_URL,
  } from "$lib/bratrax/constants";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let email = "";
  let password = "";
  let confirm = "";
  let companyName = "";
  let error = "";
  let loading = false;

  // Confirm-password match check — mirrors the accept-invite/[token] form
  // so the two account-creation surfaces validate identically. Empty
  // password means "not yet typed" — don't surface the mismatch message
  // until the user has typed something in BOTH fields.
  $: passwordsMatch = password.length > 0 && password === confirm;

  // ToS + Privacy consent — gates both the open-mode signup submit and the
  // invite-only Request Access CTA. Required on /signup regardless of mode
  // (legal requirement: explicit consent before any account creation or
  // waitlist signup).
  let agreedToTerms = false;

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

  // Waitlist state (doors-closed mode). Replaces the old Request Access modal:
  // the visitor joins inline, we store + email-confirm them, then redirect to
  // the Typeform survey.
  let waitlistEmail = "";
  let waitlistFirstName = "";
  let waitlistSubmitting = false;
  let waitlistError = "";

  onMount(async () => {
    const cfg = await getAuthConfig();
    inviteOnly = cfg.invite_only;
    configLoaded = true;
  });

  async function handleJoinWaitlist() {
    const value = waitlistEmail.trim().toLowerCase();
    if (!value || !value.includes("@")) {
      waitlistError = "Enter a valid email";
      return;
    }
    waitlistSubmitting = true;
    waitlistError = "";
    // Best-effort: the backend stores the email + first name and fires the
    // confirmation email. A failure must NOT block the survey — the Typeform is
    // the real capture. The 2xx shape is identical for new/pending/already-user,
    // so there's nothing to branch on.
    try {
      await requestAccess(value, waitlistFirstName.trim());
    } catch {
      // Non-fatal — still send them to the survey.
    }
    window.location.href = `${WAITLIST_TYPEFORM_URL}#email=${encodeURIComponent(value)}`;
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

      // Redirect to Connect your store (Screen 2)
      await goto("/onboard/store");
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
            Doors are closed for now
          </p>
          <p class="text-sm font-light text-bratrax-text-body">
            We open Bratrax in waves — we'd rather get each merchant set up right
            than throw the doors open. Join the waitlist and we'll bring you in
            when the next cohort opens.
          </p>
        </div>

        {#if waitlistError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {waitlistError}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label for="waitlist-email" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Email
          </label>
          <input
            id="waitlist-email"
            type="email"
            bind:value={waitlistEmail}
            required
            autocomplete="email"
            placeholder="you@example.com"
            class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="waitlist-first-name" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            First name <span class="normal-case tracking-normal text-bratrax-text-muted/70">(optional)</span>
          </label>
          <input
            id="waitlist-first-name"
            type="text"
            bind:value={waitlistFirstName}
            autocomplete="given-name"
            placeholder="Jane"
            class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
        </div>

        <label class="flex items-start gap-2 text-sm font-light text-bratrax-text-body">
          <input
            type="checkbox"
            bind:checked={agreedToTerms}
            class="mt-1 h-4 w-4 flex-none accent-bratrax-acid"
          />
          <span>
            I agree to the
            <a
              href={TERMS_OF_SERVICE_URL}
              target="_blank"
              rel="noreferrer noopener"
              class="underline">Terms of Service</a
            >
            and
            <a
              href={PRIVACY_POLICY_URL}
              target="_blank"
              rel="noreferrer noopener"
              class="underline">Privacy Policy</a
            >
          </span>
        </label>

        <button
          type="button"
          on:click={handleJoinWaitlist}
          disabled={!agreedToTerms || waitlistSubmitting}
          class="block w-full bg-bratrax-acid px-4 py-3 text-center font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50 disabled:hover:translate-y-0"
        >
          {waitlistSubmitting ? "Joining…" : "Join the waitlist →"}
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

        <div class="flex flex-col gap-1">
          <label for="confirm-password" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Confirm password
          </label>
          <input
            id="confirm-password"
            type="password"
            bind:value={confirm}
            required
            minlength="8"
            autocomplete="new-password"
            class="signup-input border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
          {#if confirm.length > 0 && !passwordsMatch}
            <p class="font-mono text-[10px] text-bratrax-tomato">Passwords don't match</p>
          {/if}
        </div>

        <label class="flex items-start gap-2 text-sm font-light text-bratrax-text-body">
          <input
            type="checkbox"
            bind:checked={agreedToTerms}
            class="mt-1 h-4 w-4 flex-none accent-bratrax-acid"
          />
          <span>
            I agree to the
            <a
              href={TERMS_OF_SERVICE_URL}
              target="_blank"
              rel="noreferrer noopener"
              class="underline">Terms of Service</a
            >
            and
            <a
              href={PRIVACY_POLICY_URL}
              target="_blank"
              rel="noreferrer noopener"
              class="underline">Privacy Policy</a
            >
          </span>
        </label>

        <button
          type="submit"
          disabled={loading || companyCheckBusy || !companyChecked || !!companyCheckError || !agreedToTerms || !passwordsMatch}
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

<style>
  .signup-page {
    background-color: var(--color-bg);
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

  /* The acid-dot texture is tuned for the dark canvas; hide it in light so the
     background matches the rest of the app. */
  :global(html:not(.dark)) .halftone-bg {
    background-image: none;
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
