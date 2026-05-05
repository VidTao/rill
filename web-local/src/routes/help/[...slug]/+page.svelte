<script lang="ts">
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { canAccess, getHelpPage } from "$lib/help";
  import Markdown from "@rilldata/web-common/components/markdown/Markdown.svelte";

  $: slug = $page.params.slug ?? "";
  $: helpPage = getHelpPage(slug);
  $: role = $bratraxUser?.role ?? null;

  // If a viewer hits an admin-only page, send them to /help.
  $: if (helpPage && !canAccess(helpPage, role)) {
    goto("/help", { replaceState: true });
  }
</script>

{#if !helpPage}
  <article class="help-page not-found">
    <h1>Not found</h1>
    <p>
      No help page matches <code>{slug}</code>. Pick a page from the sidebar.
    </p>
  </article>
{:else if canAccess(helpPage, role)}
  <article class="help-page">
    <div class="meta">
      <span class="audience-tag audience-{helpPage.audience}">
        {helpPage.audience === "shared" ? "Reference" : `For ${helpPage.audience}s`}
      </span>
      {#if helpPage.status === "stub"}
        <span class="status-tag">stub — content coming</span>
      {:else if helpPage.status === "draft"}
        <span class="status-tag">draft</span>
      {/if}
    </div>

    <Markdown content={helpPage.body} />
  </article>
{/if}

<style lang="postcss">
  .help-page {
    max-width: 760px;
    margin: 0 auto;
    padding: 32px 32px 96px;
  }
  .meta {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
  }
  .audience-tag,
  .status-tag {
    font-family: "Space Mono", monospace;
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 1.5px;
    padding: 3px 8px;
    border-radius: 3px;
  }
  .audience-tag {
    background: var(--color-acid-mid);
    color: var(--color-acid-text);
  }
  .audience-admin {
    background: var(--color-acid-dim);
  }
  .status-tag {
    background: transparent;
    border: 1px dashed var(--color-text-muted);
    color: var(--color-text-muted);
  }
  .help-page :global(.chat-markdown h1) {
    font-size: 1.75rem;
    font-weight: 700;
    margin-bottom: 12px;
    letter-spacing: -0.02em;
  }
  .help-page :global(.chat-markdown h2) {
    font-size: 1.25rem;
    margin-top: 1.75rem;
  }
  .help-page :global(.chat-markdown h3) {
    font-size: 1rem;
    margin-top: 1.25rem;
  }
  .help-page :global(.chat-markdown p),
  .help-page :global(.chat-markdown li) {
    font-size: 0.9375rem;
    line-height: 1.6;
  }
  .help-page :global(.chat-markdown blockquote) {
    border-left-color: var(--color-acid);
    color: var(--color-text);
    background: var(--color-acid-dim);
    padding: 8px 12px;
    margin: 12px 0;
  }
  .not-found h1 {
    font-size: 1.5rem;
    margin-bottom: 8px;
  }
  .not-found code {
    background: var(--color-acid-dim);
    padding: 2px 6px;
    border-radius: 3px;
    font-family: "Space Mono", monospace;
  }
</style>
