<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { onboardMe } from "$lib/bratrax/onboarding/api";

  interface Advertiser {
    id: string;
    name: string;
    currency: string;
    timezone: string;
  }

  let status = "Completing TikTok connection...";
  let error = "";
  let loaded = false;
  let submitting = false;
  let state = "";
  let advertisers: Advertiser[] = [];
  let selected: Set<string> = new Set();
  let clientId = "";

  onMount(async () => {
    const code = $page.url.searchParams.get("auth_code") || $page.url.searchParams.get("code");
    const stateParam = $page.url.searchParams.get("state");
    const expectedState = sessionStorage.getItem("tiktok_ads_oauth_state");

    if (!code || !stateParam) {
      error = "Missing authorization code. Please try connecting again.";
      return;
    }
    if (expectedState && stateParam !== expectedState) {
      error = "State mismatch. Possible CSRF — please try connecting again.";
      return;
    }

    try {
      const host = get(runtime).host;

      const me = await onboardMe();
      clientId = me?.client_id || sessionStorage.getItem("onboard_client_id") || "";
      if (!clientId) {
        error = "Could not determine client. Please restart onboarding.";
        return;
      }

      status = "Exchanging authorization code...";
      const accountsUrl = `${host}/bratrax/onboard/tiktok/accounts?code=${encodeURIComponent(
        code,
      )}&state=${encodeURIComponent(stateParam)}`;
      const res = await fetch(accountsUrl, { credentials: "include" });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? "Failed to list TikTok advertisers");
      }

      const body = (await res.json()) as { state: string; advertisers: Advertiser[] };
      state = body.state;
      advertisers = body.advertisers ?? [];
      selected = new Set(advertisers.map((a) => a.id));
      loaded = true;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  });

  function toggle(id: string) {
    if (selected.has(id)) selected.delete(id);
    else selected.add(id);
    selected = selected;
  }

  async function confirm() {
    if (selected.size === 0) {
      error = "Select at least one advertiser.";
      return;
    }
    error = "";
    submitting = true;

    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}/bratrax/onboard/tiktok/connect`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          state,
          client_id: clientId,
          selected_advertiser_ids: [...selected],
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? "Failed to save TikTok advertisers");
      }

      sessionStorage.removeItem("tiktok_ads_oauth_state");
      await goto("/onboard/stack?connected=tiktok_ads");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      submitting = false;
    }
  }
</script>

<div class="flex h-screen w-screen items-start justify-center overflow-y-auto bg-surface py-12">
  <div class="w-full max-w-lg rounded-lg border border-border bg-white p-8 shadow-sm">
    <h1 class="mb-1 text-xl font-bold">Select TikTok advertisers</h1>
    <p class="mb-4 text-sm text-fg-secondary">
      Choose which advertiser accounts to connect to Bratrax. You can change this later.
    </p>

    {#if error}
      <div class="mb-4 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
        {error}
      </div>
      <a href="/onboard/stack" class="text-sm text-indigo-600 hover:underline">
        Back to onboarding
      </a>
    {:else if !loaded}
      <div class="flex items-center gap-3 py-6">
        <div class="h-5 w-5 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600"></div>
        <span class="text-sm text-fg-secondary">{status}</span>
      </div>
    {:else}
      <ul class="mb-6 max-h-80 divide-y divide-border overflow-y-auto rounded border border-border">
        {#each advertisers as a (a.id)}
          <li class="flex items-center gap-3 px-3 py-2">
            <input
              id={`adv-${a.id}`}
              type="checkbox"
              class="h-4 w-4 rounded border-border text-indigo-600 focus:ring-indigo-500"
              checked={selected.has(a.id)}
              on:change={() => toggle(a.id)}
            />
            <label for={`adv-${a.id}`} class="flex flex-1 items-center justify-between text-sm">
              <span class="font-medium">{a.name || a.id}</span>
              <span class="font-mono text-xs text-fg-secondary">
                {a.id}{a.currency ? ` · ${a.currency}` : ""}
              </span>
            </label>
          </li>
        {/each}
      </ul>

      <div class="flex justify-end gap-2">
        <a href="/onboard/stack" class="rounded border border-border bg-white px-4 py-2 text-sm hover:bg-surface">
          Cancel
        </a>
        <button
          type="button"
          class="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          disabled={submitting || selected.size === 0}
          on:click={confirm}
        >
          {submitting ? "Saving…" : `Connect ${selected.size} advertiser${selected.size === 1 ? "" : "s"}`}
        </button>
      </div>
    {/if}
  </div>
</div>
