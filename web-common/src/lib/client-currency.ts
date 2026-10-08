import { get, writable } from "svelte/store";

// Bratrax fork: ISO 4217 code of the active store's currency (`currency:` in
// clients/<name>/config.yaml). web-local's root layout sets it from
// /onboard/me on every navigation, before any page renders.
//
// Dashboard measures already carry their currency (the compiler writes it into
// each metrics view's format). This covers money formatted outside a measure —
// the order drill-down modals, metric trees, the financial statement's
// fallback — which used to hardcode USD, so a EUR, CZK or HUF store saw "$" on
// its own revenue. It lives in web-common because the metric-tree and
// financial-statement canvas components need it and cannot import web-local.
export const clientCurrency = writable<string>("USD");

/**
 * Format `n` as money in the active store's currency (or `currency`).
 *
 * Falls back to USD when Intl rejects the code: toLocaleString throws a
 * RangeError on an unknown currency, and a typo in a config.yaml must not take
 * a whole modal down mid-render.
 */
export function formatMoney(
  n: number,
  options: Intl.NumberFormatOptions = {},
  currency: string = get(clientCurrency),
): string {
  try {
    return n.toLocaleString(undefined, { ...options, style: "currency", currency });
  } catch {
    return n.toLocaleString(undefined, {
      ...options,
      style: "currency",
      currency: "USD",
    });
  }
}
