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

export interface BillingSummary {
  plan: string;
  price_cents: number;
  currency: string;
  interval: string;
  status: string;
  current_period_end: string | null;
}

export interface InvitationPreview {
  email: string;
  role: Role;
  // kind distinguishes the flavors of invitation:
  //   team       — joining an existing client as admin/viewer
  //   superadmin — cross-client super_admin (client_id=NULL)
  //   signup     — pre-authorized public signup; creates a brand-new client
  //   demo       — self-serve "Try demo"; viewer on the Bratrax Demo Account
  kind: "team" | "superadmin" | "signup" | "demo";
  company_name: string;
  inviter_name: string;
  expired: boolean;
  accepted: boolean;
}

export interface AISettings {
  key_set: boolean;
  key_preview: string | null;
}

export interface MCPSettings {
  token_set: boolean;
  token: string | null;
  mcp_url: string;
  created_at: string | null;
  claude_desktop_config: Record<string, unknown> | null;
}

export interface SlackWorkspace {
  team_id: string;
  team_name: string | null;
  installed_at: string | null;
  installed_by_email: string | null;
}

export interface SlackSettings {
  workspaces: SlackWorkspace[];
  // false when the server has no Slack app credentials configured
  configured: boolean;
}

export interface SlackInstallLink {
  install_url: string;
  expires_in_minutes: number;
}
