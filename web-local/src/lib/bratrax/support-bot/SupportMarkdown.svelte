<script lang="ts">
  import DOMPurify from "dompurify";
  import { marked } from "marked";

  export let content: string;

  // The bot answers in Markdown (the KB articles it quotes are Markdown), so
  // rendering raw text left `**bold**` and `[label](url)` visible in the panel.
  // Rill's shared components/markdown/Markdown.svelte hardcodes light-theme
  // greys and blue links, so this is a Bratrax-themed sibling rather than a
  // reuse.
  $: html = marked.parse(content, { async: false }) as string;
  $: safeHtml = DOMPurify.sanitize(html);

  // marked emits bare anchors; send external links to a new tab and leave
  // relative ones alone so SvelteKit keeps handling them as client-side nav.
  function linkTargets(node: HTMLElement) {
    const apply = () => {
      for (const a of node.querySelectorAll("a")) {
        if (/^https?:/i.test(a.getAttribute("href") ?? "")) {
          a.setAttribute("target", "_blank");
          a.setAttribute("rel", "noreferrer noopener");
        }
      }
    };
    apply();
    return { update: apply };
  }
</script>

<div class="support-md" use:linkTargets>
  {@html safeHtml}
</div>

<style lang="postcss">
  .support-md {
    @apply text-sm leading-relaxed break-words;
  }

  /* Tight vertical rhythm: the panel is narrow, so default prose spacing
     wastes most of the visible answer area. */
  .support-md :global(p),
  .support-md :global(ul),
  .support-md :global(ol),
  .support-md :global(pre),
  .support-md :global(blockquote),
  .support-md :global(table) {
    @apply mb-2;
  }
  .support-md :global(> *:last-child) {
    @apply mb-0;
  }

  .support-md :global(h1),
  .support-md :global(h2),
  .support-md :global(h3),
  .support-md :global(h4) {
    @apply mb-1 mt-3 font-semibold text-bratrax-text-headline;
  }
  .support-md :global(h1) {
    @apply text-base;
  }
  .support-md :global(h2) {
    @apply text-sm;
  }
  .support-md :global(h3),
  .support-md :global(h4) {
    @apply font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted;
  }
  .support-md :global(> *:first-child) {
    @apply mt-0;
  }

  .support-md :global(ul) {
    @apply list-disc pl-5;
  }
  .support-md :global(ol) {
    @apply list-decimal pl-5;
  }
  .support-md :global(li) {
    @apply mb-1;
  }
  .support-md :global(li > ul),
  .support-md :global(li > ol) {
    @apply mb-0 mt-1;
  }

  .support-md :global(a) {
    @apply text-bratrax-acid underline underline-offset-2;
  }
  .support-md :global(a:hover) {
    @apply opacity-80;
  }

  .support-md :global(strong) {
    @apply font-semibold text-bratrax-text-headline;
  }
  .support-md :global(em) {
    @apply italic;
  }

  .support-md :global(code) {
    @apply border border-bratrax-border bg-bratrax-bg px-1 py-0.5 font-mono text-[11px];
  }
  .support-md :global(pre) {
    @apply overflow-x-auto border border-bratrax-border bg-bratrax-bg p-2;
  }
  .support-md :global(pre code) {
    @apply border-0 bg-transparent p-0;
  }

  .support-md :global(blockquote) {
    @apply border-l-2 border-bratrax-acid-muted pl-3 text-bratrax-text-muted;
  }

  .support-md :global(hr) {
    @apply my-3 border-t border-bratrax-border;
  }

  .support-md :global(table) {
    @apply block w-full overflow-x-auto text-xs;
    border-collapse: collapse;
  }
  .support-md :global(th),
  .support-md :global(td) {
    @apply border border-bratrax-border px-2 py-1 text-left;
  }
  .support-md :global(th) {
    @apply font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted;
  }
</style>
