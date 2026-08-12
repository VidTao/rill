import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
import { startRuntimeSessionSync } from "$lib/bratrax/shopify-app-bridge";

export const ssr = false;

export const load = async () => {
  // Authenticate the Rill runtime client BEFORE the page renders.
  //
  // This page's verification loop calls runtimeServiceListResources() to decide
  // when the client's dashboards have finished reconciling, and then picks the
  // redirect target from that list. Those calls go through the runtime client's
  // own transport, which reads its token from `runtime.jwt` — NOT through
  // apiFetch, and therefore NOT through installEmbeddedAuthFetch.
  //
  // Inside the Shopify admin iframe there is no bratrax_auth cookie, so without
  // this the queries went out unauthenticated, InstanceRouterMiddleware resolved
  // no client, and they fell through to the empty "default" instance — which
  // contains Rill's bundled demo project (margin_scorecard,
  // metrics_margin_explore). Two things broke as a result:
  //
  //   1. The redirect landed on /files/dashboards/<demo>.yaml — the file editor
  //      showing a Rill example — because no non-demo Canvas was found.
  //   2. "Dashboards ready" was a false green: the loop verified the demo
  //      project's resources, never the client's.
  //
  // In `load` rather than onMount so SvelteKit guarantees it resolves before
  // the component's first query. No-ops when not embedded, so every existing
  // user is unaffected.
  const { host, instanceId } = get(runtime);
  await startRuntimeSessionSync(queryClient, host, instanceId);
};
