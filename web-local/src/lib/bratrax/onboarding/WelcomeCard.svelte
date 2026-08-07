<script lang="ts">
  import { onMount } from "svelte";
  import { BarChart3, Search, Sparkles } from "lucide-svelte";
  import {
    Dialog,
    DialogContent,
  } from "@rilldata/web-common/components/dialog";
  import {
    bratraxUser,
    bratraxShowWelcomeCard,
    bratraxIsDemo,
  } from "$lib/bratrax/auth-store";
  import { HELP_CENTER_URL, SUPPORT_EMAIL } from "$lib/bratrax/constants";
  import { getChecklist, dismissWelcome } from "./checklist";

  let open = true;
  let workspaceName = "";
  let adminEmail = "";

  $: firstName = ($bratraxUser?.name || $bratraxUser?.email || "").split(" ")[0];

  onMount(async () => {
    try {
      const c = await getChecklist();
      workspaceName = c.workspace_name;
      adminEmail = c.contact_admin?.email ?? "";
    } catch {
      // Non-blocking — render with what we have.
    }
  });

  async function gotIt() {
    open = false;
    bratraxShowWelcomeCard.set(false);
    try {
      await dismissWelcome();
    } catch {
      // Best-effort; the worst case is the card shows once more next login.
    }
  }
</script>

<Dialog bind:open onOpenChange={(v) => { if (!v) void gotIt(); }}>
  <DialogContent class="welcome-card">
    <!-- Demo visitors get their own card. The owner copy is wrong for them line
         by line — it is not their workspace, the data is synthetic, and they
         cannot act on any of the three prompts — and the admin-contact line
         showed the shared demo account's admin address to every visitor. -->
    {#if $bratraxIsDemo}
      <h2 class="title">
        You're in the Bratrax demo{firstName ? `, ${firstName}` : ""}.
      </h2>
      <p class="lede">
        A working Bratrax workspace running on a synthetic store. The dashboards
        are real; the orders, ad spend and customers are generated. Nothing you
        click can break it.
      </p>

      <div class="bullet">
        <Sparkles size={18} class="bicon" />
        <span>
          <a href="/help/demo-tour">Take the tour</a> — seven stops, about
          fifteen minutes
        </span>
      </div>
      <div class="bullet">
        <BarChart3 size={18} class="bicon" />
        <span>
          Or go straight to
          <a href="/canvas/campaign_deep_dive">Attribution</a>
          or <a href="/canvas/performance_overview">Store Performance</a>
        </span>
      </div>
    {:else}
      <h2 class="title">
        Welcome to {workspaceName || "Bratrax"} on Bratrax{firstName ? `, ${firstName}` : ""}.
      </h2>
      <p class="lede">
        Your workspace is set up and ready. Here's how to get value out of it:
      </p>

      <div class="bullet">
        <BarChart3 size={18} class="bicon" />
        <span>
          Open your
          <a href="/canvas/campaign_deep_dive">Attribution dashboard</a>
        </span>
      </div>
      <div class="bullet">
        <Search size={18} class="bicon" />
        <span>
          <a href="/canvas/customer_analytics">Dive into customer analytics</a>
        </span>
      </div>
      <div class="bullet">
        <Sparkles size={18} class="bicon" />
        <span>Ask the AI a question — try "What's my best ROAS day?"</span>
      </div>

      {#if adminEmail}
        <p class="contact">
          Need access changes? Contact a workspace admin:
          <a href={`mailto:${adminEmail}`}>{adminEmail}</a>
        </p>
      {/if}
    {/if}

    <p class="contact">
      Need help using Bratrax?<br />
      → Browse the
      <a href={HELP_CENTER_URL} target="_blank" rel="noreferrer noopener">help center</a><br />
      → Still stuck? Email
      <a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>
    </p>

    <div class="footer">
      <button class="got-it" on:click={gotIt}>Got it</button>
    </div>
  </DialogContent>
</Dialog>

<style>
  :global(.welcome-card) {
    border-top: 4px solid var(--color-acid) !important;
  }
  .title {
    font-family: "Outfit", sans-serif;
    font-weight: 900;
    font-size: 20px;
    color: var(--color-text);
    margin: 0 0 12px;
  }
  .lede,
  .contact {
    font-family: "Outfit", sans-serif;
    font-weight: 300;
    font-size: 15px;
    color: var(--color-text-secondary);
    margin: 0 0 14px;
  }
  .bullet {
    display: flex;
    align-items: center;
    gap: 10px;
    font-family: "Outfit", sans-serif;
    font-size: 15px;
    color: var(--color-text);
    padding: 6px 0;
  }
  :global(.bicon) {
    color: var(--color-acid-text);
    flex: 0 0 auto;
  }
  a {
    color: var(--color-acid-text);
    text-decoration: underline;
  }
  .footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 18px;
  }
  .got-it {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    background: var(--color-acid);
    color: #0a0a0a;
    border: 1px solid var(--color-acid);
    padding: 8px 16px;
    cursor: pointer;
  }
  .got-it:hover {
    opacity: 0.88;
  }
</style>
