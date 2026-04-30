<script lang="ts">
  export let open = false;
  export let loading = false;
  export let proposed = "";
  export let conflicts: Array<{
    customer_id: string;
    descriptive_name: string;
    current: string;
  }> = [];
  export let onReplaceAll: () => void = () => {};
  export let onSkip: () => void = () => {};
  export let onCancel: () => void = () => {};
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
    <div
      class="relative w-full max-w-lg border border-bratrax-border bg-bratrax-surface p-6"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">
        Replace existing tracking templates?
      </h2>
      <p class="mb-4 text-sm text-bratrax-text-muted">
        The accounts below already have a tracking template configured. To
        enable Bratrax attribution, we need to replace these with our standard
        template.
      </p>

      <div class="mb-4">
        <div class="mb-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          Bratrax tracking template
        </div>
        <pre
          class="overflow-x-auto border border-bratrax-border bg-bratrax-bg p-2 font-mono text-[11px] text-bratrax-text-body">{proposed}</pre>
      </div>

      <div class="mb-4">
        <div class="mb-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-muted">
          {conflicts.length} account{conflicts.length === 1 ? "" : "s"} with existing template
        </div>
        <div class="max-h-56 overflow-auto border border-bratrax-border">
          {#each conflicts as c (c.customer_id)}
            <div class="border-b border-bratrax-border p-3 last:border-0">
              <div class="text-sm text-bratrax-text-primary">
                {c.descriptive_name || c.customer_id}
                <span class="font-mono text-[10px] text-bratrax-text-muted">
                  ({c.customer_id})
                </span>
              </div>
              <pre
                class="mt-1 overflow-x-auto whitespace-pre-wrap break-all font-mono text-[11px] text-bratrax-text-muted">{c.current}</pre>
            </div>
          {/each}
        </div>
      </div>

      <div class="flex justify-end gap-2">
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
          on:click={onSkip}
          disabled={loading}
          title="Attribution will be disabled for these accounts."
        >
          Skip these
        </button>
        <button
          type="button"
          class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
          on:click={onReplaceAll}
          disabled={loading}
        >
          {loading ? "APPLYING..." : "REPLACE ALL"}
        </button>
      </div>
    </div>
  </div>
{/if}
