import type { HelpPage } from "$lib/help";

// Destinations for the ⌘K palette: config pages that have no nav entry of their
// own (the six settings tabs sit behind one SETTINGS ▾ trigger) or are buried in
// a dropdown. Dashboards are deliberately absent — the sidebar/tabs strip
// preserves dashboard state (tr, grain, compare_tr, compare_dim), which a
// palette jump would drop.
//
// Visibility mirrors the root-layout nav in routes/+layout.svelte (the
// isViewer / mid-onboarding / post-onboarding branches) plus SettingsDropdown.
// If you change who sees what there, change destinationsFor() to match; the
// spec pins the current mapping. Pages hidden from the nav still exist and the
// backend still enforces access — this is about not advertising what a role
// can't use, not about security.
//
// Not listed: /workshop and /workshop/catalogs. workshop/+layout.ts
// 307-redirects every /workshop/* URL to "/" ("parked").

export type PaletteRole = "viewer" | "admin" | "super_admin";

export interface Destination {
  id: string;
  label: string;
  href: string;
  /** Short context shown beside the label. */
  section: string;
  /** Extra words that should find this page ("invoice" → Billing). */
  keywords: string[];
}

export interface PaletteContext {
  role: PaletteRole | null;
  /** $bratraxOnboarded — mid-onboarding admins are bounced off most routes. */
  onboarded: boolean;
  /** Inside the Shopify admin iframe, where the nav hides Settings. */
  shopifyEmbedded: boolean;
}

const GETTING_STARTED: Destination = {
  id: "getting-started",
  label: "Getting started",
  href: "/onboarding",
  section: "Setup",
  keywords: ["onboarding", "checklist", "setup"],
};

// Labels match SettingsDropdown so the two read as the same list.
const SETTINGS_TABS: Destination[] = [
  {
    id: "settings-account",
    label: "Account settings",
    href: "/settings/account",
    section: "Settings",
    keywords: ["profile", "name", "email", "password"],
  },
  {
    id: "settings-team",
    label: "Team settings",
    href: "/settings/team",
    section: "Settings",
    keywords: ["users", "invite", "members", "teammates", "people", "roles"],
  },
  {
    id: "settings-billing",
    label: "Billing",
    href: "/settings/billing",
    section: "Settings",
    keywords: [
      "plan",
      "subscription",
      "invoice",
      "payment",
      "pricing",
      "upgrade",
      "cancel",
    ],
  },
  {
    id: "settings-ai",
    label: "AI settings",
    href: "/settings/ai",
    section: "Settings",
    keywords: ["assistant", "chat", "anthropic", "claude", "api key"],
  },
  {
    id: "settings-mcp",
    label: "MCP",
    href: "/settings/mcp",
    section: "Settings",
    keywords: ["model context protocol", "claude", "token", "integration"],
  },
  {
    id: "settings-slack",
    label: "Slack",
    href: "/settings/slack",
    section: "Settings",
    keywords: ["notifications", "bot", "assistant"],
  },
];

const CONNECTORS: Destination = {
  id: "connectors",
  label: "Connectors",
  href: "/connectors",
  section: "Settings",
  keywords: [
    "integrations",
    "data sources",
    "connect",
    "reconnect",
    "sync",
    "shopify",
    "woocommerce",
    "facebook",
    "meta",
    "google ads",
    "tiktok",
    "klaviyo",
    "amazon",
  ],
};

const COST_SETTINGS: Destination = {
  id: "cost-settings",
  label: "Cost settings",
  href: "/cost-settings",
  section: "Settings",
  keywords: [
    "cogs",
    "cost of goods",
    "gateway fees",
    "transaction fees",
    "expenses",
    "shipping",
    "profit",
    "margin",
  ],
};

const METRIC_TREES: Destination = {
  id: "metric-trees",
  label: "Metric trees",
  href: "/metric-trees",
  section: "Analysis",
  keywords: ["kpi", "drivers", "root cause", "goals"],
};

const CUSTOMIZE: Destination = {
  id: "customize",
  label: "Customize dashboards",
  href: "/customize",
  section: "Dashboards",
  keywords: ["reorder", "hide", "tabs", "layout"],
};

const SUPER_ADMIN: Destination[] = [
  {
    id: "clients",
    label: "Clients",
    href: "/clients",
    section: "Super admin",
    keywords: ["workspaces", "stores", "accounts"],
  },
  {
    id: "superadmins",
    label: "Super admins",
    href: "/superadmins",
    section: "Super admin",
    keywords: ["invite", "signup links"],
  },
  {
    id: "email-log",
    label: "Email log",
    href: "/superadmins/email-log",
    section: "Super admin",
    keywords: ["sendgrid", "emails", "lifecycle"],
  },
  {
    id: "client-stats",
    label: "Client stats",
    href: "/superadmins/client-stats",
    section: "Super admin",
    keywords: ["usage", "statistics", "fleet"],
  },
];

export function destinationsFor({
  role,
  onboarded,
  shopifyEmbedded,
}: PaletteContext): Destination[] {
  // Viewers' nav is Dashboards + Help; "Getting started" is their avatar-menu
  // link. Settings is not offered to them.
  if (role === "viewer") return [GETTING_STARTED];
  if (role !== "admin" && role !== "super_admin") return [];

  const isSuper = role === "super_admin";
  // Mid-onboarding: only /settings/* survives the +layout.ts isAlwaysAllowed
  // gate; anything else would bounce straight back into the funnel.
  if (!onboarded && !isSuper) return shopifyEmbedded ? [] : SETTINGS_TABS;

  return [
    // The Shopify iframe hides the whole Settings dropdown.
    ...(shopifyEmbedded
      ? []
      : [GETTING_STARTED, CONNECTORS, COST_SETTINGS, ...SETTINGS_TABS]),
    METRIC_TREES,
    CUSTOMIZE,
    ...(isSuper ? SUPER_ADMIN : []),
  ];
}

function words(text: string): string[] {
  return text
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(Boolean);
}

/**
 * Word-prefix matching: every query token must start some word in the item's
 * searchable text. Prefix rather than substring so "ai" finds AI settings but
 * not "Email log". Ranked: label starts with the query, then a label word
 * starts with the first token, then keyword-only matches; stable within a rank.
 * An empty query returns everything in its original order.
 */
function match<T>(
  query: string,
  items: T[],
  label: (item: T) => string,
  searchable: (item: T) => string,
): T[] {
  const tokens = words(query);
  if (tokens.length === 0) return items;
  const phrase = tokens.join(" ");

  const ranked: { item: T; index: number; rank: number }[] = [];
  items.forEach((item, index) => {
    const haystack = words(searchable(item));
    if (!tokens.every((t) => haystack.some((w) => w.startsWith(t)))) return;
    const labelWords = words(label(item));
    const rank = labelWords.join(" ").startsWith(phrase)
      ? 0
      : labelWords.some((w) => w.startsWith(tokens[0]))
        ? 1
        : 2;
    ranked.push({ item, index, rank });
  });
  return ranked
    .sort((a, b) => a.rank - b.rank || a.index - b.index)
    .map((r) => r.item);
}

export function matchDestinations(
  query: string,
  destinations: Destination[],
): Destination[] {
  return match(
    query,
    destinations,
    (d) => d.label,
    (d) => [d.label, d.section, ...d.keywords].join(" "),
  );
}

/**
 * Title-only for now. Bodies are already eager-loaded on every HelpPage, and
 * searchHelp() in $lib/help does body matching with snippets, but that needs
 * snippet rendering to not feel noisy here. Callers must pass pagesFor(role) —
 * the same set the help sidebar shows — never HELP_PAGES directly.
 */
export function matchHelpPages(query: string, pages: HelpPage[]): HelpPage[] {
  return match(
    query,
    pages,
    (p) => p.title,
    (p) => p.title,
  );
}
