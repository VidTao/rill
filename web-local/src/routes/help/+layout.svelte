<script lang="ts">
  import { page } from "$app/stores";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import HelpSearch from "$lib/help/HelpSearch.svelte";
  import { pagesFor, type HelpPage } from "$lib/help";

  $: role = $bratraxUser?.role ?? null;
  $: visiblePages = pagesFor(role);

  $: viewerPages = visiblePages.filter((p) => p.audience === "viewer");
  $: adminPages = visiblePages.filter((p) => p.audience === "admin");
  $: sharedPages = visiblePages.filter((p) => p.audience === "shared");

  $: currentSlug = $page.url.pathname.replace(/^\/help\/?/, "");

  let searchQuery = "";
  $: searching = searchQuery.trim().length > 0;

  function isActive(p: HelpPage): boolean {
    return currentSlug === p.slug;
  }
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
            <a
              class="nav-link ask-link"
              class:active={currentSlug === "ask"}
              href="/help/ask"
            >
              Ask support
            </a>
          </li>
        </ul>
      </div>
    {/if}

    {#if !searching && viewerPages.length > 0}
      <div class="nav-section">
        <div class="nav-header">For everyone</div>
        <ul class="nav-list">
          {#each viewerPages as p (p.slug)}
            <li>
              <a
                class="nav-link"
                class:active={isActive(p)}
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
    {/if}

    {#if !searching && adminPages.length > 0}
      <div class="nav-section">
        <div class="nav-header">For admins</div>
        <ul class="nav-list">
          {#each adminPages as p (p.slug)}
            <li>
              <a
                class="nav-link"
                class:active={isActive(p)}
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
    {/if}

    {#if !searching && sharedPages.length > 0}
      <div class="nav-section">
        <div class="nav-header">Reference</div>
        <ul class="nav-list">
          {#each sharedPages as p (p.slug)}
            <li>
              <a
                class="nav-link"
                class:active={isActive(p)}
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
