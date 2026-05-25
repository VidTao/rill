<script lang="ts">
  import { onMount } from "svelte";
  import {
    listAllClients,
    enableMultiStoreForClient,
    type SuperadminClientRow,
  } from "$lib/bratrax/superadmins/api";

  let clients: SuperadminClientRow[] = [];
  let topError = "";
  let loading = true;

  // Confirm modal state — clicking "Enable multi-store" pops a confirmation
  // before promoting (irreversible on the user's side; only a superadmin
  // could later detach via SQL, so a confirm step matches the gravity).
  let confirmOpen = false;
  let confirmTarget: SuperadminClientRow | null = null;
  let enabling = false;

  // Per-row toast surface — after a successful promote we keep the row but
  // flip the badge + disable the button. Errors live in topError so they
  // mirror the existing /superadmins UX.

  onMount(load);

  async function load() {
    topError = "";
    loading = true;
    try {
      const data = await listAllClients();
      clients = data.clients;
    } catch (e: any) {
      topError = e?.message ?? "Failed to load clients";
    } finally {
      loading = false;
    }
  }

  function requestEnable(c: SuperadminClientRow) {
    confirmTarget = c;
    confirmOpen = true;
  }

  async function confirmEnable() {
    if (!confirmTarget) return;
    enabling = true;
    topError = "";
    try {
      const result = await enableMultiStoreForClient(confirmTarget.client_id);
      // Patch the in-memory row so the badge updates without a full reload.
      // The next mount-refresh (or page reload) replaces this with the
      // canonical backend response.
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

  function formatDate(iso: string | null): string {
    if (!iso) return "";
    return new Date(iso).toLocaleDateString();
  }
</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="relative w-full max-w-3xl border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        CLIENTS
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        All Bratrax clients
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Promote a single-store client into a multi-store parent so the owner can connect additional Shopify brands without a new login.
      </p>
    </div>

    {#if topError}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {topError}
      </div>
    {/if}

    <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
      Clients ({clients.length})
    </h2>

    {#if loading}
      <p class="mt-3 font-mono text-[10px] text-bratrax-text-muted">Loading…</p>
    {:else if clients.length === 0}
      <p class="mt-3 font-mono text-[10px] text-bratrax-text-muted">
        No clients yet.
      </p>
    {:else}
      <ul class="mt-3 flex flex-col gap-2">
        {#each clients as c (c.client_id)}
          <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="truncate font-mono text-xs font-bold text-bratrax-text-headline">
                  {c.company_name}
                </span>
                {#if c.multi_client_id}
                  <span class="shrink-0 border border-bratrax-acid/40 px-1.5 py-0.5 font-mono text-[9px] text-bratrax-acid">
                    MULTI-STORE
                  </span>
                {:else}
                  <span class="shrink-0 border border-bratrax-border px-1.5 py-0.5 font-mono text-[9px] text-bratrax-text-muted">
                    SINGLE STORE
                  </span>
                {/if}
                <span class="shrink-0 border px-1.5 py-0.5 font-mono text-[9px] {c.is_paid_subscriber ? 'border-bratrax-acid/40 text-bratrax-acid' : 'border-bratrax-border text-bratrax-text-muted'}">
                  {c.is_paid_subscriber ? "PAID" : "UNPAID"}
                </span>
              </div>
              <div class="truncate font-mono text-[10px] text-bratrax-text-muted">
                {c.admin_email ?? "no admin"} · created {formatDate(c.created_at)}
              </div>
            </div>

            {#if !c.multi_client_id}
              <button
                type="button"
                on:click={() => requestEnable(c)}
                class="btn-bratrax btn-primary btn-compact"
              >
                Enable multi-store
              </button>
            {:else}
              <span class="btn-bratrax btn-neutral btn-compact" style="pointer-events: none; opacity: 0.55;">
                Multi-store enabled
              </span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<!-- Confirm modal -->
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

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        ENABLE MULTI-STORE
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Promote {confirmTarget.company_name} to multi-store?
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        This creates a parent record above the existing client. Every admin/viewer on this client will see the client switcher and an "Add store" button on their next request. Their current store remains unchanged. The parent inherits this client's paid status, so new sub-stores skip the payment screen.
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
