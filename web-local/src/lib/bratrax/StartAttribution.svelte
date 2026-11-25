<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { page } from "$app/stores";
  import { portal } from "@rilldata/web-common/lib/actions/portal";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
  import { bratraxUser } from "./auth-store";
  import { checkCompanyAvailable, onboardStart } from "./onboarding/api";
  import { seedConnectedSourcesBaseline, trackOnce } from "./analytics";
  import { START_ATTRIBUTION_EVENT, startAttribution } from "./members";

  // "Start attribution": turns a viewer on the shared demo workspace (a demo
  // visitor or a directory member from bratrax.com/stores) into a customer,
  // with the same login. Mounted once, by the root layout, for those viewers
  // only. Opens from its own header button, from openStartAttribution()
  // (the demo AI-allowance gate) and from any URL carrying
  // ?start-attribution=1 (the demo tour's closing link).
  //
  // Flow: company name (same availability check as /signup) → Flask converts
  // the account in place (admin, no workspace) → onboardStart creates the
  // workspace → hard reload into /onboard/store, so the layout re-reads the
  // user's new role and the onboarding guard takes over (payment first).

  /** Show the header button; the modal works either way. */
  export let showButton = true;

  let open = false;
  let companyName = "";
  let busy = false;
  let error = "";
  let inputEl: HTMLInputElement | null = null;
  // Set once the account is converted: a retry after a failed workspace step
  // must not convert again (it would be refused: no longer a demo viewer).
  let converted = false;

  function openModal() {
    if (busy) return;
    error = "";
    open = true;
    queueMicrotask(() => inputEl?.focus());
  }

  function closeModal() {
    if (busy) return;
    open = false;
  }

  function describeCheck(reason: string): string {
    switch (reason) {
      case "taken":
        return "That company name is already in use. Try another.";
      case "invalid":
        return "Use letters or numbers — that doesn't produce a valid name.";
      default:
        return "Couldn't check that name right now. Try again.";
    }
  }

  async function submit() {
    const name = companyName.trim();
    if (name.length < 3) {
      error = "Company name must be at least 3 characters.";
      return;
    }
    busy = true;
    error = "";
    try {
      if (!converted) {
        const check = await checkCompanyAvailable(name);
        if (!check.available) {
          error = describeCheck(check.reason);
          busy = false;
          return;
        }
        await startAttribution(name);
        converted = true;
        const user = $bratraxUser;
        if (user) {
          trackOnce(`signup_${user.id}`, "sign_up", { method: "start_attribution" });
        }
      }
      const result = await onboardStart(name);
      sessionStorage.setItem("onboard_client_id", result.client_id);
      sessionStorage.setItem("onboard_client_name", result.client_name);
      seedConnectedSourcesBaseline(result.client_id);
      queryClient.clear();
      window.location.href = "/onboard/store";
    } catch (e) {
      busy = false;
      const message = e instanceof Error ? e.message : String(e);
      error = converted
        ? `Your account is ready, but setting up the workspace failed: ${message}. Try again.`
        : message;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") closeModal();
  }

  const onEvent = () => openModal();
  onMount(() => window.addEventListener(START_ATTRIBUTION_EVENT, onEvent));
  onDestroy(() => {
    if (typeof window !== "undefined") {
      window.removeEventListener(START_ATTRIBUTION_EVENT, onEvent);
    }
  });

  // The demo tour links here with ?start-attribution=1. Once per URL, so
  // closing the modal doesn't reopen it while the query is still there.
  let autoOpenedFor = "";
  $: if (
    $page.url.searchParams.get("start-attribution") === "1" &&
    $page.url.href !== autoOpenedFor
  ) {
    autoOpenedFor = $page.url.href;
    openModal();
  }
</script>

{#if showButton}
  <button
    type="button"
    on:click={openModal}
    class="bratrax-start-attribution-btn"
    disabled={busy}
    title="Connect your own store and see your own numbers"
  >
    Start attribution
  </button>
{/if}

{#if open}
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
      aria-labelledby="start-attribution-title"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div
        class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        START ATTRIBUTION
      </div>
      <h2
        id="start-attribution-title"
        class="text-lg font-black text-bratrax-text-headline"
      >
        See your own numbers
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        We'll set up your own Bratrax workspace on this same login. Next you
        connect your store and ad accounts; your history backfills within the
        hour. $1 for your first month, then $99/mo flat.
      </p>

      <div class="mt-4 flex flex-col gap-3">
        <div>
          <label
            for="start-attribution-company"
            class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
          >
            Company name
          </label>
          <input
            id="start-attribution-company"
            type="text"
            bind:value={companyName}
            bind:this={inputEl}
            on:input={() => (error = "")}
            on:keydown={(e) => e.key === "Enter" && !busy && submit()}
            class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            placeholder="Your brand name"
            autocomplete="organization"
            disabled={busy || converted}
          />
        </div>

        {#if error}
          <div
            class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
            role="alert"
          >
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
          {busy ? "Setting up…" : converted ? "Try again" : "Start attribution →"}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Same weight as the header's Add store button, filled acid: it is the
     demo workspace's one call to action. */
  .bratrax-start-attribution-btn {
    display: inline-flex;
    align-items: center;
    height: 32px;
    padding: 0 12px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    background: var(--color-acid, #d4ff00);
    color: #0a0a0a;
    border: 1px solid var(--color-acid, #d4ff00);
    cursor: pointer;
    white-space: nowrap;
    transition: opacity 120ms ease;
  }
  .bratrax-start-attribution-btn:hover:not(:disabled) {
    opacity: 0.9;
  }
  .bratrax-start-attribution-btn:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
  .bratrax-start-attribution-btn:disabled {
    opacity: 0.5;
    cursor: wait;
  }
</style>
