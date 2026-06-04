<script lang="ts">
  import { Handle, Position } from "@xyflow/svelte";
  import { sourceColor, type PathNodeData } from "./order-path-layout";

  export let data: PathNodeData;

  // XYFlow injects these layout props; declared so Svelte doesn't warn about
  // unknown props on the custom node component.
  export let id: string | undefined = undefined;
  export let type: string | undefined = undefined;
  export let selected = false;
  export let width: number | undefined = undefined;
  export let height: number | undefined = undefined;
  export let sourcePosition: Position | undefined = undefined;
  export let targetPosition: Position | undefined = undefined;
  export let dragHandle: string | undefined = undefined;
  export let parentId: string | undefined = undefined;
  export let dragging = false;
  export let zIndex = 0;
  export let selectable = true;
  export let deletable = true;
  export let draggable = false;
  export let isConnectable = true;
  export let positionAbsoluteX = 0;
  export let positionAbsoluteY = 0;

  const ensureFlowProps = (..._args: unknown[]) => {};
  $: ensureFlowProps(
    id,
    type,
    selected,
    width,
    height,
    sourcePosition,
    targetPosition,
    dragHandle,
    parentId,
    dragging,
    zIndex,
    selectable,
    deletable,
    draggable,
    isConnectable,
    positionAbsoluteX,
    positionAbsoluteY,
  );

  $: accent = sourceColor(data.source);
  $: isConversion = data.kind === "conversion";
  // Behavior events + collapsed runs render as small chips in compact mode.
  $: isChip = data.compact === true;
  $: isCollapsed = data.kind === "collapsed";

  function fmtTime(iso: string | undefined): string {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }
</script>

{#if isChip}
  <div
    class="chip"
    class:collapsed={isCollapsed}
    style:--accent={accent}
    title={data.subtitle || data.title}
  >
    <Handle type="target" position={Position.Left} isConnectable={false} />
    <Handle type="source" position={Position.Right} isConnectable={false} />
    <span class="chip-dot" />
    <span class="chip-title">{data.title}</span>
    {#if isCollapsed}<span class="chip-expand">expand</span>{/if}
  </div>
{:else}
  <div
    class="pnode"
    class:winner={data.isWinner}
    class:conversion={isConversion}
    style:--accent={accent}
  >
    <Handle type="target" position={Position.Left} isConnectable={false} />
    <Handle type="source" position={Position.Right} isConnectable={false} />

    <div class="row-top">
      <span class="src-tag">{data.sourceLabel}</span>
      {#if data.isWinner}<span class="winner-badge">winner</span>{/if}
    </div>
    <div class="title" title={data.title}>{data.title}</div>
    {#if data.subtitle}
      <div class="sub" title={data.subtitle}>{data.subtitle}</div>
    {/if}
    {#if data.ts}
      <div class="ts">{fmtTime(data.ts)}</div>
    {/if}
  </div>
{/if}

<style>
  /* Compact chip — behavior events + collapsed runs. */
  .chip {
    box-sizing: border-box;
    width: 158px;
    height: 46px;
    padding: 0 10px;
    display: flex;
    align-items: center;
    gap: 7px;
    border-radius: 23px;
    border: 1px solid color-mix(in srgb, var(--accent) 40%, transparent);
    background: var(--color-surface, #ffffff);
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.06);
    cursor: pointer;
    overflow: hidden;
    transition:
      box-shadow 120ms ease,
      transform 120ms ease;
  }
  .chip:hover {
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.16);
    transform: translateY(-1px);
  }
  .chip.collapsed {
    border-style: dashed;
    border-width: 1.5px;
  }
  .chip-dot {
    flex-shrink: 0;
    width: 8px;
    height: 8px;
    border-radius: 9999px;
    background: var(--accent);
  }
  .chip-title {
    flex: 1;
    min-width: 0;
    font-size: 12px;
    font-weight: 500;
    line-height: 1.2;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--color-text, #1a1a18);
  }
  .chip-expand {
    flex-shrink: 0;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 8px;
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--accent);
    opacity: 0.8;
  }

  .pnode {
    box-sizing: border-box;
    width: 248px;
    height: 104px;
    padding: 10px 12px;
    border-radius: 8px;
    border: 1px solid color-mix(in srgb, var(--accent) 45%, transparent);
    border-left: 4px solid var(--accent);
    background: var(--color-surface, #ffffff);
    box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
    display: flex;
    flex-direction: column;
    gap: 3px;
    overflow: hidden;
    cursor: pointer;
    transition:
      box-shadow 120ms ease,
      transform 120ms ease;
  }
  .pnode:hover {
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.18);
    transform: translateY(-1px);
  }
  .pnode.conversion {
    border-left-color: #16a34a;
    border-color: #16a34a;
    background: color-mix(in srgb, #16a34a 6%, #ffffff);
  }
  /* Dashboard themes can pin --color-* to their light values inside the
     canvas, so the raw-var fallbacks aren't reliable in dark mode. Set
     readable colors explicitly when the app is in dark mode. */
  :global(.dark) .pnode,
  :global(.dark) .chip {
    background: #161616;
    border-color: color-mix(in srgb, var(--accent) 55%, #2a2a2a);
  }
  :global(.dark) .chip-title {
    color: #ece7dd;
  }
  :global(.dark) .pnode.conversion {
    background: color-mix(in srgb, #16a34a 16%, #141414);
    border-color: #3fbf64;
    border-left-color: #3fbf64;
  }
  .pnode.winner {
    border-color: var(--color-acid, #d4ff00);
    box-shadow: 0 0 0 2px rgba(181, 159, 0, 0.45);
  }
  .row-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
  }
  .src-tag {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--accent);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .winner-badge {
    flex-shrink: 0;
    padding: 1px 5px;
    background: var(--color-acid, #d4ff00);
    color: #1a1a18;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    border-radius: 2px;
  }
  .title {
    font-size: 14px;
    font-weight: 600;
    line-height: 1.25;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    color: var(--color-text, #1a1a18);
  }
  .sub {
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ts {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    color: var(--color-text-muted, #9ca3af);
  }
  :global(.dark) .title {
    color: #ece7dd;
  }
  :global(.dark) .sub {
    color: #c2bcb0;
  }
  :global(.dark) .ts {
    color: #a39d90;
  }
  :global(.svelte-flow__handle) {
    width: 6px;
    height: 6px;
    min-width: 6px;
    min-height: 6px;
    border-radius: 9999px;
    background: color-mix(in srgb, var(--accent) 30%, #ffffff);
    border: 1px solid var(--accent);
    opacity: 1;
  }
</style>
