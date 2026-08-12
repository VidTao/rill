import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { installEmbeddedAuthFetch } from "$lib/bratrax/shopify-app-bridge";

/** INITIALIZE RUNTIME STORE **/
// Always use relative paths — Vite proxy handles dev, same-origin handles prod
const HOST = "";
const INSTANCE_ID = "default";

const runtimeInit = {
  host: HOST,
  instanceId: INSTANCE_ID,
};

runtime.set(runtimeInit);

// Inside the Shopify admin iframe, bratrax_auth is a third-party cookie and is
// never sent, so every /bratrax/* call would arrive unauthenticated. This
// attaches the embedded credential to those requests at one choke point rather
// than in each of the eleven fetch helpers. No-ops entirely when not embedded.
installEmbeddedAuthFetch();
