export type Role = "super_admin" | "admin" | "viewer";

export interface AccountInfo {
  company_name: string;
  email: string;
  timezone: string;
  role: Role;
}

export interface TeamMember {
  id: number;
  email: string;
  name: string;
  role: Role;
}

export interface PendingInvite {
  token: string;
  email: string;
  name: string;
  role: Exclude<Role, "super_admin">;
  expires_at: string;
  created_at: string;
  accept_url: string;
}

export interface TeamData {
  members: TeamMember[];
  pending: PendingInvite[];
}

export interface InviteResult {
  token: string;
  accept_url: string;
  expires_at: string;
  email: string;
  role: Exclude<Role, "super_admin">;
}

export interface PaymentMethod {
  brand: string;
  last4: string;
}

export interface Invoice {
  id: string;
  date: string;
  amount_cents: number;
  status: string;
  url: string;
}

export interface BillingSummary {
  plan: string;
  price_cents: number;
  currency: string;
  interval: string;
  status: string;
  current_period_end: string;
  payment_method: PaymentMethod | null;
  invoices: Invoice[];
}

export interface InvitationPreview {
  email: string;
  role: Exclude<Role, "super_admin">;
  company_name: string;
  inviter_name: string;
  expired: boolean;
  accepted: boolean;
}
