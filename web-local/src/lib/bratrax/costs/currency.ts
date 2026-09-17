/**
 * Currency symbol for money inputs on the cost-settings page.
 *
 * These form inputs used to hardcode their prefix, which meant a USD store was
 * shown "€" on the shipping tab and "$" everywhere else. Rill's own
 * number-formatting package only knows "$" and "€" (see
 * web-common/src/lib/number-formatting), and it formats dashboard measures
 * rather than form fields, so it does not help here.
 *
 * Intl gives us every currency the browser knows without shipping a table.
 * The Python side keeps an equivalent map for compile-time Rill metric YAML in
 * ontology/compiler/generators/rill_project.py (_CURRENCY_SYMBOLS).
 */
export function currencySymbol(code: string): string {
  const currency = (code || "USD").toUpperCase();

  // narrowSymbol keeps USD as "$" rather than "US$" under non-US locales, but
  // it is not universally supported and throws where it is not; retry plain.
  for (const currencyDisplay of ["narrowSymbol", "symbol"] as const) {
    try {
      const symbol = new Intl.NumberFormat(undefined, {
        style: "currency",
        currency,
        currencyDisplay,
      })
        .formatToParts(0)
        .find((part) => part.type === "currency")?.value;
      if (symbol) return symbol;
    } catch {
      /* unsupported option or unknown code — try the next form */
    }
  }

  return currency;
}
