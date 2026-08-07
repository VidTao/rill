import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
import { startRuntimeSessionSync } from "$lib/bratrax/shopify-app-bridge";

export const load = async ({ params }) => {
  const canvasName = params.name;

  // Authenticate the runtime client BEFORE the page renders.
  //
  // The dashboard's data does not come through apiFetch — the Rill runtime
  // client has its own transport and reads its token from `runtime.jwt`, which
  // web-local normally leaves unset because the cookie suffices. Inside the
  // Shopify iframe there is no cookie, so a query firing before the token is in
  // place would be unauthenticated, resolve no client, and read the empty
  // "default" instance. Doing it in `load` rather than onMount is what
  // guarantees the ordering: SvelteKit resolves this before the component runs.
  //
  // No-ops when not embedded, so /canvas/[name] and every existing user are
  // unaffected.
  const { host, instanceId } = get(runtime);
  await startRuntimeSessionSync(queryClient, host, instanceId);

  return { canvasName };
};
