<script lang="ts">
  import { page } from "$app/stores";
  import { onMount, tick } from "svelte";
  import * as DropdownMenu from "@rilldata/web-common/components/dropdown-menu";

  // Phase 3: Ribbon 2 respects per-user prefs.
  //
  // The merge (canvases × prefs) happens in the root layout (web-local), which
  // owns the dashboardPrefs store import. This component receives the result
  // as a flat `tabs` prop so it stays web-common-safe (no $lib imports).
  //
  // Layout:
  //   - Up to USER_PRIMARY_CAP visible tabs render inline; the rest collapse
  //     into a MORE ▾ dropdown
  //   - A ResizeObserver tracks Ribbon 2 width and lowers the effective cap
  //     on narrow viewports so tabs never wrap to a second row

  interface DashboardTab {
    key: string;
    label: string;
  }

  export let tabs: DashboardTab[] = [];
  export let isAdminOrSuper: boolean = false;

  const USER_PRIMARY_CAP = 6;

  let containerEl: HTMLDivElement;
  let widthBudget = Infinity;

  $: activeCanvasName = $page.params.name ?? "";
  $: onCustomize = $page.url.pathname.startsWith("/customize");

  // Effective primary cap = min(user cap, width-based cap). The width-based
  // cap is a coarse estimate: assume each tab averages 220px (Space Mono
  // 11px uppercase title in a 28px-padded button). We reserve ~200px on the
  // right for CUSTOMIZE + MORE ▾ + padding.
  $: estimatedTabWidth = 220;
  $: reservedRight = 200;
  $: widthCap = isFinite(widthBudget)
    ? Math.max(1, Math.floor((widthBudget - reservedRight) / estimatedTabWidth))
    : USER_PRIMARY_CAP;
  $: primaryCap = Math.min(USER_PRIMARY_CAP, widthCap);

  $: primary = tabs.slice(0, primaryCap);
  $: overflow = tabs.slice(primaryCap);

  onMount(() => {
    // Observe width changes so MORE ▾ kicks in before tabs overflow.
    let observer: ResizeObserver | null = null;
    void tick().then(() => {
      if (!containerEl || typeof ResizeObserver === "undefined") return;
      observer = new ResizeObserver((entries) => {
        for (const entry of entries) {
          widthBudget = entry.contentRect.width;
        }
      });
      observer.observe(containerEl);
    });
    return () => observer?.disconnect();
  });
</script>

<div class="dashboard-tabs-strip" bind:this={containerEl}>
  <nav class="tabs">
    {#each primary as tab (tab.key)}
      <a
        href={`/canvas/${tab.key}`}
        class="dashboard-tab"
        class:active={!onCustomize && tab.key === activeCanvasName}
      >
        {tab.label}
      </a>
    {/each}

    {#if overflow.length > 0}
      {@const overflowActive = overflow.some(
        (t) => !onCustomize && t.key === activeCanvasName,
      )}
      <DropdownMenu.Root>
        <DropdownMenu.Trigger
          class={overflowActive
            ? "dashboard-tab more-trigger active"
            : "dashboard-tab more-trigger"}
        >
          More <span class="more-chev">▾</span>
        </DropdownMenu.Trigger>
        <DropdownMenu.Content class="more-dropdown-content" sideOffset={4}>
          {#each overflow as tab (tab.key)}
            <DropdownMenu.Item
              href={`/canvas/${tab.key}`}
              class="more-dropdown-item"
            >
              {tab.label}
            </DropdownMenu.Item>
          {/each}
        </DropdownMenu.Content>
      </DropdownMenu.Root>
    {/if}
  </nav>

  {#if isAdminOrSuper}
    <a
      href="/customize"
      class="dashboard-tab customize-tab"
      class:active={onCustomize}
    >
      Customize
    </a>
  {/if}
</div>

<style lang="postcss">
  .dashboard-tabs-strip {
    @apply flex items-stretch w-full px-6 border-b;
    border-color: var(--color-border, var(--border));
    background: var(--color-surface-base, var(--background));
  }

  .tabs {
    @apply flex items-stretch flex-1 min-w-0;
  }

  /* Plain anchor tabs */
  .dashboard-tab {
    @apply inline-flex items-center px-4 py-3.5 whitespace-nowrap;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    letter-spacing: 0.12em;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--color-text-muted);
    text-decoration: none;
    border-bottom: 2px solid transparent;
    transition:
      color 0.2s,
      border-color 0.2s;
    background: transparent;
    cursor: pointer;
  }

  .dashboard-tab:hover {
    color: var(--color-text);
  }

  /* Use :global on .active for the dropdown trigger button (bits-ui renders
     the button outside this component's scoped CSS tree). */
  :global(.dashboard-tab.active) {
    color: var(--color-acid);
    border-bottom-color: var(--color-acid);
  }

  .customize-tab {
    border-left: 1px solid var(--color-border, var(--border));
    margin-left: 12px;
    padding-left: 28px;
    flex-shrink: 0;
  }

  :global(.more-trigger) {
    /* same as .dashboard-tab — duplicated as :global so bits-ui's <button>
       picks it up, since scoping doesn't cross the component boundary. */
    display: inline-flex;
    align-items: center;
    padding: 14px 16px;
    white-space: nowrap;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    letter-spacing: 0.12em;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--color-text-muted);
    text-decoration: none;
    border: none;
    border-bottom: 2px solid transparent;
    background: transparent;
    cursor: pointer;
    transition:
      color 0.2s,
      border-color 0.2s;
  }
  :global(.more-trigger:hover) {
    color: var(--color-text);
  }
  :global(.more-trigger .more-chev) {
    margin-left: 4px;
    font-size: 10px;
  }

  :global(.more-dropdown-content) {
    background: var(--color-surface, var(--background));
    border: 1px solid var(--color-border-strong, var(--color-border));
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6);
    min-width: 240px;
  }
  :global(.more-dropdown-item) {
    padding: 10px 16px !important;
    font-family: "Space Mono", "JetBrains Mono", monospace !important;
    font-size: 11px !important;
    letter-spacing: 0.1em !important;
    font-weight: 700 !important;
    text-transform: uppercase !important;
    color: var(--color-text) !important;
    cursor: pointer;
  }
  :global(.more-dropdown-item:hover) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06)) !important;
  }
</style>
