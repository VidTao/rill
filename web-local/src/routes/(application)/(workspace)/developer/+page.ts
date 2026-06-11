import { redirect } from "@sveltejs/kit";
import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { runtimeServiceListResources } from "@rilldata/web-common/runtime-client";
import { isRillDemoCanvas } from "$lib/bratrax/dashboardPrefs";

// Load-time redirect: every role lands on the first Canvas dashboard in
// preview mode instead of seeing the file-editor/welcome workspace. This
// fires BEFORE any UI renders, so the user never sees a /developer flash on
// the way to /canvas/<name>. Falls through to +page.svelte when no canvases
// exist (renders the "No dashboards yet" placeholder).
export const load = async () => {
  const instanceId = get(runtime).instanceId;
  if (!instanceId) return {};

  try {
    const response = await runtimeServiceListResources(instanceId, {
      kind: "rill.runtime.v1.Canvas",
    });
    const resources = response.resources ?? [];
    for (const r of resources) {
      const name = r.meta?.name?.name;
      // Skip Rill bundled-example canvases (margin_scorecard etc.) — they
      // can leak in during instance warm-up; see RILL_DEMO_CANVAS_NAMES in
      // dashboardPrefs.ts.
      if (name && !isRillDemoCanvas(name)) {
        throw redirect(307, `/canvas/${name}`);
      }
    }
  } catch (e) {
    // Rethrow SvelteKit redirects; swallow runtime errors so the page can
    // still render its fallback UI.
    if (e && typeof e === "object" && "status" in e && "location" in e) {
      throw e;
    }
  }

  return {};
};
