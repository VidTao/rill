<script lang="ts">
  import type { AdAccountInfo } from "$lib/bratrax/connectors/api";

  export let open = false;
  export let loading = false;
  export let platformName = "";
  export let accounts: Array<{ id: string; name: string }> = [];
  export let onSubmit: (selected: AdAccountInfo[]) => void = () => {};
  export let onClose: () => void = () => {};

  let selectedIds: Set<string> = new Set();

  function toggleAccount(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    selectedIds = next;
  }

  function toggleAll() {
    if (selectedIds.size === accounts.length) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(accounts.map((a) => a.id));
    }
  }

  function handleSubmit() {
    const selected: AdAccountInfo[] = accounts
      .filter((a) => selectedIds.has(a.id))
      .map((a) => ({ accountId: a.id, accountName: a.name }));
    onSubmit(selected);
  }

  $: allSelected = accounts.length > 0 && selectedIds.size === accounts.length;
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" on:click={onClose}>
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <h2 class="mb-1 text-lg font-bold text-bratrax-text-headline">
        Select {platformName} Accounts
      </h2>
      <p class="mb-4 text-sm text-bratrax-text-muted">
        Choose which accounts to connect
      </p>

      {#if accounts.length === 0}
        <p class="py-4 text-center text-sm text-bratrax-text-muted">No accounts found.</p>
      {:else}
        <div class="mb-4 max-h-64 overflow-auto border border-bratrax-border">
          <div class="sticky top-0 border-b border-bratrax-border bg-bratrax-bg px-3 py-2">
            <label class="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={allSelected}
                on:change={toggleAll}
                class="accent-bratrax-acid"
              />
              <span class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body">Select All</span>
            </label>
          </div>
          {#each accounts as account (account.id)}
            <label class="flex cursor-pointer items-center gap-2 border-b border-bratrax-border px-3 py-2 text-sm last:border-0 hover:bg-bratrax-hover">
              <input
                type="checkbox"
                checked={selectedIds.has(account.id)}
                on:change={() => toggleAccount(account.id)}
                class="accent-bratrax-acid"
              />
              <span class="text-bratrax-text-primary">{account.name}</span>
              <span class="font-mono text-[10px] text-bratrax-text-muted">({account.id})</span>
            </label>
          {/each}
        </div>
      {/if}

      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover"
          on:click={onClose}
          disabled={loading}
        >
          Cancel
        </button>
        <button
          type="button"
          class="bg-bratrax-acid px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
          on:click={handleSubmit}
          disabled={loading || selectedIds.size === 0}
        >
          {loading ? "CONNECTING..." : `CONNECT ${selectedIds.size} ACCOUNT${selectedIds.size !== 1 ? "S" : ""}`}
        </button>
      </div>
    </div>
  </div>
{/if}
