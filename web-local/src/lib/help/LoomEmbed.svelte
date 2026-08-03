<script lang="ts">
  // Renders a Bratrax walkthrough video as an inline Loom player.
  // Used only by the help center (see help/[...slug]/+page.svelte). The generic
  // Markdown component strips iframes via DOMPurify, so video embeds are handled
  // here as a trusted, code-owned component rather than as raw HTML in markdown.
  //
  // The iframe is created on click, not on mount. Loom's embed boots a heavy
  // player per instance and some articles carry a dozen videos (start-here has
  // 13), so mounting them all made the page crawl even with loading="lazy".
  // Until a visitor asks for a video, this costs one button and no network.
  export let id: string;

  let playing = false;

  // autoplay, so the click that reveals the player also starts it
  $: src =
    `https://www.loom.com/embed/${id}` +
    `?hide_owner=true&hide_share=true&hide_title=true&hideEmbedTopBar=true&autoplay=1`;
</script>

<div class="loom-embed">
  {#if playing}
    <iframe
      {src}
      title="Bratrax walkthrough video"
      frameborder="0"
      allow="autoplay; fullscreen"
      allowfullscreen
    ></iframe>
  {:else}
    <button
      type="button"
      on:click={() => (playing = true)}
      aria-label="Play walkthrough video"
    >
      <span class="play-icon" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="currentColor" width="28" height="28">
          <path d="M8 5.14v13.72L19 12 8 5.14z" />
        </svg>
      </span>
      <span class="play-label">Play walkthrough</span>
    </button>
  {/if}
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
  .loom-embed button {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    border: 0;
    cursor: pointer;
    background: theme("colors.slate.100");
    color: theme("colors.slate.600");
    transition: background-color 150ms ease;
  }
  .loom-embed button:hover {
    background: theme("colors.slate.200");
  }
  .loom-embed button:focus-visible {
    outline: 2px solid theme("colors.blue.500");
    outline-offset: -2px;
  }
  .play-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 56px;
    border-radius: 9999px;
    background: theme("colors.slate.900");
    color: white;
    padding-left: 4px; /* optically centre the triangle */
  }
  .loom-embed button:hover .play-icon {
    background: theme("colors.slate.700");
  }
  .play-label {
    font-size: 13px;
    font-weight: 500;
  }
</style>
