<script lang="ts">
  import type { AdvertisingConnection, CrmConnection } from "$lib/bratrax/connectors/types";

  export let open = false;
  export let platformName = "";
  export let adAccounts: AdvertisingConnection[] = [];
  export let crmAccounts: CrmConnection[] = [];
  export let onClose: () => void = () => {};

  $: hasAds = adAccounts.length > 0;
  $: hasCrm = crmAccounts.length > 0;
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" on:click={onClose}>
    <div
      class="relative w-full max-w-lg border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-bold text-bratrax-text-headline">{platformName} Connections</h2>
        <button class="text-bratrax-text-muted hover:text-bratrax-text-primary" on:click={onClose}>
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {#if hasAds}
        <h3 class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">Ad Accounts</h3>
        <div class="mb-4 overflow-auto border border-bratrax-border">
          <table class="w-full text-left text-sm">
            <thead class="border-b border-bratrax-border bg-bratrax-bg font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
              <tr>
                <th class="px-3 py-2">Account</th>
                <th class="px-3 py-2">ID</th>
                <th class="px-3 py-2">Currency</th>
                <th class="px-3 py-2">Timezone</th>
              </tr>
            </thead>
            <tbody>
              {#each adAccounts as acc}
                <tr class="border-b border-bratrax-border last:border-0">
                  <td class="px-3 py-2 font-medium text-bratrax-text-primary">{acc.accountName}</td>
                  <td class="px-3 py-2 font-mono text-xs text-bratrax-text-muted">{acc.accountId}</td>
                  <td class="px-3 py-2 text-bratrax-text-muted">{acc.currency}</td>
                  <td class="px-3 py-2 text-bratrax-text-muted">{acc.timezone}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if hasCrm}
        <h3 class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">Connected Stores</h3>
        <div class="overflow-auto border border-bratrax-border">
          <table class="w-full text-left text-sm">
            <thead class="border-b border-bratrax-border bg-bratrax-bg font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
              <tr>
                <th class="px-3 py-2">Store</th>
                <th class="px-3 py-2">URL</th>
                <th class="px-3 py-2">Currency</th>
                <th class="px-3 py-2">Timezone</th>
              </tr>
            </thead>
            <tbody>
              {#each crmAccounts as acc}
                <tr class="border-b border-bratrax-border last:border-0">
                  <td class="px-3 py-2 font-medium text-bratrax-text-primary">{acc.storeName}</td>
                  <td class="px-3 py-2 font-mono text-xs text-bratrax-text-muted">{acc.storeUrl}</td>
                  <td class="px-3 py-2 text-bratrax-text-muted">{acc.currency}</td>
                  <td class="px-3 py-2 text-bratrax-text-muted">{acc.timezone}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if !hasAds && !hasCrm}
        <p class="text-sm text-bratrax-text-muted">No connection details available.</p>
      {/if}

      <div class="mt-4 flex justify-end">
        <button
          class="border border-bratrax-border px-4 py-2 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover"
          on:click={onClose}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
