import { error } from "@sveltejs/kit";

// Reject any URL segment that isn't a real settings tab — prevents stray
// strings from rendering the page with `activeTab` falling back to "account".
// Keep this set in sync with TABS in [tab]/+page.svelte.
const VALID_TABS = new Set(["account", "team", "billing", "ai", "mcp"]);

export const load = ({ params }) => {
  if (!VALID_TABS.has(params.tab)) {
    throw error(404, `Unknown settings tab: ${params.tab}`);
  }
  return { tab: params.tab };
};
