import { createAndExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
import { V1Operation } from "@rilldata/web-common/runtime-client";
import { describe, expect, it } from "vitest";
import { buildFinancialStatementWhere } from "./query";

describe("buildFinancialStatementWhere", () => {
  it("removes the empty default canvas filter before adding Daily grain", () => {
    const result = buildFinancialStatementWhere(createAndExpression([]));

    expect(result?.cond?.op).toBe(V1Operation.OPERATION_AND);
    expect(result?.cond?.exprs).toHaveLength(1);
    expect(result?.cond?.exprs?.[0]?.cond?.op).toBe(V1Operation.OPERATION_IN);
    expect(result?.cond?.exprs?.[0]?.cond?.exprs).toEqual([
      { ident: "period_grain" },
      { val: "Daily" },
    ]);
  });
});
