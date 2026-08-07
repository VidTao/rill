// App-wide Bratrax constants. Yuliya owns the getting-started guide content;
// engineering owns the link target. The guide URL is a placeholder ("coming
// soon") until the guide is published.
export const GETTING_STARTED_URL = "https://bratrax.com/help/start-here";
export const HELP_CENTER_URL = "https://bratrax.com/help";
export const SUPPORT_EMAIL = "support@bratrax.com";

// Legal pages — referenced by the ToS+Privacy consent checkbox on every
// account-creation surface (signup form, request-access CTA, invite
// acceptance).
export const TERMS_OF_SERVICE_URL = "https://bratrax.com/terms-of-service";
export const PRIVACY_POLICY_URL = "https://bratrax.com/privacy-policy";

// Doors-closed waitlist survey (Typeform). After a visitor joins the waitlist
// on /signup, the browser redirects here with their email appended as
// #email=<URL-encoded>. The survey has a matching `email` parameter configured.
export const WAITLIST_TYPEFORM_URL = "https://form.typeform.com/to/NEugqBiK";

/**
 * The Shopify app's handle, as it appears in admin deep links:
 *   https://admin.shopify.com/store/<store>/apps/<handle>
 *
 * Matches `handle` in the bratrax repo's shopify.app.toml. Shopify rewrites
 * the per-install app handle in Liquid block types, but the admin URL uses
 * this stable one.
 */
export const SHOPIFY_APP_HANDLE = "platform-connector-2";
