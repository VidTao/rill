import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

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

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface SuperadminMember {
  id: number;
  email: string;
  name: string;
  created_at: string | null;
}

export interface SuperadminPendingInvite {
  token: string;
  email: string;
  name: string;
  role: "super_admin";
  expires_at: string | null;
  created_at: string | null;
  accept_url: string;
}

export interface SuperadminListResponse {
  members: SuperadminMember[];
  pending: SuperadminPendingInvite[];
}

export interface SuperadminInviteResult {
  token: string;
  accept_url: string;
  expires_at: string | null;
  email: string;
  role: "super_admin";
}

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

export function getSuperadmins(): Promise<SuperadminListResponse> {
  return apiFetch<SuperadminListResponse>("/bratrax/superadmins");
}

export function inviteSuperadmin(
  email: string,
  name = "",
): Promise<SuperadminInviteResult> {
  return apiFetch<SuperadminInviteResult>("/bratrax/superadmins/invite", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, name }),
  });
}

export function removeSuperadmin(userId: number): Promise<{ removed: number }> {
  return apiFetch(`/bratrax/superadmins/${userId}`, { method: "DELETE" });
}

export function revokeSuperadminInvitation(
  token: string,
): Promise<{ revoked: string }> {
  return apiFetch(`/bratrax/superadmins/invitations/${token}`, {
    method: "DELETE",
  });
}
