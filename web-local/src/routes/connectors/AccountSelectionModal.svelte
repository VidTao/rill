<script lang="ts">
  import type { AdAccountInfo } from "$lib/bratrax/connectors/api";

  export let open = false;
  export let loading = false;
  export let platformName = "";
  // Optional `group` field on each account lets the modal section the list
  // by parent context (Facebook Business Manager, Google MCC manager). When
  // every entry's `group` is absent or empty, the modal renders a single
  // ungrouped list — same UX as callers passing the historical flat shape.
  export let accounts: Array<{ id: string; name: string; group?: string }> = [];
  export let onSubmit: (selected: AdAccountInfo[]) => void = () => {};
  export let onClose: () => void = () => {};

  let selectedIds: Set<string> = new Set();
  let searchQuery = "";

  // Reset selection and search whenever the modal (re)opens. The component
  // is mounted once and reused for every platform: without this, picking 1
  // FB account, closing, then opening the Google modal would carry the
  // previous selection into the new platform's counter.
  $: if (open) {
    selectedIds = new Set();
    searchQuery = "";
  }

  function toggleAccount(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    selectedIds = next;
  }

  function toggleAllVisible() {
    const next = new Set(selectedIds);
    const allInView =
      filteredAccounts.length > 0 &&
      filteredAccounts.every((a) => next.has(a.id));
    if (allInView) {
      for (const a of filteredAccounts) next.delete(a.id);
    } else {
      for (const a of filteredAccounts) next.add(a.id);
    }
    selectedIds = next;
  }

  function isGroupAllSelected(groupAccounts: Array<{ id: string }>): boolean {
    return (
      groupAccounts.length > 0 &&
      groupAccounts.every((a) => selectedIds.has(a.id))
    );
  }

  function toggleGroup(groupAccounts: Array<{ id: string }>) {
    const next = new Set(selectedIds);
    if (groupAccounts.every((a) => next.has(a.id))) {
      for (const a of groupAccounts) next.delete(a.id);
    } else {
      for (const a of groupAccounts) next.add(a.id);
    }
    selectedIds = next;
  }

  function handleSubmit() {
    const selected: AdAccountInfo[] = accounts
      .filter((a) => selectedIds.has(a.id))
      .map((a) => ({ accountId: a.id, accountName: a.name }));
    onSubmit(selected);
  }

  $: filteredAccounts = (() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return accounts;
    return accounts.filter(
      (a) =>
        a.name.toLowerCase().includes(q) || a.id.toLowerCase().includes(q),
    );
  })();

  // Preserves insertion order from the input list — callers control the
  // section ordering by pre-sorting their entries.
  $: groupedAccounts = (() => {
    const groups = new Map<string, Array<{ id: string; name: string }>>();
    for (const a of filteredAccounts) {
      const key = a.group ?? "";
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key)!.push({ id: a.id, name: a.name });
    }
    return [...groups.entries()];
  })();

  $: allVisibleSelected =
    filteredAccounts.length > 0 &&
    filteredAccounts.every((a) => selectedIds.has(a.id));
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
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
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search by name or ID…"
          class="mb-3 w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body placeholder:text-bratrax-text-muted/60 focus:border-bratrax-acid focus:outline-none"
        />

        {#if filteredAccounts.length === 0}
          <p class="py-4 text-center text-sm text-bratrax-text-muted">No accounts match this search.</p>
        {:else}
          <div class="mb-4 max-h-96 overflow-auto border border-bratrax-border">
            <div class="sticky top-0 z-10 border-b border-bratrax-border bg-bratrax-bg px-3 py-2">
              <label class="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={allVisibleSelected}
                  on:change={toggleAllVisible}
                  class="accent-bratrax-acid"
                />
                <span class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body">
                  Select All {searchQuery ? "Matching" : ""}
                </span>
              </label>
            </div>

            {#each groupedAccounts as [groupLabel, groupAccounts] (groupLabel)}
              {#if groupLabel}
                <div class="border-b border-bratrax-border bg-bratrax-surface px-3 py-2">
                  <label class="flex items-center gap-2 text-xs">
                    <input
                      type="checkbox"
                      checked={isGroupAllSelected(groupAccounts)}
                      on:change={() => toggleGroup(groupAccounts)}
                      class="accent-bratrax-acid"
                    />
                    <span class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/80">
                      {groupLabel}
                    </span>
                    <span class="font-mono text-[10px] text-bratrax-text-muted">
                      ({groupAccounts.length})
                    </span>
                  </label>
                </div>
              {/if}
              {#each groupAccounts as account (account.id)}
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
            {/each}
          </div>
        {/if}
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
