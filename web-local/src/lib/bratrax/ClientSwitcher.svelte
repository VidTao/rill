<script lang="ts">
  import { onMount, tick } from "svelte";
  import {
    bratraxListClients,
    bratraxSwitchClient,
    type BratraxClientSummary,
  } from "./auth";

  // Powers the top-right client switcher shown only to super_admin / multi-store
  // users — they can jump between any client without re-logging-in. The Go proxy
  // resolves the active client via the bratrax_active_client cookie and rebuilds
  // the per-client Rill instance lazily, so a hard reload after switch is enough
  // to re-init the runtime stores cleanly.
  //
  // Rendered as a custom popover (not a native <select>) so it can carry a search
  // field and a two-line, readable row per client — super_admins routinely have
  // dozens of clients with similar names, so type-to-filter + the admin email as
  // a secondary line are what make the list usable.

  let clients: BratraxClientSummary[] = [];
  let activeClientId = "";
  let switching = false;
  let loadError = "";

  let open = false;
  let query = "";
  let highlightIndex = 0;

  let rootEl: HTMLDivElement | null = null;
  let triggerEl: HTMLButtonElement | null = null;
  let searchEl: HTMLInputElement | null = null;
  let rowEls: HTMLButtonElement[] = [];

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

  $: activeClient = clients.find((c) => c.client_id === activeClientId);
  $: q = query.trim().toLowerCase();
  $: filtered = q
    ? clients.filter(
        (c) =>
          c.company_name.toLowerCase().includes(q) ||
          (c.admin_email ?? "").toLowerCase().includes(q),
      )
    : clients;

  async function toggle() {
    if (switching) return;
    open = !open;
    if (open) {
      query = "";
      await tick();
      searchEl?.focus();
      // Start the highlight on the currently active client so Enter is a no-op
      // until the user actually moves or types.
      const idx = clients.findIndex((c) => c.client_id === activeClientId);
      highlightIndex = idx >= 0 ? idx : 0;
      rowEls[highlightIndex]?.scrollIntoView({ block: "nearest" });
    }
  }

  function close() {
    open = false;
  }

  function moveHighlight(delta: number) {
    if (!filtered.length) return;
    highlightIndex =
      (highlightIndex + delta + filtered.length) % filtered.length;
    rowEls[highlightIndex]?.scrollIntoView({ block: "nearest" });
  }

  async function selectClient(id: string) {
    if (switching) return;
    if (id === activeClientId) {
      close();
      return;
    }
    switching = true;
    try {
      await bratraxSwitchClient(id);
      // Hard reload so Rill stores re-init for the new instance (per-client
      // DuckDB cache, metrics views, dashboards). Same effect as fresh login.
      window.location.href = "/";
    } catch (e: any) {
      switching = false;
      open = false;
      // eslint-disable-next-line no-alert
      alert(e?.message ?? "Failed to switch client");
    }
  }

  function onSearchInput() {
    highlightIndex = 0;
  }

  function onSearchKeydown(e: KeyboardEvent) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      moveHighlight(1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      moveHighlight(-1);
    } else if (e.key === "Enter") {
      e.preventDefault();
      const c = filtered[highlightIndex];
      if (c) selectClient(c.client_id);
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
      triggerEl?.focus();
    }
  }

  function onTriggerKeydown(e: KeyboardEvent) {
    if (!open && (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      void toggle();
    }
  }

  function onWindowClick(e: MouseEvent) {
    if (open && rootEl && !rootEl.contains(e.target as Node)) close();
  }

  function onWindowKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && open) {
      close();
      triggerEl?.focus();
    }
  }

  // Split a label around the current query so the matched run can be visually
  // emphasised — helps the eye land on the right row when names are similar.
  function highlightParts(text: string): { text: string; match: boolean }[] {
    if (!q) return [{ text, match: false }];
    const lower = text.toLowerCase();
    const parts: { text: string; match: boolean }[] = [];
    let i = 0;
    while (i < text.length) {
      const idx = lower.indexOf(q, i);
      if (idx === -1) {
        parts.push({ text: text.slice(i), match: false });
        break;
      }
      if (idx > i) parts.push({ text: text.slice(i, idx), match: false });
      parts.push({ text: text.slice(idx, idx + q.length), match: true });
      i = idx + q.length;
    }
    return parts;
  }
</script>

<svelte:window on:click={onWindowClick} on:keydown={onWindowKeydown} />

{#if loadError}
  <span class="cs-error" title={loadError}>—</span>
{:else if clients.length > 0}
  <div class="cs-root" bind:this={rootEl}>
    <button
      type="button"
      class="cs-trigger"
      class:open
      bind:this={triggerEl}
      on:click={toggle}
      on:keydown={onTriggerKeydown}
      disabled={switching}
      aria-haspopup="listbox"
      aria-expanded={open}
      aria-label="Switch active client"
      title={activeClient?.company_name ?? "Select client"}
    >
      <svg
        class="cs-trigger-icon"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <rect x="4" y="2" width="16" height="20" />
        <path d="M9 22v-4h6v4" />
        <path
          d="M8 6h.01M16 6h.01M8 10h.01M16 10h.01M8 14h.01M16 14h.01"
        />
      </svg>
      <span class="cs-trigger-name">
        {switching ? "Switching…" : (activeClient?.company_name ?? "Select client")}
      </span>
      <svg
        class="cs-chevron"
        width="12"
        height="12"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </button>

    {#if open}
      <div class="cs-panel">
        <div class="cs-search">
          <svg
            class="cs-search-icon"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <circle cx="11" cy="11" r="7" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            class="cs-search-input"
            type="text"
            placeholder="Search clients…"
            bind:value={query}
            bind:this={searchEl}
            on:input={onSearchInput}
            on:keydown={onSearchKeydown}
            aria-label="Search clients"
            autocomplete="off"
            spellcheck="false"
          />
        </div>

        <div class="cs-list" role="listbox" aria-label="Clients">
          {#each filtered as c, i (c.client_id)}
            <button
              type="button"
              class="cs-option"
              class:active={c.client_id === activeClientId}
              class:highlighted={i === highlightIndex}
              role="option"
              aria-selected={c.client_id === activeClientId}
              bind:this={rowEls[i]}
              on:click={() => selectClient(c.client_id)}
              on:mouseenter={() => (highlightIndex = i)}
            >
              <span class="cs-option-text">
                <span class="cs-option-name">
                  {#each highlightParts(c.company_name) as part}
                    {#if part.match}<mark class="cs-match">{part.text}</mark
                      >{:else}{part.text}{/if}
                  {/each}
                </span>
                {#if c.admin_email}
                  <span class="cs-option-email">
                    {#each highlightParts(c.admin_email) as part}
                      {#if part.match}<mark class="cs-match">{part.text}</mark
                        >{:else}{part.text}{/if}
                    {/each}
                  </span>
                {/if}
              </span>
              {#if c.client_id === activeClientId}
                <svg
                  class="cs-check"
                  width="15"
                  height="15"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              {/if}
            </button>
          {:else}
            <div class="cs-empty">No clients match “{query}”</div>
          {/each}
        </div>

        <div class="cs-footer">
          {#if q}{filtered.length} / {clients.length} clients{:else}{clients.length}
            {clients.length === 1 ? "client" : "clients"}{/if}
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  /* Editorial client switcher — sharp edges, Space Mono caps, acid accents.
     Sits in the header next to .bratrax-add-store-btn / .bratrax-theme-toggle
     and reuses the same 32px control height + neutral border + acid focus ring. */
  .cs-root {
    position: relative;
    display: inline-flex;
  }

  /* -- Trigger -- */
  .cs-trigger {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 32px;
    min-width: 160px;
    max-width: 240px;
    padding: 0 10px;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text, var(--fg-primary));
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.5px;
    cursor: pointer;
    transition: background-color 120ms ease, color 120ms ease,
      border-color 120ms ease;
  }
  .cs-trigger:hover:not(:disabled),
  .cs-trigger.open {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    border-color: var(--color-text, var(--fg-primary));
  }
  .cs-trigger:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
  .cs-trigger:disabled {
    opacity: 0.5;
    cursor: wait;
  }
  .cs-trigger-icon {
    flex: none;
    color: var(--color-text-secondary, var(--fg-secondary));
  }
  .cs-trigger-name {
    flex: 1;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    text-align: left;
  }
  .cs-chevron {
    flex: none;
    color: var(--color-text-muted, var(--fg-muted));
    transition: transform 140ms ease;
  }
  .cs-trigger.open .cs-chevron {
    transform: rotate(180deg);
  }

  /* -- Panel -- */
  .cs-panel {
    position: absolute;
    top: calc(100% + 6px);
    right: 0;
    z-index: 90;
    width: 320px;
    max-width: 80vw;
    background: var(--color-elevated, #fff);
    border: 1px solid var(--color-border-strong, var(--border));
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
  }
  /* Acid top bar — same card signature used across the design system. */
  .cs-panel::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: var(--color-acid, #d4ff00);
    pointer-events: none;
  }
  :global(.dark) .cs-panel {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  }

  /* -- Search -- */
  .cs-search {
    position: relative;
    padding: 10px;
  }
  .cs-search-icon {
    position: absolute;
    left: 20px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--color-text-muted, var(--fg-muted));
    pointer-events: none;
  }
  .cs-search-input {
    width: 100%;
    padding: 8px 10px 8px 32px;
    font-family: "Outfit", "Helvetica Neue", sans-serif;
    font-size: 14px;
  }

  /* -- List -- */
  .cs-list {
    max-height: 320px;
    overflow-y: auto;
    border-top: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
    padding: 4px;
  }
  .cs-option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    width: 100%;
    padding: 9px 10px;
    background: transparent;
    border: none;
    border-left: 2px solid transparent;
    cursor: pointer;
    text-align: left;
    transition: background-color 100ms ease;
  }
  .cs-option.highlighted {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.08));
  }
  .cs-option.active {
    background: var(--color-acid-mid, rgba(212, 255, 0, 0.16));
    border-left-color: var(--color-acid, #d4ff00);
  }
  .cs-option-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .cs-option-name {
    font-family: "Outfit", "Helvetica Neue", sans-serif;
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text, var(--fg-primary));
    line-height: 1.25;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .cs-option-email {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    color: var(--color-text-muted, var(--fg-muted));
    line-height: 1.3;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .cs-match {
    background: var(--color-acid, #d4ff00);
    color: #0a0a0a;
    font-weight: 700;
    padding: 0 1px;
  }
  .cs-check {
    flex: none;
    color: var(--color-text, var(--fg-primary));
  }

  .cs-empty {
    padding: 18px 12px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 12px;
    text-align: center;
    color: var(--color-text-muted, var(--fg-muted));
  }

  .cs-footer {
    padding: 7px 12px;
    border-top: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-text-muted, var(--fg-muted));
  }

  .cs-error {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 12px;
    color: var(--color-text-muted, #858585);
    padding: 6px 10px;
  }
</style>
