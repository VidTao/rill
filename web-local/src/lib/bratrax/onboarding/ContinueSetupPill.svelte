<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { getChecklist, progressCounts } from "./checklist";

  // Phase 2 of the nav restructure (NAV_RESTRUCTURE_HANDOFF.MD). Conditional
  // CTA sitting in Ribbon 1 between the client switcher and SETTINGS ▾.
  //
  // Visibility rules:
  //   - role === "admin" (NOT super_admin — they're acting on someone else's
  //     workspace; NOT viewer — they can't action setup items)
  //   - workspace step === "ready" (pre-compilation the funnel handles it)
  //   - at least one checklist item is still "todo"/"in_progress"/"error"
  //   - !checklist_dismissed (clicking "Hide for now" on /onboarding kills it)
  //
  // Completion does NOT auto-hide the pill: a fully-✓ checklist still surfaces
  // the pill until the user explicitly dismisses, by design (per the spec).

  let visible = false;

  $: role = $bratraxUser?.role ?? null;
  $: isAdmin = role === "admin";
  // Refetch when navigating off /onboarding back to a dashboard — picks up the
  // user's just-clicked "Hide for now" without a page reload.
  $: path = $page.url.pathname;

  let lastCheckedPath: string | null = null;

  async function check() {
    if (!isAdmin) {
      visible = false;
      return;
    }
    try {
      const c = await getChecklist();
      if (c.checklist_dismissed) {
        visible = false;
        return;
      }
      if (c.workspace_step !== "ready") {
        visible = false;
        return;
      }
      const { done, total } = progressCounts(c);
      visible = done < total;
    } catch {
      visible = false;
    }
  }

  onMount(() => {
    lastCheckedPath = path;
    void check();
  });

  $: if (isAdmin && path !== lastCheckedPath) {
    lastCheckedPath = path;
    void check();
  }
</script>

{#if visible}
  <!-- Styling follows the canonical Bratrax acid-CTA pattern (see
       BratraxChatGate.svelte, WelcomeBanner.svelte): bg-bratrax-acid +
       text-bratrax-bg. Both tokens flip through bratrax-theme.css per the
       active light/dark theme, so the pill auto-adapts. -->
  <a
    href="/onboarding"
    class="continue-setup-pill bg-bratrax-acid text-bratrax-bg hover:opacity-90"
  >
    Continue setup →
  </a>
{/if}

<style lang="postcss">
  .continue-setup-pill {
    @apply inline-flex items-center;
    @apply px-3.5 py-2;
    @apply font-mono text-[11px] font-bold uppercase;
    letter-spacing: 0.12em;
    text-decoration: none;
    cursor: pointer;
    transition: opacity 120ms ease;
  }
  .continue-setup-pill:focus-visible {
    outline: 2px solid var(--color-acid);
    outline-offset: 2px;
  }
</style>
