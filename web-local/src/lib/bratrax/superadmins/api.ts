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

// ---------------------------------------------------------------------------
// Customer signup invitations (kind='signup')
//
// Used when ONLY_INVITATION_LINK=true gates public signup. A super_admin
// generates a single-use link that lets the recipient set a password +
// company name and proceed through onboarding.
// ---------------------------------------------------------------------------

export interface SignupPendingInvite {
  token: string;
  email: string;
  expires_at: string | null;
  created_at: string | null;
  accept_url: string;
  invited_by_email: string | null;
  // True = real paying customer (LS checkout required); false = inceptly
  // internal team (skip payment).
  requires_payment: boolean;
  // True = multi-store invite (creates a rill_multi_clients parent + first
  // sub-store on acceptance, lets the user add more stores from the dashboard).
  // Orthogonal to requires_payment.
  is_multi_store: boolean;
}

export interface SignupInviteResult {
  token: string;
  accept_url: string;
  expires_at: string | null;
  email: string;
  requires_payment: boolean;
  is_multi_store: boolean;
}

export function listSignupInvitations(): Promise<{
  pending: SignupPendingInvite[];
}> {
  return apiFetch("/bratrax/superadmins/signup-invitations");
}

export function createSignupInvite(
  email: string,
  requiresPayment: boolean = true,
  isMultiStore: boolean = false,
): Promise<SignupInviteResult> {
  return apiFetch<SignupInviteResult>("/bratrax/superadmins/signup-invite", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email,
      requires_payment: requiresPayment,
      is_multi_store: isMultiStore,
    }),
  });
}

export function revokeSignupInvitation(
  token: string,
): Promise<{ revoked: string }> {
  return apiFetch(`/bratrax/superadmins/signup-invitations/${token}`, {
    method: "DELETE",
  });
}

// ---------------------------------------------------------------------------
// Access requests — submitted from /signup's invite-only screen, reviewed by
// super_admin. Approve generates a kind='signup' invite (same shape as
// createSignupInvite); dismiss closes without generating.
// ---------------------------------------------------------------------------

export interface AccessRequest {
  id: number;
  email: string;
  created_at: string;
}

export function listAccessRequests(): Promise<{ pending: AccessRequest[] }> {
  return apiFetch("/bratrax/superadmins/access-requests");
}

export function approveAccessRequest(id: number): Promise<SignupInviteResult> {
  return apiFetch<SignupInviteResult>(
    `/bratrax/superadmins/access-requests/${id}/approve`,
    { method: "POST" },
  );
}

export function dismissAccessRequest(
  id: number,
): Promise<{ dismissed: number }> {
  return apiFetch(`/bratrax/superadmins/access-requests/${id}`, {
    method: "DELETE",
  });
}

// ---------------------------------------------------------------------------
// Clients overview (super_admin Clients tab)
//
// Lists every rill_clients row with its multi-store status + lets a super_admin
// flip a single-store client into a multi-store parent in one click.
// ---------------------------------------------------------------------------

export interface SuperadminClientRow {
  client_id: string;
  company_name: string;
  created_at: string | null;
  admin_email: string | null;
  multi_client_id: string | null;
  multi_client_display_name: string | null;
  // Per-client billing flag (legacy single-store source of truth).
  is_paid_subscriber: boolean;
  // Parent's billing flag — present only when multi_client_id is set.
  multi_client_is_paid_subscriber: boolean | null;
}

export function listAllClients(): Promise<{ clients: SuperadminClientRow[] }> {
  return apiFetch("/bratrax/superadmins/clients");
}

export interface EnableMultiStoreResult {
  client_id: string;
  multi_client_id: string;
  linked_users: number;
}

export function enableMultiStoreForClient(
  clientId: string,
): Promise<EnableMultiStoreResult> {
  return apiFetch<EnableMultiStoreResult>(
    `/bratrax/superadmins/clients/${encodeURIComponent(clientId)}/enable-multi-store`,
    { method: "POST" },
  );
}
