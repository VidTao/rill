<script lang="ts">
  import { onDestroy } from "svelte";
  import { createEventDispatcher } from "svelte";
  import {
    wooPluginDownloadUrl,
    verifyWooInstall,
    type WooVerifyInstallResult,
  } from "./api";

  // The active client to verify events for. Required.
  export let clientId: string;

  const dispatch = createEventDispatcher<{
    verified: WooVerifyInstallResult;
  }>();

  type Status = "idle" | "checking" | "waiting" | "connected" | "error";
  let status: Status = "idle";
  let result: WooVerifyInstallResult | null = null;
  let error = "";
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let pollsLeft = 0;

  const EXPECTED = [
    "page_viewed",
    "product_viewed",
    "product_added_to_cart",
    "checkout_started",
    "checkout_completed",
  ];

  const downloadUrl = wooPluginDownloadUrl();

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  async function checkOnce(): Promise<void> {
    if (!clientId) {
      error = "No active client to verify.";
      status = "error";
      return;
    }
    try {
      result = await verifyWooInstall(clientId);
      error = "";
      if (result.connected) {
        status = "connected";
        stopPolling();
        dispatch("verified", result);
      } else if (status !== "waiting") {
        status = "idle";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Verification failed";
      status = "error";
      stopPolling();
    }
  }

  // Arm a short auto-poll window (the merchant browses their store while we watch).
  function startVerifying(): void {
    status = "waiting";
    pollsLeft = 24; // ~2 min at 5s
    void checkOnce();
    stopPolling();
    pollTimer = setInterval(() => {
      pollsLeft -= 1;
      if (pollsLeft <= 0) {
        stopPolling();
        if (status === "waiting") status = "idle";
        return;
      }
      void checkOnce();
    }, 5000);
  }

  onDestroy(stopPolling);
</script>

<div class="flex flex-col gap-4 rounded-lg border border-gray-200 p-5">
  <div>
    <h3 class="text-base font-semibold">
      Install the WooCommerce tracking plugin
    </h3>
    <p class="mt-1 text-sm text-gray-600">
      Download the plugin, install it on your WordPress store, and connect it to
      Bratrax. We'll confirm it's working when the first events arrive.
    </p>
  </div>

  <ol class="flex flex-col gap-2 text-sm text-gray-700">
    <li class="flex items-start gap-2">
      <span class="font-semibold">1.</span>
      <span>
        <a
          href={downloadUrl}
          class="font-semibold text-blue-600 underline"
          download
        >
          Download the plugin (.zip)
        </a>
      </span>
    </li>
    <li class="flex items-start gap-2">
      <span class="font-semibold">2.</span>
      <span
        >In WordPress admin, go to <strong
          >Plugins &rarr; Add New &rarr; Upload Plugin</strong
        >, choose the zip, install and activate.</span
      >
    </li>
    <li class="flex items-start gap-2">
      <span class="font-semibold">3.</span>
      <span
        >Go to <strong>WooCommerce &rarr; Bratrax</strong> and click
        <strong>"Connect to Bratrax"</strong>. Your workspace links
        automatically &mdash; no IDs to copy.</span
      >
    </li>
    <li class="flex items-start gap-2">
      <span class="font-semibold">4.</span>
      <span
        >Visit a product page and add it to the cart, then verify below.</span
      >
    </li>
  </ol>

  <div class="flex items-center gap-3">
    <button
      type="button"
      class="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
      on:click={startVerifying}
      disabled={status === "waiting" || status === "connected"}
    >
      {#if status === "waiting"}
        Watching for events&hellip;
      {:else if status === "connected"}
        Connected
      {:else}
        Verify install
      {/if}
    </button>

    {#if status === "connected"}
      <span class="text-sm font-semibold text-green-600">
        &#10003; Receiving events ({result?.events_seen})
      </span>
    {:else if status === "waiting"}
      <span class="text-sm text-gray-500">
        Browse your store in another tab &mdash; events appear within a minute.
      </span>
    {/if}
  </div>

  {#if error}
    <p class="text-sm text-red-600">{error}</p>
  {/if}

  {#if result}
    <div class="rounded-md bg-gray-50 p-3 text-sm">
      <p class="mb-1 font-semibold text-gray-700">
        Events seen (last {result.lookback_minutes} min): {result.events_seen}
      </p>
      <ul class="grid grid-cols-1 gap-0.5 sm:grid-cols-2">
        {#each EXPECTED as ev}
          <li class="flex items-center gap-2">
            <span
              class={result.by_type[ev] ? "text-green-600" : "text-gray-400"}
            >
              {result.by_type[ev] ? "✓" : "—"}
            </span>
            <span class="font-mono text-xs">{ev}</span>
            {#if result.by_type[ev]}
              <span class="text-xs text-gray-500">({result.by_type[ev]})</span>
            {/if}
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
