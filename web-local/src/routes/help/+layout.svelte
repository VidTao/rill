<script lang="ts">
  import { page } from "$app/stores";
  import { bratraxIsDemo, bratraxUser } from "$lib/bratrax/auth-store";
  import HelpSearch from "$lib/help/HelpSearch.svelte";
  import { groupedPagesFor, pagesFor } from "$lib/help";
  import { supportChatActions } from "@rilldata/web-common/features/chat/layouts/sidebar/support-chat-store";

  $: role = $bratraxUser?.role ?? null;
  // Search reads this list too, so demo-only pages stay out of a paying
  // customer's results, not just their sidebar.
  $: visiblePages = pagesFor(role, $bratraxIsDemo);

  // Sidebar sections come from each page's `group`, not its `audience` — see
  // the HelpGroup docs in $lib/help.
  $: navGroups = groupedPagesFor(role, $bratraxIsDemo);

  // Compared inline at each nav link (`p.slug === currentSlug`) rather than
  // through a helper. Svelte's update guard tracks the identifiers appearing in
  // the template expression, not what a function reads from its closure — so
  // `class:active={isActive(p)}` left `currentSlug` out of the guard entirely
  // and the highlight stayed on whichever page was open when the sidebar mounted.
  $: currentSlug = $page.url.pathname.replace(/^\/help\/?/, "");

  let searchQuery = "";
  $: searching = searchQuery.trim().length > 0;
</script>

<div class="flex h-full w-full overflow-hidden bg-bratrax-bg">
  <nav class="help-nav">
    <a class="help-home" href="/help" class:active={currentSlug === ""}>
      Help
    </a>

    <HelpSearch
      pages={visiblePages}
      variant="sidebar"
      placeholder="Search help…"
      bind:query={searchQuery}
    />

    {#if !searching}
      <div class="nav-section">
        <ul class="nav-list">
          <li>
            <!-- Opens the support-bot sidebar (same panel as the "?" header
                 button) instead of navigating — one chat, two doors. -->
            <button
              type="button"
              class="nav-link ask-link"
              on:click={supportChatActions.open}
            >
              Ask support
            </button>
          </li>
        </ul>
      </div>
    {/if}

    {#if !searching}
      {#each navGroups as g (g.id)}
        <div class="nav-section">
          <div class="nav-header">{g.label}</div>
          <ul class="nav-list">
            {#each g.pages as p (p.slug)}
              <li>
                <a
                  class="nav-link"
                  class:active={p.slug === currentSlug}
                  href="/help/{p.slug}"
                >
                  {p.title}
                  {#if p.status === "stub"}
                    <span class="status-tag">stub</span>
                  {/if}
                </a>
              </li>
            {/each}
          </ul>
        </div>
      {/each}
    {/if}
  </nav>

  <section class="help-content">
    <slot />
  </section>
</div>

<style lang="postcss">
  .help-nav {
    @apply flex flex-col h-full w-64 border-r border-bratrax-border bg-bratrax-surface py-3 px-2 overflow-y-auto flex-none gap-3;
  }

  .help-home {
    font-family: "Space Mono", monospace;
    font-size: 0.875rem;
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-acid-text);
    padding: 6px 8px;
    text-decoration: none;
    border-left: 2px solid transparent;
  }
  .help-home.active {
    background-color: var(--color-acid-mid);
    border-left-color: var(--color-acid);
  }

  .nav-section {
    @apply flex flex-col gap-1;
  }

  .nav-header {
    font-family: "Space Mono", monospace;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted);
    padding: 0 8px 2px;
  }

  .nav-list {
    @apply flex flex-col gap-0.5;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .nav-link {
    @apply block px-2 py-1.5 text-sm;
    color: var(--color-text);
    text-decoration: none;
    border-left: 2px solid transparent;
    transition:
      background-color 0.15s,
      color 0.15s,
      border-color 0.15s;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
  }
  .nav-link:hover {
    background-color: var(--color-acid-dim);
  }
  .nav-link.active {
    background-color: var(--color-acid-mid);
    border-left-color: var(--color-acid);
    font-weight: 500;
  }

  .ask-link {
    font-family: "Space Mono", monospace;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-acid-text);
    /* button reset — this entry is a <button>, its siblings are <a> */
    width: 100%;
    background: none;
    border: none;
    border-left: 2px solid transparent;
    text-align: left;
    cursor: pointer;
  }

  .status-tag {
    font-family: "Space Mono", monospace;
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--color-text-muted);
    background: var(--color-acid-dim);
    padding: 1px 6px;
    border-radius: 3px;
  }

  .help-content {
    @apply flex-1 overflow-y-auto;
  }
</style>
