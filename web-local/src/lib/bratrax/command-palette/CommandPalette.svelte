<script lang="ts">
  import { goto } from "$app/navigation";
  import * as Command from "@rilldata/web-common/components/command";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import { bratraxIsDemo } from "$lib/bratrax/auth-store";
  import { pagesFor } from "$lib/help";
  import {
    destinationsFor,
    matchDestinations,
    matchHelpPages,
    type PaletteRole,
  } from "./destinations";
  import { commandPaletteOpen, isMac } from "./store";

  /**
   * Global ⌘K / Ctrl+K search over config destinations and help articles.
   * Mounted once in the root layout. RangePickerV2 used to own ⌘K for the
   * time picker; it moved to ⌘⇧K.
   *
   * Composed from Dialog + Command directly rather than web-common's
   * CommandDialog: that wrapper spreads $$restProps into both Dialog.Root and
   * Command (so shouldFilter would land on the dialog too) and can't pass
   * noClose. Filtering is ours (shouldFilter={false}) because cmdk-sv 0.0.19
   * has no `keywords` prop and its fuzzy scorer would match on item ids.
   */

  export let role: PaletteRole | null;
  export let onboarded: boolean;
  export let shopifyEmbedded: boolean;

  let query = "";

  $: destinations = matchDestinations(
    query,
    destinationsFor({ role, onboarded, shopifyEmbedded }),
  );
  // Same set the help sidebar shows, so the two never disagree.
  $: helpPages = matchHelpPages(query, pagesFor(role, $bratraxIsDemo));
  $: noResults = destinations.length === 0 && helpPages.length === 0;
  $: if (!$commandPaletteOpen) query = "";

  function onKeydown(e: KeyboardEvent) {
    // Something upstream (an editor, a BlockingOverlay) already claimed it.
    if (e.defaultPrevented) return;
    const mod = isMac ? e.metaKey : e.ctrlKey;
    if (!mod || e.shiftKey || e.altKey || e.key.toLowerCase() !== "k") return;
    // Also stops the browser's own Ctrl+K (focus the search bar).
    e.preventDefault();
    commandPaletteOpen.update((open) => !open);
  }

  function go(href: string) {
    commandPaletteOpen.set(false);
    void goto(href);
  }
</script>

<svelte:window on:keydown={onKeydown} />

<Dialog.Root bind:open={$commandPaletteOpen}>
  <Dialog.Content
    noClose
    class="command-palette top-[15%] max-w-lg translate-y-0 gap-0 overflow-hidden p-0"
  >
    <Dialog.Title class="sr-only">Search settings and help</Dialog.Title>
    <Command.Root shouldFilter={false} loop label="Search settings and help">
      <!-- px-3: bratrax-theme.css gives every <input> its own elevated box and
           acid focus ring, so without inner padding the text sits flush
           against that box's left edge. -->
      <Command.Input
        class="px-3"
        bind:value={query}
        placeholder="Search settings and help…"
        autofocus
      />
      <Command.List class="max-h-[360px] p-1">
        {#if noResults}
          <p class="command-palette-empty">
            No settings or help articles match “{query.trim()}”.
          </p>
        {/if}
        {#if destinations.length > 0}
          <Command.Group heading="Pages & settings">
            {#each destinations as d (d.id)}
              <Command.Item value={`dest:${d.id}`} onSelect={() => go(d.href)}>
                <span class="flex-1 truncate">{d.label}</span>
                <span class="command-palette-meta">{d.section}</span>
              </Command.Item>
            {/each}
          </Command.Group>
        {/if}
        {#if helpPages.length > 0}
          <Command.Group heading="Help">
            {#each helpPages as p (p.slug)}
              <Command.Item
                value={`help:${p.slug}`}
                onSelect={() => go(`/help/${p.slug}`)}
              >
                <span class="flex-1 truncate">{p.title}</span>
                <span class="command-palette-meta">Help</span>
              </Command.Item>
            {/each}
          </Command.Group>
        {/if}
      </Command.List>
    </Command.Root>
  </Dialog.Content>
</Dialog.Root>

<style lang="postcss">
  /* font-size set here, not via Tailwind: bratrax-theme.css floors every
     sub-14px text utility with !important. */
  .command-palette-meta {
    flex: none;
    font-size: 12px;
    color: var(--color-text-muted, var(--fg-muted));
  }

  .command-palette-empty {
    padding: 24px 12px;
    text-align: center;
    font-size: 13px;
    color: var(--color-text-muted, var(--fg-muted));
  }
</style>
