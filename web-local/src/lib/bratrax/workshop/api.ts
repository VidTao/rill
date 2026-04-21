import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import type {
  ClientSummary,
  ClientFiles,
  FileKey,
  ValidationResult,
  CompileResult,
  DeployResult,
  TemplateSummary,
  CatalogSource,
  CatalogSearchResult,
  KnowledgeFile,
  KnowledgeFileContent,
} from "./types";

function getBaseUrl(): string {
  return get(runtime).host;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${getBaseUrl()}${path}`, {
    credentials: "include",
    ...init,
  });
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body.error) message = body.error;
      if (body.message) message = body.message;
    } catch {
      // use default message
    }
    throw new Error(message);
  }
  return res.json() as Promise<T>;
}

// ── Clients ──

export async function listClients(): Promise<ClientSummary[]> {
  const res = await apiFetch<{ clients: ClientSummary[] }>("/bratrax/api/v1/clients");
  return res.clients;
}

export async function createClient(
  template: string,
): Promise<{ name: string }> {
  return apiFetch<{ name: string }>("/bratrax/api/v1/clients", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ template }),
  });
}

export async function getClientFiles(name: string): Promise<ClientFiles> {
  const res = await apiFetch<{ files: ClientFiles }>(`/bratrax/api/v1/clients/${encodeURIComponent(name)}`);
  return res.files;
}

export async function saveClientFile(
  name: string,
  fileKey: FileKey,
  content: string,
): Promise<void> {
  await apiFetch(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/${fileKey}`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content }),
    },
  );
}

// ── Validate / Compile / Deploy ──

export async function validateClient(
  name: string,
): Promise<ValidationResult> {
  return apiFetch<ValidationResult>(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/validate`,
    { method: "POST" },
  );
}

export async function compileClient(name: string): Promise<CompileResult> {
  return apiFetch<CompileResult>(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/compile`,
    { method: "POST" },
  );
}

export async function deployClient(
  name: string,
  options: { apply: boolean; create_pr?: boolean } = { apply: false },
): Promise<DeployResult> {
  return apiFetch<DeployResult>(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/deploy`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(options),
    },
  );
}

// ── Templates ──

export async function listTemplates(): Promise<TemplateSummary[]> {
  const res = await apiFetch<{ templates: TemplateSummary[] }>("/bratrax/api/v1/templates");
  return res.templates;
}

// ── Catalogs ──

export async function listCatalogs(): Promise<CatalogSource[]> {
  const res = await apiFetch<{ catalogs: CatalogSource[] }>("/bratrax/api/v1/catalogs");
  return res.catalogs;
}

export async function searchCatalog(
  query: string,
): Promise<{ results: CatalogSearchResult[]; count: number }> {
  return apiFetch<{ results: CatalogSearchResult[]; count: number }>(
    `/bratrax/api/v1/catalogs/search?q=${encodeURIComponent(query)}`,
  );
}

// ── Knowledge ──

export async function listKnowledgeFiles(
  name: string,
): Promise<KnowledgeFile[]> {
  const res = await apiFetch<{ client: string; files: KnowledgeFile[] }>(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/knowledge`,
  );
  return res.files;
}

export async function readKnowledgeFile(
  name: string,
  filepath: string,
): Promise<KnowledgeFileContent> {
  return apiFetch<KnowledgeFileContent>(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/knowledge/${filepath}`,
  );
}

export async function getSystemPrompt(name: string): Promise<string> {
  const res = await apiFetch<{ client: string; content: string }>(
    `/bratrax/api/v1/clients/${encodeURIComponent(name)}/system-prompt`,
  );
  return res.content;
}
