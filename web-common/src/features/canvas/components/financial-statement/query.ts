import { mergeFilters } from "@rilldata/web-common/features/dashboards/pivot/pivot-merge-filters";
import { createInExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
import type { V1Expression } from "@rilldata/web-common/runtime-client";

export function buildFinancialStatementWhere(
  where: V1Expression | undefined,
): V1Expression | undefined {
  return mergeFilters(where, createInExpression("period_grain", ["Daily"]));
}
