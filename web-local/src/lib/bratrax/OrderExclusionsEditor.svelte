<script lang="ts">
  import { onMount } from "svelte";
  import {
    getOrderExclusions,
    updateOrderExclusions,
  } from "$lib/bratrax/settings/api";
  import type { BrandDomainsApplyStatus } from "$lib/bratrax/settings/types";

  // Editor for the merchant order tags that take an order out of scope.
  //
  // What this is for: a store selling DTC and wholesale through one Shopify
  // storefront tags the wholesale orders. Left in, those orders dominate every
  // blended number — they typically carry a far higher average order value, so
  // AOV, MER, LTV and the metric trees all describe a business the merchant
  // isn't trying to measure. Declaring the tags removes those orders, and the
  // line items and refunds attached to them, from everything we compute.

  /** Rendered without the surrounding card when embedded in a modal. */
  export let embedded = false;

  let tags: string[] = [];
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
      const data = await getOrderExclusions();
      tags = [...(data.excluded_order_tags || [])];
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
        const data = await getOrderExclusions();
        applyStatus = data.apply_status ?? null;
        // Don't clobber edits made while the deploy was running.
        if (!dirty) tags = [...(data.excluded_order_tags || [])];
        schedulePoll();
      } catch {
        // Transient — stop polling rather than spinning on an error.
      }
    }, 5000);
  }

  function addRow() {
    tags = [...tags, ""];
    dirty = true;
    notice = "";
  }

  function removeRow(i: number) {
    tags = tags.filter((_, idx) => idx !== i);
    dirty = true;
    notice = "";
  }

  function onInput(i: number, value: string) {
    tags[i] = value;
    tags = tags;
    dirty = true;
    notice = "";
  }

  async function save() {
    saving = true;
    error = "";
    notice = "";
    try {
      // Send the raw strings — the server owns normalisation so the stored
      // value can never disagree with what the filter was built from.
      const data = await updateOrderExclusions(tags.filter((t) => t.trim()));
      tags = [...(data.excluded_order_tags || [])];
      applyStatus = data.apply_status ?? null;
      dirty = false;
      notice = data.pending_activation
        ? "Saved. These apply when your dashboard finishes setting up."
        : data.applied
          ? "Saved. Rebuilding your data — this takes a few minutes, and your historical totals will change."
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
      ? "Rebuilding your data…"
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
      Exclude orders by tag
    </h3>
  {/if}

  <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
    If you sell wholesale or B2B through the same store as your DTC orders, add
    the tags you put on those orders. We'll leave them out of your reporting
    entirely — the orders, their products and their refunds — so your numbers
    describe only the side of the business you're measuring.
  </p>

  <p class="mb-3 font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
    Tags must match whole tags on the order, but capitalisation doesn't matter.
    <span class="text-bratrax-text-body">B2B</span> won't match a tag named
    <span class="text-bratrax-text-body">B2B-NET30</span>, so add each one you use.
  </p>

  {#if loading}
    <p class="font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
      Loading…
    </p>
  {:else}
    <div class="mb-3 flex flex-col gap-2">
      {#each tags as tag, i}
        <div class="flex items-center gap-2">
          <input
            type="text"
            value={tag}
            on:input={(e) => onInput(i, e.currentTarget.value)}
            placeholder="Wholesale"
            spellcheck="false"
            autocapitalize="none"
            class="flex-1 border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
          />
          <button
            on:click={() => removeRow(i)}
            aria-label="Remove tag"
            class="border border-bratrax-border px-3 py-2 font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted hover:border-bratrax-tomato hover:text-bratrax-tomato"
          >
            Remove
          </button>
        </div>
      {/each}
      {#if tags.length === 0}
        <p class="font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
          No excluded tags — every order is counted.
        </p>
      {/if}
    </div>

    <div class="flex flex-wrap items-center justify-between gap-2">
      <button on:click={addRow} class="btn-bratrax btn-neutral btn-compact">
        + Add tag
      </button>
      <button
        on:click={save}
        disabled={saving || !dirty}
        class="btn-bratrax btn-primary btn-compact"
      >
        {saving ? "Saving…" : "Save tags"}
      </button>
    </div>

    {#if dirty && tags.some((t) => t.trim())}
      <p class="mt-3 font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
        Saving rebuilds your warehouse. Past totals will drop by whatever these
        tagged orders were worth.
      </p>
    {/if}
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
