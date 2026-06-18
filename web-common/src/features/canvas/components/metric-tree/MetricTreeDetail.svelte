<script lang="ts">
  import { statusTheme } from "./status";
  import type { MetricTreeNodeData } from "./types";

  export let node: MetricTreeNodeData | null;
  export let onClose: () => void = () => {};

  $: theme = node ? statusTheme(node.status) : statusTheme(null);

  function fmtValue(
    v: number | string | null | undefined,
    unit: string | null | undefined,
  ): string {
    if (v == null || v === "") return "—";
    const n = typeof v === "number" ? v : Number(v);
    if (Number.isFinite(n)) {
      if (unit === "percent") return `${n.toLocaleString()}%`;
      if (unit === "usd" || unit === "currency") {
        return n.toLocaleString(undefined, {
          style: "currency",
          currency: "USD",
          maximumFractionDigits: 0,
        });
      }
      return `${n.toLocaleString()}${unit && unit !== "count" ? ` ${unit}` : ""}`;
    }
    return String(v);
  }

  // Sections that only render when the node carries the field.
  $: sections = node
    ? (
        [
          ["Observation", node.observation],
          ["Suggested test", node.suggestedTest],
          ["Success metric", node.successMetric],
          ["Limitation", node.limitation],
        ] as const
      ).filter(([, v]) => !!v)
    : [];
</script>

{#if node}
  <!-- scrim over the graph -->
  <button class="scrim" aria-label="Close detail" on:click={onClose} />

  <aside
    class="panel"
    style:--accent={theme.light}
    style:--accent-dark={theme.dark}
  >
    <header>
      <div class="title" title={node.label}>{node.label}</div>
      <button class="close" aria-label="Close" on:click={onClose}>✕</button>
    </header>

    <div class="status-row">
      {#if node.status}
        <span class="dot" />
        <span class="status">{theme.label}</span>
      {/if}
      {#if node.confidence}
        <span class="conf">· {node.confidence} confidence</span>
      {/if}
    </div>

    <div class="value">
      {fmtValue(node.value, node.unit)}
      {#if node.delta != null}
        <span class="delta">
          {node.delta > 0 ? "▲" : node.delta < 0 ? "▼" : "•"}
          {Math.abs(node.delta).toLocaleString()}
        </span>
      {/if}
    </div>

    {#each sections as [label, value] (label)}
      <section>
        <div class="section-label">{label}</div>
        <div class="section-body">{value}</div>
      </section>
    {/each}
  </aside>
{/if}

<style lang="postcss">
  .scrim {
    @apply absolute inset-0 z-10 cursor-default border-0 bg-black/10;
  }
  .panel {
    @apply absolute inset-y-0 right-0 z-20 flex flex-col gap-3 overflow-y-auto p-4;
    width: min(340px, 85%);
    background: var(--color-surface-card, #ffffff);
    border-left: 1px solid var(--color-border, #e5e7eb);
    box-shadow: -8px 0 24px rgba(15, 23, 42, 0.12);
  }
  header {
    @apply flex items-start justify-between gap-2;
  }
  .title {
    font-size: 15px;
    font-weight: 700;
    line-height: 1.25;
    color: var(--color-text, #1a1a18);
  }
  .close {
    @apply shrink-0 rounded px-1 text-sm;
    color: var(--color-text-muted, #6b7280);
  }
  .close:hover {
    color: var(--color-text, #1a1a18);
  }
  .status-row {
    @apply flex items-center gap-1.5;
    font-size: 11px;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 9999px;
    background: var(--accent);
  }
  .status {
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--accent);
  }
  .conf {
    color: var(--color-text-muted, #6b7280);
  }
  .value {
    font-size: 22px;
    font-weight: 700;
    color: var(--color-text, #1a1a18);
  }
  .delta {
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-muted, #6b7280);
    margin-left: 6px;
  }
  .section-label {
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.6px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    margin-bottom: 2px;
  }
  .section-body {
    font-size: 13px;
    line-height: 1.4;
    color: var(--color-text, #1a1a18);
  }

  :global(.dark) .panel {
    background: #161616;
    border-left-color: #2a2a2a;
  }
  :global(.dark) .title,
  :global(.dark) .value,
  :global(.dark) .section-body {
    color: #ece7dd;
  }
  :global(.dark) .status,
  :global(.dark) .dot {
    color: var(--accent-dark);
  }
  :global(.dark) .dot {
    background: var(--accent-dark);
  }
  :global(.dark) .conf,
  :global(.dark) .delta,
  :global(.dark) .section-label {
    color: #a39d90;
  }
  :global(.dark) .close {
    color: #a39d90;
  }
</style>
