// Help content loader.
//
// Stub markdown lives in `./content/{viewer,admin,_shared}/*.md`. Each file has
// flat YAML-ish frontmatter (title, audience, order, status). We load every
// file at build time via Vite's import.meta.glob and parse the frontmatter with
// a tiny inline scanner — the schema is simple enough that a real YAML parser
// would be overkill.

export type Audience = "viewer" | "admin" | "shared";
export type Status = "stub" | "draft" | "ready";

export interface HelpFrontmatter {
  title: string;
  audience: Audience;
  order: number;
  status: Status;
}

export interface HelpPage extends HelpFrontmatter {
  /** Slug relative to /help, e.g. "viewer/reading-dashboards" */
  slug: string;
  /** Body markdown with frontmatter stripped */
  body: string;
}

const RAW_MODULES = import.meta.glob("./content/**/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

function parseFrontmatter(raw: string): { fm: Partial<HelpFrontmatter>; body: string } {
  const match = raw.match(/^---\s*\n([\s\S]*?)\n---\s*\n?([\s\S]*)$/);
  if (!match) return { fm: {}, body: raw };
  const [, header, body] = match;
  const fm: Record<string, string | number> = {};
  for (const line of header.split("\n")) {
    const m = line.match(/^([a-zA-Z_][\w-]*)\s*:\s*(.*?)(?:\s*#.*)?$/);
    if (!m) continue;
    const [, key, valRaw] = m;
    const val = valRaw.replace(/^["']|["']$/g, "");
    fm[key] = key === "order" ? Number(val) : val;
  }
  return { fm: fm as Partial<HelpFrontmatter>, body };
}

function pathToSlug(path: string): string {
  // ./content/viewer/02-reading-dashboards.md → viewer/reading-dashboards
  return path
    .replace(/^\.\/content\//, "")
    .replace(/^_shared\//, "")
    .replace(/\.md$/, "")
    .replace(/\/_/, "/")
    .replace(/(^|\/)\d+-/, "$1");
}

export const HELP_PAGES: HelpPage[] = Object.entries(RAW_MODULES)
  .map(([path, raw]) => {
    const { fm, body } = parseFrontmatter(raw);
    return {
      slug: pathToSlug(path),
      title: fm.title ?? "Untitled",
      audience: (fm.audience ?? "shared") as Audience,
      order: fm.order ?? 999,
      status: (fm.status ?? "stub") as Status,
      body,
    };
  })
  .sort((a, b) => {
    if (a.audience !== b.audience) return a.audience.localeCompare(b.audience);
    return a.order - b.order;
  });

const BY_SLUG = new Map(HELP_PAGES.map((p) => [p.slug, p]));

export function getHelpPage(slug: string): HelpPage | null {
  return BY_SLUG.get(slug) ?? null;
}

/**
 * Pages an audience can see. Admins see admin + shared (and viewer too — admin
 * docs reference viewer concepts). Viewers see viewer + shared only.
 */
export function pagesFor(role: "viewer" | "admin" | "super_admin" | null): HelpPage[] {
  if (role === "viewer") {
    return HELP_PAGES.filter((p) => p.audience === "viewer" || p.audience === "shared");
  }
  // admin + super_admin see everything
  return HELP_PAGES;
}

export function canAccess(
  page: HelpPage,
  role: "viewer" | "admin" | "super_admin" | null,
): boolean {
  if (page.audience === "shared") return true;
  if (page.audience === "viewer") return true; // viewer pages are universal
  if (page.audience === "admin") return role === "admin" || role === "super_admin";
  return false;
}
