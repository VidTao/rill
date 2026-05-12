<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { acceptInvitation, getInvitation } from "$lib/bratrax/settings/api";
  import type { InvitationPreview } from "$lib/bratrax/settings/types";
  import { bratraxLogin } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import {
    onboardStart,
    checkCompanyAvailable,
    type CompanyCheckResult,
  } from "$lib/bratrax/onboarding/api";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let invite: InvitationPreview | null = null;
  let loadError = "";
  let loading = true;

  let name = "";
  let password = "";
  let confirm = "";
  let companyName = "";
  let submitError = "";
  let submitting = false;

  $: token = $page.params.token;

  $: isSignup = invite?.kind === "signup";
  $: passwordsMatch = password.length > 0 && password === confirm;
  $: canSubmit =
    !!invite &&
    !invite.expired &&
    !invite.accepted &&
    password.length >= 8 &&
    passwordsMatch &&
    (!isSignup ||
      (companyName.trim().length > 0 &&
        companyChecked &&
        !companyCheckError &&
        !companyCheckBusy));

  // Company-name uniqueness check (signup-kind invites only).
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

  onMount(async () => {
    try {
      invite = await getInvitation(token);
    } catch (e: any) {
      loadError = e.message ?? "Invitation not found";
    } finally {
      loading = false;
    }
  });

  async function submit() {
    if (!canSubmit || !invite) return;
    submitError = "";
    submitting = true;
    try {
      const result = await acceptInvitation(
        token,
        password,
        name.trim(),
        isSignup ? companyName.trim() : undefined,
      );

      // Signup invites: log the new user in immediately and run onboard_start
      // so they land on /onboard/shopify with a real client. Doing this in one
      // submit handler keeps the user from getting orphaned (account exists,
      // no client yet) if they navigate away before completing onboarding.
      if (result.kind === "signup") {
        const { user } = await bratraxLogin(result.email, password);
        bratraxUser.set(user);
        queryClient.clear();
        // Forward requires_payment from the accept response so the new
        // rill_clients row is stamped with the right is_paid_subscriber. If
        // the invite was inceptly (requires_payment=false), the user skips
        // /onboard/payment and goes straight to /onboard/shopify.
        const started = await onboardStart(
          result.company_name ?? companyName.trim(),
          { requiresPayment: result.requires_payment ?? true },
        );
        sessionStorage.setItem("onboard_client_id", started.client_id);
        sessionStorage.setItem("onboard_client_name", started.client_name);
        // Layout's resume-onboarding logic will redirect to /onboard/payment
        // for paying users (step=payment_pending) or stay on /onboard/shopify
        // for inceptly users (step=created).
        await goto("/onboard/shopify");
        return;
      }

      // team / superadmin invites: existing flow — user logs in manually.
      await goto("/login?invited=1");
    } catch (e: any) {
      submitError = e.message ?? "Failed to accept invitation";
    } finally {
      submitting = false;
    }
  }

  function roleLabel(r: string): string {
    if (r === "admin") return "Admin";
    if (r === "viewer") return "Viewer";
    if (r === "super_admin") return "Super admin";
    return r;
  }
</script>

<div class="flex h-screen w-screen items-center justify-center bg-bratrax-bg">
  <div class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    {#if loading}
      <p class="font-mono text-xs text-bratrax-text-muted">Loading invitation…</p>
    {:else if loadError}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Invitation
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">Link not valid</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">{loadError}</p>
      <a
        href="/login"
        class="mt-6 inline-block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-acid hover:underline"
      >
        Go to login →
      </a>
    {:else if invite && invite.accepted}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
        Already accepted
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">This invitation has already been used</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Log in with the email <strong>{invite.email}</strong>, or ask support@bratrax.com
        for a fresh invite.
      </p>
      <a
        href="/login"
        class="mt-6 inline-block bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
      >
        Go to login
      </a>
    {:else if invite && invite.expired}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Expired
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">This invitation has expired</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Ask {invite.inviter_name || "your admin"} at <strong>{invite.company_name}</strong> for a new invite link.
      </p>
    {:else if invite}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        {isSignup ? "Welcome to Bratrax" : "You're invited"}
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">
        {#if isSignup}
          Create your account
        {:else}
          Join {invite.company_name}
        {/if}
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        {#if isSignup}
          Set a password and your company name to start onboarding{invite.inviter_name ? `, invited by ${invite.inviter_name}` : ""}.
        {:else}
          Joining as <strong>{roleLabel(invite.role)}</strong>{invite.inviter_name ? `, invited by ${invite.inviter_name}` : ""}.
        {/if}
      </p>

      <form on:submit|preventDefault={submit} class="mt-6 flex flex-col gap-3">
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Email
          </label>
          <input
            type="email"
            value={invite.email}
            readonly
            class="w-full cursor-not-allowed border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-muted opacity-80"
          />
        </div>
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Your name
          </label>
          <input
            type="text"
            bind:value={name}
            placeholder="Joe Smith"
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          />
        </div>
        {#if isSignup}
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Company name
            </label>
            <input
              type="text"
              bind:value={companyName}
              on:input={onCompanyInput}
              on:blur={runCompanyCheck}
              placeholder="Your brand name"
              required
              class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
            />
            {#if companyCheckBusy}
              <p class="mt-1 font-mono text-[10px] text-bratrax-text-muted">Checking availability…</p>
            {:else if companyCheckError}
              <p class="mt-1 font-mono text-[10px] text-bratrax-tomato">{companyCheckError}</p>
            {/if}
          </div>
        {/if}
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Password
          </label>
          <input
            type="password"
            bind:value={password}
            placeholder="At least 8 characters"
            minlength="8"
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          />
        </div>
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Confirm password
          </label>
          <input
            type="password"
            bind:value={confirm}
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          />
          {#if confirm.length > 0 && !passwordsMatch}
            <p class="mt-1 font-mono text-[10px] text-bratrax-tomato">Passwords don't match</p>
          {/if}
        </div>

        {#if submitError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {submitError}
          </div>
        {/if}

        <button
          type="submit"
          disabled={!canSubmit || submitting}
          class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {#if submitting}
            {isSignup ? "Setting up…" : "Creating account…"}
          {:else}
            {isSignup ? "Create account & start →" : "Accept invitation"}
          {/if}
        </button>
      </form>
    {/if}
  </div>
</div>
