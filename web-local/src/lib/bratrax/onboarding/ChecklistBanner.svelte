<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { Info } from "lucide-svelte";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { getChecklist, coreSetupIncomplete } from "./checklist";

  let incomplete = false;

  $: role = $bratraxUser?.role ?? null;
  $: isAdmin = role === "admin" || role === "super_admin";
  // Dashboard surfaces only. Never on the checklist page itself or setup pages.
  $: path = $page.url.pathname;
  $: onDashboard =
    path === "/developer" ||
    path.startsWith("/canvas") ||
    path.startsWith("/explore") ||
    path.startsWith("/files");
  $: visible = isAdmin && onDashboard && incomplete;

  async function check() {
    if (!isAdmin || !onDashboard) {
      incomplete = false;
      return;
    }
    try {
      const c = await getChecklist();
      incomplete = coreSetupIncomplete(c);
    } catch {
      incomplete = false;
    }
  }

  onMount(check);
  // Re-check when navigating between dashboard surfaces (e.g. after completing
  // a setup item and returning).
  $: if (isAdmin && onDashboard) void check();
</script>

{#if visible}
  <div class="banner">
    <Info size={16} class="binfo" />
    <span class="msg">Your setup isn't finished yet — data may be incomplete.</span>
    <button class="resume" on:click={() => goto("/onboarding")}>Resume setup →</button>
  </div>
{/if}

<style>
  .banner {
    display: flex;
    align-items: center;
    gap: 10px;
    background: var(--color-surface);
    border-left: 2px solid var(--color-acid);
    border-bottom: 1px solid var(--color-border);
    padding: 10px 16px;
  }
  :global(.binfo) {
    color: var(--color-acid-text);
    flex: 0 0 auto;
  }
  .msg {
    font-family: "Outfit", sans-serif;
    font-size: 14px;
    color: var(--color-text);
  }
  .resume {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    background: transparent;
    border: none;
    color: var(--color-acid-text);
    cursor: pointer;
    margin-left: auto;
  }
</style>
