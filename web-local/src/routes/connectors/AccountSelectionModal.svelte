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
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click={onClose}>
    <div
      class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
      on:click|stopPropagation
    >
      <h2 class="mb-1 text-lg font-semibold text-gray-800">
        Select {platformName} Accounts
      </h2>
      <p class="mb-4 text-sm text-gray-500">
        Choose which accounts to connect
      </p>

      {#if accounts.length === 0}
        <p class="py-4 text-center text-sm text-gray-500">No accounts found.</p>
      {:else}
        <div class="mb-4 max-h-64 overflow-auto rounded border">
          <div class="sticky top-0 border-b bg-gray-50 px-3 py-2">
            <label class="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={allSelected}
                on:change={toggleAll}
                class="rounded"
              />
              <span class="font-medium text-gray-600">Select All</span>
            </label>
          </div>
          {#each accounts as account (account.id)}
            <label class="flex items-center gap-2 border-b px-3 py-2 last:border-0 hover:bg-gray-50 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={selectedIds.has(account.id)}
                on:change={() => toggleAccount(account.id)}
                class="rounded"
              />
              <span class="text-gray-800">{account.name}</span>
              <span class="text-xs text-gray-400">({account.id})</span>
            </label>
          {/each}
        </div>
      {/if}

      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="rounded bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
          on:click={onClose}
          disabled={loading}
        >
          Cancel
        </button>
        <button
          type="button"
          class="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
          on:click={handleSubmit}
          disabled={loading || selectedIds.size === 0}
        >
          {loading ? "Connecting..." : `Connect ${selectedIds.size} Account${selectedIds.size !== 1 ? "s" : ""}`}
        </button>
      </div>
    </div>
  </div>
{/if}
