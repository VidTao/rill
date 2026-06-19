<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import {
    getClientDetail,
    enableMultiStoreForClient,
    type ClientDetail,
    type ClientConnection,
    type ClientStatus,
  } from "$lib/bratrax/superadmins/api";
  import ClientEmailAutomations from "$lib/bratrax/superadmins/ClientEmailAutomations.svelte";
  import ClientEmailHistory from "$lib/bratrax/superadmins/ClientEmailHistory.svelte";

  const clientId = $page.params.client_id;

  let detail: ClientDetail | null = null;
  let topError = "";
  let loading = true;

  // Multi-store promotion — same action as the /clients row "Enable multi-store".
  let confirmOpen = false;
  let enabling = false;
  async function confirmEnable() {
    if (!detail) return;
    enabling = true;
    topError = "";
    try {
      const result = await enableMultiStoreForClient(detail.client_id);
      detail = { ...detail, multi_client_id: result.multi_client_id };
      confirmOpen = false;
    } catch (e: any) {
      topError = e?.message ?? "Failed to enable multi-store";
    } finally {
      enabling = false;
    }
  }

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

  const STATUS_ICON: Record<ClientStatus, string> = {
    healthy: "●",
    running: "▶",
    waiting: "⏸",
    needs_handoff: "⚠",
    stuck: "⚠",
    error: "✕",
    cancelled: "◐",
    expired: "◌",
  };
  const STATUS_LABEL: Record<ClientStatus, string> = {
    healthy: "Healthy",
    running: "Running",
    waiting: "Waiting",
    needs_handoff: "Needs handoff",
    stuck: "Stuck",
    error: "Error",
    cancelled: "Cancelled",
    expired: "Expired",
  };

  // Page-level state pill — color-matched to the status (mockup §§2-5).
  function pillStyle(s: ClientStatus): string {
    switch (s) {
      case "needs_handoff":
        return "background: var(--bratrax-acid); color: #000;";
      case "error":
      case "stuck":
        return "background: var(--bratrax-tomato); color: #000;";
      case "cancelled":
        return "background: var(--bratrax-lavender); color: #000;";
      case "expired":
        return "background: var(--bratrax-gray); color: var(--bratrax-text-headline);";
      case "healthy":
        return "background: var(--color-acid-dim); color: var(--color-acid-text); border: 1px solid var(--color-acid-mid);";
      case "running":
        return "background: transparent; color: var(--bratrax-cyan); border: 1px solid var(--bratrax-cyan);";
      default: // waiting
        return "background: var(--bratrax-bg); color: var(--bratrax-text-muted); border: 1px solid var(--bratrax-border);";
    }
  }

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

  function relativeTime(iso: string | null): string {
    if (!iso) return "— tracking pending";
    const then = new Date(iso).getTime();
    if (Number.isNaN(then)) return "— tracking pending";
    const days = Math.floor((Date.now() - then) / 86400000);
    if (days <= 0) return "today";
    if (days === 1) return "yesterday";
    if (days < 30) return `${days} days ago`;
    return fmtDate(iso);
  }

  // Normalize the external-landing-pages alias: connected_platforms stores
  // either "external_pages" (legacy) or "funnelish" (builder sub-picker); the
  // registry id is "external_pages". Mirrors the /onboard/me read-time rewrite.
  $: connectionByPlatform = new Map<string, ClientConnection>(
    (detail?.onboarding.connected_platforms ?? [])
      .filter((c) => c && typeof c === "object")
      .map((c) => [
        c.platform === "funnelish" ? "external_pages" : c.platform,
        c,
      ]),
  );

  $: primaryIntegrations =
    detail?.supported_integrations?.filter((i) => i.category === "primary") ??
    [];
  $: optionalIntegrations =
    detail?.supported_integrations?.filter((i) => i.category === "optional") ??
    [];

  // Card top-accent colors, varying by each card's contribution to state.
  $: onboardingAccent = !detail
    ? "transparent"
    : detail.onboarding.step === "error"
      ? "var(--bratrax-tomato)"
      : detail.status === "needs_handoff" || detail.onboarding.step === "ready"
        ? "var(--bratrax-acid)"
        : "var(--bratrax-border)";

  $: subscriptionAccent = !detail
    ? "transparent"
    : detail.status === "cancelled"
      ? "var(--bratrax-lavender)"
      : detail.status === "expired"
        ? "var(--bratrax-gray)"
        : detail.subscription.is_paid_subscriber &&
            detail.subscription.status === "active"
          ? "var(--bratrax-acid)"
          : "var(--bratrax-border)";

  $: allPrimaryConnected =
    primaryIntegrations.length > 0 &&
    primaryIntegrations.every((i) => connectionByPlatform.has(i.id));
  $: connectionsAccent = allPrimaryConnected
    ? "var(--bratrax-acid)"
    : "var(--bratrax-border)";

  $: stepIndex = detail ? STEP_FLOW.indexOf(detail.onboarding.step ?? "") : -1;

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
      class="font-mono text-[11px] uppercase tracking-[2px] text-bratrax-text-muted hover:text-bratrax-acid"
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
      <div class="mt-4 flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="text-2xl font-black text-bratrax-text-headline">
            {detail.company_name}
          </h1>
          <div class="mt-1 font-mono text-[11px] text-bratrax-text-muted">
            {detail.client_id} · Signed up {fmtDate(detail.signed_up_at)}
            {#if detail.timezone}· {detail.timezone}{/if}
            · Shopify embed {detail.shopify_embed_enabled
              ? "enabled"
              : "not enabled"}
          </div>
          <!-- Page-level state pill -->
          <div
            class="mt-3 inline-block font-mono text-[12px] font-bold uppercase tracking-[1.5px]"
            style={`${pillStyle(detail.status)} padding: 5px 10px;`}
          >
            {STATUS_ICON[detail.status]}
            {STATUS_LABEL[detail.status]}{detail.status_sub_label
              ? ` · ${detail.status_sub_label}`
              : ""}
          </div>
        </div>

        <!-- Right-side action: enable multi-store (or badge when already on) -->
        <div class="shrink-0 pt-1">
          {#if detail.multi_client_id}
            <span
              class="inline-block border border-bratrax-acid/40 px-2 py-1 font-mono text-[10px] uppercase tracking-[1px] text-bratrax-acid"
            >
              Multi-store
            </span>
          {:else}
            <button
              type="button"
              on:click={() => (confirmOpen = true)}
              class="btn-bratrax btn-primary btn-compact"
            >
              Enable multi-store
            </button>
          {/if}
        </div>
      </div>

      <!-- Onboarding -->
      <section
        class="relative mt-6 border border-bratrax-border bg-bratrax-surface p-5"
      >
        <div
          class="absolute left-0 right-0 top-0 h-1"
          style={`background: ${onboardingAccent}`}
        ></div>
        <h2
          class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
        >
          Onboarding
        </h2>

        <div class="flex flex-wrap items-center gap-1 font-mono text-[11px]">
          {#each STEP_FLOW as step, i}
            <span
              class:font-bold={i === stepIndex}
              class:text-bratrax-acid={i === stepIndex &&
                detail.onboarding.step !== "error"}
              class:text-bratrax-text-body={i < stepIndex}
              class:text-bratrax-text-muted={i > stepIndex}
            >
              {i === stepIndex ? "● " : ""}{step}
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
          {#if detail.onboarding.step_age_hours != null}
            <div>
              <dt class="inline text-bratrax-text-muted">Step age:</dt>
              {Math.round(detail.onboarding.step_age_hours)} hours
            </div>
          {/if}
          <div>
            <dt class="inline text-bratrax-text-muted">Error:</dt>
            {detail.onboarding.error_message ?? "—"}
          </div>
        </dl>
      </section>

      <!-- Connections — registry-driven, two columns (primary | optional) -->
      <section
        class="relative mt-4 border border-bratrax-border bg-bratrax-surface p-5"
      >
        <div
          class="absolute left-0 right-0 top-0 h-1"
          style={`background: ${connectionsAccent}`}
        ></div>
        <h2
          class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
        >
          Connections
        </h2>
        <div class="grid gap-x-8 gap-y-1 md:grid-cols-2">
          {#each [primaryIntegrations, optionalIntegrations] as column}
            <ul class="space-y-1 font-mono text-[11px]">
              {#each column as integ (integ.id)}
                {@const conn = connectionByPlatform.get(integ.id)}
                <li class="flex items-baseline gap-3 py-1">
                  <span
                    class={conn
                      ? "text-bratrax-acid"
                      : "text-bratrax-text-muted"}
                  >
                    {conn ? "✓" : "○"}
                  </span>
                  <span
                    class="w-40 shrink-0 font-bold text-bratrax-text-headline"
                  >
                    {integ.display_name}
                  </span>
                  <span class="flex-1">
                    {#if conn}
                      <span class="text-bratrax-text-body">
                        {conn.account_name || conn.account_id || "connected"}
                      </span>
                      {#if conn.connected_at}
                        <span class="block text-bratrax-text-muted"
                          >Connected {fmtDate(conn.connected_at)}</span
                        >
                      {/if}
                    {:else}
                      <span class="text-bratrax-text-muted"
                        >— not connected —</span
                      >
                    {/if}
                  </span>
                </li>
              {/each}
            </ul>
          {/each}
        </div>
      </section>

      <!-- Subscription + Team -->
      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <section
          class="relative border border-bratrax-border bg-bratrax-surface p-5"
        >
          <div
            class="absolute left-0 right-0 top-0 h-1"
            style={`background: ${subscriptionAccent}`}
          ></div>
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
                {detail.subscription.status ?? "—"}
              </span>
            </div>
            {#if detail.status === "cancelled" || detail.status === "expired"}
              <div>
                <dt class="inline text-bratrax-text-muted">Access ends:</dt>
                {fmtDate(detail.subscription_ends_at)}
              </div>
            {:else}
              <div>
                <dt class="inline text-bratrax-text-muted">Next billing:</dt>
                {fmtDate(detail.subscription_renews_at)}
              </div>
            {/if}
            <div>
              <dt class="inline text-bratrax-text-muted">Customer ID:</dt>
              {detail.subscription.lemon_customer_id ?? "—"}
            </div>
            <div>
              <dt class="inline text-bratrax-text-muted">Subscription:</dt>
              {detail.subscription.lemon_subscription_id ?? "—"}
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

          <div
            class="mt-4 border-t border-dashed border-bratrax-border pt-3 font-mono text-[11px] text-bratrax-text-muted"
          >
            Last sign-in (any user):
            <span class:text-bratrax-lavender={!detail.workspace_last_seen}>
              {relativeTime(detail.workspace_last_seen)}
            </span>
          </div>
        </section>
      </div>

      <!-- Email automations + Email history (Phase 4c sections) -->
      <ClientEmailAutomations {clientId} />
      <ClientEmailHistory {clientId} />

      <!-- Confirm modal (multi-store promotion) -->
      {#if confirmOpen}
        <div
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
          on:click={() => !enabling && (confirmOpen = false)}
          on:keydown={(e) =>
            e.key === "Escape" && !enabling && (confirmOpen = false)}
          role="presentation"
        >
          <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
          <div
            class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
            on:click|stopPropagation
            on:keydown|stopPropagation
            role="dialog"
            aria-modal="true"
            tabindex="-1"
          >
            <div
              class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"
            ></div>

            <div
              class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
            >
              ENABLE MULTI-STORE
            </div>
            <h2 class="text-lg font-black text-bratrax-text-headline">
              Promote {detail.company_name} to multi-store?
            </h2>
            <p class="mt-2 text-sm font-light text-bratrax-text-body">
              This creates a parent record above the existing client. Every
              admin/viewer on this client will see the client switcher and an
              "Add store" button on their next request. Their current store
              remains unchanged. The parent inherits this client's paid status,
              so new sub-stores skip the payment screen.
            </p>

            <div class="mt-6 flex items-center justify-end gap-2">
              <button
                type="button"
                on:click={() => (confirmOpen = false)}
                disabled={enabling}
                class="btn-bratrax btn-neutral btn-compact"
              >
                Cancel
              </button>
              <button
                type="button"
                on:click={confirmEnable}
                disabled={enabling}
                class="btn-bratrax btn-primary btn-compact"
              >
                {enabling ? "Enabling…" : "Enable multi-store"}
              </button>
            </div>
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>
