import { BaseCanvasComponent } from "@rilldata/web-common/features/canvas/components/BaseCanvasComponent";
import {
  commonOptions,
  getFilterOptions,
} from "@rilldata/web-common/features/canvas/components/util";
import type { InputParams } from "@rilldata/web-common/features/canvas/inspector/types";
import type {
  V1MetricsViewSpec,
  V1Resource,
} from "@rilldata/web-common/runtime-client";
import type { CanvasEntity, ComponentPath } from "../../stores/canvas-entity";
import type {
  CanvasComponentType,
  ComponentCommonProperties,
  ComponentFilterProperties,
} from "../types";
import FinancialStatementDisplay from "./FinancialStatementDisplay.svelte";

export interface FinancialStatementSpec
  extends ComponentCommonProperties,
    ComponentFilterProperties {
  metrics_view: string;
  currency?: string;
  grain?: "Daily" | "Monthly";
}

export class FinancialStatementComponent extends BaseCanvasComponent<FinancialStatementSpec> {
  minSize = { width: 6, height: 8 };
  defaultSize = { width: 12, height: 14 };
  resetParams: string[] = [];
  type: CanvasComponentType = "financial_statement";
  component = FinancialStatementDisplay;

  constructor(resource: V1Resource, parent: CanvasEntity, path: ComponentPath) {
    super(resource, parent, path, {
      metrics_view: "",
      currency: "USD",
      grain: "Daily",
    });
  }

  isValid(spec: FinancialStatementSpec): boolean {
    return (
      typeof spec.metrics_view === "string" && spec.metrics_view.length > 0
    );
  }

  inputParams(): InputParams<FinancialStatementSpec> {
    return {
      options: {
        metrics_view: { type: "metrics", label: "Metrics view" },
        currency: { type: "text", optional: true, label: "Currency" },
        grain: { type: "text", optional: true, label: "Default grain" },
        ...commonOptions,
      },
      filter: getFilterOptions(false, false),
    };
  }

  static newComponentSpec(
    metricsViewName: string,
    _metricsViewSpec?: V1MetricsViewSpec,
  ): FinancialStatementSpec {
    return { metrics_view: metricsViewName, currency: "USD", grain: "Daily" };
  }
}

export { default as FinancialStatementDisplay } from "./FinancialStatementDisplay.svelte";
