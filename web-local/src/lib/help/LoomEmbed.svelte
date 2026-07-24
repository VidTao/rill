<script lang="ts">
  // Renders a Bratrax walkthrough video as an inline Loom player.
  // Used only by the help center (see help/[...slug]/+page.svelte). The generic
  // Markdown component strips iframes via DOMPurify, so video embeds are handled
  // here as a trusted, code-owned component rather than as raw HTML in markdown.
  export let id: string;

  $: src =
    `https://www.loom.com/embed/${id}` +
    `?hide_owner=true&hide_share=true&hide_title=true&hideEmbedTopBar=true`;
</script>

<div class="loom-embed">
  <iframe
    {src}
    title="Bratrax walkthrough video"
    frameborder="0"
    allowfullscreen
    loading="lazy"
  ></iframe>
</div>

<style lang="postcss">
  .loom-embed {
    position: relative;
    padding-bottom: 56.25%; /* 16:9 */
    height: 0;
    margin: 16px 0 20px;
    border-radius: 6px;
    overflow: hidden;
    background: theme("colors.slate.100");
  }
  .loom-embed iframe {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
  }
</style>
