// Signup page doesn't need server-side data loading
export const ssr = false;

import { redirect } from "@sveltejs/kit";
import { bratraxGetMe, bratraxLogout } from "$lib/bratrax/auth";

export async function load({ url, fetch }) {
  // If the user is already authenticated, don't show the signup form —
  // bounce them to wherever they were headed (?redirect=...) or to /developer,
  // which then handles per-role landing (super_admin stays on the welcome,
  // admin/viewer redirect to first dashboard) via its own onMount logic.
  const user = await bratraxGetMe(fetch);
  if (user) {
    // ?logout=1 opts out of that bounce by ending the session first. It exists
    // for demo users: the demo tour's closing call to action is to start a real
    // workspace, and every route that could show them a signup form is closed
    // to an authenticated visitor — this bounce, and the apex handler in
    // +layout.ts, which sends a logged-in visitor from / to their dashboard.
    // Without this they had to know to sign out by hand.
    if (url.searchParams.get("logout")) {
      await bratraxLogout();
      // Reload rather than returning {} and rendering: +layout.ts short-circuits
      // /signup as a public route without touching `bratraxUser`, so an SPA
      // navigation would leave the old identity in the store while the form is
      // on screen. The cookie is gone by now, so the reload lands on the form
      // with `user` null. `replace` keeps this URL out of history, so Back
      // cannot re-trigger a logout.
      window.location.replace("/signup");
      return {};
    }
    const target = url.searchParams.get("redirect") || "/developer";
    throw redirect(307, target);
  }
  return {};
}
