import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import type {
  AccountInfo,
  AISettings,
  BillingSummary,
  InvitationPreview,
  InviteResult,
  MCPSettings,
  Role,
  TeamData,
} from "./types";

function host(): string {
  return get(runtime).host;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${host()}${path}`, {
    credentials: "include",
    ...init,
  });
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body.error) message = body.error;
    } catch {
      // ignore parse errors
    }
    throw new Error(message);
  }
  return res.json();
}

// ----- Account ---------------------------------------------------------------

export function getAccount(): Promise<AccountInfo> {
  return apiFetch<AccountInfo>("/bratrax/settings/account");
}

export function updateAccount(payload: Partial<AccountInfo>): Promise<AccountInfo> {
  return apiFetch<AccountInfo>("/bratrax/settings/account", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

// ----- Team ------------------------------------------------------------------

export function getTeam(): Promise<TeamData> {
  return apiFetch<TeamData>("/bratrax/settings/team");
}

export function inviteMember(
  email: string,
  role: Exclude<Role, "super_admin">,
  name = "",
): Promise<InviteResult> {
  return apiFetch<InviteResult>("/bratrax/settings/team/invite", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, role, name }),
  });
}

export function updateRole(
  userId: number,
  role: Exclude<Role, "super_admin">,
): Promise<{ id: number; role: Role }> {
  return apiFetch(`/bratrax/settings/team/${userId}/role`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ role }),
  });
}

export function removeMember(userId: number): Promise<{ removed: number }> {
  return apiFetch(`/bratrax/settings/team/${userId}`, { method: "DELETE" });
}

export function revokeInvitation(token: string): Promise<{ revoked: string }> {
  return apiFetch(`/bratrax/settings/team/invitations/${token}`, {
    method: "DELETE",
  });
}

// ----- Billing ---------------------------------------------------------------

export function getBilling(): Promise<BillingSummary> {
  return apiFetch<BillingSummary>("/bratrax/settings/billing");
}

// ----- AI (BYOK) -------------------------------------------------------------

export function getAISettings(): Promise<AISettings> {
  return apiFetch<AISettings>("/bratrax/settings/ai");
}

export function updateAISettings(anthropic_api_key: string): Promise<AISettings> {
  return apiFetch<AISettings>("/bratrax/settings/ai", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ anthropic_api_key }),
  });
}

export function deleteAISettings(): Promise<AISettings> {
  return apiFetch<AISettings>("/bratrax/settings/ai", { method: "DELETE" });
}

// ----- MCP (Claude Desktop bridge) -------------------------------------------

export function getMCPSettings(): Promise<MCPSettings> {
  return apiFetch<MCPSettings>("/bratrax/settings/mcp");
}

export function regenerateMCPToken(): Promise<MCPSettings> {
  return apiFetch<MCPSettings>("/bratrax/settings/mcp/regenerate", {
    method: "POST",
  });
}

export function deleteMCPToken(): Promise<MCPSettings> {
  return apiFetch<MCPSettings>("/bratrax/settings/mcp", { method: "DELETE" });
}

// ----- Invitation acceptance (public, used by /accept-invite/[token]) --------

export function getInvitation(token: string): Promise<InvitationPreview> {
  return apiFetch<InvitationPreview>(`/bratrax/invitations/${token}`);
}

export function acceptInvitation(
  token: string,
  password: string,
  name = "",
): Promise<{ user_id: number }> {
  return apiFetch(`/bratrax/invitations/${token}/accept`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password, name }),
  });
}
