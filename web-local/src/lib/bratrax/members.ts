import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { apiFetch } from "./onboarding/api";

/**
 * Directory member accounts: the emailed sign-in link and "Start attribution".
 *
 * A demo visitor or directory member is a viewer on the shared demo workspace
 * with no password they know. They sign in by emailed link, and "Start
 * attribution" turns the same account into a customer (admin, no workspace
 * yet); onboarding then creates their workspace.
 */

/**
 * Ask for a one-time sign-in link (public; existing accounts only). The answer
 * is the same whether or not the address has an account, so this never tells
 * anyone which emails are registered.
 */
export async function requestSigninLink(email: string, next = "/"): Promise<void> {
  const res = await fetch(`${get(runtime).host}/bratrax/signin-link`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email: email.trim(), next }),
  });
  if (!res.ok && res.status !== 429) {
    throw new Error("Couldn't send the email just now. Try again in a minute.");
  }
}

/**
 * Convert the signed-in demo-workspace viewer into a customer, in place (same
 * login). The caller then runs onboardStart to create the workspace.
 */
export async function startAttribution(companyName: string): Promise<void> {
  await apiFetch<{ ok: boolean }>("/bratrax/members/start-attribution", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ company_name: companyName }),
  });
}

/** Opens the shared Start attribution modal (mounted once, in the root layout). */
export const START_ATTRIBUTION_EVENT = "bratrax:start-attribution";

export function openStartAttribution(): void {
  window.dispatchEvent(new CustomEvent(START_ATTRIBUTION_EVENT));
}
