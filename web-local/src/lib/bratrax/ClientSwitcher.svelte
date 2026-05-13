<script lang="ts">
  import { onMount } from "svelte";
  import {
    bratraxListClients,
    bratraxSwitchClient,
    type BratraxClientSummary,
  } from "./auth";

  // Powers the top-right dropdown shown only to super_admin users — they can
  // jump between any client without re-logging-in. The Go proxy resolves the
  // active client via the bratrax_active_client cookie and rebuilds the per-
  // client Rill instance lazily, so a hard reload after switch is enough to
  // re-init the runtime stores cleanly.

  let clients: BratraxClientSummary[] = [];
  let activeClientId = "";
  let switching = false;
  let loadError = "";

  onMount(async () => {
    try {
      const data = await bratraxListClients();
      clients = data.clients;
      activeClientId = data.active_client_id;
    } catch (e: any) {
      // Non-fatal: swallow the error so the rest of the header still renders.
      loadError = e?.message ?? "Failed to load clients";
    }
  });

  async function handleChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    const newId = target.value;
    if (!newId || newId === activeClientId) return;
    switching = true;
    try {
      await bratraxSwitchClient(newId);
      // Hard reload so Rill stores re-init for the new instance (per-client
      // DuckDB cache, metrics views, dashboards). Same effect as fresh login.
      window.location.href = "/";
    } catch (e: any) {
      switching = false;
      // Reset to active so the dropdown doesn't sit in a stale half-state.
      target.value = activeClientId;
      // eslint-disable-next-line no-alert
      alert(e?.message ?? "Failed to switch client");
    }
  }
</script>

{#if loadError}
  <span class="bratrax-client-switcher-error" title={loadError}>—</span>
{:else if clients.length > 0}
  <select
    class="bratrax-client-switcher"
    value={activeClientId}
    on:change={handleChange}
    disabled={switching}
    aria-label="Active client"
  >
    {#each clients as c (c.client_id)}
      <option value={c.client_id}
        >{c.company_name}{c.admin_email ? ` — ${c.admin_email}` : ""}</option
      >
    {/each}
  </select>
{/if}

<style>
  /* Editorial dropdown — Space Mono caps, neutral border, acid focus ring.
     Matches the .btn-bratrax + .bratrax-status-pill family established in
     the round-3 design pass. */
  .bratrax-client-switcher {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    padding: 6px 28px 6px 12px;
    background: var(--color-elevated, #fff)
      url("data:image/svg+xml;charset=utf-8,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 8 5'%3E%3Cpath fill='%231A1A18' d='M0 0l4 5 4-5z'/%3E%3C/svg%3E")
      no-repeat right 10px center / 8px 5px;
    border: 1px solid rgba(0, 0, 0, 0.18);
    color: #1a1a18;
    cursor: pointer;
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    transition: border-color 120ms ease, background-color 120ms ease;
    max-width: 220px;
    text-overflow: ellipsis;
  }
  .bratrax-client-switcher:hover:not(:disabled) {
    border-color: #1a1a18;
  }
  .bratrax-client-switcher:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
  .bratrax-client-switcher:disabled {
    opacity: 0.5;
    cursor: wait;
  }
  :global(.dark) .bratrax-client-switcher {
    background-color: #141414;
    color: #e8e4dc;
    border-color: #3a3a3a;
    background-image: url("data:image/svg+xml;charset=utf-8,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 8 5'%3E%3Cpath fill='%23E8E4DC' d='M0 0l4 5 4-5z'/%3E%3C/svg%3E");
  }
  :global(.dark) .bratrax-client-switcher:hover:not(:disabled) {
    border-color: #e8e4dc;
  }

  .bratrax-client-switcher-error {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 12px;
    color: #858585;
    padding: 6px 10px;
  }
</style>
