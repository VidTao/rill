<script lang="ts">
  import { Check, Copy } from "lucide-svelte";
  import {
    GOOGLE_ACCOUNT_TRACKING_TEMPLATE,
    META_URL_PARAMETERS,
    googleTrackingWarnings,
    metaTrackingWarnings,
    trackingVerificationChecklist,
  } from "$lib/bratrax/tracking-templates";

  type CopyKey = "google-combined" | "meta-params";

  let copied: CopyKey | "" = "";

  async function copyValue(key: CopyKey, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      copied = key;
      setTimeout(() => {
        if (copied === key) copied = "";
      }, 1800);
    } catch (e) {
      console.warn("Failed to copy tracking template", e);
    }
  }

  const flowSteps = [
    "Ad click",
    "URL parameters",
    "Bratrax attribution",
    "Campaign reporting",
  ];

  const metaFields = [
    {
      key: "meta-params" as const,
      label: "URL Parameters",
      value: META_URL_PARAMETERS,
    },
  ];
</script>

<section class="tracking-guide" aria-labelledby="tracking-setup-title">
  <div class="tracking-guide-header">
    <div>
      <div class="tracking-eyebrow">Tracking setup</div>
      <h2 id="tracking-setup-title">Paid media URL templates</h2>
      <p>
        Add these URL settings in Google Ads and Meta Ads so Bratrax can resolve
        revenue to campaign, ad set or asset group, and ad level wherever the ad
        platform supports it.
      </p>
    </div>
  </div>

  <div class="tracking-flow" aria-label="Tracking data flow">
    {#each flowSteps as step, idx}
      <div class="flow-step">
        <span>{idx + 1}</span>
        {step}
      </div>
    {/each}
  </div>

  <div class="platform-grid">
    <article class="platform-panel google-panel">
      <div class="platform-header">
        <span class="platform-mark google-mark"></span>
        <div>
          <div class="platform-kicker">Google Ads</div>
          <h3>Account global tracking template</h3>
        </div>
      </div>

      <p class="platform-note">
        Paste this combined value into Google Ads account Global settings, under
        Tracking template. It includes the landing page token and the full final
        URL suffix parameters.
      </p>

      <div class="copy-block">
        <div class="copy-block-top">
          <span>Tracking template</span>
          <button
            type="button"
            class="copy-button"
            on:click={() =>
              copyValue("google-combined", GOOGLE_ACCOUNT_TRACKING_TEMPLATE)}
            title="Copy Google Ads tracking template"
            aria-label="Copy Google Ads tracking template"
          >
            {#if copied === "google-combined"}
              <Check size={14} />
              Copied
            {:else}
              <Copy size={14} />
              Copy
            {/if}
          </button>
        </div>
        <pre>{GOOGLE_ACCOUNT_TRACKING_TEMPLATE}</pre>
      </div>

      <ul class="warning-list">
        {#each googleTrackingWarnings as warning}
          <li>{warning}</li>
        {/each}
      </ul>
    </article>

    <article class="platform-panel meta-panel">
      <div class="platform-header">
        <span class="platform-mark meta-mark"></span>
        <div>
          <div class="platform-kicker">Meta Ads</div>
          <h3>Campaign-level URL Parameters</h3>
        </div>
      </div>

      <p class="platform-note">
        Paste this into URL Parameters on each campaign in Meta Ads Manager.
        Meta does not provide a reliable account-level URL parameter setting for
        this setup.
      </p>

      {#each metaFields as field}
        <div class="copy-block">
          <div class="copy-block-top">
            <span>{field.label}</span>
            <button
              type="button"
              class="copy-button"
              on:click={() => copyValue(field.key, field.value)}
              title={`Copy ${field.label}`}
              aria-label={`Copy Meta Ads ${field.label}`}
            >
              {#if copied === field.key}
                <Check size={14} />
                Copied
              {:else}
                <Copy size={14} />
                Copy
              {/if}
            </button>
          </div>
          <pre>{field.value}</pre>
        </div>
      {/each}

      <ul class="warning-list">
        {#each metaTrackingWarnings as warning}
          <li>{warning}</li>
        {/each}
      </ul>
    </article>
  </div>

  <div class="verification-panel">
    <div>
      <div class="tracking-eyebrow">Verify after setup</div>
      <h3>Checklist</h3>
    </div>
    <ol>
      {#each trackingVerificationChecklist as item}
        <li>{item}</li>
      {/each}
    </ol>
  </div>
</section>

<style lang="postcss">
  .tracking-guide {
    @apply mt-8;
  }

  .tracking-guide-header {
    @apply mb-4 flex items-start justify-between gap-4;
  }

  .tracking-eyebrow,
  .platform-kicker {
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-acid-text);
  }

  .tracking-guide h2 {
    font-family: "Outfit", sans-serif;
    font-size: 22px;
    font-weight: 900;
    color: var(--color-text);
    line-height: 1.2;
    margin-top: 4px;
  }

  .tracking-guide h3 {
    font-family: "Outfit", sans-serif;
    font-size: 15px;
    font-weight: 800;
    color: var(--color-text);
    line-height: 1.2;
  }

  .tracking-guide p {
    max-width: 720px;
    margin-top: 8px;
    color: var(--color-text-secondary);
    font-size: 14px;
    line-height: 1.6;
  }

  .tracking-flow {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 8px;
    margin-bottom: 16px;
  }

  .flow-step {
    display: flex;
    min-height: 56px;
    align-items: center;
    gap: 10px;
    border: 0.5px solid var(--color-border);
    background: var(--color-elevated);
    padding: 12px;
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    color: var(--color-text);
  }

  .flow-step span {
    display: grid;
    width: 24px;
    height: 24px;
    flex: 0 0 auto;
    place-items: center;
    background: var(--color-acid);
    color: #0a0a0a;
    font-size: 10px;
  }

  .platform-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  .platform-panel,
  .verification-panel {
    position: relative;
    border: 0.5px solid var(--color-border);
    background: var(--color-elevated);
    padding: 20px;
  }

  .platform-panel::before,
  .verification-panel::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: var(--color-acid);
  }

  .platform-header {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 16px;
  }

  .platform-mark {
    width: 24px;
    height: 24px;
    flex: 0 0 auto;
  }

  .google-mark {
    background: #4285f4;
  }

  .meta-mark {
    background: #1877f2;
  }

  .copy-block {
    margin-top: 12px;
    border: 0.5px solid var(--color-border);
    background: var(--color-bg);
  }

  .copy-block-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    border-bottom: 0.5px solid var(--color-border);
    padding: 10px 12px;
  }

  .copy-block-top span {
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }

  .copy-button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    border: 0.5px solid var(--color-border);
    padding: 6px 9px;
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-text);
    background: var(--color-elevated);
    transition: border-color 0.15s, background-color 0.15s;
  }

  .copy-button:hover {
    border-color: var(--color-acid);
    background: var(--color-acid-dim);
  }

  pre {
    max-height: 170px;
    overflow: auto;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    padding: 12px;
    font-family: "Space Mono", monospace;
    font-size: 11px;
    line-height: 1.65;
    color: var(--color-text);
  }

  .warning-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 14px;
    color: var(--color-text-secondary);
    font-size: 13px;
    line-height: 1.45;
  }

  .warning-list li {
    border-left: 3px solid var(--color-acid);
    padding-left: 10px;
  }

  .verification-panel {
    display: grid;
    grid-template-columns: 180px minmax(0, 1fr);
    gap: 20px;
    margin-top: 16px;
  }

  .verification-panel ol {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    counter-reset: verify;
  }

  .verification-panel li {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    color: var(--color-text-secondary);
    font-size: 13px;
    line-height: 1.45;
    counter-increment: verify;
  }

  .verification-panel li::before {
    content: counter(verify);
    display: grid;
    width: 20px;
    height: 20px;
    flex: 0 0 auto;
    place-items: center;
    background: var(--color-acid-dim);
    color: var(--color-acid-text);
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
  }

  @media (max-width: 860px) {
    .tracking-flow,
    .platform-grid,
    .verification-panel,
    .verification-panel ol {
      grid-template-columns: 1fr;
    }
  }
</style>
