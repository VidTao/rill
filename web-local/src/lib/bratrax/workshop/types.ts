export interface ClientSummary {
  name: string;
  has_config: boolean;
  has_ontology: boolean;
  has_sources: boolean;
  has_tracking_plan: boolean;
}

export interface ClientFiles {
  config: string;
  ontology: string;
  sources: string;
  tracking_plan: string;
}

export type FileKey = keyof ClientFiles;

export const FILE_KEYS: FileKey[] = [
  "config",
  "sources",
  "ontology",
  "tracking_plan",
];

export const FILE_LABELS: Record<FileKey, string> = {
  config: "Config",
  sources: "Sources",
  ontology: "Ontology",
  tracking_plan: "Tracking Plan",
};

export interface ValidationIssue {
  severity: "error" | "warning";
  code: string;
  message: string;
  context: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: number;
  warnings: number;
  resolved_refs: number;
  issues: ValidationIssue[];
}

export interface CompileArtifact {
  path: string;
  type: string;
  layer: string;
  content: string;
}

export interface CompileResult {
  success: boolean;
  artifacts: CompileArtifact[];
  summary: {
    rill_source: number;
    rill_model: number;
    rill_metrics: number;
    ch_flatten: number;
    ch_activity_stream: number;
    ch_dims: number;
    total: number;
  };
  validation: ValidationResult;
}

export interface DeployAction {
  action: "CREATE" | "UPDATE" | "EXECUTE";
  target: string;
}

export interface ChExecutionResult {
  file: string;
  statement: number;
  type: string;
  ok: boolean;
  error: string;
}

export interface DeployResult {
  plan: DeployAction[];
  applied: boolean;
  pr_url: string | null;
  ch_results: ChExecutionResult[];
}

export interface TemplateSummary {
  name: string;
  display_name: string;
  description: string;
}

// ── Catalog types ──

export interface CatalogField {
  name: string;
  type: string;
  nullable: boolean;
  singer_types: string[];
}

export interface CatalogStream {
  name: string;
  fields: CatalogField[];
}

export interface CatalogSource {
  source_name: string;
  label: string;
  category: string;
  source_type: "meltano" | "webhook";
  raw_table: string;
  streams: CatalogStream[];
}

export interface CatalogSearchResult {
  field: string;
  type: string;
  stream: string;
  tap: string;
  label: string;
  source_ref: string;
}

export type WorkshopPhase = "idle" | "validating" | "compiling" | "deploying";

export interface WorkshopResults {
  phase: "validation" | "compile" | "deploy";
  validation: ValidationResult | null;
  compile: CompileResult | null;
  deploy: DeployResult | null;
}
