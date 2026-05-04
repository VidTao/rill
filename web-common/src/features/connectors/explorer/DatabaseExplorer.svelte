<script lang="ts">
  import { onMount } from "svelte";
  import { slide } from "svelte/transition";
  import { LIST_SLIDE_DURATION as duration } from "../../../layout/config";
  import type { V1AnalyzedConnector } from "../../../runtime-client";
  import { runtime } from "../../../runtime-client/runtime-store";
  import { get } from "svelte/store";
  import DatabaseEntry from "./DatabaseEntry.svelte";
  import { useListDatabaseSchemas } from "../selectors";
  import type { ConnectorExplorerStore } from "./connector-explorer-store";

  export let instanceId: string;
  export let connector: V1AnalyzedConnector;
  export let store: ConnectorExplorerStore;

  $: connectorName = connector?.name as string;
  $: hasError = !!connector?.errorMessage;

  $: queryEnabled = !hasError;

  $: databaseSchemasQuery = useListDatabaseSchemas(
    instanceId,
    connectorName,
    undefined,
    queryEnabled,
  );

  $: ({ data: rawData, error, isLoading } = $databaseSchemasQuery);

  // TanStack Query returns cached data even when disabled
  $: data = queryEnabled ? rawData : undefined;

  // Bratrax E20: scope ClickHouse databases to the active client only.
  // Without this filter, users (esp. super_admins) see every database on the
  // shared ClickHouse server (vyne, vyne_raw, ziva, ziva_raw, cod, ...).
  // /bratrax/onboard/me returns the active client's clickhouse_db; we keep
  // only `{db}` and `{db}_raw`. DuckDB is unaffected.
  let activeClientDB = "";
  $: isClickHouse = connector?.driver?.name === "clickhouse";

  onMount(async () => {
    if (connector?.driver?.name !== "clickhouse") return;
    try {
      const host = get(runtime).host || "";
      const res = await fetch(`${host}/bratrax/onboard/me`, {
        credentials: "include",
      });
      if (!res.ok) return;
      const j = await res.json();
      if (j?.clickhouse_db) activeClientDB = j.clickhouse_db;
    } catch {
      // Non-fatal: fall back to showing all databases (legacy behavior).
    }
  });

  $: filteredData =
    isClickHouse && activeClientDB
      ? data?.filter(
          (db) => db === activeClientDB || db === `${activeClientDB}_raw`,
        )
      : data;
</script>

<div class="wrapper">
  {#if hasError}
    <span class="message pl-6">Error: {connector.errorMessage}</span>
  {:else if isLoading && queryEnabled}
    <span class="message pl-6">Loading tables...</span>
  {:else if error && queryEnabled}
    <span class="message pl-6"
      >Error: {error.message || error.response?.data?.message}</span
    >
  {:else if filteredData}
    {#if filteredData.length === 0}
      <span class="message pl-6">No tables found</span>
    {:else}
      <ol transition:slide={{ duration }}>
        {#each filteredData as database (database)}
          <DatabaseEntry {instanceId} {connector} {database} {store} />
        {/each}
      </ol>
    {/if}
  {/if}
</div>

<style lang="postcss">
  .wrapper {
    @apply flex flex-col overflow-y-auto;
  }

  .message {
    @apply pr-3.5 py-2; /* left-padding is set inline above */
    @apply text-fg-secondary;
  }
</style>
