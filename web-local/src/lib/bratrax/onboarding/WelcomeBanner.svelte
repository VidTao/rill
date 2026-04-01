<script lang="ts">
  import { onMount } from "svelte";

  const STORAGE_KEY = "bratrax_welcome_dismissed";

  let visible = false;

  onMount(() => {
    visible = localStorage.getItem(STORAGE_KEY) !== "true";
  });

  function dismiss() {
    visible = false;
    localStorage.setItem(STORAGE_KEY, "true");
  }

  function openChat() {
    // Rill's DashboardChat component listens for this custom event
    window.dispatchEvent(new CustomEvent("bratrax:open-chat"));
    dismiss();
  }
</script>

{#if visible}
  <div
    class="relative mx-4 mt-4 rounded-lg border border-indigo-200 bg-indigo-50 px-5 py-4"
  >
    <button
      on:click={dismiss}
      class="absolute right-3 top-3 text-indigo-400 hover:text-indigo-600"
      aria-label="Dismiss"
    >
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>

    <h2 class="text-base font-semibold text-indigo-900">
      Your analytics are live!
    </h2>
    <p class="mt-1 text-sm text-indigo-700">
      Your data refreshes every 1–3 hours. Attribution improves over the next
      few days as we collect browsing behavior from your store.
    </p>
    <div class="mt-3">
      <button
        on:click={openChat}
        class="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
      >
        Ask Claude →
      </button>
      <span class="ml-3 text-xs text-indigo-500">
        Try: "How is my ROAS this week?"
      </span>
    </div>
  </div>
{/if}
