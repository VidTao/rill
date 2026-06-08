export const ssr = false;

import { redirect } from "@sveltejs/kit";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { get } from "svelte/store";
import { bratraxGetMe } from "$lib/bratrax/auth";
import {
  bratraxUser,
  bratraxAuthChecked,
  bratraxOnboarded,
  bratraxOnboardResumeRoute,
  bratraxShowWelcomeCard,
  bratraxViewerMidOnboarding,
} from "$lib/bratrax/auth-store";
import {
  onboardMe,
  getOnboardResumeRoute,
  getOnboardRouteIndex,
} from "$lib/bratrax/onboarding/api";
import { getChecklist, hasIncompleteSetup } from "$lib/bratrax/onboarding/checklist";
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient.js";
import {
  getRuntimeServiceListFilesQueryKey,
  runtimeServiceListFiles,
  type V1ListFilesResponse,
} from "@rilldata/web-common/runtime-client/index.js";
import { handleUninitializedProject } from "@rilldata/web-common/features/welcome/is-project-initialized.js";
import { Settings } from "luxon";

Settings.defaultLocale = "en";

export async function load({ url, depends, untrack, fetch }) {
  depends("init");

  // Apex: marketing homepage, public to everyone. Authed users skip the
  // marketing page and go straight to their dashboard — preserves the
  // "logged-in users never see the marketing site" behavior the previous
  // Go-proxy apex handler enforced. Must run before the isPublicRoute
  // block below — every path startsWith("/") so a naive prefix match
  // there would swallow the whole app.
  if (url.pathname === "/") {
    const user = await bratraxGetMe(fetch);
    if (user) {
      bratraxUser.set(user);
      bratraxAuthChecked.set(true);
      throw redirect(307, "/developer");
    }
    return { initialized: false };
  }

  // /login, /signup, /accept-invite, password-reset, and the legal pages are
  // reachable without authentication. Every other path (including /onboard)
  // requires an authenticated session; unauthenticated visitors are bounced
  // to /login with the original destination preserved as a ?redirect= param.
  const isPublicRoute =
    url.pathname.startsWith("/login") ||
    url.pathname.startsWith("/signup") ||
    url.pathname.startsWith("/accept-invite") ||
    url.pathname.startsWith("/forgot-password") ||
    url.pathname.startsWith("/reset-password") ||
    url.pathname.startsWith("/privacy-policy") ||
    url.pathname.startsWith("/terms-of-service") ||
    url.pathname.startsWith("/vs/") ||
    url.pathname.startsWith("/faq");

  if (isPublicRoute) {
    return { initialized: false };
  }

  const user = await bratraxGetMe(fetch);
  bratraxUser.set(user);
  bratraxAuthChecked.set(true);
  if (!user) {
    const redirectTo = encodeURIComponent(url.pathname + url.search);
    throw redirect(307, `/login?redirect=${redirectTo}`);
  }

  // Resume incomplete onboarding: if the user's rill_onboarding_state.step
  // maps to a specific /onboard/* page, hold them at or after that page.
  // Forward navigation along the onboard flow is allowed (e.g. clicking
  // "Set up my analytics" on /onboard/stack must reach /onboard/business
  // even though the server step is still "platforms_connected"); only
  // backward jumps and non-onboard routes get bounced back.
  //
  // HARD-GATE STEPS override the forward-nav allowance — the user must be
  // exactly on the target path, not anywhere downstream:
  //   - payment_pending: must complete LS checkout before any onboarding
  //   - embed_pending:   must enable the Shopify Theme App Embed before
  //                      proceeding (so attribution doesn't silently miss
  //                      email signups / Klaviyo onsite events)
  //   - created:         must connect Shopify next; transitioning out of
  //                      this step is server-driven (OAuth callback flips
  //                      to embed_pending), so the forward-nav allowance
  //                      isn't needed and lets users jump into
  //                      half-broken downstream pages otherwise.
  // Without these gates a user could type /onboard/shopify directly while
  // payment_pending and the forward-nav rule would let them stay there.
  const HARD_GATE_STEPS = new Set([
    "payment_pending",
    "embed_pending",
    "created",
  ]);
  // Settings and Help are reachable at every onboarding step — they have no
  // Rill-project dependency, and a user stuck mid-onboarding (even unpaid)
  // must be able to reach their account settings and help. The minimal nav in
  // the root layout surfaces them; this exemption stops the resume-redirect
  // below from bouncing the user straight back into /onboard/*.
  const isAlwaysAllowed =
    url.pathname.startsWith("/settings") || url.pathname.startsWith("/help");

  try {
    const me = await onboardMe();

    // Surface onboarding completion to the layout's nav. "Onboarded" = no
    // onboarding page left to resume to (step "ready", or no record at all
    // for super-admins / cross-client users). The resume route also feeds the
    // minimal nav's "Return to onboarding" link.
    const resumeRoute = getOnboardResumeRoute(me?.step);
    bratraxOnboarded.set(resumeRoute == null);
    bratraxOnboardResumeRoute.set(resumeRoute);

    // Fully-onboarded users typing /onboard/* directly: bounce to the
    // workspace landing. Without this they'd land on stale onboarding
    // screens — most worryingly /onboard/business which would happily
    // re-submit the questionnaire and trigger another _run_activation.
    //
    // Exceptions (all gated on `onboard_oauth_return` being set, so we
    // only bypass when /connectors initiated the flow):
    //   - /onboard/shopify  — reused as the shop-URL prompt when a ready
    //     user reconnects Shopify from /connectors.
    //   - /onboard/embed    — detour when the Theme App Embed got disabled.
    //   - /onboard/stack    — redirect-OAuth landing for TikTok / Klaviyo /
    //     Microsoft Bing Ads when initiated from /connectors. The page
    //     enters `isOAuthBounce` mode, opens AccountSelectionModal over a
    //     minimal "Completing connection…" screen, then goto(returnTo)
    //     once the user submits.
    // handleOAuthConnect / handleShopifyConnect set onboard_oauth_return
    // before navigating; the destination page clears it on success.
    //
    // ⚠️ MAINTENANCE: this exception list is the most-forgotten step when
    //   adding a new redirect-OAuth platform whose provider redirect URI
    //   lands on /onboard/stack (TikTok, Klaviyo, Bing all do). If a new
    //   platform's redirect URI is anywhere else under /onboard/* and
    //   /connectors will initiate it, add the path here too — otherwise
    //   ready users get bounced to /developer mid-OAuth and the
    //   AccountSelectionModal never opens. See the full new-connector
    //   cascade in docs/lite/CLAUDE.md ("Adding a new OAuth connector").
    // NOTE: the post-funnel checklist lives at /onboarding, which also matches
    // startsWith("/onboard"). Scope the funnel bounce to the real funnel paths
    // (/onboard/*) so a ready admin isn't kicked off their checklist.
    const isOnboardFunnel =
      url.pathname === "/onboard" || url.pathname.startsWith("/onboard/");
    if (me?.step === "ready" && isOnboardFunnel) {
      const isOAuthCallbackBounce =
        (url.pathname.startsWith("/onboard/shopify") ||
          url.pathname.startsWith("/onboard/embed") ||
          url.pathname.startsWith("/onboard/stack")) &&
        typeof sessionStorage !== "undefined" &&
        !!sessionStorage.getItem("onboard_oauth_return");
      if (!isOAuthCallbackBounce) {
        throw redirect(307, "/developer");
      }
    }

    const isViewer = user.role === "viewer";

    // Viewers never get pushed into the /onboard/* funnel — they can't action
    // any of it. A mid-onboarding viewer sees a placeholder instead (rendered
    // by the root layout, gated on this store flag); a ready viewer falls
    // through to the welcome-card logic below.
    bratraxViewerMidOnboarding.set(isViewer && me?.step !== "ready");

    const target = getOnboardResumeRoute(me?.step);
    if (target && !isAlwaysAllowed && !isViewer) {
      const isHardGate = me?.step ? HARD_GATE_STEPS.has(me.step) : false;
      if (isHardGate) {
        // Must be on the gate page exactly. Anything else bounces back.
        if (!url.pathname.startsWith(target)) {
          throw redirect(307, target);
        }
      } else {
        const targetIdx = getOnboardRouteIndex(target);
        const currentIdx = getOnboardRouteIndex(url.pathname);
        if (currentIdx === -1 || currentIdx < targetIdx) {
          throw redirect(307, target);
        }
      }
    }

    // -------- Onboarding checklist (Project C) --------
    // Sits on top of the funnel: only relevant once step === 'ready'.
    if (me?.step === "ready") {
      const isAdminRole = user.role === "admin" || user.role === "super_admin";
      // Admins land on the checklist when setup is unfinished and not dismissed.
      // Gated on the default post-login landing (/developer) ONLY — so clicking
      // a checklist action that navigates to /settings, /connectors, /cost-
      // settings, or a dashboard is never bounced back into the checklist.
      if (isAdminRole && url.pathname === "/developer") {
        try {
          const checklist = await getChecklist();
          if (!checklist.checklist_dismissed && hasIncompleteSetup(checklist)) {
            throw redirect(307, "/onboarding");
          }
        } catch (err) {
          if (err && typeof err === "object" && "status" in err && "location" in err) {
            throw err; // propagate the redirect
          }
          // Checklist fetch failed — don't block the landing.
        }
      }
      // Viewers see a one-time welcome-card overlay (no redirect).
      if (isViewer) {
        try {
          const checklist = await getChecklist();
          bratraxShowWelcomeCard.set(!checklist.welcome_card_dismissed);
        } catch {
          bratraxShowWelcomeCard.set(false);
        }
      }
    }
  } catch (e) {
    // Rethrow SvelteKit redirects; swallow anything else so a transient
    // /onboard/me failure doesn't block the rest of the app from loading.
    if (e && typeof e === "object" && "status" in e && "location" in e) throw e;
  }

  // Onboarding routes are auth-gated but don't require a Rill project yet.
  if (url.pathname.startsWith("/onboard")) {
    return { initialized: false };
  }

  // Skip Rill file initialization for /connectors pages
  if (url.pathname.startsWith("/connectors")) {
    return { initialized: false };
  }

  const instanceId = get(runtime).instanceId;

  const files = await queryClient.fetchQuery<V1ListFilesResponse>({
    queryKey: getRuntimeServiceListFilesQueryKey(instanceId, undefined),
    queryFn: ({ signal }) => {
      return runtimeServiceListFiles(instanceId, undefined, signal);
    },
  });

  const firstDashboardFile = files.files?.find((file) =>
    file.path?.startsWith("/dashboards/"),
  );

  let initialized = !!files.files?.some(({ path }) => path === "/rill.yaml");

  const redirectPath = untrack(() => {
    return (
      !!url.searchParams.get("redirect") &&
      url.pathname !== `/files${firstDashboardFile?.path}` &&
      `/files${firstDashboardFile?.path}`
    );
  });

  if (!initialized) {
    initialized = await handleUninitializedProject(instanceId);
  } else if (redirectPath) {
    throw redirect(303, redirectPath);
  }

  return { initialized };
}
