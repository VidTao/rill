import { redirect } from "@sveltejs/kit";

// Bare /settings → default to the Account tab. The real tab pages live under
// /settings/[tab]/ and validate the segment against the known TabId set.
export const load = () => {
  throw redirect(307, "/settings/account");
};
