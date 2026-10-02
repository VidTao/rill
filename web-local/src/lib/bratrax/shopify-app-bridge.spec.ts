// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from "vitest";

// Inside the Shopify admin iframe, a merchant who signed in carries a bratrax
// JWT that names them but not the store. The interceptor adds the shop's
// session token beside it so the Go proxy can pick the shop's store
// (clientFromShopifySessionHeader in rill/bratrax/auth.go). Without it a
// multi-store merchant in their second shop's admin saw their first store.

const embedded = { jwt: null as string | null };

vi.mock("./shopify-embed", () => ({
  isShopifyEmbedded: () => true,
  getEmbeddedToken: () => embedded.jwt,
}));

import { installEmbeddedAuthFetch } from "./shopify-app-bridge";

const sent: { url: string; headers: Headers }[] = [];

window.shopify = { idToken: async () => "shop-session" };
window.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
  sent.push({ url: String(input), headers: new Headers(init?.headers) });
  return new Response("{}");
}) as typeof fetch;
installEmbeddedAuthFetch();

const runtimeUrl = `${window.location.origin}/v1/instances/default/resources`;

function last() {
  return sent[sent.length - 1];
}

beforeEach(() => {
  sent.length = 0;
  embedded.jwt = null;
});

describe("embedded fetch interceptor, signed in with a bratrax JWT", () => {
  beforeEach(() => {
    embedded.jwt = "bratrax-jwt";
  });

  it("sends the JWT and names the shop on /bratrax/ calls", async () => {
    await fetch("/bratrax/settings/account");
    expect(last().headers.get("Authorization")).toBe("Bearer bratrax-jwt");
    expect(last().headers.get("X-Shopify-Session-Token")).toBe("shop-session");
  });

  it("names the shop on runtime calls without touching their Authorization", async () => {
    await fetch(runtimeUrl, { headers: { Authorization: "Bearer runtime-jwt" } });
    expect(last().headers.get("Authorization")).toBe("Bearer runtime-jwt");
    expect(last().headers.get("X-Shopify-Session-Token")).toBe("shop-session");
  });

  it("keeps an Authorization header the caller set and still names the shop", async () => {
    await fetch("/bratrax/multi-client/add-store", {
      method: "POST",
      headers: { Authorization: "Bearer explicit", "Content-Type": "application/json" },
      body: "{}",
    });
    expect(last().headers.get("Authorization")).toBe("Bearer explicit");
    expect(last().headers.get("Content-Type")).toBe("application/json");
    expect(last().headers.get("X-Shopify-Session-Token")).toBe("shop-session");
  });
});

describe("embedded fetch interceptor, session token as the bearer", () => {
  it("authenticates /bratrax/ calls with the session token and adds no shop header", async () => {
    await fetch("/bratrax/settings/account");
    expect(last().headers.get("Authorization")).toBe("Bearer shop-session");
    expect(last().headers.has("X-Shopify-Session-Token")).toBe(false);
  });

  it("leaves runtime calls exactly as sent", async () => {
    await fetch(runtimeUrl, { headers: { Authorization: "Bearer runtime" } });
    expect(last().headers.get("Authorization")).toBe("Bearer runtime");
    expect(last().headers.has("X-Shopify-Session-Token")).toBe(false);
  });
});

describe("embedded fetch interceptor, other requests", () => {
  it("never touches other paths or origins", async () => {
    embedded.jwt = "bratrax-jwt";
    await fetch("/api/something");
    expect(last().headers.has("Authorization")).toBe(false);
    expect(last().headers.has("X-Shopify-Session-Token")).toBe(false);

    await fetch("https://cdn.example.com/v1/thing");
    expect(last().headers.has("X-Shopify-Session-Token")).toBe(false);
  });
});
