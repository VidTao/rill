import { redirect } from "@sveltejs/kit";

export const ssr = false;

// Workshop is parked until we ship the per-client editor refresh (E19, 2026-05-04).
// Until then, every visit to /workshop or /workshop/* is bounced to the developer
// surface so users can't reach the page by typing the URL or following a stale link.
//
// To restore Workshop access: delete this load() function. The role-gating in
// routes/+layout.svelte (currently disabled tab + commented {#if isSuper} block)
// also needs to be reverted in the same change.
export function load() {
  throw redirect(307, "/");
}
