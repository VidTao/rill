import { redirect } from "@sveltejs/kit";
import { bratraxGetMe } from "$lib/bratrax/auth";
import { safeRedirect } from "$lib/bratrax/safe-redirect";

export async function load({ url, fetch }) {
  // If the user is already authenticated, don't show the login form again —
  // bounce them to wherever they were headed (?redirect=...) or to /developer,
  // which then handles per-role landing (super_admin stays on the welcome,
  // admin/viewer redirect to first dashboard) via its own onMount logic.
  const user = await bratraxGetMe(fetch);
  if (user) {
    // Same-site paths only: never forward a signed-in visitor off the site.
    throw redirect(307, safeRedirect(url.searchParams.get("redirect")));
  }
  return {};
}
