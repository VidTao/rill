<script lang="ts">
  // Renders a Bratrax walkthrough video as a disclosure: a compact titled row
  // the reader clicks to open the player. Used only by the help center (see
  // help/[...slug]/+page.svelte). The generic Markdown component strips iframes
  // via DOMPurify, so video embeds are handled here as a trusted, code-owned
  // component rather than as raw HTML in markdown.
  //
  // The iframe is created on open, not on mount. Loom's embed boots a heavy
  // player per instance and some articles carry a dozen videos (start-here has
  // 13), so mounting them all made the page crawl even with loading="lazy".
  // Until a visitor asks for a video, this costs one button and no network.
  //
  // Colors come from the Bratrax theme tokens (src/bratrax-theme.css), not from
  // Tailwind's slate palette: the help center runs in both themes, and hardcoded
  // light-theme greys painted a washed-out placeholder onto the dark surface.
  export let id: string;
  export let label = "";
  export let duration = "";

  let open = false;

  const panelId = `loom-panel-${id}`;

  // Deliberately no `autoplay` param: opening a disclosure loads the player and
  // stops, so the reader presses play themselves. Autoplaying meant expanding
  // two stops of a tour started two videos talking over each other, and it also
  // made the open feel slow — the player could not paint until it had fetched
  // enough video to begin, where a paused player shows Loom's poster frame as
  // soon as the embed lands.
  //
  // `allow="autoplay"` stays on the iframe. It grants nothing on its own (the
  // URL decides), and it keeps Loom's own controls working for anything that
  // resumes playback without a fresh click inside the frame.
  $: src =
    `https://www.loom.com/embed/${id}` +
    `?hide_owner=true&hide_share=true&hide_title=true&hideEmbedTopBar=true`;
</script>

<div class="loom">
  <button
    type="button"
    class="loom-toggle"
    class:is-open={open}
    aria-expanded={open}
    aria-controls={panelId}
    on:click={() => (open = !open)}
  >
    <span class="caret" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
        <path d="M8 5.14v13.72L19 12 8 5.14z" />
      </svg>
    </span>
    <span class="text">
      {open ? "Hide" : "Watch"}{label ? ` — ${label}` : " walkthrough"}
    </span>
    {#if duration}<span class="duration">({duration})</span>{/if}
  </button>

  {#if open}
    <div class="frame" id={panelId}>
      <iframe
        {src}
        title={label ? `${label} walkthrough` : "Bratrax walkthrough video"}
        frameborder="0"
        allow="autoplay; fullscreen"
        allowfullscreen
      ></iframe>
    </div>
  {/if}
</div>

<style lang="postcss">
  .loom {
    margin: 14px 0 20px;
  }
  .loom-toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 7px 13px;
    border: 1px solid var(--color-border-strong);
    border-radius: 6px;
    background: var(--color-surface);
    color: var(--color-text);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition:
      border-color 150ms ease,
      background-color 150ms ease;
  }
  .loom-toggle:hover {
    background: var(--color-elevated);
    border-color: var(--color-acid);
  }
  .loom-toggle:focus-visible {
    outline: 2px solid var(--color-acid);
    outline-offset: 2px;
  }
  .caret {
    display: flex;
    color: var(--color-acid);
    transition: transform 150ms ease;
  }
  .is-open .caret {
    transform: rotate(90deg);
  }
  .duration {
    color: var(--color-text-muted);
    font-weight: 400;
    font-variant-numeric: tabular-nums;
  }
  .frame {
    position: relative;
    padding-bottom: 56.25%; /* 16:9 */
    height: 0;
    margin-top: 10px;
    border-radius: 6px;
    overflow: hidden;
    background: var(--color-surface);
  }
  .frame iframe {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
  }
</style>
