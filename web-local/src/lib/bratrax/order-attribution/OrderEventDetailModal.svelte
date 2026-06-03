<script lang="ts">
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import { formatOrderLabel, humanizeResolutionReason } from "./api";
  import { humanizeSource } from "./order-path-layout";
  import type { OrderTimelineRow } from "./types";

  export let open = false;
  export let row: OrderTimelineRow | null = null;

  type Field = { label: string; value: string };
  type Section = { title: string; fields: Field[] };

  function fmtDateTime(iso: string | undefined): string {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  function fmtMoney(n: number | undefined): string {
    if (n == null) return "";
    return n.toLocaleString(undefined, {
      style: "currency",
      currency: "USD",
      maximumFractionDigits: 2,
    });
  }

  function fmtWeight(n: number | undefined): string {
    if (n == null) return "";
    return n.toLocaleString(undefined, { maximumFractionDigits: 4 });
  }

  // Humanize the row_type for the title.
  function rowTypeLabel(r: OrderTimelineRow): string {
    switch (r.row_type) {
      case "attribution_touchpoint":
        return "Touchpoint";
      case "behavior_event":
        return r.event_type || "Behavior event";
      case "conversion":
        return "Conversion";
      case "resolver_evidence":
        return "Resolver evidence";
      default:
        return r.event_type || r.row_type || "Event";
    }
  }

  // Drop empty / nullish fields so sections stay tight.
  function keep(fields: Field[]): Field[] {
    return fields.filter((f) => f.value !== "" && f.value != null);
  }

  // raw_params can arrive as a JSON string or an already-parsed object;
  // pretty-print whichever it is, fall back to the raw string.
  function prettyRaw(raw: unknown): string {
    if (raw == null || raw === "") return "";
    if (typeof raw === "object") {
      try {
        return JSON.stringify(raw, null, 2);
      } catch {
        return String(raw);
      }
    }
    if (typeof raw === "string") {
      try {
        return JSON.stringify(JSON.parse(raw), null, 2);
      } catch {
        return raw;
      }
    }
    return String(raw);
  }

  $: rawParams = row
    ? prettyRaw((row as Record<string, unknown>).raw_params)
    : "";

  $: sections = (
    row
      ? [
          {
            title: "Event",
            fields: keep([
              { label: "Type", value: rowTypeLabel(row) },
              { label: "Event name", value: row.event_type ?? "" },
              {
                label: "Source",
                value: row.event_source ? humanizeSource(row.event_source) : "",
              },
              { label: "Time", value: fmtDateTime(row.event_ts) },
              {
                label: "Rank",
                value: row.event_rank != null ? String(row.event_rank) : "",
              },
              { label: "Activity ID", value: row.activity_id ?? "" },
              { label: "Anonymous ID", value: row.anonymous_id ?? "" },
            ]),
          },
          {
            title: "Channel & campaign",
            fields: keep([
              { label: "Channel group", value: row.channel_group ?? "" },
              { label: "Source", value: row.source ?? "" },
              { label: "Medium", value: row.medium ?? "" },
              { label: "Campaign", value: row.campaign ?? "" },
              { label: "Campaign ID", value: row.campaign_id ?? "" },
              { label: "Adset", value: row.adset ?? "" },
              { label: "Ad", value: row.ad ?? "" },
            ]),
          },
          {
            title: "Page",
            fields: keep([
              { label: "URL", value: row.url ?? "" },
              { label: "Referrer", value: row.referrer ?? "" },
            ]),
          },
          {
            title: "Order",
            fields: keep([
              {
                label: "Order",
                value:
                  row.order_number || row.order_id
                    ? formatOrderLabel(row.order_number, row.order_id)
                    : "",
              },
              { label: "Revenue", value: fmtMoney(row.revenue) },
              { label: "Email", value: row.email ?? "" },
              { label: "Customer ID", value: row.customer_id ?? "" },
              { label: "Conversion time", value: fmtDateTime(row.conversion_ts) },
            ]),
          },
          {
            title: "Resolution",
            fields: keep([
              { label: "Status", value: row.resolution_status ?? "" },
              {
                label: "Reason",
                value: row.resolution_reason
                  ? `${humanizeResolutionReason(row.resolution_reason)}  ·  ${row.resolution_reason}`
                  : "",
              },
              { label: "Confidence", value: row.resolution_confidence ?? "" },
            ]),
          },
          {
            title: "Attribution weights",
            fields: keep([
              { label: "First touch", value: fmtWeight(row.first_touch_weight) },
              { label: "Last touch", value: fmtWeight(row.last_touch_weight) },
              { label: "Linear", value: fmtWeight(row.linear_weight) },
              { label: "Time decay", value: fmtWeight(row.time_decay_weight) },
              { label: "Position", value: fmtWeight(row.position_weight) },
              {
                label: "Winning models",
                value: [
                  row.is_first_touch_winner ? "first" : "",
                  row.is_last_touch_winner ? "last" : "",
                  row.is_linear_winner ? "linear" : "",
                  row.is_time_decay_winner ? "time-decay" : "",
                  row.is_position_winner ? "position" : "",
                ]
                  .filter(Boolean)
                  .join(", "),
              },
            ]),
          },
        ]
      : []
  ).filter((s: Section) => s.fields.length > 0);

  $: title = row ? rowTypeLabel(row) : "Event";
  $: subtitle = row?.event_source ? humanizeSource(row.event_source) : "";
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-xl">
    <Dialog.Header>
      <Dialog.Title>{title}</Dialog.Title>
      {#if subtitle || row?.event_ts}
        <Dialog.Description>
          {#if subtitle}{subtitle}{/if}
          {#if subtitle && row?.event_ts}<span class="dot-sep">·</span>{/if}
          {#if row?.event_ts}{fmtDateTime(row.event_ts)}{/if}
        </Dialog.Description>
      {/if}
    </Dialog.Header>

    {#if !row}
      <div class="empty">No event selected.</div>
    {:else}
      <div class="detail-body">
        {#each sections as section (section.title)}
          <section class="group">
            <h4 class="group-title">{section.title}</h4>
            <dl class="grid">
              {#each section.fields as f (f.label)}
                <dt>{f.label}</dt>
                <dd>{f.value}</dd>
              {/each}
            </dl>
          </section>
        {/each}

        {#if rawParams}
          <section class="group">
            <h4 class="group-title">Raw params</h4>
            <pre class="raw">{rawParams}</pre>
          </section>
        {/if}
      </div>
    {/if}
  </Dialog.Content>
</Dialog.Root>

<style>
  .empty {
    padding: 32px 8px;
    text-align: center;
    color: var(--color-text-muted, #6b7280);
    font-size: 14px;
  }
  .detail-body {
    max-height: 60vh;
    overflow-y: auto;
    padding-right: 4px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .group-title {
    margin: 0 0 6px 0;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
  }
  .grid {
    margin: 0;
    display: grid;
    grid-template-columns: 130px 1fr;
    gap: 4px 12px;
    align-items: baseline;
  }
  dt {
    font-size: 12px;
    color: var(--color-text-muted, #6b7280);
  }
  dd {
    margin: 0;
    font-size: 13px;
    color: var(--color-text, #1a1a18);
    word-break: break-word;
  }
  .raw {
    margin: 0;
    padding: 8px 10px;
    background: var(--color-surface-subtle, #f5f5f4);
    border: 1px solid var(--color-bratrax-border, #e5e7eb);
    border-radius: 4px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    line-height: 1.4;
    color: var(--color-text, #1a1a18);
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .dot-sep {
    margin: 0 4px;
    color: var(--color-text-muted, #9ca3af);
  }
  /* Dashboard theme boundary pins --color-* to light values; force readable
     colors in dark mode (same reason as the node cards). */
  :global(.dark) dd,
  :global(.dark) .raw {
    color: #ece7dd;
  }
  :global(.dark) dt,
  :global(.dark) .group-title {
    color: #a39d90;
  }
  :global(.dark) .raw {
    background: #161616;
    border-color: #2e2e2e;
  }
</style>
