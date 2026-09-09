<script lang="ts">
  import { onMount } from "svelte";
  import {
    getBrandDomains,
    updateBrandDomains,
  } from "$lib/bratrax/settings/api";
  import type { BrandDomainsApplyStatus } from "$lib/bratrax/settings/types";

  // Editor for the extra domains the merchant owns. Shared by the Custom /
  // Other install modal and the Settings page so both write through the same
  // endpoint and show the same apply state.
  //
  // What this is for: attribution's self-referral guard matches a referrer by
  // its registrable core, so it recognises the storefront domain on its own but
  // cannot know that e.g. "healthy-mornings.com" is the same brand's
  // advertorial. Undeclared, that hop into the store is credited as a Referral
  // and steals the paid touchpoint that actually drove the visit.

  /** Rendered without the surrounding card when embedded in a modal. */
  export let embedded = false;

  let domains: string[] = [];
  let brandDomain = "";
  let loading = true;
  let saving = false;
  let error = "";
  let notice = "";
  let applyStatus: BrandDomainsApplyStatus | null = null;
  let pollTimer: ReturnType<typeof setTimeout> | null = null;
  let dirty = false;

  const IN_FLIGHT = ["queued", "compiling", "deploying", "refreshing"];

  onMount(() => {
    void load();
    return () => {
      if (pollTimer) clearTimeout(pollTimer);
    };
  });

  async function load() {
    loading = true;
    error = "";
    try {
      const data = await getBrandDomains();
      brandDomain = data.brand_domain || "";
      domains = [...(data.brand_domains || [])];
      applyStatus = data.apply_status ?? null;
      dirty = false;
      schedulePoll();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  // A compile+deploy takes minutes, so the row is polled rather than awaited.
  function schedulePoll() {
    if (pollTimer) clearTimeout(pollTimer);
    if (!applyStatus || !IN_FLIGHT.includes(applyStatus.state)) return;
    pollTimer = setTimeout(async () => {
      try {
        const data = await getBrandDomains();
        applyStatus = data.apply_status ?? null;
        // Don't clobber edits made while the deploy was running.
        if (!dirty) domains = [...(data.brand_domains || [])];
        schedulePoll();
      } catch {
        // Transient — stop polling rather than spinning on an error.
      }
    }, 5000);
  }

  function addRow() {
    domains = [...domains, ""];
    dirty = true;
    notice = "";
  }

  function removeRow(i: number) {
    domains = domains.filter((_, idx) => idx !== i);
    dirty = true;
    notice = "";
  }

  function onInput(i: number, value: string) {
    domains[i] = value;
    domains = domains;
    dirty = true;
    notice = "";
  }

  async function save() {
    saving = true;
    error = "";
    notice = "";
    try {
      // Send the raw strings — the server owns normalisation so the stored
      // value can never disagree with what the guard was built from.
      const data = await updateBrandDomains(domains.filter((d) => d.trim()));
      domains = [...(data.brand_domains || [])];
      brandDomain = data.brand_domain || "";
      applyStatus = data.apply_status ?? null;
      dirty = false;
      notice = data.pending_activation
        ? "Saved. These apply when your dashboard finishes setting up."
        : data.applied
          ? "Saved. Rebuilding your attribution — this takes a few minutes."
          : "Saved. No changes to apply.";
      schedulePoll();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      saving = false;
    }
  }

  $: statusLabel = !applyStatus
    ? ""
    : applyStatus.state === "compiling" || applyStatus.state === "queued"
      ? "Rebuilding attribution…"
      : applyStatus.state === "deploying"
        ? "Updating your warehouse…"
        : applyStatus.state === "refreshing"
          ? "Refreshing your data…"
          : applyStatus.state === "applied"
            ? "Applied"
            : applyStatus.state === "failed"
              ? `Could not apply${applyStatus.error ? `: ${applyStatus.error}` : ""}`
              : "";
</script>

<div class={embedded ? "" : "border border-bratrax-border bg-bratrax-surface p-6"}>
  {#if !embedded}
    <h3 class="mb-1 text-base font-black text-bratrax-text-headline">
      Your other domains
    </h3>
  {/if}

  <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
    Add any other domains you own — advertorials, pre-sell and bridge pages,
    custom checkouts, regional stores. We treat traffic arriving from them as
    internal, so your own pages don't get credited as a referral and steal the
    ad that actually drove the visit.
  </p>

  {#if brandDomain}
    <p class="mb-3 font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
      Your store domain <span class="text-bratrax-text-body">{brandDomain}</span>
      is already covered — no need to add it.
    </p>
  {/if}

  {#if loading}
    <p class="font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
      Loading…
    </p>
  {:else}
    <div class="mb-3 flex flex-col gap-2">
      {#each domains as domain, i}
        <div class="flex items-center gap-2">
          <input
            type="text"
            value={domain}
            on:input={(e) => onInput(i, e.currentTarget.value)}
            placeholder="offers.example.com"
            spellcheck="false"
            autocapitalize="none"
            class="flex-1 border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
          />
          <button
            on:click={() => removeRow(i)}
            aria-label="Remove domain"
            class="border border-bratrax-border px-3 py-2 font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted hover:border-bratrax-tomato hover:text-bratrax-tomato"
          >
            Remove
          </button>
        </div>
      {/each}
      {#if domains.length === 0}
        <p class="font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
          No extra domains yet.
        </p>
      {/if}
    </div>

    <div class="flex flex-wrap items-center justify-between gap-2">
      <button
        on:click={addRow}
        class="btn-bratrax btn-neutral btn-compact"
      >
        + Add domain
      </button>
      <button
        on:click={save}
        disabled={saving || !dirty}
        class="btn-bratrax btn-primary btn-compact"
      >
        {saving ? "Saving…" : "Save domains"}
      </button>
    </div>
  {/if}

  {#if statusLabel}
    <p
      class="mt-3 font-mono text-[10px] uppercase tracking-wider {applyStatus?.state ===
      'failed'
        ? 'text-bratrax-tomato'
        : 'text-bratrax-acid/80'}"
    >
      {statusLabel}
    </p>
  {/if}

  {#if notice}
    <p class="mt-2 font-mono text-[10px] uppercase tracking-wider text-bratrax-acid/80">
      {notice}
    </p>
  {/if}

  {#if error}
    <div
      class="mt-3 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
    >
      {error}
    </div>
  {/if}
</div>
