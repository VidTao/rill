<script lang="ts">
  import { bratraxAddStore } from "./multiClient";
  import { bratraxSwitchClient } from "./auth";

  // Header button — visible only when the parent layout has determined the
  // current user belongs to a multi-client. Click prompts for the new
  // store's display name, provisions it via /bratrax/multi-client/add-store,
  // then switches the active-client cookie to the new sub-store and
  // hard-reloads into /onboard/shopify (the route guard handles the rest).

  let busy = false;

  async function handleClick() {
    if (busy) return;
    // Quick prompt for the new store's name. Matches what the user enters
    // for the first store on the accept-invite page (the "company_name"
    // input). Replace with a proper modal later if we want richer UX.
    const name = window.prompt("Name of the new store?");
    if (!name) return;
    const trimmed = name.trim();
    if (!trimmed) return;

    busy = true;
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
      // eslint-disable-next-line no-alert
      alert(e?.message ?? "Failed to add store");
    }
  }
</script>

<button
  type="button"
  on:click={handleClick}
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
