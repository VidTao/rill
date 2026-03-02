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
    flatten: number;
    activity_stream: number;
    dim: number;
    meltano: number;
    total: number;
  };
  validation: ValidationResult;
}

export interface DeployAction {
  action: "CREATE" | "UPDATE";
  target: string;
}

export interface DeployResult {
  plan: DeployAction[];
  applied: boolean;
  pr_url: string | null;
}

export interface TemplateSummary {
  name: string;
  display_name: string;
  description: string;
}

export type WorkshopPhase = "idle" | "validating" | "compiling" | "deploying";

export interface WorkshopResults {
  phase: "validation" | "compile" | "deploy";
  validation: ValidationResult | null;
  compile: CompileResult | null;
  deploy: DeployResult | null;
}
