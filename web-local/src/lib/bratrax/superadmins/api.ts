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
  // --- CRM fields (derived from rill_onboarding_state) ---------------------
  signed_up_at: string | null;
  admin_name: string | null;
  user_count: number;
  pending_invites: number;
  onboarding_step: string | null;
  step_age_hours: number | null;
  template_name: string | null;
  compile_status: string | null;
  deploy_status: string | null;
  error_message: string | null;
  subscription_status: string | null;
  shopify_embed_enabled: boolean;
  connected_platform_count: number;
  connected_platforms: string[];
  updated_at: string | null;
}

export interface ClientFilters {
  step?: string[];
  subscription_status?: string[];
  paid?: "true" | "false";
  search?: string;
  stuck_hours?: number;
}

function buildClientsQuery(filters?: ClientFilters): string {
  if (!filters) return "";
  const qs = new URLSearchParams();
  filters.step?.forEach((s) => qs.append("step", s));
  filters.subscription_status?.forEach((s) =>
    qs.append("subscription_status", s),
  );
  if (filters.paid) qs.set("paid", filters.paid);
  if (filters.search) qs.set("search", filters.search);
  if (filters.stuck_hours != null)
    qs.set("stuck_hours", String(filters.stuck_hours));
  const s = qs.toString();
  return s ? `?${s}` : "";
}

export function listAllClients(
  filters?: ClientFilters,
): Promise<{ clients: SuperadminClientRow[] }> {
  return apiFetch(`/bratrax/superadmins/clients${buildClientsQuery(filters)}`);
}

// ---------------------------------------------------------------------------
// CRM dashboard — summary tiles + per-client detail.
// ---------------------------------------------------------------------------

export interface ClientsSummary {
  total: number;
  active_paid: number;
  stuck_over_24h: number;
  in_error: number;
  ready: number;
  signed_up_this_week: number;
}

export function getClientsSummary(): Promise<ClientsSummary> {
  return apiFetch("/bratrax/superadmins/clients/summary");
}

export interface ClientConnection {
  platform: string;
  account_id?: string;
  account_name?: string;
  connected_at?: string;
}

export interface ClientTeamMember {
  id: number;
  email: string;
  name: string;
  role: string;
  created_at: string | null;
}

export interface ClientPendingInvite {
  token: string;
  email: string;
  name: string;
  role: string;
  expires_at: string | null;
  created_at: string | null;
}

export interface ClientDetail {
  client_id: string;
  company_name: string;
  signed_up_at: string | null;
  updated_at: string | null;
  timezone: string | null;
  clickhouse_db: string | null;
  rill_project_id: string | null;
  shopify_embed_enabled: boolean;
  multi_client_id: string | null;
  onboarding: {
    step: string | null;
    step_age_hours: number | null;
    template_name: string | null;
    compile_status: string | null;
    deploy_status: string | null;
    extraction_status: Record<string, unknown>;
    connected_platforms: ClientConnection[];
    stack_selections: Record<string, unknown>;
    error_message: string | null;
    updated_at: string | null;
  };
  subscription: {
    is_paid_subscriber: boolean;
    status: string | null;
    lemon_subscription_id: string | null;
    lemon_customer_id: string | null;
  };
  team: ClientTeamMember[];
  pending_invitations: ClientPendingInvite[];
  connected_platforms_oauth: string[];
}

export function getClientDetail(clientId: string): Promise<ClientDetail> {
  return apiFetch(
    `/bratrax/superadmins/clients/${encodeURIComponent(clientId)}`,
  );
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
