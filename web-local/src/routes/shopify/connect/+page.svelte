<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { bratraxLogin } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import {
    onboardStart,
    checkCompanyAvailable,
    createShopifyAccount,
  } from "$lib/bratrax/onboarding/api";
  import { markShopifyEmbedded } from "$lib/bratrax/shopify-embed";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
  import {
    TERMS_OF_SERVICE_URL,
    PRIVACY_POLICY_URL,
  } from "$lib/bratrax/constants";

  // Entry point for a merchant who installed Bratrax from the Shopify App
  // Store. Their store is already connected — the token was parked by
  // /shopify/install/callback before any account existed — so this screen only
  // has to establish who they are.
  //
  // Deliberately NOT /signup: that page is waitlist-gated, and someone who
  // found us on the App Store, installed, and granted scopes has already
  // chosen Bratrax. The parked install token is their authorisation.
  //
  // Two paths:
  //   new account  -> POST /shopify/install/account, log in, onboardStart
  //   existing user -> log in, then attach the shop as another store
  //
  // Either way onboardStart / the attach flow carries shopify_install_token,
  // which adopts the parked connection so /onboard/store is skipped entirely.

  $: installToken = $page.url.searchParams.get("shopify_install") ?? "";
  $: shop = $page.url.searchParams.get("shop") ?? "";
  $: shopLabel = shop.replace(/\.myshopify\.com$/i, "");

  type Tab = "create" | "login";
  let tab: Tab = "create";

  let email = "";
  let password = "";
  let companyName = "";
  let agreedToTerms = false;
  let error = "";
  let loading = false;

  let companyCheckBusy = false;
  let companyCheckError = "";
  let companyChecked = false;

  onMount(() => {
    // Everything downstream — the checkout break-out, the final dashboard —
    // keys off this. Shopify sent us here with embedded=1; make it sticky so
    // it survives the OAuth and checkout round-trips.
    markShopifyEmbedded();
  });

  $: canCreate =
    !!installToken &&
    email.includes("@") &&
    password.length >= 8 &&
    companyChecked &&
    agreedToTerms &&
    !loading;
  $: canLogin =
    !!installToken && email.includes("@") && password.length > 0 && !loading;

  async function verifyCompany() {
    const name = companyName.trim();
    companyChecked = false;
    companyCheckError = "";
    if (!name) return;
    companyCheckBusy = true;
    try {
      const res = await checkCompanyAvailable(name);
      if (res.available) {
        companyChecked = true;
      } else {
        companyCheckError =
          res.reason === "taken"
            ? "That workspace name is already in use. Try another."
            : "Use letters or numbers — that doesn't produce a valid name.";
      }
    } catch {
      companyCheckError = "Couldn't check that name right now.";
    } finally {
      companyCheckBusy = false;
    }
  }

  async function handleCreate() {
    error = "";
    loading = true;
    try {
      const res = await createShopifyAccount({
        shopifyInstallToken: installToken,
        email: email.trim().toLowerCase(),
        password,
      });
      if (res.status === "already_user") {
        tab = "login";
        error =
          "You already have a Bratrax account — log in to add this store.";
        return;
      }
      if (res.status === "already_claimed") {
        tab = "login";
        error =
          "This store is already connected to a Bratrax account. Log in to continue.";
        return;
      }
      await loginThenOnboard();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function loginThenOnboard() {
    const user = await bratraxLogin(email.trim().toLowerCase(), password);
    bratraxUser.set(user);
    queryClient.clear();

    // Adoption happens inside onboard_start: it claims the parked install and
    // writes the Shopify connection onto the brand-new client, which is what
    // lets the merchant skip /onboard/store.
    await onboardStart(companyName.trim() || shopLabel, {
      shopifyInstallToken: installToken,
    });
    await goto("/onboard/payment");
  }

  async function handleLogin() {
    error = "";
    loading = true;
    try {
      const user = await bratraxLogin(email.trim().toLowerCase(), password);
      bratraxUser.set(user);
      queryClient.clear();

      // Existing customer adding another store: promote to multi-store if
      // needed (no-op when they already are), create the sub-store, switch to
      // it, then adopt the parked install onto it. Kept as separate calls so a
      // failure at any step leaves the earlier ones intact and retryable.
      const host = get(runtime).host;
      const call = async (path: string, body?: unknown) => {
        const r = await fetch(`${host}${path}`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: body ? JSON.stringify(body) : undefined,
        });
        if (!r.ok)
          throw new Error((await r.json().catch(() => ({}))).error ?? path);
        return r.json();
      };

      await call("/bratrax/multi-client/promote");
      const store = await call("/bratrax/multi-client/add-store", {
        company_name: shopLabel,
      });
      await call("/bratrax/auth/switch-client", { client_id: store.client_id });
      await call("/bratrax/onboard/shopify/adopt", {
        shopify_install_token: installToken,
      });
      await goto("/onboard/payment");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Connect Bratrax — Shopify</title></svelte:head>

<div
  class="onboard-page flex min-h-screen w-full items-center justify-center py-10"
>
  <div class="halftone-bg"></div>

  <div
    class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6 text-center">
      <p
        class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid"
      >
        Store connected
      </p>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        {shopLabel || "Your store"} is ready
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Create your Bratrax account to finish setup. We've already got your
        store — you won't need to connect it again.
      </p>
    </div>

    {#if !installToken}
      <div
        class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
      >
        This link is missing its install reference. Please reopen Bratrax from
        your Shopify admin.
      </div>
    {:else}
      <div class="mb-5 grid grid-cols-2 gap-2">
        <button
          type="button"
          on:click={() => {
            tab = "create";
            error = "";
          }}
          class="border px-3 py-2 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors
            {tab === 'create'
            ? 'border-bratrax-acid bg-bratrax-acid/10 bratrax-acid-text'
            : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-text-muted'}"
        >
          New account
        </button>
        <button
          type="button"
          on:click={() => {
            tab = "login";
            error = "";
          }}
          class="border px-3 py-2 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors
            {tab === 'login'
            ? 'border-bratrax-acid bg-bratrax-acid/10 bratrax-acid-text'
            : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-text-muted'}"
        >
          I have one
        </button>
      </div>

      {#if error}
        <div
          class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
        >
          {error}
        </div>
      {/if}

      <form
        class="flex flex-col gap-4"
        on:submit|preventDefault={tab === "create" ? handleCreate : handleLogin}
      >
        {#if tab === "create"}
          <div class="flex flex-col gap-1">
            <label
              for="company"
              class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
            >
              Workspace name
            </label>
            <input
              id="company"
              type="text"
              bind:value={companyName}
              on:blur={verifyCompany}
              placeholder={shopLabel}
              class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            />
            {#if companyCheckBusy}
              <span class="font-mono text-[10px] text-bratrax-text-muted"
                >Checking…</span
              >
            {:else if companyCheckError}
              <span class="font-mono text-[10px] text-bratrax-tomato"
                >{companyCheckError}</span
              >
            {:else if companyChecked}
              <span class="font-mono text-[10px] bratrax-acid-text"
                >Available</span
              >
            {/if}
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
            autocomplete="email"
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="password"
            class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Password
          </label>
          <input
            id="password"
            type="password"
            bind:value={password}
            autocomplete={tab === "create"
              ? "new-password"
              : "current-password"}
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
          />
          {#if tab === "create"}
            <span class="font-mono text-[10px] text-bratrax-text-muted"
              >At least 8 characters</span
            >
          {/if}
        </div>

        {#if tab === "create"}
          <label class="flex items-start gap-2 text-xs text-bratrax-text-muted">
            <input
              type="checkbox"
              bind:checked={agreedToTerms}
              class="mt-0.5"
            />
            <span>
              I agree to the
              <a
                href={TERMS_OF_SERVICE_URL}
                target="_blank"
                rel="noopener noreferrer"
                class="bratrax-acid-text underline">Terms</a
              >
              and
              <a
                href={PRIVACY_POLICY_URL}
                target="_blank"
                rel="noopener noreferrer"
                class="bratrax-acid-text underline">Privacy Policy</a
              >.
            </span>
          </label>
        {/if}

        <button
          type="submit"
          disabled={tab === "create" ? !canCreate : !canLogin}
          class="w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 disabled:opacity-40"
        >
          {loading
            ? "Working…"
            : tab === "create"
              ? "Create account →"
              : "Log in and add store →"}
        </button>
      </form>
    {/if}
  </div>
</div>

<style>
  .onboard-page {
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

  :global(html:not(.dark)) .halftone-bg {
    background-image: none;
  }
</style>
