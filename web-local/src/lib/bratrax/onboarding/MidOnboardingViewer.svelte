<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { bratraxUser, bratraxViewerMidOnboarding } from "$lib/bratrax/auth-store";
  import { onboardMe } from "./api";
  import { HELP_CENTER_URL } from "$lib/bratrax/constants";

  let workspaceName = "";
  let adminEmail = "";
  let timer: ReturnType<typeof setInterval> | undefined;

  $: firstName = ($bratraxUser?.name || $bratraxUser?.email || "").split(" ")[0];

  async function poll() {
    const me = await onboardMe();
    if (!me) return;
    workspaceName = me.company_name || workspaceName;
    if (me.step === "ready") {
      // Workspace finished setting up — flip out of the placeholder. The layout
      // guard sets the welcome card on the next navigation/refresh; force a
      // reload so the dashboards + welcome card render cleanly.
      bratraxViewerMidOnboarding.set(false);
      if (typeof location !== "undefined") location.reload();
    }
  }

  onMount(() => {
    void poll();
    timer = setInterval(poll, 30000); // 30s, per spec
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<div class="wrap">
  <div class="card">
    <h2 class="title">
      Hi{firstName ? ` ${firstName}` : ""}. {workspaceName || "Your workspace"} is still being set up.
    </h2>
    <p>
      Setup is in progress. We'll let you know when dashboards are ready —
      usually within an hour or two.
    </p>
    {#if adminEmail}
      <p>
        Need access changes? Contact a workspace admin:
        <a href={`mailto:${adminEmail}`}>{adminEmail}</a>
      </p>
    {/if}
    <p>
      Have questions about Bratrax? Browse the
      <a href={HELP_CENTER_URL} target="_blank" rel="noreferrer noopener">help center</a>.
    </p>
  </div>
</div>

<style>
  .wrap {
    flex: 1 1 auto;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg);
    padding: 24px;
  }
  .card {
    max-width: 520px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-top: 4px solid var(--color-acid);
    padding: 28px 30px;
  }
  .title {
    font-family: "Outfit", sans-serif;
    font-weight: 900;
    font-size: 20px;
    color: var(--color-text);
    margin: 0 0 14px;
  }
  p {
    font-family: "Outfit", sans-serif;
    font-weight: 300;
    font-size: 15px;
    color: var(--color-text-secondary);
    margin: 0 0 12px;
  }
  a {
    color: var(--color-acid-text);
    text-decoration: underline;
  }
</style>
