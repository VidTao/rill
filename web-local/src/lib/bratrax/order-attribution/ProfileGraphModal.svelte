<script lang="ts">
  import type { MeasureCellClickContext } from "@rilldata/web-common/features/canvas/components/pivot/drilldown-context";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import {
    createQueryServiceMetricsViewRows,
    type V1MetricsViewRowsResponseDataItem,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { createInExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
  import { extractCellLabels, formatOrderLabel } from "./api";

  export let open = false;
  export let cellContext: MeasureCellClickContext;

  type ProfileRow = V1MetricsViewRowsResponseDataItem & {
    profile_id?: string;
    display_name?: string;
    profile_type?: string;
    segment?: string;
    recommended_action?: string;
    revenue?: number;
    order_count?: number;
    acquisition_channel?: string;
    acquisition_campaign?: string;
    acquisition_confidence?: string;
    product_interest?: string;
    product_views?: number;
    add_to_carts?: number;
    checkout_events?: number;
    email_clicked?: number;
    evidence_gap_orders?: number;
  };

  type EdgeRow = V1MetricsViewRowsResponseDataItem & {
    from_identifier_type?: string;
    from_identifier_value?: string;
    to_identifier_type?: string;
    to_identifier_value?: string;
    edge_source?: string;
    confidence?: string;
    last_seen_at?: string;
  };

  type TimelineRow = V1MetricsViewRowsResponseDataItem & {
    event_ts?: string;
    event_rank?: number;
    row_type?: string;
    event_type?: string;
    event_source?: string;
    title?: string;
    detail?: string;
    order_id?: string;
    order_number?: string | number;
    revenue?: number;
    channel_group?: string;
    source?: string;
    campaign?: string;
    resolution_confidence?: string;
    resolution_reason?: string;
    url?: string;
  };

  const TIMELINE_LIMIT = 500;
  const EDGE_LIMIT = 200;

  $: ({ instanceId } = $runtime);
  $: labels = extractCellLabels(cellContext.filters);
  $: profileId = labels["profile_id"];
  $: whereProfile = profileId
    ? createInExpression("profile_id", [profileId])
    : undefined;

  $: profileQuery = createQueryServiceMetricsViewRows(
    instanceId,
    "profile_directory_metrics",
    { where: whereProfile, limit: 1 },
    { query: { enabled: open && !!instanceId && !!profileId } },
  );

  $: edgesQuery = createQueryServiceMetricsViewRows(
    instanceId,
    "identity_edges_metrics",
    { where: whereProfile, limit: EDGE_LIMIT },
    { query: { enabled: open && !!instanceId && !!profileId } },
  );

  $: timelineQuery = createQueryServiceMetricsViewRows(
    instanceId,
    "profile_timeline_metrics",
    { where: whereProfile, limit: TIMELINE_LIMIT },
    { query: { enabled: open && !!instanceId && !!profileId } },
  );

  $: profile = (($profileQuery.data?.data as ProfileRow[] | undefined) ?? [])[0];
  $: edges = (($edgesQuery.data?.data as EdgeRow[] | undefined) ?? [])
    .slice()
    .sort((a, b) => (b.last_seen_at ?? "").localeCompare(a.last_seen_at ?? ""));
  $: timeline = (($timelineQuery.data?.data as TimelineRow[] | undefined) ?? [])
    .slice()
    .sort((a, b) => {
      const ats = a.event_ts ?? "";
      const bts = b.event_ts ?? "";
      if (ats !== bts) return bts.localeCompare(ats);
      return (b.event_rank ?? 0) - (a.event_rank ?? 0);
    });

  function errorMessage(err: unknown): string {
    if (!err) return "unknown error";
    const e = err as {
      response?: { data?: { message?: string } | string };
      message?: string;
    };
    if (typeof e.response?.data === "string") return e.response.data;
    if (e.response?.data?.message) return e.response.data.message;
    return e.message ?? String(err);
  }

  function fmtDate(iso: string | undefined): string {
    if (!iso) return "-";
    const d = new Date(iso);
    return Number.isNaN(d.getTime())
      ? iso
      : d.toLocaleString(undefined, {
          month: "short",
          day: "numeric",
          year: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        });
  }

  function fmtMoney(n: number | undefined): string {
    if (n == null) return "-";
    return n.toLocaleString(undefined, {
      style: "currency",
      currency: "USD",
      maximumFractionDigits: 2,
    });
  }

  function value(v: string | number | undefined): string {
    if (v === undefined || v === null || v === "") return "-";
    return String(v);
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-6xl">
    <Dialog.Header>
      <Dialog.Title>Profile Graph</Dialog.Title>
      <Dialog.Description>{profile?.display_name ?? profileId ?? "Select a single profile row"}</Dialog.Description>
    </Dialog.Header>

    {#if !profileId}
      <div class="state">Click a row that includes Profile ID to open a single profile graph.</div>
    {:else if $profileQuery.isLoading || $edgesQuery.isLoading || $timelineQuery.isLoading}
      <div class="state">Loading profile...</div>
    {:else if $profileQuery.isError}
      <div class="state error">Couldn't load profile: {errorMessage($profileQuery.error)}</div>
    {:else if $edgesQuery.isError}
      <div class="state error">Couldn't load identity edges: {errorMessage($edgesQuery.error)}</div>
    {:else if $timelineQuery.isError}
      <div class="state error">Couldn't load timeline: {errorMessage($timelineQuery.error)}</div>
    {:else}
      <section class="summary">
        <div><span>Type</span><strong>{value(profile?.profile_type)}</strong></div>
        <div><span>Segment</span><strong>{value(profile?.segment)}</strong></div>
        <div><span>Revenue</span><strong>{fmtMoney(profile?.revenue)}</strong></div>
        <div><span>Orders</span><strong>{value(profile?.order_count)}</strong></div>
        <div><span>Acquisition</span><strong>{value(profile?.acquisition_channel)}</strong></div>
        <div><span>Confidence</span><strong>{value(profile?.acquisition_confidence)}</strong></div>
      </section>

      <section class="panel">
        <div class="panel-title">Recommended Action</div>
        <div class="action">{value(profile?.recommended_action)}</div>
        <div class="muted">{value(profile?.product_interest)} / {value(profile?.acquisition_campaign)}</div>
      </section>

      <div class="columns">
        <section class="panel">
          <div class="panel-title">Identity Edges</div>
          {#if edges.length === 0}
            <div class="empty">No deterministic identity edges for this profile.</div>
          {:else}
            <div class="edge-list">
              {#each edges as edge, i (i)}
                <div class="edge">
                  <div class="mono">{value(edge.from_identifier_type)}:{value(edge.from_identifier_value)}</div>
                  <div class="arrow">-&gt;</div>
                  <div class="mono">{value(edge.to_identifier_type)}:{value(edge.to_identifier_value)}</div>
                  <div class="meta">{value(edge.edge_source)} / {value(edge.confidence)}</div>
                </div>
              {/each}
            </div>
          {/if}
        </section>

        <section class="panel">
          <div class="panel-title">Activity</div>
          <div class="metrics">
            <div><span>Views</span><strong>{value(profile?.product_views)}</strong></div>
            <div><span>Cart</span><strong>{value(profile?.add_to_carts)}</strong></div>
            <div><span>Checkout</span><strong>{value(profile?.checkout_events)}</strong></div>
            <div><span>Email Clicks</span><strong>{value(profile?.email_clicked)}</strong></div>
            <div><span>Evidence Gaps</span><strong>{value(profile?.evidence_gap_orders)}</strong></div>
          </div>
        </section>
      </div>

      <section class="panel timeline-panel">
        <div class="panel-title">Timeline</div>
        {#if timeline.length === 0}
          <div class="empty">No timeline events for this profile.</div>
        {:else}
          <div class="timeline">
            {#each timeline as row, i (i)}
              <div class="event">
                <div class="event-time">{fmtDate(row.event_ts)}</div>
                <div class="event-body">
                  <div class="event-head">
                    <span class="badge">{value(row.row_type)}</span>
                    <strong>{row.row_type === "conversion" ? formatOrderLabel(row.order_number, row.order_id) : value(row.title || row.event_type)}</strong>
                    {#if row.revenue}
                      <span class="money">{fmtMoney(row.revenue)}</span>
                    {/if}
                  </div>
                  <div class="event-detail">{value(row.detail || row.campaign || row.source || row.url || row.resolution_reason)}</div>
                  <div class="meta">{value(row.event_source)} / {value(row.channel_group)} / {value(row.resolution_confidence)}</div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  </Dialog.Content>
</Dialog.Root>

<style>
  .state {
    padding: 32px 8px;
    text-align: center;
    color: var(--color-text-muted, #6b7280);
    font-size: 14px;
  }
  .state.error {
    color: #b91c1c;
  }
  .summary {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 1px;
    border: 1px solid var(--color-bratrax-border, #e5e7eb);
    background: var(--color-bratrax-border, #e5e7eb);
    margin-bottom: 12px;
  }
  .summary > div,
  .panel {
    background: var(--color-bratrax-surface, #fff);
  }
  .summary > div {
    padding: 12px;
    min-width: 0;
  }
  span,
  .muted,
  .meta,
  .event-time,
  .empty {
    color: var(--color-text-muted, #6b7280);
    font-size: 12px;
  }
  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .panel {
    border: 1px solid var(--color-bratrax-border, #e5e7eb);
    padding: 12px;
    margin-bottom: 12px;
  }
  .panel-title {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    margin-bottom: 8px;
  }
  .action {
    font-weight: 700;
    margin-bottom: 4px;
  }
  .columns {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(0, 0.65fr);
    gap: 12px;
  }
  .edge-list {
    display: grid;
    gap: 8px;
    max-height: 220px;
    overflow: auto;
  }
  .edge {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 20px minmax(0, 1fr);
    gap: 8px;
    align-items: center;
    border-bottom: 1px solid var(--color-bratrax-border-faint, #f3f4f6);
    padding-bottom: 8px;
  }
  .edge .meta {
    grid-column: 1 / -1;
  }
  .arrow {
    text-align: center;
    color: var(--color-text-muted, #6b7280);
  }
  .mono {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .metrics {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .metrics strong {
    font-size: 18px;
  }
  .timeline-panel {
    max-height: 48vh;
    overflow: auto;
  }
  .timeline {
    display: grid;
    gap: 10px;
  }
  .event {
    display: grid;
    grid-template-columns: 150px minmax(0, 1fr);
    gap: 12px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--color-bratrax-border-faint, #f3f4f6);
  }
  .event-head {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }
  .event-head strong {
    display: inline;
  }
  .badge {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    text-transform: uppercase;
    background: var(--color-acid-dim, #f5f5dc);
    padding: 2px 6px;
    color: var(--color-text, #111827);
  }
  .money {
    margin-left: auto;
    font-variant-numeric: tabular-nums;
  }
  .event-detail {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin: 2px 0;
  }
  @media (max-width: 900px) {
    .summary {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .columns {
      grid-template-columns: 1fr;
    }
    .event {
      grid-template-columns: 1fr;
    }
  }
</style>
