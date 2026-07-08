<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import {
    bratraxLogin,
    getStoreSignupLink,
    consumeStoreSignupLink,
  } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import {
    onboardStart,
    checkCompanyAvailable,
    type CompanyCheckResult,
  } from "$lib/bratrax/onboarding/api";
  import { TERMS_OF_SERVICE_URL, PRIVACY_POLICY_URL } from "$lib/bratrax/constants";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  // Public entry for a store-locked, multi-use signup link. Unlike accept-invite
  // (prefilled, single-use email), the user types their OWN email; the link only
  // carries the store platform + a usage cap. Store lock flows to onboarding via
  // onboardStart({ lockedStore }) → stack_selections.locked_store_platform, which
  // /onboard/store reads to preselect + disable the other store.
  $: token = $page.params.token;

  let linkLoading = true;
  let linkValid = false;
  let storePlatform: "shopify" | "woocommerce" | "" = "";

  let email = "";
  let password = "";
  let confirm = "";
  let companyName = "";
  let agreedToTerms = false;
  let error = "";
  let loading = false;

  let companyCheckBusy = false;
  let companyCheckError = "";
  let companyChecked = false;

  $: passwordsMatch = password.length > 0 && password === confirm;
  $: storeLabel = storePlatform === "woocommerce" ? "WooCommerce" : "Shopify";
  $: canSubmit =
    linkValid &&
    email.includes("@") &&
    password.length >= 8 &&
    passwordsMatch &&
    companyChecked &&
    agreedToTerms &&
    !loading;

  onMount(async () => {
    try {
      const info = await getStoreSignupLink(token);
      linkValid = info.valid;
      storePlatform = info.store_platform ?? "";
    } catch {
      linkValid = false;
    } finally {
      linkLoading = false;
    }
  });

  function describeCompanyCheck(reason: CompanyCheckResult["reason"]): string {
    switch (reason) {
      case "ok":
      case "empty":
        return "";
      case "taken":
        return "That company name is already in use. Try another.";
      case "invalid":
        return "Use letters or numbers — that doesn't produce a valid name.";
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
    companyCheckError = "";
    companyChecked = false;
  }

  async function handleSubmit() {
    if (!canSubmit) return;
    const value = email.trim().toLowerCase();
    error = "";
    loading = true;
    try {
      // 1) Reserve a slot AND create the account server-side (Flask bypasses the
      //    invite-only /auth/signup gate). 2) Log in for the JWT cookie.
      const { store_platform } = await consumeStoreSignupLink(
        token,
        value,
        password,
        companyName.trim(),
      );
      const { user } = await bratraxLogin(value, password);
      bratraxUser.set(user);
      queryClient.clear();
      // 3) Create the client, stamping the store lock so /onboard/store locks it.
      const started = await onboardStart(companyName.trim(), {
        lockedStore: store_platform,
      });
      sessionStorage.setItem("onboard_client_id", started.client_id);
      sessionStorage.setItem("onboard_client_name", started.client_name);
      await goto("/onboard/store");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg === "already_user") {
        error = "An account with that email already exists — please sign in.";
      } else if (msg === "link_full_or_expired") {
        error = "This signup link is full or has expired.";
        linkValid = false;
      } else {
        error = msg || "Something went wrong. Try again.";
      }
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Join Bratrax</title>
</svelte:head>

<div class="flex h-screen w-screen items-center justify-center bg-bratrax-bg p-4">
  <div class="relative w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    {#if linkLoading}
      <p class="text-center font-mono text-[11px] text-bratrax-text-muted">Loading…</p>
    {:else if !linkValid}
      <div class="text-center">
        <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato">
          Link unavailable
        </div>
        <h1 class="mb-3 text-xl font-black text-bratrax-text-headline">
          This signup link isn't available
        </h1>
        <p class="text-sm font-light text-bratrax-text-body">
          It may have reached its limit or expired. Ask whoever shared it for a fresh one.
        </p>
        <a
          href="/login"
          class="mt-6 inline-block font-mono text-[11px] text-bratrax-acid hover:underline"
        >
          Already have an account? Sign in
        </a>
      </div>
    {:else}
      <div class="mb-6 text-center">
        <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          {storeLabel} store
        </div>
        <h1 class="text-2xl font-black text-bratrax-text-headline">
          Create your Bratrax account
        </h1>
        <p class="mt-2 text-sm font-light text-bratrax-text-body">
          Get your analytics live in under 60&nbsp;minutes.
        </p>
      </div>

      {#if error}
        <div
          class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
        >
          {error}
        </div>
      {/if}

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label
            for="join-email"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Email
          </label>
          <input
            id="join-email"
            type="email"
            bind:value={email}
            required
            autocomplete="email"
            placeholder="you@example.com"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="join-company"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Company name
          </label>
          <input
            id="join-company"
            type="text"
            bind:value={companyName}
            required
            autocomplete="organization"
            on:input={onCompanyInput}
            on:blur={runCompanyCheck}
            placeholder="Acme Co"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
          {#if companyCheckBusy}
            <p class="font-mono text-[11px] text-bratrax-text-muted">Checking…</p>
          {:else if companyCheckError}
            <p class="font-mono text-[11px] text-bratrax-tomato">{companyCheckError}</p>
          {/if}
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="join-password"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Password
          </label>
          <input
            id="join-password"
            type="password"
            bind:value={password}
            required
            autocomplete="new-password"
            placeholder="At least 8 characters"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="join-confirm"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Confirm password
          </label>
          <input
            id="join-confirm"
            type="password"
            bind:value={confirm}
            required
            autocomplete="new-password"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
          {#if confirm.length > 0 && !passwordsMatch}
            <p class="font-mono text-[11px] text-bratrax-tomato">Passwords don't match.</p>
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
            <a href={TERMS_OF_SERVICE_URL} target="_blank" rel="noreferrer noopener" class="underline">Terms of Service</a>
            and
            <a href={PRIVACY_POLICY_URL} target="_blank" rel="noreferrer noopener" class="underline">Privacy Policy</a>
          </span>
        </label>

        <button
          type="submit"
          disabled={!canSubmit}
          class="block w-full bg-bratrax-acid px-4 py-3 text-center font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 disabled:opacity-50"
        >
          {loading ? "Creating…" : "Create account →"}
        </button>

        <p class="text-center font-mono text-[11px] text-bratrax-text-muted">
          Already have an account?
          <a href="/login" class="text-bratrax-acid hover:underline">Sign in</a>
        </p>
      </form>
    {/if}
  </div>
</div>
