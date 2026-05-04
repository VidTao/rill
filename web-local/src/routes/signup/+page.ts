// Signup page doesn't need server-side data loading
export const ssr = false;

import { redirect } from "@sveltejs/kit";
import { bratraxGetMe } from "$lib/bratrax/auth";

export async function load({ url, fetch }) {
  // If the user is already authenticated, don't show the signup form —
  // bounce them to wherever they were headed (?redirect=...) or to /developer,
  // which then handles per-role landing (super_admin stays on the welcome,
  // admin/viewer redirect to first dashboard) via its own onMount logic.
  const user = await bratraxGetMe(fetch);
  if (user) {
    const target = url.searchParams.get("redirect") || "/developer";
    throw redirect(307, target);
  }
  return {};
}
