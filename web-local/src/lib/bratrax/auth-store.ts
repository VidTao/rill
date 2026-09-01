import { writable } from "svelte/store";
import type { BratraxUser } from "./auth";

export const bratraxUser = writable<BratraxUser | null>(null);
export const bratraxAuthChecked = writable<boolean>(false);
// True when the user has no onboarding page left to resume to (step === "ready"
// or no onboarding record, e.g. super-admins). Drives the minimal-nav vs
// full-nav decision in the root layout. Default true so real users never have
// tabs hidden before the guard resolves.
export const bratraxOnboarded = writable<boolean>(true);
// The /onboard/* page a non-onboarded user should resume at (null once
// onboarded). Lets the minimal nav offer a "Return to onboarding" link so
// Settings/Help aren't a dead-end mid-flow.
export const bratraxOnboardResumeRoute = writable<string | null>(null);
// Viewer-only: render the one-time welcome-card overlay (set by the layout
// guard when role === 'viewer' and the card hasn't been dismissed yet).
export const bratraxShowWelcomeCard = writable<boolean>(false);
// Viewer-only: their workspace is still mid-onboarding (step !== 'ready').
// The layout renders a placeholder instead of dashboards; viewers can't fix
// anything themselves, so they must never be sent into the /onboard/* funnel.
export const bratraxViewerMidOnboarding = writable<boolean>(false);
// True when the user sits on the shared /try-demo workspace (from
// /onboard/me's `is_demo`). Gates demo-only surfaces — today the
// /help/demo-tour page, which is the welcome email's primary CTA and would
// be noise in a paying customer's help sidebar. Defaults false so nobody
// sees demo-only content before the guard resolves.
export const bratraxIsDemo = writable<boolean>(false);
// Platform IDs whose stored OAuth token has gone bad and needs the merchant to
// reconnect (e.g. ["facebook_ads"]). Set by the root layout guard from the
// /onboard/me it already fetches on every navigation, so the header CTA costs
// no extra request. Defaults [] — the layout swallows /onboard/me failures, and
// a transient error must not accuse a healthy connector of being broken.
export const bratraxNeedsReconnect = writable<string[]>([]);
