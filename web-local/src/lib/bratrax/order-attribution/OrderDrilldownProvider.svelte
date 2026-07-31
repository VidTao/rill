<script lang="ts">
  // Wraps the dashboard area, registers the drilldown handler on Svelte
  // context, and owns the two modals' open state. Mount this once at the
  // workspace layout level. Runtime resource discovery enables each
  // drilldown only when the active client exposes its backing metrics view.

  import {
    ORDER_DRILLDOWN_CONTEXT,
    type MeasureCellClickContext,
    type OrderDrilldownContext,
  } from "@rilldata/web-common/features/canvas/components/pivot/drilldown-context";
  import { createRuntimeServiceListResources } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { createInExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
  import { setContext } from "svelte";
  import { writable } from "svelte/store";
  import { CANCELLED_ORDER_ITEMS_METRICS_VIEW, extractCellLabels } from "./api";
  import CancelledOrderListModal from "./CancelledOrderListModal.svelte";
  import OrderListModal from "./OrderListModal.svelte";
  import OrderTimelineModal from "./OrderTimelineModal.svelte";
  import ProfileGraphModal from "./ProfileGraphModal.svelte";
  import ProfileListModal from "./ProfileListModal.svelte";

  // Modal 1 state: cell click context drives the orders-list query.
  let cellContext: MeasureCellClickContext | null = null;
  let listOpen = false;

  let cancelledContext: MeasureCellClickContext | null = null;
  let cancelledOpen = false;

  $: ({ instanceId } = $runtime);
  $: metricsViewsQuery = createRuntimeServiceListResources(instanceId, {
    kind: "rill.runtime.v1.MetricsView",
  });
  const supportedMeasures = writable<ReadonlySet<string>>(
    new Set(["metric_attributed_orders", "profiles"]),
  );
  $: {
    const hasCancelledItems = ($metricsViewsQuery.data?.resources ?? []).some(
      (resource) =>
        resource.meta?.name?.name === CANCELLED_ORDER_ITEMS_METRICS_VIEW,
    );
    supportedMeasures.set(
      new Set([
        "metric_attributed_orders",
        "profiles",
        ...(hasCancelledItems ? ["metric_cancelled_orders"] : []),
      ]),
    );
  }

  // Profile explorer state: grouped profile cells open a list; a single
  // profile row opens the identity graph directly.
  let profileListContext: MeasureCellClickContext | null = null;
  let profileListOpen = false;
  let profileContext: MeasureCellClickContext | null = null;
  let profileOpen = false;

  // Modal 2 state: a selected order from Modal 1 drives the timeline query.
  // The attribution model is captured at click time so the winner banner
  // tracks Modal 1's filter even if the user later changes the dashboard
  // filter while the timeline is still open.
  let selectedOrderId: string | null = null;
  let selectedAttributionModel: string | undefined = undefined;
  let timelineOpen = false;

  const drilldownContext: OrderDrilldownContext = {
    supportedMeasures,
    open(ctx) {
      if (ctx.measureName === "profiles") {
        const profileId = extractCellLabels(ctx.filters)["profile_id"];
        if (profileId) {
          profileContext = ctx;
          profileOpen = true;
        } else {
          profileListContext = ctx;
          profileListOpen = true;
        }
        return;
      }
      if (ctx.measureName === "metric_cancelled_orders") {
        cancelledContext = ctx;
        cancelledOpen = true;
        return;
      }
      cellContext = ctx;
      listOpen = true;
    },
  };

  setContext(ORDER_DRILLDOWN_CONTEXT, drilldownContext);

  function handleOrderSelect(
    orderId: string,
    context: MeasureCellClickContext | null,
  ) {
    selectedOrderId = orderId;
    selectedAttributionModel = context
      ? extractCellLabels(context.filters)["attribution_model"]
      : undefined;
    timelineOpen = true;
  }

  function handleProfileSelect(profileId: string) {
    const timeRange = profileListContext?.timeRange ?? {
      start: undefined,
      end: undefined,
    };
    profileContext = {
      measureName: "profiles",
      filters: createInExpression("profile_id", [profileId]),
      timeRange,
    };
    profileListOpen = false;
    profileOpen = true;
  }

  // When Modal 1 closes, drop the cell context so its query doesn't refire
  // if the user re-opens with a different cell. Modal 2's order_id stays
  // available until it closes too.
  $: if (!listOpen) cellContext = null;
  $: if (!cancelledOpen) cancelledContext = null;
  $: if (!profileListOpen) profileListContext = null;
  $: if (!profileOpen) profileContext = null;
  $: if (!timelineOpen) {
    selectedOrderId = null;
    selectedAttributionModel = undefined;
  }
</script>

<slot />

{#if cellContext}
  <OrderListModal
    bind:open={listOpen}
    {cellContext}
    onSelectOrder={(orderId) => handleOrderSelect(orderId, cellContext)}
  />
{/if}

{#if cancelledContext}
  <CancelledOrderListModal
    bind:open={cancelledOpen}
    cellContext={cancelledContext}
    onSelectOrder={(orderId) => handleOrderSelect(orderId, cancelledContext)}
  />
{/if}

{#if selectedOrderId}
  <OrderTimelineModal
    bind:open={timelineOpen}
    orderId={selectedOrderId}
    attributionModel={selectedAttributionModel}
  />
{/if}

{#if profileListContext}
  <ProfileListModal
    bind:open={profileListOpen}
    cellContext={profileListContext}
    onSelectProfile={handleProfileSelect}
  />
{/if}

{#if profileContext}
  <ProfileGraphModal bind:open={profileOpen} cellContext={profileContext} />
{/if}
