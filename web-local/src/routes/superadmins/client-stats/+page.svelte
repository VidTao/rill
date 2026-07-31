<script lang="ts">
  import { onMount } from "svelte";
  import {
    listAttributionStats,
    recomputeAttributionStats,
    type AttributionStatRow,
  } from "$lib/bratrax/superadmins/api";

  let stats: AttributionStatRow[] = [];
  let lastComputedAt: string | null = null;
  let loading = true;
  let recomputing = false;
  let error = "";
  let notice = "";

  // Arrives pre-sorted worst-first on 30d from the API; these track
  // client-side re-sorts when a percentage header is clicked.
  type SortKey = "direct_pct_7d" | "direct_pct_30d" | "direct_pct_all";
  let sortKey: SortKey = "direct_pct_30d";

  // Above this share of attributed orders landing in Direct, a client is
  // almost certainly mis-tracking rather than genuinely direct-heavy.
  const ALERT_THRESHOLD = 0.4;

  onMount(load);

  async function load() {
    loading = true;
    error = "";
    try {
      const resp = await listAttributionStats();
      stats = resp.stats;
      lastComputedAt = resp.last_computed_at;
      applySort();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function recompute() {
    recomputing = true;
    error = "";
    notice = "";
    try {
      const result = await recomputeAttributionStats();
      notice = result.skipped
        ? `Skipped: ${result.skipped}`
        : `Recomputed ${result.clients_evaluated} clients` +
          (result.errors ? ` · ${result.errors} failed` : "");
      await load();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      recomputing = false;
    }
  }

  function sortBy(key: SortKey) {
    sortKey = key;
    applySort();
  }

  // Nulls last in every ordering — "no attributed orders in this window" is not
  // the same as a perfect 0% and must not float to the top of a worst-first list.
  function applySort() {
    stats = [...stats].sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (av === null && bv === null) return 0;
      if (av === null) return 1;
      if (bv === null) return -1;
      return bv - av;
    });
  }

  function pct(v: number | null): string {
    return v === null ? "—" : `${(v * 100).toFixed(1)}%`;
  }

  function num(v: number | null): string {
    return v === null ? "—" : Math.round(v).toLocaleString();
  }

  function fmt(iso: string | null): string {
    if (!iso) return "—";
    return new Date(iso).toLocaleString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function pctClass(v: number | null): string {
    if (v === null) return "text-bratrax-text-muted";
    return v >= ALERT_THRESHOLD
      ? "text-bratrax-tomato"
      : "text-bratrax-text-primary";
  }
</script>

<div
  class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12"
>
  <div class="w-full max-w-[1400px] px-4">
    <a
      href="/superadmins"
      class="font-mono text-[11px] text-bratrax-acid hover:underline"
    >
      ← Super-admin
    </a>

    <div class="mt-4 flex items-baseline justify-between gap-4">
      <h1
        class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted"
      >
        Client stats — Direct attribution share
      </h1>
      <button
        type="button"
        on:click={recompute}
        disabled={recomputing || loading}
        class="font-mono text-[11px] text-bratrax-acid hover:underline disabled:opacity-50"
      >
        {recomputing ? "Recomputing…" : "Recompute now"}
      </button>
    </div>

    <p class="mt-2 max-w-3xl font-mono text-[11px] leading-relaxed text-bratrax-text-muted">
      Share of attributed orders landing in the <span
        class="text-bratrax-text-primary">Direct</span
      >
      channel, on the last-touch model — the same measure the client's Attribution
      canvas shows. Direct is also the fallback bucket for orders with no usable
      marketing signal, so read a high number as
      <span class="text-bratrax-text-primary">unattributed</span>, not as
      genuine direct traffic. Recomputed every 4 hours.
    </p>

    <section class="mt-3 border border-bratrax-border bg-bratrax-surface p-5">
      {#if error}
        <div
          class="mb-3 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
        >
          {error}
        </div>
      {/if}

      {#if notice}
        <div
          class="mb-3 border border-bratrax-border px-3 py-2 font-mono text-xs text-bratrax-text-muted"
        >
          {notice}
        </div>
      {/if}

      <div class="mb-3 font-mono text-[11px] text-bratrax-text-muted">
        {stats.length} active client{stats.length === 1 ? "" : "s"} · last computed
        {fmt(lastComputedAt)}
      </div>

      {#if loading}
        <div class="space-y-2">
          {#each Array(6) as _}
            <div class="h-8 animate-pulse bg-bratrax-bg"></div>
          {/each}
        </div>
      {:else if stats.length === 0}
        <div
          class="border border-bratrax-border px-4 py-6 text-center font-mono text-xs text-bratrax-text-muted"
        >
          No stats yet. Hit “Recompute now” to populate the table.
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left font-mono text-xs">
            <thead class="text-bratrax-text-muted">
              <tr class="border-b border-bratrax-border">
                <th class="th">Client</th>
                <th class="th text-right">Attr. orders 30d</th>
                <th class="th text-right">Direct orders 30d</th>
                <th class="th text-right">
                  <button
                    type="button"
                    class="sort-btn"
                    class:sort-active={sortKey === "direct_pct_30d"}
                    on:click={() => sortBy("direct_pct_30d")}>Direct % 30d</button
                  >
                </th>
                <th class="th text-right">
                  <button
                    type="button"
                    class="sort-btn"
                    class:sort-active={sortKey === "direct_pct_7d"}
                    on:click={() => sortBy("direct_pct_7d")}>Direct % 7d</button
                  >
                </th>
                <th class="th text-right">
                  <button
                    type="button"
                    class="sort-btn"
                    class:sort-active={sortKey === "direct_pct_all"}
                    on:click={() => sortBy("direct_pct_all")}
                    >Direct % all-time</button
                  >
                </th>
                <th class="th-last">Computed</th>
              </tr>
            </thead>
            <tbody>
              {#each stats as s (s.client_id)}
                <tr class="border-b border-bratrax-border hover:bg-bratrax-bg">
                  <td class="py-2 pr-3">
                    <div class="text-bratrax-text-primary">
                      {s.company_name}
                    </div>
                    <div class="text-[10px] text-bratrax-text-muted">
                      {s.clickhouse_db}
                    </div>
                  </td>
                  {#if s.error}
                    <td
                      class="py-2 pr-3 text-right text-bratrax-text-muted"
                      colspan="5"
                      title={s.error}
                    >
                      query failed — {s.error.slice(0, 80)}
                    </td>
                  {:else}
                    <td class="py-2 pr-3 text-right text-bratrax-text-primary">
                      {num(s.attributed_orders_30d)}
                    </td>
                    <td class="py-2 pr-3 text-right text-bratrax-text-primary">
                      {num(s.direct_orders_30d)}
                    </td>
                    <td
                      class="py-2 pr-3 text-right font-bold {pctClass(
                        s.direct_pct_30d,
                      )}"
                    >
                      {pct(s.direct_pct_30d)}
                    </td>
                    <td class="py-2 pr-3 text-right {pctClass(s.direct_pct_7d)}">
                      {pct(s.direct_pct_7d)}
                    </td>
                    <td class="py-2 pr-3 text-right {pctClass(s.direct_pct_all)}">
                      {pct(s.direct_pct_all)}
                    </td>
                  {/if}
                  <td class="py-2 text-bratrax-text-muted">
                    {fmt(s.computed_at)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </section>
  </div>
</div>

<style>
  .th,
  .th-last {
    padding: 8px 12px 8px 0;
    font-family: ui-monospace, monospace;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1.5px;
  }
  .th-last {
    padding-right: 0;
  }
  .sort-btn {
    font: inherit;
    letter-spacing: inherit;
    text-transform: inherit;
    color: inherit;
  }
  .sort-btn:hover {
    color: var(--bratrax-text-primary, #f5f5f7);
  }
  .sort-active {
    color: #d4ff00;
  }
</style>
