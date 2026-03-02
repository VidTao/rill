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
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click={onClose}>
    <div
      class="w-full max-w-lg rounded-lg bg-white p-6 shadow-xl"
      on:click|stopPropagation
    >
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold text-gray-800">{platformName} Connections</h2>
        <button class="text-gray-400 hover:text-gray-600" on:click={onClose}>
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {#if hasAds}
        <h3 class="mb-2 text-sm font-medium text-gray-600">Ad Accounts</h3>
        <div class="mb-4 overflow-auto rounded border">
          <table class="w-full text-left text-sm">
            <thead class="border-b bg-gray-50 text-xs text-gray-500">
              <tr>
                <th class="px-3 py-2">Account</th>
                <th class="px-3 py-2">ID</th>
                <th class="px-3 py-2">Currency</th>
                <th class="px-3 py-2">Timezone</th>
              </tr>
            </thead>
            <tbody>
              {#each adAccounts as acc}
                <tr class="border-b last:border-0">
                  <td class="px-3 py-2 font-medium">{acc.accountName}</td>
                  <td class="px-3 py-2 text-gray-500">{acc.accountId}</td>
                  <td class="px-3 py-2 text-gray-500">{acc.currency}</td>
                  <td class="px-3 py-2 text-gray-500">{acc.timezone}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if hasCrm}
        <h3 class="mb-2 text-sm font-medium text-gray-600">Connected Stores</h3>
        <div class="overflow-auto rounded border">
          <table class="w-full text-left text-sm">
            <thead class="border-b bg-gray-50 text-xs text-gray-500">
              <tr>
                <th class="px-3 py-2">Store</th>
                <th class="px-3 py-2">URL</th>
                <th class="px-3 py-2">Currency</th>
                <th class="px-3 py-2">Timezone</th>
              </tr>
            </thead>
            <tbody>
              {#each crmAccounts as acc}
                <tr class="border-b last:border-0">
                  <td class="px-3 py-2 font-medium">{acc.storeName}</td>
                  <td class="px-3 py-2 text-gray-500">{acc.storeUrl}</td>
                  <td class="px-3 py-2 text-gray-500">{acc.currency}</td>
                  <td class="px-3 py-2 text-gray-500">{acc.timezone}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if !hasAds && !hasCrm}
        <p class="text-sm text-gray-500">No connection details available.</p>
      {/if}

      <div class="mt-4 flex justify-end">
        <button
          class="rounded bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
          on:click={onClose}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
