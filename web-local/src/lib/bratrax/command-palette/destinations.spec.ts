import { existsSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { pagesFor } from "$lib/help";
import {
  destinationsFor,
  matchDestinations,
  matchHelpPages,
  type PaletteContext,
} from "./destinations";

const ADMIN: PaletteContext = {
  role: "admin",
  onboarded: true,
  shopifyEmbedded: false,
};
const SUPER: PaletteContext = { ...ADMIN, role: "super_admin" };
const VIEWER: PaletteContext = { ...ADMIN, role: "viewer" };

const ALL_CONTEXTS: PaletteContext[] = (
  ["viewer", "admin", "super_admin", null] as const
).flatMap((role) =>
  [true, false].flatMap((onboarded) =>
    [true, false].map((shopifyEmbedded) => ({
      role,
      onboarded,
      shopifyEmbedded,
    })),
  ),
);

const SUPER_ONLY = [
  "/clients",
  "/superadmins",
  "/superadmins/email-log",
  "/superadmins/client-stats",
];

function hrefs(ctx: PaletteContext): string[] {
  return destinationsFor(ctx).map((d) => d.href);
}

describe("destinationsFor — mirrors the root-layout nav", () => {
  it("viewers get only Getting started (no Settings, like their nav)", () => {
    expect(hrefs(VIEWER)).toEqual(["/onboarding"]);
  });

  it("admins get every settings tab, config page and workspace page", () => {
    expect(hrefs(ADMIN)).toEqual([
      "/onboarding",
      "/connectors",
      "/cost-settings",
      "/settings/account",
      "/settings/team",
      "/settings/billing",
      "/settings/ai",
      "/settings/mcp",
      "/settings/slack",
      "/metric-trees",
      "/customize",
    ]);
  });

  it("super admins additionally get the super-admin pages", () => {
    expect(hrefs(SUPER)).toEqual([...hrefs(ADMIN), ...SUPER_ONLY]);
  });

  it("super-admin pages never leak to any other role or state", () => {
    for (const ctx of ALL_CONTEXTS) {
      if (ctx.role === "super_admin") continue;
      for (const href of SUPER_ONLY) expect(hrefs(ctx)).not.toContain(href);
    }
  });

  it("mid-onboarding admins get only /settings/*, the one place not bounced", () => {
    const mid = hrefs({ ...ADMIN, onboarded: false });
    expect(mid.length).toBe(6);
    for (const href of mid) expect(href.startsWith("/settings/")).toBe(true);
  });

  it("super admins ignore onboarding state, like the nav", () => {
    expect(hrefs({ ...SUPER, onboarded: false })).toEqual(hrefs(SUPER));
  });

  it("the Shopify iframe drops the Settings dropdown's destinations", () => {
    const embedded = hrefs({ ...ADMIN, shopifyEmbedded: true });
    expect(embedded).toEqual(["/metric-trees", "/customize"]);
    expect(
      hrefs({ ...ADMIN, onboarded: false, shopifyEmbedded: true }),
    ).toEqual([]);
  });

  it("no role, nothing", () => {
    expect(hrefs({ ...ADMIN, role: null })).toEqual([]);
  });

  it("ids are unique (cmdk selects by value)", () => {
    const ids = destinationsFor(SUPER).map((d) => d.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});

describe("every destination is a real route", () => {
  const routes = fileURLToPath(new URL("../../../routes", import.meta.url));
  const settingsTabs = readFileSync(
    `${routes}/settings/[tab]/+page.ts`,
    "utf8",
  );

  for (const d of destinationsFor(SUPER)) {
    it(d.href, () => {
      const tab = d.href.match(/^\/settings\/([a-z]+)$/)?.[1];
      if (tab) {
        // One dynamic route; its load() 404s any tab not in VALID_TABS.
        expect(existsSync(`${routes}/settings/[tab]/+page.svelte`)).toBe(true);
        expect(settingsTabs).toContain(`"${tab}"`);
      } else {
        expect(existsSync(`${routes}${d.href}/+page.svelte`)).toBe(true);
      }
    });
  }
});

describe("matchDestinations", () => {
  const admin = destinationsFor(ADMIN);

  it('"billing" lands on the billing settings tab first', () => {
    expect(matchDestinations("billing", admin)[0].href).toBe(
      "/settings/billing",
    );
  });

  it("keywords find pages whose label doesn't say it", () => {
    expect(matchDestinations("invoice", admin)[0].href).toBe(
      "/settings/billing",
    );
    expect(matchDestinations("cogs", admin)[0].href).toBe("/cost-settings");
    expect(matchDestinations("invite", admin)[0].href).toBe("/settings/team");
    expect(matchDestinations("klaviyo", admin)[0].href).toBe("/connectors");
  });

  it("label matches outrank keyword-only matches", () => {
    // "claude" is a keyword on AI and MCP; "AI settings" wins "ai" on label.
    expect(matchDestinations("ai", admin)[0].href).toBe("/settings/ai");
  });

  it("matches word prefixes, not arbitrary substrings", () => {
    // "ai" must not drag in anything via "email" / "domain".
    for (const d of matchDestinations("ai", admin)) {
      expect(d.href).toMatch(/\/settings\/(ai|mcp)$/);
    }
  });

  it("every token must match", () => {
    expect(matchDestinations("team invite", admin).map((d) => d.href)).toEqual([
      "/settings/team",
    ]);
    expect(matchDestinations("team billing", admin)).toEqual([]);
  });

  it("is case- and whitespace-insensitive", () => {
    expect(matchDestinations("  BILL  ", admin)[0].href).toBe(
      "/settings/billing",
    );
  });

  it("an empty query returns the list unchanged", () => {
    expect(matchDestinations("", admin)).toEqual(admin);
    expect(matchDestinations("   ", admin)).toEqual(admin);
  });
});

describe("matchHelpPages — over pagesFor(role), titles only", () => {
  it("a topic query finds its article", () => {
    const pages = matchHelpPages("attribution", pagesFor("viewer"));
    expect(pages.map((p) => p.slug)).toContain("viewer/attribution");
  });

  it("admin-audience articles never reach a viewer", () => {
    const hits = matchHelpPages("billing", pagesFor("viewer"));
    expect(hits).toEqual([]);
    for (const p of matchHelpPages("", pagesFor("viewer"))) {
      expect(p.audience).not.toBe("admin");
    }
  });

  it("admins do get admin articles", () => {
    const hits = matchHelpPages("billing", pagesFor("admin"));
    expect(hits.map((p) => p.title)).toEqual(["Billing & subscription"]);
  });

  it("does not match on article bodies (yet)", () => {
    // "cohort" appears in bodies but in no title.
    const bodyOnly = pagesFor("admin").filter(
      (p) =>
        !p.title.toLowerCase().includes("cohort") &&
        p.body.toLowerCase().includes("cohort"),
    );
    expect(bodyOnly.length).toBeGreaterThan(0);
    expect(matchHelpPages("cohort", pagesFor("admin"))).toEqual([]);
  });
});
