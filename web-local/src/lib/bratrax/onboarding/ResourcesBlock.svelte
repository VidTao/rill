<script lang="ts">
  import { goto } from "$app/navigation";
  import { HelpCircle, Sparkles, Mail } from "lucide-svelte";
  import { HELP_CENTER_URL, SUPPORT_EMAIL } from "$lib/bratrax/constants";
  // NOTE: the "Check out the getting-started guide" row was removed
  // 2026-06-10 because the guide isn't published yet. When the URL is live
  // again, re-import GETTING_STARTED_URL + BookOpen and restore the row at
  // the top of the resources list.
</script>

<div class="card">
  <div class="title">Resources</div>

  <div class="resource">
    <HelpCircle size={18} class="ricon" />
    <span>
      Browse the
      <a href={HELP_CENTER_URL} target="_blank" rel="noreferrer noopener">help center</a>
    </span>
  </div>

  <div class="resource">
    <Sparkles size={18} class="ricon" />
    <button class="link" on:click={() => goto("/canvas/campaign_deep_dive")}>
      Ask Claude in-app
    </button>
  </div>

  <div class="resource muted">
    <Mail size={18} class="ricon" />
    <span>
      Still stuck? Email
      <a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>
    </span>
  </div>
</div>

<style>
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-top: 4px solid var(--color-acid);
    padding: 18px 20px;
    margin-top: 28px;
  }
  .title {
    font-family: "Space Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-acid-text);
    opacity: 0.85;
    margin-bottom: 14px;
  }
  .resource {
    display: flex;
    align-items: center;
    gap: 10px;
    font-family: "Outfit", sans-serif;
    font-weight: 300;
    font-size: 16px;
    color: var(--color-text-secondary);
    padding: 6px 0;
  }
  .resource.muted :global(.ricon) {
    color: var(--color-text-muted);
  }
  :global(.ricon) {
    color: var(--color-acid-text);
    flex: 0 0 auto;
  }
  a,
  .link {
    /* Match the global `a { color: var(--accent-primary) !important }`
       rule in bratrax-theme.css so the button-styled-as-link reads the
       same as the real anchors in this card — near-black on cream in
       light theme, acid yellow on near-black in dark. Using
       --color-acid-text here paints olive green in light theme (the
       AA-text variant of acid), which is inconsistent with the other
       links sitting beside it. */
    color: var(--accent-primary);
    text-decoration: underline;
    font: inherit;
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0;
  }
</style>
