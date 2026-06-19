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
    type ClientStatus,
  } from "$lib/bratrax/superadmins/api";

  let clients: SuperadminClientRow[] = [];
  let summary: ClientsSummary | null = null;
  let topError = "";
  let loading = true;

  // --- Status taxonomy (mirrors backend derive_status) ----------------------
  // icon + label + the text color and row-stripe color per state. Colors are
  // CSS vars so they flip correctly between light/dark themes. `acidText` uses
  // the olive-on-cream fallback in light theme (raw acid fails AA as text).
  const STATUS_META: Record<
    ClientStatus,
    { icon: string; label: string; color: string; stripe: string | null }
  > = {
    needs_handoff: {
      icon: "⚠",
      label: "Needs handoff",
      color: "var(--color-acid-text)",
      stripe: "var(--bratrax-acid)",
    },
    stuck: {
      icon: "⚠",
      label: "Stuck",
      color: "var(--bratrax-tomato)",
      stripe: "var(--bratrax-tomato)",
    },
    error: {
      icon: "✕",
      label: "Error",
      color: "var(--bratrax-tomato)",
      stripe: "var(--bratrax-tomato)",
    },
    cancelled: {
      icon: "◐",
      label: "Cancelled",
      color: "var(--bratrax-lavender)",
      stripe: "var(--bratrax-lavender)",
    },
    expired: {
      icon: "◌",
      label: "Expired",
      color: "var(--bratrax-text-muted)",
      stripe: "var(--bratrax-gray)",
    },
    running: {
      icon: "▶",
      label: "Running",
      color: "var(--bratrax-cyan)",
      stripe: null,
    },
    waiting: {
      icon: "⏸",
      label: "Waiting",
      color: "var(--bratrax-text-muted)",
      stripe: null,
    },
    healthy: {
      icon: "●",
      label: "Healthy",
      color: "var(--color-acid-text)",
      stripe: "var(--bratrax-acid)",
    },
  };

  // Triage priority — lower sorts to the top. Urgent states surface first.
  const STATUS_PRIORITY: Record<ClientStatus, number> = {
    error: 0,
    stuck: 1,
    needs_handoff: 2,
    cancelled: 3,
    expired: 4,
    running: 5,
    waiting: 6,
    healthy: 7,
  };

  // --- Tiles double as the status filter ------------------------------------
  // Each non-Total tile maps to a set of statuses; clicking applies ?status=.
  type TileKey =
    | "total"
    | "needs_attention"
    | "healthy"
    | "cancelled_or_expired";
  const TILE_STATUSES: Record<Exclude<TileKey, "total">, ClientStatus[]> = {
    needs_attention: ["needs_handoff", "stuck", "error"],
    healthy: ["healthy"],
    cancelled_or_expired: ["cancelled", "expired"],
  };
  const TILES: { key: TileKey; label: string; accent: string | null }[] = [
    { key: "total", label: "Total", accent: null },
    {
      key: "needs_attention",
      label: "Needs attention",
      accent: "var(--bratrax-tomato)",
    },
    { key: "healthy", label: "Healthy", accent: "var(--bratrax-acid)" },
    {
      key: "cancelled_or_expired",
      label: "Cancelled / expired",
      accent: "var(--bratrax-lavender)",
    },
  ];
  function tileValue(key: TileKey): number | undefined {
    if (!summary) return undefined;
    if (key === "total") return summary.total;
    if (key === "needs_attention") return summary.needs_attention;
    if (key === "healthy") return summary.healthy;
    return summary.cancelled_or_expired;
  }
  let activeTile: TileKey = "total";

  // --- Secondary filters ----------------------------------------------------
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
  let subStatus = "";
  let paid = "";
  let stuckHours: number | null = null;
  let search = "";
  let stepMenuOpen = false;

  // --- Sorting (client-side) ------------------------------------------------
  type SortKey = "company" | "status" | "stuck";
  let sortKey: SortKey = "status";
  let sortDir: "asc" | "desc" = "asc";

  // --- Multi-store promotion (preserved) ------------------------------------
  let confirmOpen = false;
  let confirmTarget: SuperadminClientRow | null = null;
  let enabling = false;
  let openMenuId: string | null = null;

  onMount(() => {
    loadSummary();
    loadClients();
  });

  async function loadSummary() {
    try {
      summary = await getClientsSummary();
    } catch {
      summary = null;
    }
  }

  function currentFilters(): ClientFilters {
    const f: ClientFilters = {};
    if (selectedSteps.size) f.step = [...selectedSteps];
    if (subStatus) f.subscription_status = [subStatus];
    if (activeTile !== "total") f.status = TILE_STATUSES[activeTile];
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

  let debounceTimer: ReturnType<typeof setTimeout>;
  function scheduleReload() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(loadClients, 250);
  }

  function selectTile(tile: TileKey) {
    activeTile = activeTile === tile ? "total" : tile;
    loadClients();
  }

  function toggleStep(step: string) {
    if (selectedSteps.has(step)) selectedSteps.delete(step);
    else selectedSteps.add(step);
    selectedSteps = selectedSteps;
    scheduleReload();
  }

  function clearFilters() {
    selectedSteps = new Set();
    subStatus = "";
    paid = "";
    stuckHours = null;
    search = "";
    activeTile = "total";
    loadClients();
  }

  $: hasFilters =
    selectedSteps.size > 0 ||
    !!subStatus ||
    !!paid ||
    stuckHours != null ||
    !!search.trim() ||
    activeTile !== "total";

  // --- Sorting --------------------------------------------------------------
  function setSort(key: SortKey) {
    if (sortKey === key) {
      sortDir = sortDir === "asc" ? "desc" : "asc";
    } else {
      sortKey = key;
      sortDir = key === "stuck" ? "desc" : "asc";
    }
  }

  $: sortedClients = [...clients].sort((a, b) => {
    let av: string | number;
    let bv: string | number;
    if (sortKey === "company") {
      av = a.company_name?.toLowerCase() ?? "";
      bv = b.company_name?.toLowerCase() ?? "";
    } else if (sortKey === "status") {
      av = STATUS_PRIORITY[a.status] ?? 99;
      bv = STATUS_PRIORITY[b.status] ?? 99;
      // Tie-break by step age so the longest-waiting urgent row floats up.
      if (av === bv) {
        const ah = a.step_age_hours ?? -1;
        const bh = b.step_age_hours ?? -1;
        return bh - ah;
      }
    } else {
      av = a.step_age_hours ?? -1;
      bv = b.step_age_hours ?? -1;
    }
    if (av < bv) return sortDir === "asc" ? -1 : 1;
    if (av > bv) return sortDir === "asc" ? 1 : -1;
    return 0;
  });

  // --- Display helpers ------------------------------------------------------
  function stuckLabel(c: SuperadminClientRow): string {
    if (c.step_age_hours == null) return "—";
    const h = c.step_age_hours;
    if (h < 1) return `${Math.round(h * 60)}m`;
    return `${Math.round(h)}h`;
  }

  // Color-temperature scaling on the Stuck column.
  function stuckColor(c: SuperadminClientRow): string {
    const h = c.step_age_hours;
    if (h == null) return "var(--bratrax-text-muted)";
    if (h >= 72) return "var(--bratrax-tomato)";
    if (h >= 24) return "var(--bratrax-text-headline)";
    return "var(--bratrax-text-body)";
  }

  // Track which row was just copied so the button can show "Copied" feedback.
  let copiedClientId: string | null = null;
  let copyResetTimer: ReturnType<typeof setTimeout>;
  function copyEmail(email: string | null, clientId: string, e: Event) {
    e.stopPropagation();
    if (!email) return;
    navigator.clipboard?.writeText(email);
    copiedClientId = clientId;
    clearTimeout(copyResetTimer);
    copyResetTimer = setTimeout(() => (copiedClientId = null), 1500);
  }

  // --- Multi-store handlers -------------------------------------------------
  function requestEnable(c: SuperadminClientRow, e: Event) {
    e.stopPropagation();
    openMenuId = null;
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

  function toggleMenu(id: string, e: Event) {
    e.stopPropagation();
    openMenuId = openMenuId === id ? null : id;
  }

  function openDetail(c: SuperadminClientRow) {
    goto(`/clients/${encodeURIComponent(c.client_id)}`);
  }
</script>

<svelte:window on:click={() => (openMenuId = null)} />

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
        Read-only status of every customer workspace. Each row carries one
        computed status; the four most urgent surface at the top. Click a tile
        to scope the table, click a row for full detail.
      </p>
    </div>

    <!-- Summary tiles — double as the status filter -->
    <div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      {#each TILES as tile (tile.key)}
        <button
          type="button"
          on:click={() => selectTile(tile.key)}
          class="relative border border-bratrax-border px-4 py-3 text-left transition-colors hover:bg-bratrax-surface"
          class:is-active={activeTile === tile.key}
          class:bg-bratrax-surface={activeTile === tile.key}
          class:bg-bratrax-bg={activeTile !== tile.key}
          style={activeTile === tile.key
            ? `box-shadow: inset 0 0 0 2px ${tile.accent ?? "var(--bratrax-acid)"}`
            : ""}
        >
          <span
            class="absolute left-0 right-0 top-0"
            style={`background: ${tile.accent ?? "transparent"}; height: ${activeTile === tile.key ? "5px" : "3px"}`}
          ></span>
          <div
            class="font-mono text-[9px] font-bold uppercase tracking-[1px]"
            class:text-bratrax-text-muted={activeTile !== tile.key}
            class:text-bratrax-text-headline={activeTile === tile.key}
          >
            {tile.label}
          </div>
          <div
            class="mt-1 text-2xl font-black leading-none"
            style={`color: ${tile.accent ?? "var(--bratrax-text-headline)"}`}
          >
            {summary ? (tileValue(tile.key) ?? 0) : "…"}
          </div>
        </button>
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

    <!-- Secondary filters -->
    <div class="mb-4 flex flex-wrap items-end gap-3 font-mono text-[10px]">
      <div class="relative">
        <div class="mb-1 uppercase tracking-[1px] text-bratrax-text-muted">
          Step
        </div>
        <button
          type="button"
          on:click|stopPropagation={() => (stepMenuOpen = !stepMenuOpen)}
          class="border border-bratrax-border bg-bratrax-bg px-2 py-1 text-bratrax-text-body"
        >
          {selectedSteps.size ? `${selectedSteps.size} selected` : "All"} ▾
        </button>
        {#if stepMenuOpen}
          <div
            class="absolute z-10 mt-1 max-h-64 w-52 overflow-y-auto border border-bratrax-border bg-bratrax-surface p-2 shadow-lg"
            on:click|stopPropagation
            on:keydown|stopPropagation
            role="menu"
            tabindex="-1"
          >
            {#each STEPS as step}
              <label
                class="flex cursor-pointer items-center gap-2 px-1 py-1 hover:bg-bratrax-bg"
              >
                <input
                  type="checkbox"
                  checked={selectedSteps.has(step)}
                  on:change={() => toggleStep(step)}
                />
                <span>{step}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>

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

      <div class="min-w-[180px] flex-1">
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

    <!-- List -->
    {#if loading}
      <div class="space-y-2">
        {#each Array(6) as _}
          <div class="h-12 animate-pulse bg-bratrax-bg"></div>
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
      <div class="border border-bratrax-border bg-bratrax-bg">
        <!-- header row -->
        <div
          class="client-row border-b border-bratrax-border font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
        >
          <div></div>
          <button
            type="button"
            class="px-3 py-3 text-left"
            on:click={() => setSort("company")}
          >
            Company {sortKey === "company"
              ? sortDir === "asc"
                ? "↑"
                : "↓"
              : "↕"}
          </button>
          <button
            type="button"
            class="px-3 py-3 text-left"
            on:click={() => setSort("status")}
          >
            Status {sortKey === "status"
              ? sortDir === "asc"
                ? "↑"
                : "↓"
              : "↕"}
          </button>
          <button
            type="button"
            class="px-3 py-3 text-left"
            on:click={() => setSort("stuck")}
          >
            Stuck {sortKey === "stuck" ? (sortDir === "asc" ? "↑" : "↓") : "↕"}
          </button>
          <div></div>
        </div>

        {#each sortedClients as c (c.client_id)}
          <div
            class="client-row group cursor-pointer border-b border-bratrax-border/50 last:border-b-0 hover:bg-bratrax-surface"
            on:click={() => openDetail(c)}
            on:keydown={(e) => e.key === "Enter" && openDetail(c)}
            role="button"
            tabindex="0"
          >
            <!-- left stripe -->
            <div
              class="self-stretch"
              style={`background: ${STATUS_META[c.status]?.stripe ?? "transparent"}`}
            ></div>

            <!-- company + admin -->
            <div class="px-3 py-3">
              <div
                class="font-mono text-[13px] font-bold text-bratrax-text-headline"
              >
                {c.company_name}
                {#if c.multi_client_id}
                  <span
                    class="ml-1 inline-block border border-bratrax-acid/40 px-1 py-0.5 align-middle text-[8px] text-bratrax-acid"
                  >
                    MULTI-STORE
                  </span>
                {/if}
              </div>
              <div class="mt-0.5 font-mono text-[11px] text-bratrax-text-muted">
                {c.admin_email ?? "no admin"}{c.admin_name
                  ? ` · ${c.admin_name}`
                  : ""}
                {#if c.admin_email}
                  <button
                    type="button"
                    on:click={(e) => copyEmail(c.admin_email, c.client_id, e)}
                    class="ml-1 border px-1 py-0.5 text-[8px] uppercase tracking-[1px] hover:border-bratrax-acid hover:text-bratrax-acid"
                    class:hidden={copiedClientId !== c.client_id}
                    class:group-hover:inline-block={copiedClientId !==
                      c.client_id}
                    class:inline-block={copiedClientId === c.client_id}
                    class:border-bratrax-border={copiedClientId !== c.client_id}
                    class:text-bratrax-text-muted={copiedClientId !==
                      c.client_id}
                    class:border-bratrax-acid={copiedClientId === c.client_id}
                    class:text-bratrax-acid={copiedClientId === c.client_id}
                  >
                    {copiedClientId === c.client_id ? "✓ Copied" : "⧉ Copy"}
                  </button>
                {/if}
              </div>
            </div>

            <!-- status -->
            <div
              class="px-3 py-3 font-mono text-[12px] font-bold uppercase tracking-[1.2px]"
            >
              <span
                class="flex items-center gap-1.5"
                style={`color: ${STATUS_META[c.status]?.color}`}
              >
                <span class="w-3.5 text-center not-italic"
                  >{STATUS_META[c.status]?.icon}</span
                >
                {STATUS_META[c.status]?.label ?? c.status}
              </span>
              {#if c.status_sub_label}
                <span
                  class="mt-0.5 block pl-5 text-[11px] font-normal normal-case tracking-normal text-bratrax-text-muted"
                >
                  {c.status_sub_label}
                </span>
              {/if}
            </div>

            <!-- stuck -->
            <div
              class="whitespace-nowrap px-3 py-3 font-mono text-[13px]"
              style={`color: ${stuckColor(c)}`}
            >
              {stuckLabel(c)}
            </div>

            <!-- overflow -->
            <div class="relative px-3 py-3 text-center">
              <button
                type="button"
                on:click={(e) => toggleMenu(c.client_id, e)}
                class="font-mono text-[18px] font-bold text-bratrax-text-muted hover:text-bratrax-acid"
                aria-label="Row actions"
              >
                ⋯
              </button>
              {#if openMenuId === c.client_id}
                <div
                  class="absolute right-2 top-10 z-20 w-44 border border-bratrax-border bg-bratrax-surface py-1 text-left shadow-lg"
                  on:click|stopPropagation
                  on:keydown|stopPropagation
                  role="menu"
                  tabindex="-1"
                >
                  <button
                    type="button"
                    on:click={() => openDetail(c)}
                    class="block w-full px-3 py-2 text-left font-mono text-[11px] text-bratrax-text-body hover:bg-bratrax-bg"
                  >
                    Open detail
                  </button>
                  {#if !c.multi_client_id}
                    <button
                      type="button"
                      on:click={(e) => requestEnable(c, e)}
                      class="block w-full px-3 py-2 text-left font-mono text-[11px] text-bratrax-text-body hover:bg-bratrax-bg"
                    >
                      Enable multi-store
                    </button>
                  {/if}
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>

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
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
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

<style>
  /* stripe / company / status / stuck / overflow */
  .client-row {
    display: grid;
    grid-template-columns: 4px 1fr 280px 90px 56px;
    align-items: center;
  }
</style>
