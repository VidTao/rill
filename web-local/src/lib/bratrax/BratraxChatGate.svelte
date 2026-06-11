<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { getAISettings } from "$lib/bratrax/settings/api";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import DeveloperChat from "@rilldata/web-common/features/chat/DeveloperChat.svelte";
  import {
    chatOpen,
    sidebarActions,
    sidebarWidth,
  } from "@rilldata/web-common/features/chat/layouts/sidebar/sidebar-store";

  // Gate the chat sidebar on the BYOK Anthropic key. When no key is set, render
  // a Bratrax-styled empty state with a CTA to /settings → AI; when a key IS
  // set, mount the normal DeveloperChat.

  let keySet: boolean | null = null;
  let checking = true;

  $: canManageKey =
    $bratraxUser?.role === "super_admin" || $bratraxUser?.role === "admin";

  async function refreshKeyStatus() {
    checking = true;
    try {
      const ai = await getAISettings();
      keySet = ai.key_set;
    } catch {
      // /settings/ai requires auth; if it fails treat as "not set" so the
      // user sees the CTA rather than a broken chat.
      keySet = false;
    } finally {
      checking = false;
    }
  }

  onMount(refreshKeyStatus);

  // When the user opens the chat after setting a key in another tab, re-check.
  $: if ($chatOpen) void refreshKeyStatus();

  function goToAISettings() {
    sidebarActions.closeChat();
    goto("/settings/ai");
  }
</script>

{#if !$chatOpen}
  <!-- Chat is closed; render nothing (matches DeveloperChat's behavior). -->
{:else if checking || keySet === null}
  <aside class="chat-sidebar" style="--sidebar-width: {$sidebarWidth}px;">
    <div class="grid h-full place-items-center font-mono text-xs text-bratrax-text-muted">
      Checking key…
    </div>
  </aside>
{:else if keySet}
  <slot>
    <DeveloperChat />
  </slot>
{:else}
  <aside class="chat-sidebar" style="--sidebar-width: {$sidebarWidth}px;">
    <div class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center">
      <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        AI CHAT
      </div>
      <h2 class="text-xl font-black text-bratrax-text-headline">
        Add your <span class="font-serif italic text-bratrax-acid">Anthropic key</span>
      </h2>
      <p class="text-sm font-light text-bratrax-text-body">
        Bratrax routes Claude through your own Anthropic account. Set up a key in
        Settings → AI to enable chat.
      </p>
      {#if canManageKey}
        <button
          type="button"
          on:click={goToAISettings}
          class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
        >
          Open Settings → AI
        </button>
      {:else}
        <p class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
          Ask an admin to add a key.
        </p>
      {/if}
      <button
        type="button"
        on:click={() => sidebarActions.closeChat()}
        class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
      >
        Close
      </button>
    </div>
  </aside>
{/if}

<style lang="postcss">
  .chat-sidebar {
    @apply flex flex-col relative h-full bg-surface-background border;
    width: var(--sidebar-width);
  }
</style>
