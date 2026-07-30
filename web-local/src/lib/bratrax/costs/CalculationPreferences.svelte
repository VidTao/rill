<script lang="ts">
  import { onMount } from "svelte";
  import {
    getProfitReadiness,
    getStoreSettings,
    saveStoreSettings,
  } from "./api";
  import type { ProfitReadiness, StoreSettings } from "./types";

  let loading = true;
  let saving = false;
  let error = "";
  let message = "";
  let readiness: ProfitReadiness | null = null;

  let excludeTaxes = true;
  let excludeZeroRevenue = false;
  let returnsDateBasis: "refund_date" | "order_date" = "refund_date";
  let adCostsConfirmed = false;
  let customCostsConfirmed = false;

  function settingValue(settings: StoreSettings, key: string): unknown {
    const stored = settings[key];
    if (stored && typeof stored === "object" && "value" in stored) {
      return (stored as { value?: unknown }).value;
    }
    return stored;
  }

  function booleanSetting(
    settings: StoreSettings,
    key: string,
    fallback: boolean,
  ): boolean {
    const value = settingValue(settings, key);
    if (value === undefined || value === null) return fallback;
    return value === true || value === "true" || value === 1 || value === "1";
  }

  async function load() {
    loading = true;
    error = "";
    try {
      const [settings, profitReadiness] = await Promise.all([
        getStoreSettings(),
        getProfitReadiness(),
      ]);
      excludeTaxes = booleanSetting(settings, "profit_exclude_taxes", true);
      excludeZeroRevenue = booleanSetting(
        settings,
        "profit_exclude_zero_revenue",
        false,
      );
      adCostsConfirmed = booleanSetting(settings, "ad_costs_confirmed", false);
      customCostsConfirmed = booleanSetting(
        settings,
        "custom_costs_confirmed",
        false,
      );
      const returnsBasis = settingValue(settings, "returns_date_basis");
      returnsDateBasis =
        returnsBasis === "order_date" ? "order_date" : "refund_date";
      readiness = profitReadiness;
    } catch (e) {
      error = e instanceof Error ? e.message : "Unable to load profit settings";
    } finally {
      loading = false;
    }
  }

  async function save() {
    saving = true;
    error = "";
    message = "";
    try {
      await saveStoreSettings({
        profit_exclude_taxes: excludeTaxes,
        profit_exclude_zero_revenue: excludeZeroRevenue,
        returns_date_basis: returnsDateBasis,
        ad_costs_confirmed: adCostsConfirmed,
        custom_costs_confirmed: customCostsConfirmed,
      });
      message = "Profit rules saved. The profitability reports are refreshing.";
    } catch (e) {
      error = e instanceof Error ? e.message : "Unable to save profit settings";
    } finally {
      saving = false;
    }
  }

  function percent(value: number): string {
    return `${Math.round(Math.max(0, Math.min(value, 1)) * 100)}%`;
  }

  onMount(() => void load());
</script>

{#if loading}
  <div
    class="flex min-h-[260px] items-center justify-center font-mono text-xs text-bratrax-text-muted"
  >
    Loading profit rules…
  </div>
{:else}
  <div class="space-y-8">
    <section>
      <div class="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <div
            class="font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
          >
            Profit readiness
          </div>
          <h2 class="mt-1 text-xl font-bold text-bratrax-text-headline">
            {readiness?.profit_ready
              ? "Net Profit is ready"
              : "Measured Profit"}
          </h2>
          <p class="mt-1 max-w-2xl text-sm text-bratrax-text-muted">
            Net Profit is shown only when every material cost input is complete.
            Missing costs are never silently treated as zero.
          </p>
        </div>
        {#if readiness?.available}
          <div class="border border-bratrax-border px-3 py-2 text-right">
            <div
              class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
            >
              Orders assessed
            </div>
            <div
              class="text-lg font-bold tabular-nums text-bratrax-text-headline"
            >
              {readiness.orders ?? 0}
            </div>
          </div>
        {/if}
      </div>

      {#if readiness?.available && readiness.costs}
        <div
          class="grid gap-px border border-bratrax-border bg-bratrax-border sm:grid-cols-2 lg:grid-cols-5"
        >
          {#each readiness.costs as cost (cost.key)}
            <div class="bg-bratrax-surface p-4">
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm font-semibold text-bratrax-text-headline"
                  >{cost.label}</span
                >
                <span
                  class="h-2.5 w-2.5 rounded-full"
                  class:bg-bratrax-acid={cost.status === "ready"}
                  class:bg-amber-400={cost.status === "estimated"}
                  class:bg-bratrax-tomato={cost.status === "missing"}
                  aria-label={cost.status}
                ></span>
              </div>
              <div
                class="mt-4 font-mono text-xl font-bold tabular-nums text-bratrax-text-headline"
              >
                {percent(cost.coverage)}
              </div>
              <div
                class="mt-1 font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
              >
                {cost.status === "estimated" ? "Estimate only" : cost.status}
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div
          class="border border-bratrax-border bg-bratrax-hover px-4 py-3 text-sm text-bratrax-text-muted"
        >
          Profit readiness becomes available after the profitability pilot is
          compiled for this store.
        </div>
      {/if}
    </section>

    <section class="border-t border-bratrax-border pt-6">
      <div class="mb-4">
        <div
          class="font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
        >
          Calculation preferences
        </div>
        <h2 class="mt-1 text-xl font-bold text-bratrax-text-headline">
          Accounting rules
        </h2>
      </div>

      <div class="divide-y divide-bratrax-border border border-bratrax-border">
        <label
          class="flex cursor-pointer items-start justify-between gap-6 p-4 hover:bg-bratrax-hover"
        >
          <div>
            <div class="text-sm font-semibold text-bratrax-text-headline">
              Exclude taxes collected from profit
            </div>
            <p class="mt-1 text-xs leading-5 text-bratrax-text-muted">
              Revenue still includes collected tax for Shopify reconciliation;
              profit subtracts it because it is owed to the tax authority.
            </p>
          </div>
          <input type="checkbox" class="mt-1" bind:checked={excludeTaxes} />
        </label>

        <label
          class="flex cursor-pointer items-start justify-between gap-6 p-4 hover:bg-bratrax-hover"
        >
          <div>
            <div class="text-sm font-semibold text-bratrax-text-headline">
              Ignore zero-revenue orders
            </div>
            <p class="mt-1 text-xs leading-5 text-bratrax-text-muted">
              Excludes test, replacement, and fully discounted orders from the
              order-level profit population.
            </p>
          </div>
          <input
            type="checkbox"
            class="mt-1"
            bind:checked={excludeZeroRevenue}
          />
        </label>

        <div class="flex flex-wrap items-start justify-between gap-4 p-4">
          <div>
            <div class="text-sm font-semibold text-bratrax-text-headline">
              Recognize returns on
            </div>
            <p class="mt-1 text-xs leading-5 text-bratrax-text-muted">
              Choose whether refunds affect the day they happened or restate the
              original order date.
            </p>
          </div>
          <select
            class="min-w-[210px] border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body"
            bind:value={returnsDateBasis}
          >
            <option value="refund_date">Refund date</option>
            <option value="order_date">Original order date</option>
          </select>
        </div>
      </div>
    </section>

    <section class="border-t border-bratrax-border pt-6">
      <div class="mb-4">
        <div
          class="font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted"
        >
          Zero-cost confirmations
        </div>
        <h2 class="mt-1 text-xl font-bold text-bratrax-text-headline">
          Confirm intentional zeroes
        </h2>
        <p class="mt-1 max-w-2xl text-sm text-bratrax-text-muted">
          Use these only when zero is a real business answer. They prevent a
          missing integration from being presented as a complete cost model.
        </p>
      </div>
      <div class="grid gap-3 md:grid-cols-2">
        <label
          class="flex cursor-pointer items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover"
        >
          <input type="checkbox" class="mt-1" bind:checked={adCostsConfirmed} />
          <div>
            <div class="text-sm font-semibold text-bratrax-text-headline">
              Ad spend is complete at zero
            </div>
            <p class="mt-1 text-xs leading-5 text-bratrax-text-muted">
              This store has no additional paid-media cost to connect or enter.
            </p>
          </div>
        </label>
        <label
          class="flex cursor-pointer items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover"
        >
          <input
            type="checkbox"
            class="mt-1"
            bind:checked={customCostsConfirmed}
          />
          <div>
            <div class="text-sm font-semibold text-bratrax-text-headline">
              Custom costs are complete at zero
            </div>
            <p class="mt-1 text-xs leading-5 text-bratrax-text-muted">
              There are no agency, software, fulfillment, or other custom costs
              to add.
            </p>
          </div>
        </label>
      </div>
    </section>

    {#if error}
      <div
        class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 text-sm text-bratrax-tomato"
      >
        {error}
      </div>
    {/if}
    {#if message}
      <div
        class="border border-bratrax-acid/40 bg-bratrax-acid/10 px-4 py-3 text-sm text-bratrax-acid-text"
      >
        {message}
      </div>
    {/if}

    <div class="flex justify-end border-t border-bratrax-border pt-5">
      <button class="btn-bratrax btn-primary" disabled={saving} on:click={save}>
        {saving ? "Saving…" : "Save Profit Rules"}
      </button>
    </div>
  </div>
{/if}
