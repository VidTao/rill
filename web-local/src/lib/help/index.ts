// Help content loader.
//
// Stub markdown lives in `./content/{viewer,admin,_shared}/*.md`. Each file has
// flat YAML-ish frontmatter (title, audience, order, status). We load every
// file at build time via Vite's import.meta.glob and parse the frontmatter with
// a tiny inline scanner — the schema is simple enough that a real YAML parser
// would be overkill.

// "demo" is not a role — it's the shared /try-demo workspace. Pages marked
// with it are visible ONLY to users sitting on that workspace, super_admins
// included, so demo-only copy never shows up in a paying customer's sidebar.
export type Audience = "viewer" | "admin" | "shared" | "demo";
export type Status = "stub" | "draft" | "ready";

/**
 * Sidebar grouping, deliberately decoupled from `audience`.
 *
 * `audience` answers "who is allowed to read this"; `group` answers "where does
 * it sit in the nav". Keeping them separate is what lets the guided tour
 * (audience: shared, so it keeps its /help/start-here slug — see
 * GETTING_STARTED_URL in lib/bratrax/constants.ts) open the "Start here" group
 * instead of being exiled to Reference.
 */
export type HelpGroup =
  | "start-here"
  | "dashboards"
  | "going-deeper"
  | "workspace"
  | "building"
  | "reference";

/** Render order + sidebar headings. */
export const HELP_GROUPS: { id: HelpGroup; label: string }[] = [
  { id: "start-here", label: "Start here" },
  { id: "dashboards", label: "Your dashboards" },
  { id: "going-deeper", label: "Going deeper" },
  { id: "workspace", label: "Your workspace" },
  { id: "building", label: "Building dashboards" },
  { id: "reference", label: "Reference" },
];

const GROUP_IDS = new Set<string>(HELP_GROUPS.map((g) => g.id));

/** Fallback for an entry with no (or an unknown) `group`, by audience. */
function defaultGroup(audience: Audience): HelpGroup {
  if (audience === "admin") return "workspace";
  if (audience === "shared") return "reference";
  if (audience === "demo") return "start-here";
  return "going-deeper";
}

export interface HelpFrontmatter {
  title: string;
  audience: Audience;
  group: HelpGroup;
  order: number;
  status: Status;
  /**
   * Keep this page out of the sidebar for demo-workspace visitors.
   *
   * The inverse of `audience: demo`, and deliberately weaker: the page stays
   * openable, so a link to it from demo-facing copy still resolves. It exists
   * because Start here otherwise offers a demo user three competing starting
   * points, two of which describe a workspace they do not have.
   */
  hideForDemo?: boolean;
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
  const fm: Record<string, string | number | boolean> = {};
  for (const line of header.split("\n")) {
    const m = line.match(/^([a-zA-Z_][\w-]*)\s*:\s*(.*?)(?:\s*#.*)?$/);
    if (!m) continue;
    const [, key, valRaw] = m;
    const val = valRaw.replace(/^["']|["']$/g, "");
    // The scanner is typed per key rather than by sniffing the value, so
    // `title: true` stays a string. snake_case in the file, camelCase in the
    // type, matching how the frontmatter reads in markdown.
    if (key === "order") {
      fm.order = Number(val);
    } else if (key === "hide_for_demo") {
      fm.hideForDemo = val === "true";
    } else {
      fm[key] = val;
    }
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

const GROUP_RANK = new Map(HELP_GROUPS.map((g, i) => [g.id, i]));

export const HELP_PAGES: HelpPage[] = Object.entries(RAW_MODULES)
  .map(([path, raw]) => {
    const { fm, body } = parseFrontmatter(raw);
    const audience = (fm.audience ?? "shared") as Audience;
    const group =
      fm.group && GROUP_IDS.has(fm.group)
        ? (fm.group as HelpGroup)
        : defaultGroup(audience);
    return {
      slug: pathToSlug(path),
      title: fm.title ?? "Untitled",
      audience,
      group,
      order: fm.order ?? 999,
      status: (fm.status ?? "stub") as Status,
      // Listed explicitly like every field above — this builder enumerates
      // rather than spreading `fm`, so a new frontmatter key is silently
      // dropped until it is named here.
      hideForDemo: fm.hideForDemo ?? false,
      body,
    };
  })
  .sort((a, b) => {
    const ga = GROUP_RANK.get(a.group) ?? 99;
    const gb = GROUP_RANK.get(b.group) ?? 99;
    if (ga !== gb) return ga - gb;
    if (a.order !== b.order) return a.order - b.order;
    // Stable tie-break so a duplicated `order` can never reorder the sidebar
    // between builds (the old sort left ties to glob iteration order).
    return a.title.localeCompare(b.title);
  });

/** Pages the given role may see, bucketed into sidebar groups (empty groups dropped). */
export function groupedPagesFor(
  role: "viewer" | "admin" | "super_admin" | null,
  isDemo = false,
): { id: HelpGroup; label: string; pages: HelpPage[] }[] {
  const visible = pagesFor(role, isDemo);
  return HELP_GROUPS.map(({ id, label }) => ({
    id,
    label,
    pages: visible.filter((p) => p.group === id),
  })).filter((g) => g.pages.length > 0);
}

const BY_SLUG = new Map(HELP_PAGES.map((p) => [p.slug, p]));

export function getHelpPage(slug: string): HelpPage | null {
  return BY_SLUG.get(slug) ?? null;
}

/**
 * Pages the given role may see in the sidebar. Admins see admin + shared (and
 * viewer too — admin docs reference viewer concepts). Viewers see viewer +
 * shared only. Demo pages are orthogonal to role: only the demo workspace sees
 * them, on top of its normal set.
 *
 * Delegates to `canAccess` so the sidebar and the page itself can never
 * disagree. They used to duplicate the audience rules, and the copies had
 * drifted: this returned every page for a null role, including admin pages
 * `canAccess` rejects, so the sidebar offered links that redirected straight
 * back to /help.
 */
export function pagesFor(
  role: "viewer" | "admin" | "super_admin" | null,
  isDemo = false,
): HelpPage[] {
  // `hideForDemo` narrows the listing only, never access — so the sidebar stays
  // a subset of what canAccess allows and the invariant above still holds.
  return HELP_PAGES.filter(
    (p) => canAccess(p, role, isDemo) && !(isDemo && p.hideForDemo),
  );
}

export function canAccess(
  page: HelpPage,
  role: "viewer" | "admin" | "super_admin" | null,
  isDemo = false,
): boolean {
  if (page.audience === "demo") return isDemo;
  if (page.audience === "shared") return true;
  if (page.audience === "viewer") return true; // viewer pages are universal
  if (page.audience === "admin") return role === "admin" || role === "super_admin";
  return false;
}

export type HelpSegment =
  | { kind: "markdown"; value: string }
  | { kind: "loom"; id: string; label?: string; duration?: string };

// Splits a help body into markdown runs and Loom video embeds. A video is
// authored as a fenced ```loom block, either as a bare video id:
//
//   ```loom
//   58f3efa9a75e483892ebe484a6c59afa
//   ```
//
// or with a label and duration, which turn the embed into a titled disclosure
// the reader clicks to open:
//
//   ```loom
//   id: 58f3efa9a75e483892ebe484a6c59afa
//   label: Store Performance
//   duration: 1:46
//   ```
//
// Authoring the label inside the fence rather than as markdown above it is what
// lets the title itself be the control; a markdown "▶ Watch — …" line beside the
// player looks like a disclosure but cannot be clicked.
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
    const seg = parseLoomFence(m[1]);
    if (seg) segments.push(seg);
    last = re.lastIndex;
  }
  const tail = body.slice(last).trim();
  if (tail) segments.push({ kind: "markdown", value: tail });
  return segments;
}

function parseLoomFence(raw: string): HelpSegment | null {
  const lines = raw
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
  if (!lines.length) return null;

  // Bare-id form: the whole fence is the id. A Loom id never contains a colon,
  // so any "key: value" first line means the fence is the keyed form — treating
  // it as bare would silently embed "label: …" as the video id.
  if (!/^[a-zA-Z_-]+\s*:/.test(lines[0])) {
    return { kind: "loom", id: lines[0] };
  }

  const fields: Record<string, string> = {};
  for (const line of lines) {
    const idx = line.indexOf(":");
    if (idx === -1) continue;
    fields[line.slice(0, idx).trim().toLowerCase()] = line.slice(idx + 1).trim();
  }
  if (!fields.id) return null;
  return {
    kind: "loom",
    id: fields.id,
    label: fields.label || undefined,
    duration: fields.duration || undefined,
  };
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
