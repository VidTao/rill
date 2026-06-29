import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import type { AuthoredMetricNode, AuthoredMetricTree, CreateMetricTreeEventPayload, CreateMetricTreeReviewSessionPayload, MetricTreeEvent, MetricTreeEvidence, MetricTreeReviewSession, MetricTreeValidation } from "./types";

function getBaseUrl(): string {
  return get(runtime).host;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${getBaseUrl()}${path}`, { credentials: "include", ...init });
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      message = body.error ?? body.message ?? message;
      if (body.validation) message += `: ${body.validation.errors?.[0]?.message ?? "validation failed"}`;
    } catch {
      /* use default */
    }
    throw new Error(message);
  }
  return res.json() as Promise<T>;
}

export async function listMetricTrees(): Promise<Array<{ tree_id: string; name: string; description: string; root_node_id: string; updated_at: string }>> {
  const res = await apiFetch<{ data: Array<{ tree_id: string; name: string; description: string; root_node_id: string; updated_at: string }> }>("/bratrax/metric-trees");
  return res.data;
}

export type MetricTreeTemplateSummary = {
  templateId: string;
  tree_id: string;
  name: string;
  description: string;
  nodeCount: number;
};

export async function listMetricTreeTemplates(): Promise<MetricTreeTemplateSummary[]> {
  const res = await apiFetch<{ data: MetricTreeTemplateSummary[] }>("/bratrax/metric-trees/templates");
  return res.data;
}

export async function createStarterTree(templateId = "d2c_hybrid"): Promise<AuthoredMetricTree> {
  const res = await apiFetch<{ data: AuthoredMetricTree }>("/bratrax/metric-trees/starter", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ templateId }),
  });
  return res.data;
}

export async function getMetricTree(treeId: string): Promise<AuthoredMetricTree> {
  const res = await apiFetch<{ data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}`);
  return res.data;
}

export async function updateMetricTree(treeId: string, payload: Partial<AuthoredMetricTree>): Promise<AuthoredMetricTree> {
  const res = await apiFetch<{ data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return res.data;
}

export async function createMetricNode(treeId: string, node: Partial<AuthoredMetricNode>): Promise<AuthoredMetricTree> {
  const res = await apiFetch<{ data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/nodes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(node),
  });
  return res.data;
}

export async function updateMetricNode(treeId: string, nodeId: string, node: Partial<AuthoredMetricNode>): Promise<AuthoredMetricTree> {
  const res = await apiFetch<{ data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/nodes/${encodeURIComponent(nodeId)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(node),
  });
  return res.data;
}

export async function deleteMetricNode(treeId: string, nodeId: string): Promise<AuthoredMetricTree> {
  const res = await apiFetch<{ data: AuthoredMetricTree }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/nodes/${encodeURIComponent(nodeId)}`, {
    method: "DELETE",
  });
  return res.data;
}

export async function validateMetricTree(treeId: string, nodes?: AuthoredMetricNode[]): Promise<MetricTreeValidation> {
  const res = await apiFetch<{ validation: MetricTreeValidation }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/validate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(nodes ? { nodes } : {}),
  });
  return res.validation;
}

export async function getMetricTreeEvents(treeId: string): Promise<MetricTreeEvent[]> {
  const res = await apiFetch<{ data: MetricTreeEvent[] }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/events`);
  return res.data;
}

export async function getMetricTreeEvidence(treeId: string, nodeId: string): Promise<MetricTreeEvidence> {
  const res = await apiFetch<{ data: MetricTreeEvidence }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/nodes/${encodeURIComponent(nodeId)}/evidence`);
  return res.data;
}

export async function createMetricTreeEvent(treeId: string, nodeId: string, payload: CreateMetricTreeEventPayload): Promise<void> {
  await apiFetch<{ success: boolean; data: unknown }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/nodes/${encodeURIComponent(nodeId)}/events`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}


export async function listMetricTreeReviewSessions(treeId: string): Promise<MetricTreeReviewSession[]> {
  const res = await apiFetch<{ data: MetricTreeReviewSession[] }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/review-sessions`);
  return res.data;
}

export async function getMetricTreeReviewSession(treeId: string, sessionId: string): Promise<MetricTreeReviewSession> {
  const res = await apiFetch<{ data: MetricTreeReviewSession }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/review-sessions/${encodeURIComponent(sessionId)}`);
  return res.data;
}

export async function createMetricTreeReviewSession(treeId: string, payload: CreateMetricTreeReviewSessionPayload): Promise<MetricTreeReviewSession> {
  const res = await apiFetch<{ data: MetricTreeReviewSession }>(`/bratrax/metric-trees/${encodeURIComponent(treeId)}/review-sessions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return res.data;
}
