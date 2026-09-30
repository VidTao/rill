<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { bratraxLogout } from "$lib/bratrax/auth";
  import { getOnboardResumeRoute } from "$lib/bratrax/onboarding/api";
  import {
    clearResumeHint,
    consentUrl,
    decideAuthorization,
    getPendingAuthorization,
    saveResumeHint,
    type OAuthPendingView,
  } from "$lib/bratrax/oauth-connect";

  // Where an AI assistant's "Connect" lands (see lib/bratrax/oauth-connect.ts).
  // Approving issues the one-time code and sends the browser back to the
  // assistant; a workspace mid-onboarding is sent to finish onboarding first.

  let resume = "";
  let loading = true;
  let error = "";
  let view: OAuthPendingView | null = null;
  let selectedClientId = "";
  let submitting = false;
  let redirecting = "";

  $: readyWorkspaces = view?.workspaces.filter((w) => w.ready) ?? [];

  onMount(async () => {
    resume = $page.url.searchParams.get("resume") ?? "";
    if (!resume) {
      error = "This link is missing its connection request. Go back to your AI assistant and click Connect again.";
      loading = false;
      return;
    }
    try {
      const res = await getPendingAuthorization(resume);
      if (!res.view) {
        clearResumeHint();
        error =
          res.status === 403
            ? "This connection request was started by a different Bratrax account. Sign out and sign in with the account you want to connect."
            : "This connection request has expired or was already used. Go back to your AI assistant and click Connect again.";
        return;
      }
      view = res.view;
      if (view.status === "needs_onboarding" && view.role !== "viewer") {
        // Finish onboarding first; the layout guard brings them back here.
        saveResumeHint(resume);
        redirecting = "Let's finish setting up your workspace first. You'll come back here automatically.";
        setTimeout(() => {
          goto(getOnboardResumeRoute(view?.step) ?? "/onboard/store");
        }, 1200);
        return;
      }
      selectedClientId = view.default_client_id;
    } catch {
      error = "We couldn't load this connection request. Check your connection and refresh the page.";
    } finally {
      loading = false;
    }
  });

  async function decide(approve: boolean) {
    submitting = true;
    error = "";
    try {
      const target = await decideAuthorization(resume, approve, selectedClientId);
      clearResumeHint();
      redirecting = approve ? `Connected. Returning you to ${view?.client_name ?? "your assistant"}…` : "Cancelled.";
      window.location.href = target;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      submitting = false;
    }
  }

  async function switchAccount() {
    await bratraxLogout();
    window.location.href = `/login?redirect=${encodeURIComponent(consentUrl(resume))}`;
  }
</script>

<div class="consent-page flex min-h-screen w-screen items-center justify-center px-4 py-10">
  <div class="halftone-bg"></div>

  <div class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6 text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        Connect
      </div>
      <h1 class="flex justify-center">
        <img src="/img/bratrax/bratrax-logo-light.png" alt="Bratrax" class="brand-logo brand-logo-light h-8 w-auto" />
        <img src="/img/bratrax/bratrax-logo-dark.png" alt="Bratrax" class="brand-logo brand-logo-dark h-8 w-auto" />
      </h1>
    </div>

    {#if loading}
      <p class="text-center font-mono text-[11px] text-bratrax-text-muted">Loading…</p>
    {:else if redirecting}
      <p class="text-center text-sm text-bratrax-text-body">{redirecting}</p>
    {:else if error && !view}
      <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 text-sm text-bratrax-text-primary">
        {error}
      </div>
      <p class="mt-4 text-center font-mono text-[11px] text-bratrax-text-muted">
        <a href="/developer" class="text-bratrax-acid hover:underline">Go to Bratrax</a>
      </p>
    {:else if view}
      <h2 class="mb-2 text-center text-lg font-semibold text-bratrax-text-primary">
        Connect {view.client_name} to Bratrax
      </h2>
      <p class="mb-6 text-center font-mono text-[11px] text-bratrax-text-muted">
        Signed in as {view.email} ·
        <button type="button" class="text-bratrax-acid hover:underline" on:click={switchAccount}>Not you?</button>
      </p>

      {#if view.status === "no_workspace"}
        <div class="border border-bratrax-border px-4 py-3 text-sm text-bratrax-text-body">
          Your account doesn't have a workspace yet. Contact support@bratrax.com and we'll get you set up.
        </div>
      {:else if view.status === "needs_onboarding"}
        <!-- Only viewers reach this: they can't run onboarding themselves. -->
        <div class="border border-bratrax-border px-4 py-3 text-sm text-bratrax-text-body">
          Your workspace hasn't finished setup yet. Once your workspace admin completes it, go back to
          {view.client_name} and click Connect again.
        </div>
      {:else}
        {#if error}
          <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {error}
          </div>
        {/if}

        {#if view.workspaces.length > 1}
          <fieldset class="mb-5">
            <legend class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
              Store to connect
            </legend>
            <div class="flex flex-col gap-2">
              {#each view.workspaces as ws (ws.client_id)}
                <label
                  class="flex items-center gap-3 border px-3 py-2 text-sm {ws.ready
                    ? 'cursor-pointer border-bratrax-border text-bratrax-text-primary'
                    : 'border-bratrax-border/50 text-bratrax-text-muted'}"
                >
                  <input
                    type="radio"
                    name="workspace"
                    value={ws.client_id}
                    bind:group={selectedClientId}
                    disabled={!ws.ready}
                    class="accent-bratrax-acid"
                  />
                  <span class="flex-1">{ws.company_name}</span>
                  {#if !ws.ready}<span class="font-mono text-[10px] uppercase">Setup not finished</span>{/if}
                </label>
              {/each}
            </div>
          </fieldset>
        {:else if readyWorkspaces.length === 1}
          <p class="mb-5 text-sm text-bratrax-text-body">
            Workspace: <span class="font-semibold text-bratrax-text-primary">{readyWorkspaces[0].company_name}</span>
          </p>
        {/if}

        <div class="mb-6 text-sm text-bratrax-text-body">
          <p class="mb-2">{view.client_name} will be able to:</p>
          <ul class="list-disc space-y-1 pl-5">
            <li>Read your store, ad and attribution metrics</li>
            <li>See your dashboards and chart your data</li>
            <li>Read the business notes saved in your workspace</li>
          </ul>
          <p class="mt-3 text-bratrax-text-muted">
            It can't change your connectors, settings or billing. You can disconnect it any time in
            Settings → MCP.
          </p>
        </div>

        <div class="flex gap-3">
          <button
            type="button"
            disabled={submitting}
            on:click={() => decide(false)}
            class="flex-1 border border-bratrax-border px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-text-primary transition-opacity hover:opacity-80 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={submitting || !selectedClientId}
            on:click={() => decide(true)}
            class="flex-1 bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:-translate-y-px hover:opacity-90 disabled:opacity-50"
          >
            {submitting ? "Connecting…" : "Connect →"}
          </button>
        </div>
        {#if view.redirect_host}
          <p class="mt-4 text-center font-mono text-[10px] text-bratrax-text-muted">
            You'll be sent back to {view.redirect_host}
          </p>
        {/if}
      {/if}
    {/if}
  </div>
</div>

<style>
  .consent-page {
    background-color: var(--color-bg);
    position: relative;
    overflow: hidden;
  }

  .halftone-bg {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background-image: radial-gradient(rgba(212, 255, 0, 0.06) 1.5px, transparent 1.5px);
    background-size: 8px 8px;
  }

  :global(html:not(.dark)) .halftone-bg {
    background-image: none;
  }

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
