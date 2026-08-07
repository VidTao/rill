import { describe, expect, it } from "vitest";
import { canAccess, HELP_PAGES, pagesFor, splitHelpBody } from "./index";

describe("splitHelpBody loom fences", () => {
  it("reads a bare video id, the form most articles still use", () => {
    const segments = splitHelpBody("intro\n\n```loom\nabc123\n```\n\nafter");
    expect(segments).toEqual([
      { kind: "markdown", value: "intro" },
      { kind: "loom", id: "abc123" },
      { kind: "markdown", value: "after" },
    ]);
  });

  it("reads a labelled fence, so the title can be the disclosure control", () => {
    const segments = splitHelpBody(
      "```loom\nid: abc123\nlabel: Store Performance\nduration: 1:46\n```",
    );
    expect(segments).toEqual([
      {
        kind: "loom",
        id: "abc123",
        label: "Store Performance",
        duration: "1:46",
      },
    ]);
  });

  it("keeps a label containing a colon intact", () => {
    const [segment] = splitHelpBody(
      "```loom\nid: abc123\nlabel: Attribution: last-touch\n```",
    );
    expect(segment).toEqual({
      kind: "loom",
      id: "abc123",
      label: "Attribution: last-touch",
      duration: undefined,
    });
  });

  it("drops a fence with no id rather than rendering a broken player", () => {
    expect(splitHelpBody("```loom\n\n```")).toEqual([]);
    expect(splitHelpBody("```loom\nlabel: orphan\n```")).toEqual([]);
  });

  it("keeps several fences and the prose between them in order", () => {
    const segments = splitHelpBody(
      "a\n\n```loom\none\n```\n\nb\n\n```loom\nid: two\nlabel: Two\n```",
    );
    expect(segments.map((s) => s.kind)).toEqual([
      "markdown",
      "loom",
      "markdown",
      "loom",
    ]);
    expect(segments[3]).toMatchObject({ id: "two", label: "Two" });
  });
});

describe("sidebar visibility", () => {
  const ROLES = ["viewer", "admin", "super_admin", null] as const;

  it("never lists a page the role cannot open", () => {
    // pagesFor and canAccess used to encode the audience rules separately and
    // had drifted, so an unrecognised audience produced a sidebar link that
    // redirected back to /help. This is the invariant that catches that.
    for (const role of ROLES) {
      for (const isDemo of [false, true]) {
        const unreachable = pagesFor(role, isDemo).filter(
          (p) => !canAccess(p, role, isDemo),
        );
        expect(unreachable.map((p) => p.slug), `${role}/${isDemo}`).toEqual([]);
      }
    }
  });

  it("shows the demo tour on the demo workspace and nowhere else", () => {
    // bratrax_welcome_demo's primary CTA is /help/demo-tour, so this page must
    // open for a demo user — who is a viewer on the shared demo workspace.
    // See docs/BRATRAX_DEMO_INVITE_FLOW.md.
    const tour = HELP_PAGES.find((p) => p.slug === "demo-tour");
    expect(tour).toMatchObject({ audience: "demo", group: "start-here", order: 0 });
    expect(canAccess(tour!, "viewer", true)).toBe(true);
    expect(pagesFor("viewer", true).map((p) => p.slug)).toContain("demo-tour");

    // A paying customer's workspace must not see it — including its admins and
    // a super_admin, who otherwise see every page.
    for (const role of ROLES) {
      expect(canAccess(tour!, role, false), `${role}`).toBe(false);
      expect(pagesFor(role, false).map((p) => p.slug), `${role}`).not.toContain(
        "demo-tour",
      );
    }
  });

  it("keeps hide_for_demo pages out of a demo sidebar but still openable", () => {
    // Start here otherwise offers a demo user three competing starting points.
    // The page stays accessible because the demo tour links to it for the admin
    // side of the product — hiding is a listing rule, not an access rule.
    const first = HELP_PAGES.find((p) => p.slug === "start-here");
    expect(first?.hideForDemo).toBe(true);
    expect(pagesFor("viewer", true).map((p) => p.slug)).not.toContain("start-here");
    expect(canAccess(first!, "viewer", true)).toBe(true);

    // Everyone else still sees it.
    expect(pagesFor("viewer", false).map((p) => p.slug)).toContain("start-here");
    expect(pagesFor("admin", false).map((p) => p.slug)).toContain("start-here");
  });

  it("leaves Welcome to Bratrax in a demo sidebar", () => {
    // Deliberate: demo users keep the tour plus one general orientation page.
    expect(pagesFor("viewer", true).map((p) => p.slug)).toContain("viewer/welcome");
  });

  it("still shows admin pages to admins and hides them from viewers", () => {
    expect(pagesFor("admin").some((p) => p.audience === "admin")).toBe(true);
    expect(pagesFor("viewer").some((p) => p.audience === "admin")).toBe(false);
  });
});

describe("sidebar active state", () => {
  // The sidebar marks a row active with `p.slug === currentSlug`, where
  // currentSlug is the pathname minus the /help prefix (see routes/help/+layout).
  const currentSlugFor = (pathname: string) => pathname.replace(/^\/help\/?/, "");

  it("matches exactly one page for every page's own URL", () => {
    for (const page of HELP_PAGES) {
      const matches = HELP_PAGES.filter(
        (p) => p.slug === currentSlugFor(`/help/${page.slug}`),
      );
      expect(matches.map((p) => p.slug), `for /help/${page.slug}`).toEqual([page.slug]);
    }
  });

  it("gives every page a distinct slug, so two rows can never light up at once", () => {
    const slugs = HELP_PAGES.map((p) => p.slug);
    expect(slugs.length).toBe(new Set(slugs).size);
  });

  it("treats the bare /help index as no page selected", () => {
    expect(currentSlugFor("/help")).toBe("");
    expect(currentSlugFor("/help/")).toBe("");
    expect(HELP_PAGES.some((p) => p.slug === "")).toBe(false);
  });
});

describe("help landing page links", () => {
  // The landing page hardcodes its card hrefs in +page.svelte, so
  // check_links.py (which reads markdown) cannot see them. One card pointed at
  // /help/viewer/new-customer-source-report, which has no live article.
  const LANDING_HREFS = [
    "viewer/store-performance",
    "viewer/attribution",
    "viewer/products",
    "viewer/customer-analytics",
    "viewer/email-sms",
    "viewer/welcome",
    "admin/connecting-platforms",
    "glossary",
  ];

  it("points every card at a page that exists", () => {
    const slugs = new Set(HELP_PAGES.map((p) => p.slug));
    const missing = LANDING_HREFS.filter((s) => !slugs.has(s));
    expect(missing).toEqual([]);
  });

  it("keeps every viewer-facing card readable by a viewer", () => {
    const viewerSlugs = new Set(pagesFor("viewer").map((p) => p.slug));
    const adminOnly = LANDING_HREFS.filter((s) => s.startsWith("admin/"));
    for (const slug of LANDING_HREFS.filter((s) => !adminOnly.includes(s))) {
      expect(viewerSlugs.has(slug), slug).toBe(true);
    }
  });
});
