import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

export interface BratraxUser {
  id: number;
  email: string;
  name: string;
  role: "admin" | "viewer";
  project_id: string | null;
}

function getBaseUrl(): string {
  return get(runtime).host;
}

export async function bratraxLogin(
  email: string,
  password: string,
): Promise<{ user: BratraxUser }> {
  const res = await fetch(`${getBaseUrl()}/bratrax/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Login failed (${res.status})`);
  }
  return res.json();
}

export async function bratraxSignup(
  email: string,
  password: string,
  companyName: string,
): Promise<{ user: BratraxUser }> {
  const res = await fetch(`${getBaseUrl()}/bratrax/auth/signup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password, company_name: companyName }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Signup failed (${res.status})`);
  }
  return res.json();
}

export async function bratraxLogout(): Promise<void> {
  await fetch(`${getBaseUrl()}/bratrax/auth/logout`, {
    method: "POST",
    credentials: "include",
  });
}

export async function bratraxGetMe(
  fetchFn: typeof fetch = fetch,
): Promise<BratraxUser | null> {
  const res = await fetchFn(`${getBaseUrl()}/bratrax/auth/me`, {
    credentials: "include",
  });
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`Auth check failed (${res.status})`);
  const data: { user: BratraxUser } = await res.json();
  return data.user;
}
