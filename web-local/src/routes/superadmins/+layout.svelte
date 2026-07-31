<script lang="ts">
  import { page } from "$app/stores";

  // Tab routes — keep these aligned with the actual SvelteKit routes under
  // src/routes/superadmins/. Adding a new page = adding a row here.
  const TABS = [
    { href: "/superadmins", label: "Overview" },
    { href: "/superadmins/email-log", label: "Email log" },
    { href: "/superadmins/client-stats", label: "Client stats" },
  ];

  // "Overview" matches /superadmins exactly; "Email log" matches the
  // subpath. Compare against $page.url.pathname so links highlight as
  // expected.
  function isActive(href: string, current: string): boolean {
    if (href === "/superadmins") return current === "/superadmins";
    return current.startsWith(href);
  }
</script>

<div class="flex h-full w-full flex-col overflow-auto bg-bratrax-bg">
  <nav class="border-b border-bratrax-border bg-bratrax-surface">
    <div class="mx-auto flex w-full max-w-6xl gap-1 px-4">
      {#each TABS as tab}
        <a
          href={tab.href}
          class="tab"
          class:tab-active={isActive(tab.href, $page.url.pathname)}
        >
          {tab.label}
        </a>
      {/each}
    </div>
  </nav>
  <slot />
</div>

<style>
  .tab {
    padding: 12px 16px;
    font-family: "Outfit", sans-serif;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--bratrax-text-muted, #858585);
    border-bottom: 2px solid transparent;
    transition: color 120ms ease, border-color 120ms ease;
  }
  .tab:hover {
    color: var(--bratrax-text-primary, #f5f5f7);
  }
  .tab-active {
    color: #d4ff00;
    border-bottom-color: #d4ff00;
  }
</style>
