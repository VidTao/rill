<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import {
    getClientDetail,
    type ClientDetail,
    type ClientConnection,
  } from "$lib/bratrax/superadmins/api";

  const clientId = $page.params.client_id;

  let detail: ClientDetail | null = null;
  let topError = "";
  let loading = true;

  // The canonical onboarding progression, rendered as a breadcrumb trail.
  const STEP_FLOW = [
    "payment_pending",
    "created",
    "platforms_connected",
    "activating",
    "compiling",
    "deploying",
    "extracting",
    "ready",
  ];

  // Expected platforms for the shopify-paid-media template (Shopify + 3 ads).
  const EXPECTED_PLATFORMS = [
    { id: "shopify", label: "Shopify" },
    { id: "facebook_ads", label: "Facebook Ads" },
    { id: "google_ads", label: "Google Ads" },
    { id: "tiktok_ads", label: "TikTok Ads" },
  ];

  onMount(load);

  async function load() {
    topError = "";
    loading = true;
    try {
      detail = await getClientDetail(clientId);
    } catch (e: any) {
      topError = e?.message ?? "Failed to load client";
    } finally {
      loading = false;
    }
  }

  function fmtDate(iso: string | null): string {
    if (!iso) return "—";
    return new Date(iso).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }

  function ageLabel(hours: number | null): string {
    if (hours == null) return "—";
    if (hours < 1) return `${Math.round(hours * 60)}m ago`;
    if (hours < 48) return `${Math.round(hours)}h ago`;
    return `${Math.round(hours / 24)}d ago`;
  }

  $: connectionByPlatform = new Map<string, ClientConnection>(
    (detail?.onboarding.connected_platforms ?? [])
      .filter((c) => c && typeof c === "object")
      .map((c) => [c.platform, c]),
  );

  $: extractionLabel = (() => {
    const es = detail?.onboarding.extraction_status;
    if (!es || Object.keys(es).length === 0) return "not started";
    return `${Object.keys(es).length} stream(s) tracked`;
  })();
</script>

<div
  class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12"
>
  <div class="w-full max-w-5xl px-4">
    <a
      href="/clients"
      class="font-mono text-[11px] text-bratrax-acid hover:underline"
    >
      ← All clients
    </a>

    {#if topError}
      <div
        class="mt-4 flex items-center justify-between border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
      >
        <span>{topError}</span>
        <button type="button" on:click={load} class="underline">Retry</button>
      </div>
    {/if}

    {#if loading}
      <div class="mt-6 space-y-3">
        {#each Array(4) as _}
          <div
            class="h-28 animate-pulse border border-bratrax-border bg-bratrax-surface"
          ></div>
        {/each}
      </div>
    {:else if detail}
      <!-- Header -->
      <div class="mt-4 flex items-start justify-between">
        <div>
          <h1 class="text-2xl font-black text-bratrax-text-headline">
            {detail.company_name}
          </h1>
          <div class="mt-1 font-mono text-[11px] text-bratrax-text-muted">
            {detail.client_id} · Signed up {fmtDate(detail.signed_up_at)}
            {#if detail.timezone}· {detail.timezone}{/if}
          </div>
        </div>
        <div class="font-mono text-[11px] text-bratrax-text-muted">
          ⏱ Updated {ageLabel(detail.onboarding.step_age_hours)}
        </div>
      </div>

      <!-- Onboarding -->
      <section
        class="relative mt-6 border border-bratrax-border bg-bratrax-surface p-5"
      >
        <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>
        <h2
          class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
        >
          Onboarding
        </h2>

        <div class="flex flex-wrap items-center gap-1 font-mono text-[11px]">
          {#each STEP_FLOW as step, i}
            <span
              class:font-bold={detail.onboarding.step === step}
              class:text-bratrax-acid={detail.onboarding.step === step}
              class:text-bratrax-text-muted={detail.onboarding.step !== step}
            >
              {detail.onboarding.step === step ? "● " : ""}{step}
            </span>
            {#if i < STEP_FLOW.length - 1}<span class="text-bratrax-text-muted"
                >→</span
              >{/if}
          {/each}
          {#if detail.onboarding.step === "error"}
            <span class="font-bold text-bratrax-tomato">· ✕ error</span>
          {/if}
        </div>

        <dl
          class="mt-4 grid grid-cols-2 gap-x-6 gap-y-1 font-mono text-[11px] sm:grid-cols-3"
        >
          <div>
            <dt class="inline text-bratrax-text-muted">Template:</dt>
            {detail.onboarding.template_name ?? "—"}
          </div>
          <div>
            <dt class="inline text-bratrax-text-muted">Compile:</dt>
            {detail.onboarding.compile_status ?? "—"}
          </div>
          <div>
            <dt class="inline text-bratrax-text-muted">Deploy:</dt>
            {detail.onboarding.deploy_status ?? "—"}
          </div>
          <div>
            <dt class="inline text-bratrax-text-muted">Extraction:</dt>
            {extractionLabel}
          </div>
          <div class="col-span-2 sm:col-span-3">
            <dt class="inline text-bratrax-text-muted">Error:</dt>
            <span class:text-bratrax-tomato={!!detail.onboarding.error_message}>
              {detail.onboarding.error_message ?? "—"}
            </span>
          </div>
        </dl>
      </section>

      <!-- Subscription + Team -->
      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <section class="border border-bratrax-border bg-bratrax-surface p-5">
          <h2
            class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
          >
            Subscription
          </h2>
          <dl class="grid gap-y-1 font-mono text-[11px]">
            <div>
              <dt class="inline text-bratrax-text-muted">Status:</dt>
              <span
                class:text-bratrax-acid={detail.subscription.status ===
                  "active"}
              >
                ● {detail.subscription.status ?? "—"}
              </span>
            </div>
            <div>
              <dt class="inline text-bratrax-text-muted">Paid:</dt>
              {detail.subscription.is_paid_subscriber ? "yes" : "no"}
            </div>
            <div>
              <dt class="inline text-bratrax-text-muted">Customer ID:</dt>
              {detail.subscription.lemon_customer_id ?? "—"}
            </div>
            <div>
              <dt class="inline text-bratrax-text-muted">Subscription:</dt>
              {detail.subscription.lemon_subscription_id ?? "—"}
            </div>
            <div>
              <dt class="inline text-bratrax-text-muted">Shopify embed:</dt>
              {detail.shopify_embed_enabled ? "✓ enabled" : "not enabled"}
            </div>
          </dl>
        </section>

        <section class="border border-bratrax-border bg-bratrax-surface p-5">
          <h2
            class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
          >
            Team ({detail.team.length})
          </h2>
          <ul class="space-y-1 font-mono text-[11px]">
            {#each detail.team as m (m.id)}
              <li class="flex items-center justify-between gap-2">
                <span class="text-bratrax-text-headline"
                  >{m.name || m.email}</span
                >
                <span class="text-bratrax-text-muted">{m.email} ({m.role})</span
                >
              </li>
            {/each}
          </ul>

          {#if detail.pending_invitations.length}
            <div class="mt-3 font-mono text-[11px]">
              <div class="text-bratrax-text-muted">
                {detail.pending_invitations.length} pending invite{detail
                  .pending_invitations.length === 1
                  ? ""
                  : "s"}:
              </div>
              <ul class="mt-1 space-y-0.5">
                {#each detail.pending_invitations as inv (inv.token)}
                  <li class="text-bratrax-text-body">
                    • {inv.email} ({inv.role}, expires {fmtDate(
                      inv.expires_at,
                    )})
                  </li>
                {/each}
              </ul>
            </div>
          {/if}
        </section>
      </div>

      <!-- Connections -->
      <section class="mt-4 border border-bratrax-border bg-bratrax-surface p-5">
        <h2
          class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
        >
          Connections
        </h2>
        <ul class="space-y-1 font-mono text-[11px]">
          {#each EXPECTED_PLATFORMS as p}
            {@const conn = connectionByPlatform.get(p.id)}
            <li class="flex items-center gap-3">
              <span
                class={conn ? "text-bratrax-acid" : "text-bratrax-text-muted"}
              >
                {conn ? "✓" : "○"}
              </span>
              <span class="w-32 text-bratrax-text-headline">{p.label}</span>
              {#if conn}
                <span class="text-bratrax-text-body"
                  >{conn.account_name || conn.account_id || "connected"}</span
                >
                {#if conn.connected_at}
                  <span class="text-bratrax-text-muted"
                    >· Connected {fmtDate(conn.connected_at)}</span
                  >
                {/if}
              {:else}
                <span class="text-bratrax-text-muted">— not connected —</span>
              {/if}
            </li>
          {/each}
        </ul>
      </section>

      <!-- Email history (Project B placeholder) -->
      <section class="mt-4 border border-bratrax-border bg-bratrax-surface p-5">
        <h2
          class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
        >
          Email history
        </h2>
        <p class="font-mono text-[11px] text-bratrax-text-muted">
          No email log yet.
        </p>
      </section>
    {/if}
  </div>
</div>
