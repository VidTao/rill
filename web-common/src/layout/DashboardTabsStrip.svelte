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
  // Right-edge reserve for CUSTOMIZE + breathing room. Admin/super see the
  // CUSTOMIZE pill; viewers don't, so reserve less for them.
  $: reservedRight = isAdminOrSuper ? 160 : 24;
  // MORE +N ▾ trigger width when at least one tab is overflowing — included
  // in the budget so the last visible tab doesn't get clipped by MORE. Sized
  // for the larger 13px tabs plus the "+N" count suffix.
  const MORE_TRIGGER_WIDTH = 96;

  let containerEl: HTMLDivElement;
  let measurerEl: HTMLDivElement | undefined;
  let widthBudget = Infinity;
  let primaryCap = USER_PRIMARY_CAP;

  $: activeCanvasName = $page.params.name ?? "";
  $: onCustomize = $page.url.pathname.startsWith("/customize");

  // Real DOM measurement: render every candidate tab in a hidden measurer,
  // sum their actual widths against the live container width, and only
  // collapse the surplus into MORE ▾. Beats the previous "assume 220px per
  // tab" heuristic that pushed tabs into MORE while half the strip sat
  // empty.
  function recomputeCap() {
    if (!measurerEl || !isFinite(widthBudget)) {
      primaryCap = Math.min(USER_PRIMARY_CAP, tabs.length);
      return;
    }
    const buttons = Array.from(
      measurerEl.querySelectorAll<HTMLElement>(".measure-tab"),
    );
    let sum = 0;
    let fits = 0;
    for (let i = 0; i < buttons.length; i++) {
      const w = buttons[i].offsetWidth;
      // MORE ▾ renders whenever any tab spills past the inline strip —
      // either because a candidate didn't fit, OR because the user has
      // more tabs than USER_PRIMARY_CAP (the surplus always goes to MORE).
      // If MORE will appear, its width has to fit too.
      const moreWillRender = i < buttons.length - 1 || tabs.length > buttons.length;
      const overhead =
        (moreWillRender ? MORE_TRIGGER_WIDTH : 0) + reservedRight;
      if (sum + w + overhead > widthBudget) break;
      sum += w;
      fits = i + 1;
    }
    primaryCap = Math.max(1, Math.min(USER_PRIMARY_CAP, fits));
  }

  // Recompute on any change to inputs that affect the layout.
  $: tabs, widthBudget, reservedRight, void tick().then(recomputeCap);

  // Pin the active tab. Whatever is currently selected must stay visible and
  // never collapse into MORE — otherwise selecting an overflowed dashboard
  // makes it vanish on a narrow window, which reads as broken. If the active
  // tab falls past the cap, swap it into the last visible slot and push the
  // displaced (inactive) tab into MORE. Overflow only ever hides inactive tabs.
  $: ({ primary, overflow } = partitionTabs(
    tabs,
    primaryCap,
    activeCanvasName,
    onCustomize,
  ));

  function partitionTabs(
    all: DashboardTab[],
    cap: number,
    activeName: string,
    customizeActive: boolean,
  ): { primary: DashboardTab[]; overflow: DashboardTab[] } {
    const primary = all.slice(0, cap);
    const overflow = all.slice(cap);
    if (customizeActive || overflow.length === 0) return { primary, overflow };
    const activeIdx = overflow.findIndex((t) => t.key === activeName);
    if (activeIdx === -1) return { primary, overflow };
    // Swap the active overflow tab into the last visible slot; the bumped tab
    // takes the active tab's old place in overflow (order otherwise preserved).
    const lastIdx = primary.length - 1;
    const bumped = primary[lastIdx];
    primary[lastIdx] = overflow[activeIdx];
    overflow[activeIdx] = bumped;
    return { primary, overflow };
  }

  onMount(() => {
    // Observe width changes so MORE ▾ kicks in exactly when the inline
    // strip would otherwise overflow.
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
          More +{overflow.length} <span class="more-chev">▾</span>
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

<!-- Hidden measurement strip: every candidate tab rendered with the real
     `.dashboard-tab` styling so `offsetWidth` matches what the visible
     anchors would render at. Sits off-screen, doesn't intercept pointer
     events, and is ignored by AT (aria-hidden). The visible strip's
     `recomputeCap()` reads widths from here. -->
<div
  bind:this={measurerEl}
  class="dashboard-tab-measurer"
  aria-hidden="true"
>
  {#each tabs.slice(0, USER_PRIMARY_CAP) as tab (tab.key)}
    <span class="dashboard-tab measure-tab">{tab.label}</span>
  {/each}
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
    font-size: 13px;
    letter-spacing: 2px;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--color-text-secondary);
    text-decoration: none;
    border-bottom: 2px solid transparent;
    transition:
      color 0.2s,
      border-color 0.2s;
    background: transparent;
    cursor: pointer;
  }

  .dashboard-tab:hover {
    color: var(--bratrax-text-headline, var(--color-text));
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
  /* Customize is a low-frequency action — keep it muted (matches MORE)
     even on the /customize page so it doesn't pull the eye like an active
     dashboard tab. !important is required because bratrax-theme.css has a
     global `a { color: var(--accent-primary) !important }` rule that paints
     every anchor acid — MORE escapes it (it's a <button>, not an <a>), but
     CUSTOMIZE is an <a> so it inherits the acid override unless we beat it. */
  :global(.customize-tab),
  :global(.customize-tab.active) {
    color: var(--color-text-muted) !important;
    border-bottom-color: transparent !important;
  }
  :global(.customize-tab:hover) {
    color: var(--color-text) !important;
  }

  :global(.more-trigger) {
    /* same as .dashboard-tab — duplicated as :global so bits-ui's <button>
       picks it up, since scoping doesn't cross the component boundary. */
    display: inline-flex;
    align-items: center;
    padding: 14px 16px;
    white-space: nowrap;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 13px;
    letter-spacing: 2px;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--color-text-secondary);
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
    color: var(--bratrax-text-headline, var(--color-text));
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
  /* Scoped under .more-dropdown-content (specificity 0,2,0) so it beats
     bratrax-theme.css §15, which floors every .text-xs element to 14px
     !important (bits-ui applies text-xs to each item). These are overflowed
     Tier 2 tabs, so they match the 13px / 2px-tracking section-tab size. */
  :global(.more-dropdown-content .more-dropdown-item) {
    padding: 10px 16px !important;
    font-family: "Space Mono", "JetBrains Mono", monospace !important;
    font-size: 13px !important;
    letter-spacing: 2px !important;
    font-weight: 700 !important;
    text-transform: uppercase !important;
    color: var(--color-text) !important;
    cursor: pointer;
  }
  :global(.more-dropdown-content .more-dropdown-item:hover) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06)) !important;
  }

  /* Off-screen measurer — anchors render at full size so offsetWidth
     matches what the visible strip would render at, but it doesn't paint
     and doesn't intercept events. */
  .dashboard-tab-measurer {
    position: absolute;
    left: -10000px;
    top: 0;
    visibility: hidden;
    pointer-events: none;
    white-space: nowrap;
    display: flex;
  }
</style>
