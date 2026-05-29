<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import {
    listAllClients,
    enableMultiStoreForClient,
    getClientsSummary,
    type SuperadminClientRow,
    type ClientFilters,
    type ClientsSummary,
  } from "$lib/bratrax/superadmins/api";

  let clients: SuperadminClientRow[] = [];
  let summary: ClientsSummary | null = null;
  let topError = "";
  let loading = true;

  // --- Filters --------------------------------------------------------------
  const STEPS = [
    "payment_pending",
    "created",
    "embed_pending",
    "platforms_connected",
    "activating",
    "compiling",
    "deploying",
    "extracting",
    "ready",
    "error",
  ];
  const SUB_STATUSES = [
    "active",
    "past_due",
    "cancelled",
    "paused",
    "expired",
    "inactive",
  ];

  let selectedSteps = new Set<string>();
  let subStatus = ""; // "" = all
  let paid = ""; // "" = any | "true" | "false"
  let stuckHours: number | null = null;
  let errorsOnly = false;
  let search = "";
  let stepMenuOpen = false;

  // --- Sorting (client-side) ------------------------------------------------
  type SortKey = "company" | "step" | "stuck";
  let sortKey: SortKey = "stuck";
  let sortDir: "asc" | "desc" = "desc";

  // --- Multi-store promotion (preserved from the original page) -------------
  let confirmOpen = false;
  let confirmTarget: SuperadminClientRow | null = null;
  let enabling = false;

  onMount(() => {
    loadSummary();
    loadClients();
  });

  async function loadSummary() {
    try {
      summary = await getClientsSummary();
    } catch {
      // Tiles are non-critical; the table error banner covers hard failures.
      summary = null;
    }
  }

  function currentFilters(): ClientFilters {
    const f: ClientFilters = {};
    const steps = errorsOnly ? ["error"] : [...selectedSteps];
    if (steps.length) f.step = steps;
    if (subStatus) f.subscription_status = [subStatus];
    if (paid === "true" || paid === "false") f.paid = paid;
    if (search.trim()) f.search = search.trim();
    if (stuckHours != null && !Number.isNaN(stuckHours))
      f.stuck_hours = stuckHours;
    return f;
  }

  async function loadClients() {
    topError = "";
    loading = true;
    try {
      const data = await listAllClients(currentFilters());
      clients = data.clients;
    } catch (e: any) {
      topError = e?.message ?? "Failed to load clients";
    } finally {
      loading = false;
    }
  }

  // Debounce any filter change (covers the search input's ~250ms requirement
  // and avoids a request per keystroke / checkbox toggle).
  let debounceTimer: ReturnType<typeof setTimeout>;
  function scheduleReload() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(loadClients, 250);
  }

  function toggleStep(step: string) {
    if (selectedSteps.has(step)) selectedSteps.delete(step);
    else selectedSteps.add(step);
    selectedSteps = selectedSteps; // trigger reactivity
    scheduleReload();
  }

  function clearFilters() {
    selectedSteps = new Set();
    subStatus = "";
    paid = "";
    stuckHours = null;
    errorsOnly = false;
    search = "";
    loadClients();
  }

  $: hasFilters =
    selectedSteps.size > 0 ||
    !!subStatus ||
    !!paid ||
    stuckHours != null ||
    errorsOnly ||
    !!search.trim();

  // --- Sorting --------------------------------------------------------------
  function setSort(key: SortKey) {
    if (sortKey === key) {
      sortDir = sortDir === "asc" ? "desc" : "asc";
    } else {
      sortKey = key;
      sortDir = key === "company" || key === "step" ? "asc" : "desc";
    }
  }

  $: sortedClients = [...clients].sort((a, b) => {
    let av: string | number;
    let bv: string | number;
    if (sortKey === "company") {
      av = a.company_name?.toLowerCase() ?? "";
      bv = b.company_name?.toLowerCase() ?? "";
    } else if (sortKey === "step") {
      av = a.onboarding_step ?? "";
      bv = b.onboarding_step ?? "";
    } else {
      av = a.step_age_hours ?? -1;
      bv = b.step_age_hours ?? -1;
    }
    if (av < bv) return sortDir === "asc" ? -1 : 1;
    if (av > bv) return sortDir === "asc" ? 1 : -1;
    return 0;
  });

  // --- Step glyph + helpers -------------------------------------------------
  function stepGlyph(step: string | null): string {
    switch (step) {
      case "payment_pending":
      case "created":
      case "embed_pending":
        return "⏵";
      case "platforms_connected":
        return "●";
      case "activating":
      case "compiling":
      case "deploying":
      case "extracting":
        return "◐";
      case "ready":
        return "◉";
      case "error":
        return "✕";
      default:
        return "·";
    }
  }

  function stuckLabel(c: SuperadminClientRow): string {
    if (c.step_age_hours == null) return "—";
    const h = c.step_age_hours;
    if (h < 1) return `${Math.round(h * 60)}m`;
    return `${Math.round(h)}h`;
  }

  function isStuck(c: SuperadminClientRow): boolean {
    return (
      c.onboarding_step !== "ready" &&
      c.step_age_hours != null &&
      c.step_age_hours > 24
    );
  }

  // --- Multi-store handlers -------------------------------------------------
  function requestEnable(c: SuperadminClientRow, e: Event) {
    e.stopPropagation();
    confirmTarget = c;
    confirmOpen = true;
  }

  async function confirmEnable() {
    if (!confirmTarget) return;
    enabling = true;
    topError = "";
    try {
      const result = await enableMultiStoreForClient(confirmTarget.client_id);
      clients = clients.map((c) =>
        c.client_id === confirmTarget!.client_id
          ? {
              ...c,
              multi_client_id: result.multi_client_id,
              multi_client_display_name: c.company_name,
              multi_client_is_paid_subscriber: c.is_paid_subscriber,
            }
          : c,
      );
      confirmOpen = false;
      confirmTarget = null;
    } catch (e: any) {
      topError = e?.message ?? "Failed to enable multi-store";
    } finally {
      enabling = false;
    }
  }

  function openDetail(c: SuperadminClientRow) {
    goto(`/clients/${encodeURIComponent(c.client_id)}`);
  }
</script>

<div
  class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12"
>
  <div
    class="relative w-full max-w-6xl border border-bratrax-border bg-bratrax-surface p-8"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6">
      <div
        class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        CLIENTS
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">Clients</h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Read-only status of every customer workspace — onboarding step,
        subscription, connections, errors. Click a row for full detail.
      </p>
    </div>

    <!-- Summary tiles -->
    <div
      class="mb-6 grid grid-cols-3 gap-px border border-bratrax-border bg-bratrax-border sm:grid-cols-6"
    >
      {#each [["TOTAL", summary?.total], ["ACTIVE PAID", summary?.active_paid], ["STUCK >24h", summary?.stuck_over_24h], ["IN ERROR", summary?.in_error], ["READY", summary?.ready], ["SIGNED UP THIS WK", summary?.signed_up_this_week]] as [label, value]}
        <div class="bg-bratrax-bg px-3 py-3">
          <div
            class="font-mono text-[9px] font-bold uppercase tracking-[1px] text-bratrax-text-muted"
          >
            {label}
          </div>
          <div class="mt-1 text-xl font-black text-bratrax-text-headline">
            {summary ? (value ?? 0) : "…"}
          </div>
        </div>
      {/each}
    </div>

    {#if topError}
      <div
        class="mb-4 flex items-center justify-between border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
      >
        <span>{topError}</span>
        <button type="button" on:click={loadClients} class="underline"
          >Retry</button
        >
      </div>
    {/if}

    <!-- Filters -->
    <div class="mb-4 flex flex-wrap items-end gap-3 font-mono text-[10px]">
      <!-- Step multi-select -->
      <div class="relative">
        <div class="mb-1 uppercase tracking-[1px] text-bratrax-text-muted">
          Step
        </div>
        <button
          type="button"
          on:click={() => (stepMenuOpen = !stepMenuOpen)}
          class="border border-bratrax-border bg-bratrax-bg px-2 py-1 text-bratrax-text-body"
        >
          {selectedSteps.size ? `${selectedSteps.size} selected` : "All"} ▾
        </button>
        {#if stepMenuOpen}
          <div
            class="absolute z-10 mt-1 max-h-64 w-52 overflow-y-auto border border-bratrax-border bg-bratrax-surface p-2 shadow-lg"
          >
            {#each STEPS as step}
              <label
                class="flex cursor-pointer items-center gap-2 px-1 py-1 hover:bg-bratrax-bg"
              >
                <input
                  type="checkbox"
                  checked={selectedSteps.has(step)}
                  on:change={() => toggleStep(step)}
                  disabled={errorsOnly}
                />
                <span>{step}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Subscription -->
      <div>
        <div class="mb-1 uppercase tracking-[1px] text-bratrax-text-muted">
          Subscription
        </div>
        <select
          bind:value={subStatus}
          on:change={loadClients}
          class="border border-bratrax-border bg-bratrax-bg px-2 py-1 text-bratrax-text-body"
        >
          <option value="">All</option>
          {#each SUB_STATUSES as s}
            <option value={s}>{s}</option>
          {/each}
        </select>
      </div>

      <!-- Paid -->
      <div>
        <div class="mb-1 uppercase tracking-[1px] text-bratrax-text-muted">
          Paid
        </div>
        <select
          bind:value={paid}
          on:change={loadClients}
          class="border border-bratrax-border bg-bratrax-bg px-2 py-1 text-bratrax-text-body"
        >
          <option value="">Any</option>
          <option value="true">Yes</option>
          <option value="false">No</option>
        </select>
      </div>

      <!-- Stuck > N hours -->
      <div>
        <div class="mb-1 uppercase tracking-[1px] text-bratrax-text-muted">
          Stuck &gt; (hrs)
        </div>
        <input
          type="number"
          min="0"
          bind:value={stuckHours}
          on:input={scheduleReload}
          placeholder="—"
          class="w-20 border border-bratrax-border bg-bratrax-bg px-2 py-1 text-bratrax-text-body"
        />
      </div>

      <!-- Errors only -->
      <label class="flex cursor-pointer items-center gap-2 pb-1">
        <input
          type="checkbox"
          bind:checked={errorsOnly}
          on:change={loadClients}
        />
        <span class="text-bratrax-text-body">Errors only</span>
      </label>

      <!-- Search -->
      <div class="flex-1 min-w-[180px]">
        <div class="mb-1 uppercase tracking-[1px] text-bratrax-text-muted">
          Search
        </div>
        <input
          type="text"
          bind:value={search}
          on:input={scheduleReload}
          placeholder="company or admin email"
          class="w-full border border-bratrax-border bg-bratrax-bg px-2 py-1 text-bratrax-text-body"
        />
      </div>

      {#if hasFilters}
        <button
          type="button"
          on:click={clearFilters}
          class="pb-1 text-bratrax-acid underline"
        >
          Clear filters
        </button>
      {/if}
    </div>

    <!-- Table -->
    {#if loading}
      <div class="space-y-2">
        {#each Array(6) as _}
          <div class="h-10 animate-pulse bg-bratrax-bg"></div>
        {/each}
      </div>
    {:else if sortedClients.length === 0}
      <div
        class="border border-bratrax-border bg-bratrax-bg px-4 py-8 text-center font-mono text-[11px] text-bratrax-text-muted"
      >
        No clients match these filters.
        {#if hasFilters}
          <button
            type="button"
            on:click={clearFilters}
            class="ml-1 text-bratrax-acid underline"
          >
            Clear filters
          </button>
        {/if}
      </div>
    {:else}
      <table class="w-full border-collapse font-mono text-[11px]">
        <thead>
          <tr
            class="border-b border-bratrax-border text-left uppercase tracking-[1px] text-bratrax-text-muted"
          >
            <th
              class="cursor-pointer px-2 py-2"
              on:click={() => setSort("company")}
            >
              Company {sortKey === "company"
                ? sortDir === "asc"
                  ? "↑"
                  : "↓"
                : "↕"}
            </th>
            <th
              class="cursor-pointer px-2 py-2"
              on:click={() => setSort("step")}
            >
              Step {sortKey === "step" ? (sortDir === "asc" ? "↑" : "↓") : "↕"}
            </th>
            <th
              class="cursor-pointer px-2 py-2"
              on:click={() => setSort("stuck")}
            >
              Stuck {sortKey === "stuck"
                ? sortDir === "asc"
                  ? "↑"
                  : "↓"
                : "↕"}
            </th>
            <th class="px-2 py-2">Paid</th>
            <th class="px-2 py-2"></th>
          </tr>
        </thead>
        <tbody>
          {#each sortedClients as c (c.client_id)}
            <tr
              class="cursor-pointer border-b border-bratrax-border/50 hover:bg-bratrax-bg"
              on:click={() => openDetail(c)}
            >
              <td class="px-2 py-2 align-top">
                <div class="font-bold text-bratrax-text-headline">
                  {c.company_name}
                </div>
                <div class="text-[10px] text-bratrax-text-muted">
                  {c.admin_email ?? "no admin"}{c.admin_name
                    ? ` · ${c.admin_name}`
                    : ""}
                </div>
                {#if c.multi_client_id}
                  <span
                    class="mt-1 inline-block border border-bratrax-acid/40 px-1 py-0.5 text-[8px] text-bratrax-acid"
                  >
                    MULTI-STORE
                  </span>
                {/if}
              </td>
              <td class="px-2 py-2 align-top">
                <span class:text-bratrax-tomato={c.onboarding_step === "error"}>
                  {stepGlyph(c.onboarding_step)}
                  {c.onboarding_step ?? "—"}
                </span>
              </td>
              <td class="px-2 py-2 align-top whitespace-nowrap">
                {stuckLabel(c)}
                {#if isStuck(c)}<span
                    class="text-bratrax-tomato"
                    title="Stuck > 24h">⚠</span
                  >{/if}
              </td>
              <td class="px-2 py-2 align-top">
                {c.subscription_status ??
                  (c.is_paid_subscriber ? "active" : "—")}
              </td>
              <td class="px-2 py-2 align-top text-right">
                {#if !c.multi_client_id}
                  <button
                    type="button"
                    on:click={(e) => requestEnable(c, e)}
                    class="btn-bratrax btn-neutral btn-compact"
                  >
                    Enable multi-store
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>

      <div class="mt-3 font-mono text-[10px] text-bratrax-text-muted">
        {sortedClients.length} client{sortedClients.length === 1 ? "" : "s"}
        {#if sortedClients.length >= 500}· showing first 500{/if}
      </div>
    {/if}
  </div>
</div>

<!-- Confirm modal (multi-store promotion) -->
{#if confirmOpen && confirmTarget}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => !enabling && (confirmOpen = false)}
    on:keydown={(e) => e.key === "Escape" && !enabling && (confirmOpen = false)}
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

      <div
        class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        ENABLE MULTI-STORE
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Promote {confirmTarget.company_name} to multi-store?
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        This creates a parent record above the existing client. Every
        admin/viewer on this client will see the client switcher and an "Add
        store" button on their next request. Their current store remains
        unchanged. The parent inherits this client's paid status, so new
        sub-stores skip the payment screen.
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
