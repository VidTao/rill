<script lang="ts">
  import type { OrderTimelineRow } from "./types";

  export let row: OrderTimelineRow;
  export let evidence: OrderTimelineRow[] = [];

  let evidenceOpen = false;

  $: rowType = row.row_type;
  $: isWinner = row.is_attribution_winner === 1;

  function fmtTime(iso: string | undefined): string {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  // Touchpoint subtitle: channel / campaign / adset / ad, skipping empties.
  $: touchpointLabel = [
    row.channel_group,
    row.campaign,
    row.adset,
    row.ad,
  ]
    .filter((v): v is string => !!v && v.length > 0)
    .join(" / ");
</script>

<li
  class="timeline-row"
  class:behavior={rowType === "behavior_event"}
  class:touchpoint={rowType === "attribution_touchpoint"}
  class:conversion={rowType === "conversion"}
  class:winner={isWinner}
>
  <span class="dot" aria-hidden="true" />
  <div class="body">
    <div class="head">
      <span class="ts">{fmtTime(row.event_ts)}</span>
      <span class="kind">
        {#if rowType === "attribution_touchpoint"}
          touchpoint
          {#if isWinner}<span class="winner-tag">winner</span>{/if}
        {:else if rowType === "conversion"}
          conversion
        {:else}
          {row.event_type ?? rowType ?? "event"}
        {/if}
      </span>
      {#if row.event_source}
        <span class="src">{row.event_source}</span>
      {/if}
    </div>
    {#if rowType === "attribution_touchpoint" && touchpointLabel}
      <div class="meta">{touchpointLabel}</div>
    {/if}
    {#if rowType === "conversion" && row.revenue != null}
      <div class="meta">
        Order #{row.order_number ?? row.order_id} · {row.revenue.toLocaleString(
          undefined,
          { style: "currency", currency: "USD" },
        )}
      </div>
    {/if}
    {#if rowType === "behavior_event" && row.url}
      <div class="meta url">{row.url}</div>
    {/if}
    {#if evidence.length > 0}
      <button
        type="button"
        class="evidence-toggle"
        on:click={() => (evidenceOpen = !evidenceOpen)}
        aria-expanded={evidenceOpen}
      >
        {evidenceOpen ? "Hide" : "Show"} evidence ({evidence.length})
      </button>
      {#if evidenceOpen}
        <ul class="evidence-list">
          {#each evidence as e (e.activity_id + ":" + (e.event_ts ?? ""))}
            <li class="evidence-item">
              <span class="ev-kind">{e.event_type ?? "evidence"}</span>
              <span class="ev-source">{e.event_source ?? ""}</span>
              {#if e.resolution_reason}
                <span class="ev-reason">{e.resolution_reason}</span>
              {/if}
              {#if e.url}
                <div class="ev-url">{e.url}</div>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
  </div>
</li>

<style>
  .timeline-row {
    position: relative;
    display: flex;
    gap: 10px;
    padding: 8px 0;
    list-style: none;
  }
  .dot {
    flex-shrink: 0;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    margin-top: 6px;
    background: var(--color-text-muted, #9ca3af);
  }
  .behavior .dot {
    background: #9ca3af;
  }
  .touchpoint .dot {
    background: #3b82f6;
  }
  .winner .dot {
    background: var(--color-acid, #d4ff00);
    box-shadow: 0 0 0 2px rgba(212, 255, 0, 0.3);
  }
  .conversion .dot {
    background: #16a34a;
  }
  .body {
    flex: 1;
    min-width: 0;
  }
  .head {
    display: flex;
    align-items: baseline;
    gap: 10px;
    font-size: 13px;
  }
  .ts {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
    flex-shrink: 0;
  }
  .kind {
    font-weight: 600;
  }
  .src {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
  }
  .winner-tag {
    margin-left: 6px;
    padding: 1px 6px;
    background: var(--color-acid, #d4ff00);
    color: #1a1a18;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    border-radius: 2px;
  }
  .meta {
    margin-top: 2px;
    font-size: 12px;
    color: var(--color-text-muted, #6b7280);
  }
  .meta.url {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .evidence-toggle {
    margin-top: 4px;
    padding: 0;
    background: none;
    border: none;
    color: var(--color-text-muted, #6b7280);
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    text-decoration: underline;
  }
  .evidence-toggle:hover {
    color: #1a1a18;
  }
  .evidence-list {
    margin: 6px 0 0 0;
    padding: 0 0 0 12px;
    border-left: 1px dashed var(--color-bratrax-border, #e5e7eb);
  }
  .evidence-item {
    list-style: none;
    padding: 4px 0;
    font-size: 12px;
  }
  .ev-kind {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
  }
  .ev-source,
  .ev-reason {
    margin-left: 8px;
    color: var(--color-text-muted, #6b7280);
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
  }
  .ev-url {
    margin-top: 2px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
