/**
 * Where to send someone after they sign in or sign up, from a `?redirect=`
 * query parameter. Only same-site paths survive: a leading "/", no "//"
 * (protocol-relative), no backslash (browsers read "/\" as "//"), no control
 * characters, and nothing a URL parser reads as having its own origin.
 * Anything else falls back to `fallback`, so a crafted login link can never
 * forward a signed-in user to another site.
 */
export function safeRedirect(
  raw: string | null | undefined,
  fallback = "/developer",
): string {
  if (!raw || raw.length > 2000) return fallback;
  if (!raw.startsWith("/") || raw.startsWith("//") || raw.includes("\\")) {
    return fallback;
  }
  // eslint-disable-next-line no-control-regex
  if (/[\u0000-\u001f\u007f]/.test(raw)) return fallback;
  try {
    const u = new URL(raw, "https://bratrax.invalid");
    if (u.origin !== "https://bratrax.invalid") return fallback;
  } catch {
    return fallback;
  }
  return raw;
}
