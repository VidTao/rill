<script lang="ts">
  // Wraps the dashboard area, registers the drilldown handler on Svelte
  // context, and owns the two modals' open state. Mount this once at the
  // workspace layout level for clients we've enabled the feature for —
  // currently earths_mushrooms only.

  import {
    ORDER_DRILLDOWN_CONTEXT,
    type MeasureCellClickContext,
    type OrderDrilldownContext,
  } from "@rilldata/web-common/features/canvas/components/pivot/drilldown-context";
  import { createInExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
  import { setContext } from "svelte";
  import { extractCellLabels } from "./api";
  import OrderListModal from "./OrderListModal.svelte";
  import OrderTimelineModal from "./OrderTimelineModal.svelte";
  import ProfileGraphModal from "./ProfileGraphModal.svelte";
  import ProfileListModal from "./ProfileListModal.svelte";

  // Modal 1 state: cell click context drives the orders-list query.
  let cellContext: MeasureCellClickContext | null = null;
  let listOpen = false;

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
      cellContext = ctx;
      listOpen = true;
    },
  };

  setContext(ORDER_DRILLDOWN_CONTEXT, drilldownContext);

  function handleOrderSelect(orderId: string) {
    selectedOrderId = orderId;
    selectedAttributionModel = cellContext
      ? extractCellLabels(cellContext.filters)["attribution_model"]
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
    onSelectOrder={handleOrderSelect}
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
