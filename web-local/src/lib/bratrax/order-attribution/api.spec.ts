import { describe, expect, it } from "vitest";
import { groupCancelledOrderItems } from "./api";
import type { CancelledOrderItemRow } from "./types";

describe("groupCancelledOrderItems", () => {
  it("deduplicates items while summing each attribution slice once", () => {
    const base = {
      order_id: "order-1",
      order_number: 42,
      order_created_at: "2026-07-01T10:00:00Z",
      currency: "USD",
      order_total: 120,
    };
    const rows: CancelledOrderItemRow[] = [
      {
        ...base,
        attribution_slice_id: "slice-a",
        attribution_weight: 0.4,
        line_item_id: "line-1",
        sku: "AED-A",
        quantity: 1,
        item_value: 40,
      },
      {
        ...base,
        attribution_slice_id: "slice-a",
        attribution_weight: 0.4,
        line_item_id: "line-2",
        sku: "AED-B",
        quantity: 2,
        item_value: 80,
      },
      {
        ...base,
        attribution_slice_id: "slice-b",
        attribution_weight: 0.6,
        line_item_id: "line-1",
        sku: "AED-A",
        quantity: 1,
        item_value: 40,
      },
      {
        ...base,
        attribution_slice_id: "slice-b",
        attribution_weight: 0.6,
        line_item_id: "line-2",
        sku: "AED-B",
        quantity: 2,
        item_value: 80,
      },
    ];

    const [order] = groupCancelledOrderItems(rows);

    expect(order.attribution_weight).toBeCloseTo(1);
    expect(order.items).toHaveLength(2);
    expect(order.items.map((item) => item.sku)).toEqual(["AED-A", "AED-B"]);
  });

  it("keeps a cancelled order that has no line-item match", () => {
    const [order] = groupCancelledOrderItems([
      {
        order_id: "order-2",
        attribution_slice_id: "slice-c",
        attribution_weight: 1,
        line_item_id: "",
        sku: "",
      },
    ]);

    expect(order.order_id).toBe("order-2");
    expect(order.items).toHaveLength(1);
    expect(order.items[0].sku).toBe("");
  });

  it("sorts orders newest first", () => {
    const orders = groupCancelledOrderItems([
      { order_id: "old", order_created_at: "2026-06-01T00:00:00Z" },
      { order_id: "new", order_created_at: "2026-07-01T00:00:00Z" },
    ]);

    expect(orders.map((order) => order.order_id)).toEqual(["new", "old"]);
  });
});
