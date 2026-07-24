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

export type HelpSegment =
  | { kind: "markdown"; value: string }
  | { kind: "loom"; id: string };

// Splits a help body into markdown runs and Loom video embeds. A video is
// authored as a fenced ```loom block whose only content is the Loom video id:
//
//   ```loom
//   58f3efa9a75e483892ebe484a6c59afa
//   ```
//
// The shared Markdown component sanitizes out iframes (it also renders AI chat
// output), so the help page renders loom segments with the LoomEmbed component
// instead of loosening that sanitizer.
export function splitHelpBody(body: string): HelpSegment[] {
  const re = /```loom\s*\n([\s\S]*?)```/g;
  const segments: HelpSegment[] = [];
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(body)) !== null) {
    if (m.index > last) {
      const md = body.slice(last, m.index).trim();
      if (md) segments.push({ kind: "markdown", value: md });
    }
    const id = m[1].trim();
    if (id) segments.push({ kind: "loom", id });
    last = re.lastIndex;
  }
  const tail = body.slice(last).trim();
  if (tail) segments.push({ kind: "markdown", value: tail });
  return segments;
}

// Sentinels wrap matched terms inside snippet strings so the rendering layer
// can split on them and emit <mark> without ever calling {@html}.
export const HELP_SNIPPET_MARK_START = "";
export const HELP_SNIPPET_MARK_END = "";

export interface HelpSearchResult {
  page: HelpPage;
  snippet: string;
  titleMatch: boolean;
  score: number;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function buildSnippet(body: string, terms: string[]): string {
  const lower = body.toLowerCase();

  let firstMatch = -1;
  for (const term of terms) {
    const idx = lower.indexOf(term);
    if (idx !== -1 && (firstMatch === -1 || idx < firstMatch)) firstMatch = idx;
  }

  let excerpt: string;
  if (firstMatch === -1) {
    excerpt = body.slice(0, 200);
    if (body.length > 200) excerpt += "…";
  } else {
    const start = Math.max(0, firstMatch - 80);
    const end = Math.min(body.length, firstMatch + 120);
    excerpt =
      (start > 0 ? "…" : "") +
      body.slice(start, end) +
      (end < body.length ? "…" : "");
  }

  excerpt = excerpt
    .replace(/^[#>\s]+/gm, "")
    .replace(/[*_`]/g, "")
    .replace(/\|/g, " ")
    .replace(/\s+/g, " ")
    .trim();

  if (terms.length > 0) {
    const pattern = new RegExp(`(${terms.map(escapeRegExp).join("|")})`, "gi");
    excerpt = excerpt.replace(
      pattern,
      `${HELP_SNIPPET_MARK_START}$1${HELP_SNIPPET_MARK_END}`,
    );
  }

  return excerpt;
}

export function searchHelp(query: string, pages: HelpPage[]): HelpSearchResult[] {
  const trimmed = query.trim().toLowerCase();
  if (!trimmed) return [];

  const terms = trimmed.split(/\s+/).filter((t) => t.length > 0);
  if (terms.length === 0) return [];

  const results: HelpSearchResult[] = [];

  for (const page of pages) {
    const lowerTitle = page.title.toLowerCase();
    const lowerBody = page.body.toLowerCase();

    const titleMatch = terms.some((t) => lowerTitle.includes(t));

    let bodyMatchCount = 0;
    for (const term of terms) {
      let idx = 0;
      while ((idx = lowerBody.indexOf(term, idx)) !== -1) {
        bodyMatchCount++;
        idx += term.length;
      }
    }

    if (!titleMatch && bodyMatchCount === 0) continue;

    results.push({
      page,
      snippet: buildSnippet(page.body, terms),
      titleMatch,
      score: (titleMatch ? 1000 : 0) + bodyMatchCount,
    });
  }

  results.sort((a, b) => {
    if (b.score !== a.score) return b.score - a.score;
    return a.page.order - b.page.order;
  });

  return results;
}
