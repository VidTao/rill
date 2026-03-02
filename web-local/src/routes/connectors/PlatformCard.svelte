<script lang="ts">
  export let name: string;
  export let logo: string;
  export let description: string;
  export let connected = false;
  export let connectedDate = "";
  export let loading = false;
  export let onConnect: () => void = () => {};
  export let onViewDetails: () => void = () => {};
</script>

<button
  class="group relative flex w-full flex-col rounded-lg border bg-white p-5 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md {connected
    ? 'border-green-200 hover:bg-green-50/50'
    : 'border-border hover:bg-blue-50/50'}"
  class:opacity-70={loading}
  on:click={connected ? onViewDetails : onConnect}
>
  <div class="mb-3 h-8">
    <img
      src="/img/bratrax/{logo}"
      alt={name}
      class="h-full w-auto object-contain"
    />
  </div>

  <span class="mb-1 text-sm font-semibold text-gray-800">{name}</span>

  {#if connected}
    <span class="absolute right-3 top-3 flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-green-600">
      <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
    </span>
  {/if}

  <div class="mb-3">
    {#if connected}
      <span class="text-xs text-gray-500">
        Connected on: <strong>{connectedDate}</strong>
      </span>
      <span
        class="mt-1 inline-flex items-center gap-1 rounded-full bg-green-50 px-2.5 py-1 text-xs text-green-700 group-hover:bg-green-100"
      >
        See connection details
        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </span>
    {:else}
      <span class="text-xs text-gray-500">{description}</span>
    {/if}
  </div>

  <div class="mt-auto flex items-center gap-1 text-sm font-medium {connected ? 'text-green-600' : 'text-blue-600'}">
    {#if loading}
      <span>Connecting...</span>
      <svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    {:else if connected}
      <span>Connected</span>
    {:else}
      <span>Connect Account</span>
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M14 5l7 7m0 0l-7 7m7-7H3" />
      </svg>
    {/if}
  </div>
</button>
