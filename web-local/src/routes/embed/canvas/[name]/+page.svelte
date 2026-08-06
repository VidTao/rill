<script lang="ts">
  import CanvasDashboardEmbed from "@rilldata/web-common/features/canvas/CanvasDashboardEmbed.svelte";
  import CanvasProvider from "@rilldata/web-common/features/canvas/CanvasProvider.svelte";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { openTopLevel } from "$lib/bratrax/shopify-embed";
  import { createEmbedHandoff } from "$lib/bratrax/onboarding/api";
  import type { PageData } from "./$types";

  // Single-dashboard view for the Shopify admin iframe. Same two components
  // the full /canvas/[name] page uses — CanvasDashboardEmbed already renders
  // just the dashboard and its filter bar; everything else on that page comes
  // from the root layout, which suppresses itself on /embed/* (onEmbedPage).
  //
  // Deliberately dropped versus /canvas/[name]: WelcomeBanner, the
  // "Edit dashboard" link, and the AI chat gate. None of them make sense in a
  // ~1150px-wide frame inside someone else's admin.
  //
  // Parametrised on [name] rather than hardcoding campaign_deep_dive so other
  // canvases can be embedded later without another route.

  export let data: PageData;

  $: ({ instanceId } = $runtime);
  $: ({ canvasName } = data);

  let opening = false;
  let openError = "";

  // "Open in Bratrax" — the escape hatch to the full app.
  //
  // Cannot be a plain <a href="/canvas/...">: inside the iframe the merchant is
  // authenticated by a Shopify App Bridge session token, and there is no
  // bratrax_auth cookie on this origin (that cookie is third-party here and
  // blocked). A new tab would land on /login. So we mint a single-use hand-off
  // token first and open the URL that exchanges it for a real cookie.
  async function openInBratrax() {
    opening = true;
    openError = "";
    try {
      const { url } = await createEmbedHandoff();
      if (!openTopLevel(url)) {
        // Popup blocked. The token is single-use and expires in 60s, so we
        // can't render it as a persistent link — ask for the click again.
        openError = "Allow pop-ups for this page, then try again.";
      }
    } catch {
      openError = "Couldn't open Bratrax. Please try again.";
    } finally {
      opening = false;
    }
  }
</script>

{#key instanceId}
  <div class="flex h-full flex-col overflow-hidden">
    <div class="canvas-area">
      <CanvasProvider {canvasName} {instanceId}>
        <CanvasDashboardEmbed {canvasName}>
          <!-- Rendered into CanvasDashboardWrapper's filter-right slot, the
               same absolutely-positioned anchor the "Edit dashboard" link uses
               on /canvas/[name]. It sits inside the filter bar's max-width
               wrapper, so it tracks the right edge at any viewport width. -->
          <svelte:fragment slot="filter-right">
            <div
              class="embed-open-wrap"
              style="position: absolute; top: 0; right: var(--filter-slot-right, 0); z-index: 60;"
            >
              {#if openError}
                <span class="embed-open-error">{openError}</span>
              {/if}
              <button
                type="button"
                class="embed-open-link"
                disabled={opening}
                on:click={openInBratrax}
              >
                {opening ? "Opening…" : "Open in Bratrax ↗"}
              </button>
            </div>
          </svelte:fragment>
        </CanvasDashboardEmbed>
      </CanvasProvider>
    </div>
  </div>
{/key}

<style lang="postcss">
  .canvas-area {
    @apply flex-1 overflow-hidden;
    position: relative;
  }

  .embed-open-wrap {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .embed-open-error {
    font-size: 12px;
    color: var(--color-text-secondary, var(--fg-secondary));
    white-space: nowrap;
  }

  /* Matches .canvas-edit-link on /canvas/[name] so the control reads as part
     of the filter row rather than a foreign element. */
  .embed-open-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 26px;
    padding: 0 10px;
    background: var(--color-surface, transparent);
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition:
      background-color 120ms ease,
      color 120ms ease,
      border-color 120ms ease;
  }
  .embed-open-link:hover:not(:disabled) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
    border-color: var(--color-acid, #d4ff00);
  }
  .embed-open-link:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .embed-open-link:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
</style>
