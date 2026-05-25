<script lang="ts">
  import { portal } from "@rilldata/web-common/lib/actions/portal";
  import { bratraxAddStore } from "./multiClient";
  import { bratraxSwitchClient } from "./auth";

  // Header button — visible only when the parent layout has determined the
  // current user belongs to a multi-client. Click opens a styled modal asking
  // for the new store's display name, then provisions it via
  // /bratrax/multi-client/add-store, switches the active-client cookie to the
  // new sub-store, and hard-reloads into /onboard/shopify (the route guard
  // handles the rest).

  let modalOpen = false;
  let storeName = "";
  let busy = false;
  let error = "";
  let inputEl: HTMLInputElement | null = null;

  function openModal() {
    if (busy) return;
    storeName = "";
    error = "";
    modalOpen = true;
    // Focus the input on the next tick once Svelte mounts it.
    queueMicrotask(() => inputEl?.focus());
  }

  function closeModal() {
    if (busy) return;
    modalOpen = false;
  }

  async function submit() {
    const trimmed = storeName.trim();
    if (!trimmed) {
      error = "Store name is required";
      return;
    }
    busy = true;
    error = "";
    try {
      const result = await bratraxAddStore(trimmed);
      // Flip the active-client cookie so the next page load loads the new
      // sub-store. The switch endpoint validates the target shares the
      // user's multi_client_id, so a tampered response cannot redirect us
      // to a foreign tenant.
      await bratraxSwitchClient(result.client_id);
      // Hard reload so Rill stores re-init for the new instance. The
      // resume-onboarding guard in +layout.ts sees step='created' on the
      // fresh row and sends the user to /onboard/shopify.
      window.location.href = "/onboard/shopify";
    } catch (e: any) {
      busy = false;
      error = e?.message ?? "Failed to add store";
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") closeModal();
  }
</script>

<button
  type="button"
  on:click={openModal}
  class="bratrax-add-store-btn"
  disabled={busy}
  title="Add a new store under this account"
  aria-label="Add store"
>
  <svg
    width="14"
    height="14"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <line x1="12" y1="5" x2="12" y2="19" />
    <line x1="5" y1="12" x2="19" y2="12" />
  </svg>
  <span>Add store</span>
</button>

{#if modalOpen}
  <!-- use:portal lifts the overlay out of any ancestor stacking context (the
       application header + dashboard chrome both create their own), so a high
       z-index reliably wins over the dashboard toolbar. -->
  <div
    use:portal
    class="fixed inset-0 z-[100] flex items-center justify-center bg-black/60"
    on:click={closeModal}
    on:keydown={handleKeydown}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-store-title"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        ADD STORE
      </div>
      <h2 id="add-store-title" class="text-lg font-black text-bratrax-text-headline">
        Add a new store
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        This starts a fresh onboarding flow for a new Shopify brand under your account. The new store inherits your billing — no payment screen.
      </p>

      <div class="mt-4 flex flex-col gap-3">
        <div>
          <label
            for="add-store-name"
            class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
          >
            Store name
          </label>
          <input
            id="add-store-name"
            type="text"
            bind:value={storeName}
            bind:this={inputEl}
            on:keydown={(e) => e.key === "Enter" && !busy && submit()}
            class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            placeholder="My new brand"
            disabled={busy}
          />
        </div>

        {#if error}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {error}
          </div>
        {/if}
      </div>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          type="button"
          on:click={closeModal}
          disabled={busy}
          class="btn-bratrax btn-neutral btn-compact"
        >
          Cancel
        </button>
        <button
          type="button"
          on:click={submit}
          disabled={busy}
          class="btn-bratrax btn-primary btn-compact"
        >
          {busy ? "Creating…" : "Create store"}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Matches .bratrax-theme-toggle visual weight but with a label so the
     affordance is obvious. Uses the same border treatment and acid focus
     ring so it slots next to the theme toggle without visual clash. */
  .bratrax-add-store-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 32px;
    padding: 0 10px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    cursor: pointer;
    transition: background-color 120ms ease, color 120ms ease,
      border-color 120ms ease;
  }
  .bratrax-add-store-btn:hover:not(:disabled) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
  }
  .bratrax-add-store-btn:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
  .bratrax-add-store-btn:disabled {
    opacity: 0.5;
    cursor: wait;
  }
</style>
