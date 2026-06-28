import { BaseCanvasComponent } from "@rilldata/web-common/features/canvas/components/BaseCanvasComponent";
import { commonOptions } from "@rilldata/web-common/features/canvas/components/util";
import type { InputParams } from "@rilldata/web-common/features/canvas/inspector/types";
import {
  type V1MetricsViewSpec,
  type V1Resource,
} from "@rilldata/web-common/runtime-client";
import type { CanvasEntity, ComponentPath } from "../../stores/canvas-entity";
import type { CanvasComponentType } from "../types";
import {
  MEASURE_ROLES,
  METRIC_TREE_DEFAULT_COLUMNS,
  type MetricTreeRole,
  type MetricTreeSpec,
} from "./columns";
import { getMetricTreeContextMetricsView } from "./context";
import MetricTreeDisplay from "./MetricTreeDisplay.svelte";

export { default as MetricTreeDisplay } from "./MetricTreeDisplay.svelte";
export * from "./columns";
export * from "./context";

// Role fields that reference columns of the old metrics view and must be
// cleared when the metrics_view changes.
const COLUMN_ROLE_PARAMS: string[] = [
  ...(Object.keys(METRIC_TREE_DEFAULT_COLUMNS) as MetricTreeRole[]),
  "tree_name",
  "context_metrics_view",
  "relationships_view",
  "evidence_view",
  "personas_view",
  "logs_view",
];

export class MetricTreeComponent extends BaseCanvasComponent<MetricTreeSpec> {
  minSize = { width: 4, height: 4 };
  defaultSize = { width: 8, height: 6 };
  resetParams = COLUMN_ROLE_PARAMS;
  type: CanvasComponentType = "metric_tree";
  component = MetricTreeDisplay;

  constructor(resource: V1Resource, parent: CanvasEntity, path: ComponentPath) {
    super(resource, parent, path, { metrics_view: "" });
  }

  protected getMetricsViewName(spec: MetricTreeSpec): string {
    return getMetricTreeContextMetricsView(spec);
  }

  isValid(spec: MetricTreeSpec): boolean {
    return typeof spec.tree_id === "string" || typeof spec.metrics_view === "string";
  }

  inputParams(): InputParams<MetricTreeSpec> {
    return {
      options: {
        tree_id: {
          type: "text",
          optional: true,
          label: "Authored tree ID",
          description: "Use a Bratrax-authored metric tree instead of a metrics view",
        },
        metrics_view: { type: "metrics", optional: true, label: "Metrics view" },
        context_metrics_view: {
          type: "metrics",
          optional: true,
          label: "Time context metrics view",
          description: "Metrics view used to expose canvas time controls for authored metric trees",
        },
        relationships_view: {
          type: "metrics",
          optional: true,
          label: "Relationships view",
        },
        evidence_view: {
          type: "metrics",
          optional: true,
          label: "Evidence view",
        },
        personas_view: {
          type: "metrics",
          optional: true,
          label: "Personas view",
        },
        logs_view: {
          type: "metrics",
          optional: true,
          label: "Logs view",
        },

        tree_column: {
          type: "dimension",
          optional: true,
          label: "Tree name column",
          description: "Column that names which tree each row belongs to",
        },
        tree_name: {
          type: "text",
          optional: true,
          label: "Tree",
          description: "Which tree to display (leave blank to show all)",
        },

        node_id: { type: "dimension", label: "Node ID" },
        parent_node_id: { type: "dimension", label: "Parent node ID" },
        label: { type: "dimension", label: "Label" },

        value: { type: "measure", optional: true, label: "Value" },
        value2: {
          type: "measure",
          optional: true,
          label: "Value 2 (face)",
        },
        value3: {
          type: "measure",
          optional: true,
          label: "Value 3 (face)",
        },
        value2_label: { type: "text", optional: true, label: "Value 2 label" },
        value2_unit: { type: "text", optional: true, label: "Value 2 unit" },
        value3_label: { type: "text", optional: true, label: "Value 3 label" },
        value3_unit: { type: "text", optional: true, label: "Value 3 unit" },
        delta_value: { type: "measure", optional: true, label: "Delta" },
        unit: { type: "dimension", optional: true, label: "Unit" },
        status: { type: "dimension", optional: true, label: "Status" },
        sort_order: { type: "dimension", optional: true, label: "Sort order" },

        confidence: { type: "dimension", optional: true, label: "Confidence" },
        limitation: { type: "dimension", optional: true, label: "Limitation" },
        observation: {
          type: "dimension",
          optional: true,
          label: "Observation",
        },
        suggested_test: {
          type: "dimension",
          optional: true,
          label: "Suggested test",
        },
        success_metric: {
          type: "dimension",
          optional: true,
          label: "Success metric",
        },
        driver_type: { type: "dimension", optional: true, label: "Driver type" },
        attribution_model: {
          type: "dimension",
          optional: true,
          label: "Attribution model",
        },
        comparison_model: {
          type: "dimension",
          optional: true,
          label: "Comparison model",
        },
        segment: { type: "dimension", optional: true, label: "Segment" },
        recommended_action: {
          type: "dimension",
          optional: true,
          label: "Recommended action",
        },
        evidence_label: {
          type: "dimension",
          optional: true,
          label: "Evidence label",
        },
        evidence_metric: {
          type: "dimension",
          optional: true,
          label: "Evidence metric",
        },
        log_filter_key: {
          type: "dimension",
          optional: true,
          label: "Log filter key",
        },
        log_filter_value: {
          type: "dimension",
          optional: true,
          label: "Log filter value",
        },
        persona_filter_key: {
          type: "dimension",
          optional: true,
          label: "Persona filter key",
        },
        persona_filter_value: {
          type: "dimension",
          optional: true,
          label: "Persona filter value",
        },

        orientation: {
          type: "select",
          optional: true,
          label: "Orientation",
          meta: {
            default: "TB",
            options: [
              { label: "Top-down", value: "TB" },
              { label: "Left-to-right", value: "LR" },
            ],
          },
        },

        ...commonOptions,
      },
      // A metric tree is a current-state snapshot — no time-range UI.
      filter: [],
    };
  }

  static newComponentSpec(
    metricsViewName: string,
    metricsViewSpec: V1MetricsViewSpec | undefined,
  ): MetricTreeSpec {
    const dimNames = new Set(
      (metricsViewSpec?.dimensions ?? []).map(
        (d) => (d.name || d.column) as string,
      ),
    );
    const measureNames = new Set(
      (metricsViewSpec?.measures ?? []).map((m) => m.name as string),
    );

    const spec: MetricTreeSpec = { metrics_view: metricsViewName };
    for (const role of Object.keys(
      METRIC_TREE_DEFAULT_COLUMNS,
    ) as MetricTreeRole[]) {
      const col = METRIC_TREE_DEFAULT_COLUMNS[role];
      const pool = MEASURE_ROLES.has(role) ? measureNames : dimNames;
      if (pool.has(col)) {
        (spec as Record<string, unknown>)[role] = col;
      }
    }
    return spec;
  }
}
