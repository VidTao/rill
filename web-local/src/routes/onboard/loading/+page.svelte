<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { get } from "svelte/store";
  import { goto } from "$app/navigation";
  import { onboardStatus, onboardMe } from "$lib/bratrax/onboarding/api";
  import type { OnboardStatus } from "$lib/bratrax/onboarding/api";
  import {
    // runtimeServiceCreateTrigger,  // disabled for testing — see verify() below
    runtimeServiceListResources,
    V1ReconcileStatus,
    type V1Resource,
    type V1ResourceName,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import { isRillDemoCanvas } from "$lib/bratrax/dashboardPrefs";

  type VerifyState = "pending" | "running" | "refreshing" | "done" | "error";

  let status: OnboardStatus | null = null;
  let error = "";
  let pollInterval: ReturnType<typeof setInterval> | null = null;
  let verifyInterval: ReturnType<typeof setInterval> | null = null;
  let clientId = "";
  let verifyState: VerifyState = "pending";
  let redirectTimer: ReturnType<typeof setTimeout> | null = null;

  // Once reconcile is settled, fire a one-shot source refresh so DuckDB
  // re-materializes from the freshly-populated ClickHouse tables. Without
  // this, dashboards render null until the next hourly cron tick.
  // States: false = not yet, true = in flight, "failed" = trigger errored
  // (proceed to redirect immediately).
  let triggeredRefresh: boolean | "failed" = false;
  let refreshStartMs = 0;
  // Number of consecutive verify() polls that saw zero pending resources
  // since the trigger fired. We require several in a row before declaring
  // "done" — Rill reconciles in cascade waves (Source IDLE → brief gap →
  // downstream Model goes PENDING → IDLE → MetricsView goes PENDING → …),
  // so a single all-IDLE observation can fall in the gap between two waves
  // and produce a premature redirect. Also subsumes the older "trigger
  // hasn't propagated yet" guard: if the trigger had no effect, the counter
  // simply ticks up uninterrupted and we redirect within ~6 seconds.
  let consecutiveIdlePolls = 0;
  const REFRESH_TIMEOUT_MS = 10 * 60 * 1000;
  const REQUIRED_IDLE_POLLS = 3;

  // ---------------------------------------------------------------------------
  // Step display configuration
  // ---------------------------------------------------------------------------
  type StepState = "done" | "running" | "pending" | "error";
  interface StepDisplay {
    label: string;
    // `vs` is included in the signature so Svelte tracks `verifyState` as a
    // template-level reactive dep at the call site — without it, changes to
    // verifyState (which the last two checks read from closure) wouldn't
    // re-render the step list, leaving the UI stuck on the old state.
    check: (s: OnboardStatus, vs: VerifyState) => StepState;
  }

  const steps: StepDisplay[] = [
    {
      label: "Shopify store connected",
      check: (s) => {
        const platforms = s.connected_platforms || [];
        return platforms.some(
          (p) => typeof p === "object" && p.platform === "shopify",
        )
          ? "done"
          : "pending";
      },
    },
    {
      label: "Ad platforms connected",
      check: (s) => {
        const platforms = s.connected_platforms || [];
        const adIds = new Set([
          "facebook_ads",
          "google_ads",
          "tiktok_ads",
          "pinterest_ads",
          "amazon_ads",
        ]);
        return platforms.some(
          (p) => typeof p === "object" && adIds.has(p.platform),
        )
          ? "done"
          : "pending";
      },
    },
    {
      label: "Building your data pipeline",
      check: (s) => {
        if (s.compile_status === "done") return "done";
        if (s.compile_status === "running") return "running";
        if (s.step === "compiling") return "running";
        if (s.step === "activating") return "running";
        return "pending";
      },
    },
    {
      label: "Deploying analytics tables",
      check: (s) => {
        if (s.deploy_status === "done") return "done";
        if (s.deploy_status === "running") return "running";
        if (s.step === "deploying") return "running";
        return "pending";
      },
    },
    {
      label: "Pulling your data",
      check: (s) => {
        const ext = s.extraction_status || {};
        if (s.step === "ready") return "done";
        if (s.step === "extracting") return "running";
        if (ext.status === "running") return "running";
        if (ext.status === "done") return "done";
        return "pending";
      },
    },
    {
      label: "Dashboards ready",
      check: (s) => {
        if (s.step === "ready") return "done";
        return "pending";
      },
    },
    {
      label: "Verifying dashboards",
      check: (s, vs) => {
        if (s.step !== "ready") return "pending";
        // Conceptually done once we've moved past the reconcile-wait phase
        // into the source-refresh phase (or have finished entirely).
        if (vs === "refreshing" || vs === "done") return "done";
        if (vs === "error") return "error";
        if (vs === "running") return "running";
        return "pending";
      },
    },
    {
      label: "Loading your data",
      check: (s, vs) => {
        if (s.step !== "ready") return "pending";
        if (vs === "refreshing") return "running";
        if (vs === "done") return "done";
        return "pending";
      },
    },
  ];

  // ---------------------------------------------------------------------------
  // Extraction tap details
  // ---------------------------------------------------------------------------
  interface TapProgress {
    label: string;
    ready: boolean;
    failed: boolean;
  }

  // Pretty-print "shopify/2qbgfb-3y.myshopify.com" -> "Shopify (2qbgfb-3y)".
  // Falls back to the raw key if it doesn't match the source/account pattern.
  function formatTapLabel(key: string): string {
    const slash = key.indexOf("/");
    if (slash === -1) {
      return key.replace("tap-", "").replace(/_/g, " ");
    }
    const source = key.slice(0, slash);
    const account = key.slice(slash + 1);
    const sourceLabel = source.replace(/_/g, " ");
    // Trim "myshopify.com" suffix etc. so the account is short and readable.
    const accountShort = account.replace(/\.myshopify\.com$/i, "").slice(0, 24);
    return accountShort ? `${sourceLabel} (${accountShort})` : sourceLabel;
  }

  function getTapDetails(s: OnboardStatus): TapProgress[] {
    const ext = s.extraction_status || {};
    const taps = (ext as Record<string, unknown>).taps;
    if (!taps || typeof taps !== "object") return [];
    return Object.entries(
      taps as Record<string, { ready?: boolean; status?: string }>,
    ).map(([name, info]) => ({
      label: formatTapLabel(name),
      ready: info?.ready === true,
      failed: info?.status === "failed",
    }));
  }

  // ---------------------------------------------------------------------------
  // Polling
  // ---------------------------------------------------------------------------
  async function poll() {
    if (!clientId) {
      error = "No client ID found. Please start over from signup.";
      return;
    }

    try {
      status = await onboardStatus(clientId);

      if (status.step === "ready") {
        if (pollInterval) clearInterval(pollInterval);
        pollInterval = null;
        startVerification();
      }

      if (status.step === "error") {
        error =
          status.error_message || "Something went wrong. Please try again.";
        if (pollInterval) clearInterval(pollInterval);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      if (pollInterval) clearInterval(pollInterval);
    }
  }

  // ---------------------------------------------------------------------------
  // Rill resource verification
  //
  // After the backend marks activation as "ready", the Rill runtime still needs
  // to finish parsing the generated project YAML and reconciling each resource
  // (connectors, sources, models, metrics views, dashboards). Redirecting to a
  // canvas before its upstream metrics view is idle produces empty/broken
  // charts. We poll ListResources until every resource transitively referenced
  // by a Canvas/Explore dashboard is IDLE with no reconcileError.
  // ---------------------------------------------------------------------------
  function refKey(ref: V1ResourceName | undefined): string {
    return `${ref?.kind ?? ""}/${ref?.name ?? ""}`;
  }

  function collectRequired(resources: V1Resource[]): Set<string> {
    const byKey = new Map<string, V1Resource>();
    for (const r of resources) {
      byKey.set(refKey(r.meta?.name), r);
    }

    const required = new Set<string>();
    const queue: string[] = [];
    for (const r of resources) {
      const kind = r.meta?.name?.kind;
      if (kind === ResourceKind.Canvas || kind === ResourceKind.Explore) {
        const k = refKey(r.meta?.name);
        required.add(k);
        queue.push(k);
      }
    }

    while (queue.length > 0) {
      const k = queue.shift()!;
      const res = byKey.get(k);
      for (const ref of res?.meta?.refs ?? []) {
        const rk = refKey(ref);
        if (!required.has(rk)) {
          required.add(rk);
          queue.push(rk);
        }
      }
    }

    return required;
  }

  async function verify() {
    try {
      const instanceId = get(runtime).instanceId;
      const resp = await runtimeServiceListResources(instanceId);
      const resources = resp.resources ?? [];

      const required = collectRequired(resources);

      // No dashboards in the project yet — parser hasn't caught up; keep polling.
      const hasDashboard = resources.some((r) => {
        const kind = r.meta?.name?.kind;
        return kind === ResourceKind.Canvas || kind === ResourceKind.Explore;
      });
      if (!hasDashboard) {
        verifyState = "running";
        return;
      }

      const failed: string[] = [];
      let anyPending = false;

      // Kinds whose pending state must block redirect. Walking only the
      // required-from-dashboards set is too narrow: refs on a MetricsView
      // may not always advertise every Model it transitively depends on
      // (downstream model→model refs in particular), so a re-materialization
      // triggered by our CreateTrigger can leave Models still PENDING while
      // the required set already shows IDLE. Waiting on the broader set
      // catches those. Infra kinds (ProjectParser, Connector, Theme) are
      // excluded — they reconcile briefly on every load and shouldn't gate
      // user-facing data.
      const DATA_KINDS = new Set<string>([
        ResourceKind.Source,
        ResourceKind.Model,
        ResourceKind.MetricsView,
        ResourceKind.Explore,
        ResourceKind.Canvas,
        ResourceKind.Component,
      ]);

      // A resource is treated as failed ONLY when it has settled (reconcile
      // status = IDLE) AND still carries a reconcileError. Mid-reconcile
      // errors are transient: Rill cancels and retries a resource whenever an
      // upstream's state version changes ("context canceled"), and those
      // brief error windows would otherwise trigger a permanent "Failed to
      // build" message and stop the polling cycle — even though the
      // reconciler self-heals seconds later. The failed check stays scoped
      // to the required set: an errored Model that nothing depends on
      // shouldn't block redirect.
      for (const r of resources) {
        const k = refKey(r.meta?.name);
        const kind = r.meta?.name?.kind ?? "";
        const isIdle =
          r.meta?.reconcileStatus === V1ReconcileStatus.RECONCILE_STATUS_IDLE;
        if (required.has(k) && isIdle && r.meta?.reconcileError) {
          failed.push(r.meta?.name?.name ?? k);
          continue;
        }
        if (!isIdle && DATA_KINDS.has(kind)) {
          anyPending = true;
        }
      }

      if (failed.length > 0) {
        verifyState = "error";
        error = `Failed to build: ${failed.join(", ")}`;
        if (verifyInterval) clearInterval(verifyInterval);
        verifyInterval = null;
        return;
      }

      const timedOut =
        triggeredRefresh === true &&
        Date.now() - refreshStartMs > REFRESH_TIMEOUT_MS;

      if (anyPending) {
        // Any pending resource resets the stability counter — we're not
        // close to done yet.
        consecutiveIdlePolls = 0;
        if (!timedOut) {
          verifyState = triggeredRefresh ? "refreshing" : "running";
          return;
        }
        console.warn(
          "Onboarding source refresh exceeded timeout; redirecting anyway.",
        );
      }

      // First time we see all-IDLE: reconcile is settled but DuckDB still
      // holds whatever the last hourly cron materialized — which predates the
      // just-finished onboarding extracts. Force a source refresh so the
      // dashboard renders populated data on first paint instead of waiting
      // for the next cron tick. Set the flag synchronously before awaiting
      // to keep re-entrant verify() invocations from re-triggering.
      if (!triggeredRefresh) {
        triggeredRefresh = true;
        refreshStartMs = Date.now();
        verifyState = "refreshing";
        consecutiveIdlePolls = 0;
        // DISABLED FOR TESTING (2026-06-12): the manual source-refresh trigger
        // is a DuckDB-era artifact. It forced re-materialization so the
        // dashboard wouldn't render stale DuckDB data. With ClickHouse-direct
        // (external: true models, no materialization) there is nothing to
        // re-materialize — the dashboard reads CH live — so the default
        // first-access reconcile should suffice once extraction is done.
        // We keep `triggeredRefresh = true` and the `return` so the
        // stability-counter redirect logic below is unchanged; only the
        // CreateTrigger call is skipped.
        // try {
        //   const sourceResources = resources.filter(
        //     (r) => r.meta?.name?.kind === ResourceKind.Source,
        //   );
        //   if (sourceResources.length > 0) {
        //     await runtimeServiceCreateTrigger(instanceId, {
        //       resources: sourceResources.map((r) => ({
        //         kind: ResourceKind.Source,
        //         name: r.meta!.name!.name!,
        //       })),
        //     });
        //   } else {
        //     // Sources may be listed under kind=Model with definedAsSource;
        //     // fall back to triggering all data resources so the fix still
        //     // engages instead of silently no-op'ing.
        //     await runtimeServiceCreateTrigger(instanceId, { all: true });
        //   }
        // } catch (triggerErr) {
        //   console.warn(
        //     "Onboarding source refresh trigger failed:",
        //     triggerErr,
        //   );
        //   triggeredRefresh = "failed";
        // }
        return;
      }

      // All-IDLE post-trigger. Require several consecutive idle polls before
      // redirecting — Rill reconciles in cascade waves (Source IDLE → brief
      // gap → downstream Model PENDING → IDLE → MetricsView PENDING → …),
      // and a single all-IDLE observation can fall in the gap between two
      // waves. The 1.5s redirect timer makes a premature redirect particularly
      // bad: by the time goto() fires, the next wave has started and the
      // dashboard renders mid-cascade. Timeout and "failed" trigger bypass
      // this check.
      consecutiveIdlePolls += 1;
      if (
        !timedOut &&
        triggeredRefresh === true &&
        consecutiveIdlePolls < REQUIRED_IDLE_POLLS
      ) {
        verifyState = "refreshing";
        return;
      }

      verifyState = "done";
      if (verifyInterval) clearInterval(verifyInterval);
      verifyInterval = null;

      // Pick the first Canvas dashboard and route to its preview at
      // /canvas/<name>. Land users on the dashboard view they'll actually
      // use day-to-day — the file editor used to be the post-onboarding
      // target, which was an artifact of the pre-nav-restructure world.
      // Demo canvases (margin_scorecard etc.) get filtered out so a
      // warming-up runtime never lands the user on a Rill bundled-example.
      // Explore-kind resources still fall back to the file view because
      // there's no /explore/<name> Bratrax-fork preview route.
      const firstCanvas = resources.find((r) => {
        const kind = r.meta?.name?.kind;
        const name = r.meta?.name?.name;
        return kind === ResourceKind.Canvas && !isRillDemoCanvas(name);
      });
      const firstCanvasName = firstCanvas?.meta?.name?.name;

      let target: string;
      if (firstCanvasName) {
        target = `/canvas/${firstCanvasName}`;
      } else {
        // Fallback: surface any Explore dashboard via the file editor (no
        // preview route exists for those in the Bratrax fork).
        const firstExplorePath = resources
          .filter((r) => r.meta?.name?.kind === ResourceKind.Explore)
          .flatMap((r) => r.meta?.filePaths ?? [])
          .find((p) => p.startsWith("/dashboards/"));
        target = firstExplorePath ? `/files${firstExplorePath}` : "/";
      }

      // Brief delay so user sees the final checkmark before redirect
      redirectTimer = setTimeout(() => goto(target), 1500);
    } catch (e) {
      // Swallow transient errors — the runtime may be momentarily unreachable
      // as it restarts to pick up newly-deployed project files. Keep polling.
      verifyState = triggeredRefresh ? "refreshing" : "running";
      console.warn("Rill resource verification failed, retrying:", e);
    }
  }

  function startVerification() {
    verifyState = "running";
    verify();
    verifyInterval = setInterval(verify, 2000);
  }

  onMount(async () => {
    // Fetch client_id from backend (sessionStorage may be empty)
    const me = await onboardMe();
    if (me?.client_id) {
      clientId = me.client_id;
    }
    poll();
    pollInterval = setInterval(poll, 5000);
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
    if (verifyInterval) clearInterval(verifyInterval);
    if (redirectTimer) clearTimeout(redirectTimer);
  });
</script>

<div class="loading-page flex h-screen w-screen items-center justify-center">
  <div class="halftone-bg"></div>

  <div class="relative w-full max-w-lg border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6 text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        INITIALIZING
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Setting up your analytics
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        This usually takes 10–15 minutes. You can close this tab and come back
        — we'll keep working in the background.
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <div class="flex flex-col gap-3">
      {#each steps as step}
        {@const state = status ? step.check(status, verifyState) : "pending"}
        <div class="flex items-center gap-3">
          {#if state === "done"}
            <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center bg-bratrax-acid/20">
              <svg class="h-4 w-4 text-bratrax-acid" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
          {:else if state === "running"}
            <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center">
              <div class="h-5 w-5 animate-spin border-2 border-bratrax-border border-t-bratrax-acid"></div>
            </div>
          {:else if state === "error"}
            <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center bg-bratrax-tomato/20">
              <svg class="h-4 w-4 text-bratrax-tomato" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>
          {:else}
            <div class="h-6 w-6 flex-shrink-0 border-2 border-bratrax-border"></div>
          {/if}

          <span
            class="text-sm {state === 'done'
              ? 'text-bratrax-acid'
              : state === 'running'
                ? 'font-medium text-bratrax-text-primary'
                : 'text-bratrax-text-muted'}"
          >
            {step.label}
          </span>
        </div>
      {/each}
    </div>

    <!-- Tap-level extraction details -->
    {#if status && getTapDetails(status).length > 0}
      <div class="mt-6 border-t border-bratrax-border pt-4">
        <p class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          Data sync progress
        </p>
        <div class="flex flex-col gap-1.5">
          {#each getTapDetails(status) as tap}
            <div class="flex items-center justify-between text-xs">
              <span class="capitalize text-bratrax-text-body">{tap.label}</span>
              <span class="font-mono">
                {#if tap.failed}
                  <span class="text-red-500" title="Extract reported an error">✗ error</span>
                {:else if tap.ready}
                  <span class="text-bratrax-acid">✓ ready</span>
                {:else}
                  <span class="text-bratrax-text-body/60">loading…</span>
                {/if}
              </span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .loading-page {
    background-color: #0A0A0A;
    position: relative;
    overflow: hidden;
  }

  .halftone-bg {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background-image: radial-gradient(
      rgba(212, 255, 0, 0.06) 1.5px,
      transparent 1.5px
    );
    background-size: 8px 8px;
  }
</style>
