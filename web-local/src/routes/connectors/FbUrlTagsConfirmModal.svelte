<script lang="ts">
  // Per-account view: each account block shows total/with-tags/empty counts
  // and a collapsible list of conflicting campaigns (campaigns whose existing
  // url_tags differs from ours). Mirrors the spec text:
  //   "Acme Brand (act_X): 47 campaigns. 12 already have url_tags."
  //   "[Replace all 47] [Apply only to the 35 empty] [Cancel]"

  export let open = false;
  export let loading = false;
  export let proposed = "";
  export let accounts: Array<{
    act_id: string;
    account_name: string;
    total_campaigns: number;
    with_url_tags: number;
    conflicting: Array<{ campaign_id: string; name: string; current: string }>;
    conflicting_count: number;
  }> = [];
  export let onReplaceAll: () => void = () => {};
  export let onApplyEmptyOnly: () => void = () => {};
  export let onCancel: () => void = () => {};

  const VISIBLE_PER_ACCOUNT = 5;
  let expanded: Record<string, boolean> = {};

  function toggleExpanded(actId: string) {
    expanded = { ...expanded, [actId]: !expanded[actId] };
  }

  $: totalCampaigns = accounts.reduce((s, a) => s + a.total_campaigns, 0);
  $: totalConflicts = accounts.reduce((s, a) => s + a.conflicting_count, 0);
  $: totalEmpty = totalCampaigns - totalConflicts;
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
    <div
      class="relative w-full max-w-2xl border border-bratrax-border bg-bratrax-surface p-6"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">
        Replace existing Facebook campaign URL tags?
      </h2>
      <p class="mb-4 text-sm text-bratrax-text-muted">
        Some campaigns already have <code>url_tags</code> set. To enable Bratrax
        attribution we'd replace them with our standard template. The other
        campaigns (where <code>url_tags</code> is empty) are unaffected by your choice.
      </p>

      <div class="mb-4">
        <div class="mb-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          Bratrax url_tags template
        </div>
        <pre
          class="overflow-x-auto whitespace-pre-wrap break-all border border-bratrax-border bg-bratrax-bg p-2 font-mono text-[11px] text-bratrax-text-body">{proposed}</pre>
      </div>

      <div class="mb-4 max-h-72 overflow-auto border border-bratrax-border">
        {#each accounts as a (a.act_id)}
          <div class="border-b border-bratrax-border p-3 last:border-0">
            <div class="flex items-baseline justify-between">
              <div>
                <span class="text-sm font-bold text-bratrax-text-primary">
                  {a.account_name || `act_${a.act_id}`}
                </span>
                <span class="ml-2 font-mono text-[10px] text-bratrax-text-muted">
                  (act_{a.act_id})
                </span>
              </div>
              <div class="font-mono text-[10px] text-bratrax-text-muted">
                {a.total_campaigns} campaigns · {a.conflicting_count} conflict{a.conflicting_count === 1 ? "" : "s"} · {a.total_campaigns - a.conflicting_count} empty
              </div>
            </div>

            {#if a.conflicting_count > 0}
              <div class="mt-2 space-y-1">
                {#each (expanded[a.act_id] ? a.conflicting : a.conflicting.slice(0, VISIBLE_PER_ACCOUNT)) as c (c.campaign_id)}
                  <div class="border-l-2 border-bratrax-border pl-2">
                    <div class="text-xs text-bratrax-text-primary">
                      {c.name}
                    </div>
                    <pre
                      class="whitespace-pre-wrap break-all font-mono text-[10px] text-bratrax-text-muted">{c.current}</pre>
                  </div>
                {/each}
                {#if a.conflicting.length > VISIBLE_PER_ACCOUNT}
                  <button
                    type="button"
                    class="font-mono text-[10px] uppercase tracking-wider text-bratrax-acid hover:underline"
                    on:click={() => toggleExpanded(a.act_id)}
                  >
                    {expanded[a.act_id]
                      ? `Hide`
                      : `Show all ${a.conflicting.length} conflicts`}
                  </button>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <div class="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover disabled:opacity-50"
          on:click={onCancel}
          disabled={loading}
        >
          Cancel
        </button>
        <button
          type="button"
          class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover disabled:opacity-50"
          on:click={onApplyEmptyOnly}
          disabled={loading}
          title="Skip campaigns that already have a custom url_tags. Attribution will be missing on those campaigns."
        >
          {loading ? "APPLYING..." : `APPLY ONLY TO THE ${totalEmpty} EMPTY`}
        </button>
        <button
          type="button"
          class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
          on:click={onReplaceAll}
          disabled={loading}
        >
          {loading ? "APPLYING..." : `REPLACE ALL ${totalCampaigns}`}
        </button>
      </div>
    </div>
  </div>
{/if}
