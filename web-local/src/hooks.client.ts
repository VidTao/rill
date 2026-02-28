import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

/** INITIALIZE RUNTIME STORE **/
// Always use relative paths — Vite proxy handles dev, same-origin handles prod
const HOST = "";
const INSTANCE_ID = "default";

const runtimeInit = {
  host: HOST,
  instanceId: INSTANCE_ID,
};

runtime.set(runtimeInit);
