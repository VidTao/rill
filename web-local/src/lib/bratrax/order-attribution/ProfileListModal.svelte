<script lang="ts">
  import type { MeasureCellClickContext } from "@rilldata/web-common/features/canvas/components/pivot/drilldown-context";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import {
    createQueryServiceMetricsViewRows,
    type V1MetricsViewRowsResponseDataItem,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  export let open = false;
  export let cellContext: MeasureCellClickContext;
  export let onSelectProfile: (profileId: string) => void;

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
    last_activity_at?: string;
  };

  const LIMIT = 500;

  $: ({ instanceId } = $runtime);

  $: rowsQuery = createQueryServiceMetricsViewRows(
    instanceId,
    "profile_directory_metrics",
    {
      where: cellContext.filters,
      timeStart: cellContext.timeRange.start,
      timeEnd: cellContext.timeRange.end,
      limit: LIMIT,
    },
    {
      query: { enabled: open && !!instanceId },
    },
  );

  $: profiles = (($rowsQuery.data?.data as ProfileRow[] | undefined) ?? []);
  $: hitRowLimit = profiles.length >= LIMIT;

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
      : d.toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
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

  function selectProfile(profileId: string | undefined) {
    if (!profileId) return;
    onSelectProfile(profileId);
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-6xl">
    <Dialog.Header>
      <Dialog.Title>Profiles</Dialog.Title>
      <Dialog.Description>Profiles matching the clicked pivot cell.</Dialog.Description>
    </Dialog.Header>

    {#if $rowsQuery.isLoading}
      <div class="state">Loading profiles...</div>
    {:else if $rowsQuery.isError}
      <div class="state error">
        Couldn't load profiles: {errorMessage($rowsQuery.error)}
      </div>
    {:else if profiles.length === 0}
      <div class="state">No profiles match this cell.</div>
    {:else}
      {#if hitRowLimit}
        <div class="banner">Showing the first {LIMIT} profiles. Tighten the pivot cell to narrow the list.</div>
      {/if}
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Profile</th>
              <th>Type</th>
              <th>Segment</th>
              <th class="num">Revenue</th>
              <th class="num">Orders</th>
              <th>Recommended Action</th>
              <th>Acquisition</th>
              <th>Confidence</th>
              <th>Last Activity</th>
            </tr>
          </thead>
          <tbody>
            {#each profiles as p, i (p.profile_id ?? p.display_name ?? i)}
              <tr
                tabindex="0"
                on:click={() => selectProfile(p.profile_id)}
                on:keydown={(e) =>
                  (e.key === "Enter" || e.key === " ") &&
                  selectProfile(p.profile_id)}
              >
                <td class="profile-link">{value(p.display_name || p.profile_id)}</td>
                <td>{value(p.profile_type)}</td>
                <td>{value(p.segment)}</td>
                <td class="num">{fmtMoney(p.revenue)}</td>
                <td class="num">{value(p.order_count)}</td>
                <td>{value(p.recommended_action)}</td>
                <td>{value(p.acquisition_channel)} / {value(p.acquisition_campaign)}</td>
                <td>{value(p.acquisition_confidence)}</td>
                <td>{fmtDate(p.last_activity_at)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
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
  .banner {
    padding: 8px 12px;
    background: var(--color-acid-dim, #f5f5dc);
    border-left: 3px solid var(--color-acid, #d4ff00);
    font-size: 12px;
    color: var(--color-text-muted, #6b7280);
  }
  .table-wrap {
    max-height: 60vh;
    overflow-y: auto;
    border-top: 1px solid var(--color-bratrax-border, #e5e7eb);
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  thead th {
    position: sticky;
    top: 0;
    background: var(--color-bratrax-surface, #ffffff);
    text-align: left;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--color-text-muted, #6b7280);
    padding: 8px 12px;
    border-bottom: 1px solid var(--color-bratrax-border, #e5e7eb);
  }
  tbody td {
    padding: 8px 12px;
    border-bottom: 1px solid var(--color-bratrax-border-faint, #f3f4f6);
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr:hover {
    background: var(--color-acid-dim, #f5f5dc);
  }
  tbody tr:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: -2px;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .profile-link {
    color: #2563eb;
    text-decoration: underline;
    font-family: "Space Mono", "JetBrains Mono", monospace;
  }
</style>
