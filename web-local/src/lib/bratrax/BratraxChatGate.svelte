<script lang="ts">
  import { openStartAttribution } from "./members";
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { getAISettings } from "$lib/bratrax/settings/api";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus";
  import DeveloperChat from "@rilldata/web-common/features/chat/DeveloperChat.svelte";
  import {
    chatOpen,
    sidebarActions,
    sidebarWidth,
  } from "@rilldata/web-common/features/chat/layouts/sidebar/sidebar-store";

  import type { DemoAIQuota } from "$lib/bratrax/settings/types";

  // Gate the chat sidebar on the BYOK Anthropic key. When no key is set, render
  // a Bratrax-styled empty state with a CTA to /settings → AI; when a key IS
  // set, mount the normal DeveloperChat.
  //
  // Demo visitors are a third case: they run on the platform key with a fixed
  // lifetime prompt allowance, and can never add a key of their own (they're
  // viewers on a shared workspace). Once spent, they get a sign-up CTA instead.

  let keySet: boolean | null = null;
  let demoAI: DemoAIQuota | null = null;
  let checking = true;

  $: canManageKey =
    $bratraxUser?.role === "super_admin" || $bratraxUser?.role === "admin";

  // `quiet` skips the "Checking key…" placeholder. The re-check triggered by a
  // spent allowance runs over a chat that is already on screen, and swapping it
  // for a spinner on the way to the CTA just adds a flash.
  async function refreshKeyStatus(quiet = false) {
    if (!quiet) checking = true;
    try {
      const ai = await getAISettings();
      keySet = ai.key_set;
      demoAI = ai.demo_ai ?? null;
    } catch {
      // /settings/ai requires auth; if it fails treat as "not set" so the
      // user sees the CTA rather than a broken chat.
      keySet = false;
      demoAI = null;
    } finally {
      checking = false;
    }
  }

  onMount(() => {
    void refreshKeyStatus();

    // The chat only re-checks when it opens, so a session that is already open
    // when the allowance runs out would otherwise sit on a transport error
    // until the page reloads. conversation.ts announces the 402 for us.
    return eventBus.on("ai-quota-exhausted", () => void refreshKeyStatus(true));
  });

  // When the user opens the chat after setting a key in another tab, re-check.
  $: if ($chatOpen) void refreshKeyStatus();

  function goToAISettings() {
    sidebarActions.closeChat();
    goto("/settings/ai");
  }

  function goToSignup() {
    // The demo account becomes the visitor's own, same login: "Start
    // attribution" (mounted by the root layout for demo-workspace viewers)
    // converts it in place, no sign-out and no second signup form.
    sidebarActions.closeChat();
    openStartAttribution();
  }
</script>

{#if !$chatOpen}
  <!-- Chat is closed; render nothing (matches DeveloperChat's behavior). -->
{:else if checking || keySet === null}
  <aside class="chat-sidebar" style="--sidebar-width: {$sidebarWidth}px;">
    <div
      class="grid h-full place-items-center font-mono text-xs text-bratrax-text-muted"
    >
      Checking key…
    </div>
  </aside>
{:else if demoAI?.exhausted}
  <!-- Checked before keySet: a demo user's key_set is true (the platform key
       serves them), so the allowance is what actually gates them. -->
  <aside class="chat-sidebar" style="--sidebar-width: {$sidebarWidth}px;">
    <div
      class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center"
    >
      <div
        class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        AI CHAT
      </div>
      <h2 class="text-xl font-black text-bratrax-text-headline">
        You've used your {demoAI.limit} free
        <span class="font-serif italic text-bratrax-acid">AI prompts</span>
      </h2>
      <p class="text-sm font-light text-bratrax-text-body">
        The demo account comes with {demoAI.limit} prompts on us. Start
        attribution on your own store to keep going with your own Anthropic key.
      </p>
      <button
        type="button"
        on:click={goToSignup}
        class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
      >
        Start attribution →
      </button>
      <button
        type="button"
        on:click={() => sidebarActions.closeChat()}
        class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
      >
        Close
      </button>
    </div>
  </aside>
{:else if keySet}
  <slot>
    <DeveloperChat />
  </slot>
{:else}
  <aside class="chat-sidebar" style="--sidebar-width: {$sidebarWidth}px;">
    <div
      class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center"
    >
      <div
        class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        AI CHAT
      </div>
      <h2 class="text-xl font-black text-bratrax-text-headline">
        Add your <span class="font-serif italic text-bratrax-acid"
          >Anthropic key</span
        >
      </h2>
      <p class="text-sm font-light text-bratrax-text-body">
        Bratrax routes Claude through your own Anthropic account. Set up a key
        in Settings → AI to enable chat.
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
        <p
          class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
        >
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
