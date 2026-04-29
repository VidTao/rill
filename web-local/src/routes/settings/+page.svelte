<script lang="ts">
  import { onMount } from "svelte";
  import {
    getAccount,
    updateAccount,
    getTeam,
    inviteMember,
    updateRole,
    removeMember,
    revokeInvitation,
    getBilling,
    getAISettings,
    updateAISettings,
    deleteAISettings,
    getMCPSettings,
    regenerateMCPToken,
    deleteMCPToken,
  } from "$lib/bratrax/settings/api";
  import type {
    AccountInfo,
    AISettings,
    BillingSummary,
    MCPSettings,
    PendingInvite,
    Role,
    TeamData,
    TeamMember,
  } from "$lib/bratrax/settings/types";

  type TabId = "account" | "team" | "billing" | "ai" | "mcp";

  const TABS: { id: TabId; label: string }[] = [
    { id: "account", label: "Account" },
    { id: "team", label: "Team" },
    { id: "billing", label: "Billing" },
    { id: "ai", label: "AI" },
    { id: "mcp", label: "MCP" },
  ];

  const INVITE_ROLE_OPTIONS: ("admin" | "viewer")[] = ["admin", "viewer"];

  // ---------------------------------------------------------------------------
  // Common timezones (curated list; full pytz catalog is overkill for D2C clients)
  // ---------------------------------------------------------------------------
  const TIMEZONES = [
    "UTC",
    "America/Los_Angeles",
    "America/Denver",
    "America/Chicago",
    "America/New_York",
    "America/Toronto",
    "America/Mexico_City",
    "America/Sao_Paulo",
    "Europe/London",
    "Europe/Dublin",
    "Europe/Lisbon",
    "Europe/Madrid",
    "Europe/Paris",
    "Europe/Berlin",
    "Europe/Amsterdam",
    "Europe/Stockholm",
    "Europe/Warsaw",
    "Europe/Athens",
    "Europe/Istanbul",
    "Africa/Cairo",
    "Africa/Johannesburg",
    "Asia/Dubai",
    "Asia/Kolkata",
    "Asia/Singapore",
    "Asia/Hong_Kong",
    "Asia/Shanghai",
    "Asia/Tokyo",
    "Asia/Seoul",
    "Australia/Sydney",
    "Pacific/Auckland",
  ];

  let activeTab: TabId = "account";
  let topError = "";

  // ----- Account state -------------------------------------------------------
  let account: AccountInfo | null = null;
  let accountForm = { company_name: "", email: "", timezone: "UTC" };
  let accountSaving = false;
  let accountSavedMessage = "";

  // ----- Team state ----------------------------------------------------------
  let team: TeamData = { members: [], pending: [] };
  let inviteOpen = false;
  let inviteEmail = "";
  let inviteName = "";
  let inviteRole: "admin" | "viewer" = "viewer";
  let inviteSaving = false;
  let inviteResultUrl = "";
  let inviteResultExpiresAt = "";
  let inviteResultEmail = "";
  let inviteCopied = false;

  let confirmRemoveOpen = false;
  let confirmTarget: TeamMember | null = null;

  // ----- Billing state -------------------------------------------------------
  let billing: BillingSummary | null = null;
  let billingError = "";

  // ----- AI state ------------------------------------------------------------
  let ai: AISettings | null = null;
  let aiError = "";
  let aiSavedMessage = "";
  let aiKeyInput = "";
  let aiSaving = false;
  let aiRemoving = false;
  let aiConfirmRemoveOpen = false;

  // ----- MCP state -----------------------------------------------------------
  let mcp: MCPSettings | null = null;
  let mcpError = "";
  let mcpStatusMessage = "";
  let mcpRegenerating = false;
  let mcpDeleting = false;
  let mcpConfigCopied = false;
  let mcpTokenCopied = false;
  let mcpConfirmRegenerateOpen = false;
  let mcpConfirmRemoveOpen = false;

  // ---------------------------------------------------------------------------
  // Permission helpers (mirror server rules)
  // ---------------------------------------------------------------------------
  $: isAdminOrSuper = account?.role === "super_admin" || account?.role === "admin";
  $: selfId = team.members.find((m) => m.email === account?.email)?.id ?? null;

  function canEditMember(m: TeamMember): boolean {
    if (!isAdminOrSuper) return false;
    if (m.role === "super_admin") return false;
    if (m.id === selfId) return false;
    return true;
  }

  // ---------------------------------------------------------------------------
  // Loaders
  // ---------------------------------------------------------------------------
  async function loadAccount() {
    accountSavedMessage = "";
    try {
      account = await getAccount();
      accountForm = {
        company_name: account.company_name,
        email: account.email,
        timezone: account.timezone,
      };
    } catch (e: any) {
      topError = e.message ?? "Failed to load account";
    }
  }

  async function loadTeam() {
    try {
      team = await getTeam();
    } catch (e: any) {
      topError = e.message ?? "Failed to load team";
    }
  }

  async function loadBilling() {
    billingError = "";
    try {
      billing = await getBilling();
    } catch (e: any) {
      billingError = e.message ?? "Failed to load billing";
    }
  }

  async function loadAI() {
    aiError = "";
    try {
      ai = await getAISettings();
    } catch (e: any) {
      aiError = e.message ?? "Failed to load AI settings";
    }
  }

  async function saveAIKey() {
    const key = aiKeyInput.trim();
    if (!key) return;
    aiSaving = true;
    aiError = "";
    aiSavedMessage = "";
    try {
      ai = await updateAISettings(key);
      aiKeyInput = "";
      aiSavedMessage = "Saved and verified with Anthropic";
    } catch (e: any) {
      aiError = e.message ?? "Failed to save key";
    } finally {
      aiSaving = false;
    }
  }

  function requestRemoveAIKey() {
    aiConfirmRemoveOpen = true;
  }

  async function confirmRemoveAIKey() {
    aiRemoving = true;
    aiError = "";
    aiSavedMessage = "";
    try {
      ai = await deleteAISettings();
      aiSavedMessage = "Key removed";
    } catch (e: any) {
      aiError = e.message ?? "Failed to remove key";
    } finally {
      aiRemoving = false;
      aiConfirmRemoveOpen = false;
    }
  }

  // ---------- MCP -----------------------------------------------------------
  async function loadMCP() {
    mcpError = "";
    try {
      mcp = await getMCPSettings();
    } catch (e: any) {
      mcpError = e.message ?? "Failed to load MCP settings";
    }
  }

  function requestRegenerateMCP() {
    mcpConfirmRegenerateOpen = true;
  }

  async function confirmRegenerateMCP() {
    mcpRegenerating = true;
    mcpError = "";
    mcpStatusMessage = "";
    try {
      mcp = await regenerateMCPToken();
      mcpStatusMessage = "New token generated. Old token is no longer valid.";
    } catch (e: any) {
      mcpError = e.message ?? "Failed to regenerate token";
    } finally {
      mcpRegenerating = false;
      mcpConfirmRegenerateOpen = false;
    }
  }

  function requestRemoveMCP() {
    mcpConfirmRemoveOpen = true;
  }

  async function confirmRemoveMCP() {
    mcpDeleting = true;
    mcpError = "";
    mcpStatusMessage = "";
    try {
      mcp = await deleteMCPToken();
      mcpStatusMessage = "Token revoked";
    } catch (e: any) {
      mcpError = e.message ?? "Failed to revoke token";
    } finally {
      mcpDeleting = false;
      mcpConfirmRemoveOpen = false;
    }
  }

  async function copyMCPConfig() {
    if (!mcp?.claude_desktop_config) return;
    try {
      await navigator.clipboard.writeText(
        JSON.stringify(mcp.claude_desktop_config, null, 2),
      );
      mcpConfigCopied = true;
      setTimeout(() => (mcpConfigCopied = false), 2000);
    } catch {
      // ignore
    }
  }

  async function copyMCPToken() {
    if (!mcp?.token) return;
    try {
      await navigator.clipboard.writeText(mcp.token);
      mcpTokenCopied = true;
      setTimeout(() => (mcpTokenCopied = false), 2000);
    } catch {
      // ignore
    }
  }

  // ---------------------------------------------------------------------------
  // Account actions
  // ---------------------------------------------------------------------------
  async function saveAccount() {
    if (!account) return;
    accountSaving = true;
    accountSavedMessage = "";
    topError = "";
    try {
      const payload: Record<string, string> = {};
      if (accountForm.company_name !== account.company_name)
        payload.company_name = accountForm.company_name.trim();
      if (accountForm.email !== account.email)
        payload.email = accountForm.email.trim().toLowerCase();
      if (accountForm.timezone !== account.timezone)
        payload.timezone = accountForm.timezone;

      if (Object.keys(payload).length === 0) {
        accountSavedMessage = "No changes";
        return;
      }
      account = await updateAccount(payload);
      accountForm = {
        company_name: account.company_name,
        email: account.email,
        timezone: account.timezone,
      };
      accountSavedMessage = "Saved";
    } catch (e: any) {
      topError = e.message ?? "Save failed";
    } finally {
      accountSaving = false;
    }
  }

  // ---------------------------------------------------------------------------
  // Team actions
  // ---------------------------------------------------------------------------
  function openInvite() {
    inviteEmail = "";
    inviteName = "";
    inviteRole = "viewer";
    inviteResultUrl = "";
    inviteResultEmail = "";
    inviteResultExpiresAt = "";
    inviteCopied = false;
    inviteOpen = true;
  }

  async function submitInvite() {
    if (!inviteEmail.includes("@")) return;
    inviteSaving = true;
    topError = "";
    try {
      const result = await inviteMember(inviteEmail.trim().toLowerCase(), inviteRole, inviteName.trim());
      inviteResultUrl = result.accept_url;
      inviteResultExpiresAt = result.expires_at;
      inviteResultEmail = result.email;
      await loadTeam();
    } catch (e: any) {
      topError = e.message ?? "Invite failed";
    } finally {
      inviteSaving = false;
    }
  }

  async function copyInviteUrl() {
    try {
      await navigator.clipboard.writeText(inviteResultUrl);
      inviteCopied = true;
      setTimeout(() => (inviteCopied = false), 2000);
    } catch {
      // ignore
    }
  }

  async function copyPendingUrl(p: PendingInvite) {
    try {
      await navigator.clipboard.writeText(p.accept_url);
    } catch {
      // ignore
    }
  }

  async function changeMemberRole(m: TeamMember, newRole: "admin" | "viewer") {
    if (m.role === newRole) return;
    topError = "";
    try {
      await updateRole(m.id, newRole);
      await loadTeam();
    } catch (e: any) {
      topError = e.message ?? "Role update failed";
    }
  }

  function onRoleChange(m: TeamMember, e: Event) {
    const v = (e.currentTarget as HTMLSelectElement).value;
    if (v === "admin" || v === "viewer") changeMemberRole(m, v);
  }

  function selectAllOnClick(e: MouseEvent) {
    (e.currentTarget as HTMLInputElement).select();
  }

  function requestRemove(m: TeamMember) {
    confirmTarget = m;
    confirmRemoveOpen = true;
  }

  async function confirmRemove() {
    if (!confirmTarget) return;
    topError = "";
    try {
      await removeMember(confirmTarget.id);
      confirmRemoveOpen = false;
      confirmTarget = null;
      await loadTeam();
    } catch (e: any) {
      topError = e.message ?? "Remove failed";
    }
  }

  async function revokePending(p: PendingInvite) {
    topError = "";
    try {
      await revokeInvitation(p.token);
      await loadTeam();
    } catch (e: any) {
      topError = e.message ?? "Revoke failed";
    }
  }

  // ---------------------------------------------------------------------------
  function formatDate(iso: string): string {
    if (!iso) return "";
    try {
      return new Date(iso).toLocaleDateString();
    } catch {
      return iso;
    }
  }

  function formatPrice(cents: number, currency: string): string {
    return `$${(cents / 100).toFixed(2)} ${currency}`;
  }

  function roleLabel(r: Role): string {
    if (r === "super_admin") return "OWNER";
    if (r === "admin") return "ADMIN";
    return "VIEWER";
  }

  function initialsOf(name: string, email: string): string {
    const src = (name || email || "").trim();
    if (!src) return "?";
    const parts = src.split(/\s+/);
    if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
    return src.slice(0, 2).toUpperCase();
  }

  // ---------------------------------------------------------------------------
  onMount(async () => {
    await Promise.all([loadAccount(), loadTeam(), loadBilling(), loadAI(), loadMCP()]);
  });
</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="relative w-full max-w-3xl border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        SETTINGS
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Manage your <span class="font-serif italic text-bratrax-acid">workspace</span>
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Team, billing, and account preferences.
      </p>
    </div>

    {#if topError}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {topError}
      </div>
    {/if}

    <!-- Tab strip -->
    <div class="mb-6 flex items-center gap-1 border-b border-bratrax-border">
      {#each TABS as tab}
        <button
          type="button"
          class="border-b-2 px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {activeTab === tab.id
            ? 'border-bratrax-acid text-bratrax-acid'
            : 'border-transparent text-bratrax-text-muted hover:text-bratrax-text-body'}"
          on:click={() => (activeTab = tab.id)}
        >
          {tab.label}
        </button>
      {/each}
    </div>

    <!-- Account tab -->
    {#if activeTab === "account"}
      {#if account}
        <div class="flex flex-col gap-4">
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Company name
            </label>
            <input
              type="text"
              bind:value={accountForm.company_name}
              disabled={!isAdminOrSuper}
              class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
            />
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Your email
            </label>
            <input
              type="email"
              bind:value={accountForm.email}
              class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
            />
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Timezone
            </label>
            <select
              bind:value={accountForm.timezone}
              disabled={!isAdminOrSuper}
              class="w-full border border-bratrax-border bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
            >
              {#each TIMEZONES as tz}
                <option value={tz}>{tz}</option>
              {/each}
            </select>
          </div>

          <div class="mt-2 flex items-center gap-3">
            <button
              type="button"
              on:click={saveAccount}
              disabled={accountSaving}
              class="bg-bratrax-acid px-6 py-2.5 font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
            >
              {accountSaving ? "Saving…" : "Save changes"}
            </button>
            {#if accountSavedMessage}
              <span class="font-mono text-xs text-bratrax-text-muted">{accountSavedMessage}</span>
            {/if}
          </div>
        </div>
      {:else}
        <p class="font-mono text-xs text-bratrax-text-muted">Loading…</p>
      {/if}
    {/if}

    <!-- Team tab -->
    {#if activeTab === "team"}
      <div class="flex items-baseline justify-between">
        <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
          Members ({team.members.length})
        </h2>
        {#if isAdminOrSuper}
          <button
            type="button"
            on:click={openInvite}
            class="bg-bratrax-acid px-4 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
          >
            Invite teammate
          </button>
        {/if}
      </div>

      <ul class="mt-3 flex flex-col gap-2">
        {#each team.members as m (m.id)}
          <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
            <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center bg-bratrax-acid/15 font-mono text-xs font-bold uppercase tracking-wider text-bratrax-acid">
              {initialsOf(m.name, m.email)}
            </div>
            <div class="min-w-0 flex-1">
              <div class="truncate font-mono text-xs font-bold text-bratrax-text-headline">
                {m.name || m.email}
              </div>
              <div class="truncate font-mono text-[10px] text-bratrax-text-muted">{m.email}</div>
            </div>

            {#if m.role === "super_admin"}
              <span class="border border-bratrax-acid/40 bg-bratrax-acid/10 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-acid">
                {roleLabel(m.role)}
              </span>
            {:else if canEditMember(m)}
              <select
                value={m.role}
                on:change={(e) => onRoleChange(m, e)}
                class="border border-bratrax-border bg-bratrax-surface px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
              >
                <option value="admin">ADMIN</option>
                <option value="viewer">VIEWER</option>
              </select>
            {:else}
              <span class="border border-bratrax-border bg-bratrax-surface px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-muted">
                {roleLabel(m.role)}
              </span>
            {/if}

            {#if canEditMember(m)}
              <button
                type="button"
                on:click={() => requestRemove(m)}
                class="border border-bratrax-tomato/40 bg-bratrax-surface px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/10"
              >
                Remove
              </button>
            {/if}
          </li>
        {/each}
      </ul>

      {#if team.pending.length > 0}
        <h2 class="mt-6 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
          Pending invitations ({team.pending.length})
        </h2>
        <ul class="mt-3 flex flex-col gap-2">
          {#each team.pending as p (p.token)}
            <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
              <div class="min-w-0 flex-1">
                <div class="truncate font-mono text-xs font-bold text-bratrax-text-headline">{p.email}</div>
                <div class="truncate font-mono text-[10px] text-bratrax-text-muted">
                  {roleLabel(p.role)} · expires {formatDate(p.expires_at)}
                </div>
              </div>
              <button
                type="button"
                on:click={() => copyPendingUrl(p)}
                class="border border-bratrax-border bg-bratrax-surface px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-body hover:bg-bratrax-hover"
              >
                Copy link
              </button>
              {#if isAdminOrSuper}
                <button
                  type="button"
                  on:click={() => revokePending(p)}
                  class="border border-bratrax-tomato/40 bg-bratrax-surface px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/10"
                >
                  Revoke
                </button>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    {/if}

    <!-- Billing tab -->
    {#if activeTab === "billing"}
      {#if billingError}
        <div class="border border-bratrax-border bg-bratrax-bg px-4 py-6 text-center font-mono text-xs text-bratrax-text-muted">
          {billingError}
        </div>
      {:else if billing}
        <div class="flex flex-col gap-4">
          <div class="border border-bratrax-border bg-bratrax-bg p-4">
            <div class="flex items-baseline justify-between">
              <div>
                <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
                  Current plan
                </div>
                <div class="mt-1 text-xl font-black text-bratrax-text-headline">
                  {billing.plan} <span class="font-mono text-sm font-normal text-bratrax-text-muted">— {formatPrice(billing.price_cents, billing.currency)}/{billing.interval}</span>
                </div>
              </div>
              <span class="border border-bratrax-acid/40 bg-bratrax-acid/10 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-acid">
                {billing.status}
              </span>
            </div>
            <p class="mt-2 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
              Renews {formatDate(billing.current_period_end)}
            </p>
            <div class="mt-3 flex items-center gap-2">
              <button
                type="button"
                disabled
                class="cursor-not-allowed border border-bratrax-border bg-bratrax-surface px-4 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted opacity-60"
                title="Coming soon"
              >
                Manage Subscription · Coming soon
              </button>
            </div>
          </div>

          {#if billing.payment_method}
            <div class="border border-bratrax-border bg-bratrax-bg p-4">
              <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
                Payment method
              </div>
              <div class="mt-2 flex items-center justify-between">
                <span class="font-mono text-sm text-bratrax-text-headline">
                  {billing.payment_method.brand.toUpperCase()} ending {billing.payment_method.last4}
                </span>
                <button
                  type="button"
                  disabled
                  class="cursor-not-allowed border border-bratrax-border bg-bratrax-surface px-4 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted opacity-60"
                  title="Coming soon"
                >
                  Update card · Coming soon
                </button>
              </div>
            </div>
          {/if}

          <div>
            <h3 class="mb-3 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
              Invoices
            </h3>
            {#if billing.invoices.length === 0}
              <p class="font-mono text-xs text-bratrax-text-muted">No invoices yet.</p>
            {:else}
              <ul class="flex flex-col gap-1.5">
                {#each billing.invoices as inv}
                  <li class="flex items-center justify-between border border-bratrax-border bg-bratrax-bg px-4 py-2">
                    <div class="font-mono text-xs text-bratrax-text-body">
                      {formatDate(inv.date)}
                    </div>
                    <div class="font-mono text-xs text-bratrax-text-headline">
                      {formatPrice(inv.amount_cents, billing.currency)}
                    </div>
                    <span class="border border-bratrax-border bg-bratrax-surface px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-text-muted">
                      {inv.status}
                    </span>
                    <a
                      href={inv.url}
                      class="font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-acid hover:underline"
                    >
                      View
                    </a>
                  </li>
                {/each}
              </ul>
            {/if}
          </div>
        </div>
      {:else}
        <p class="font-mono text-xs text-bratrax-text-muted">Loading…</p>
      {/if}
    {/if}

    <!-- AI tab -->
    {#if activeTab === "ai"}
      {#if aiError && !ai}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {aiError}
        </div>
      {:else if ai}
        <div class="flex flex-col gap-4">
          <div>
            <h2 class="text-lg font-black text-bratrax-text-headline">
              Anthropic API key
            </h2>
            <p class="mt-2 text-sm font-light text-bratrax-text-body">
              Bratrax routes Claude chat through your own Anthropic account. Paste an API key
              from
              <a
                href="https://console.anthropic.com/settings/keys"
                target="_blank"
                rel="noopener"
                class="text-bratrax-acid hover:underline"
              >console.anthropic.com</a>.
              We verify it works before saving.
            </p>
          </div>

          {#if ai.key_set}
            <div class="border border-bratrax-border bg-bratrax-bg p-4">
              <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
                Current key
              </div>
              <div class="mt-2 flex items-center justify-between gap-3">
                <span class="font-mono text-sm text-bratrax-text-headline truncate">
                  {ai.key_preview}
                </span>
                <span class="border border-bratrax-acid/40 bg-bratrax-acid/10 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-acid flex-shrink-0">
                  Active
                </span>
              </div>
              <div class="mt-3 flex items-center gap-2">
                <button
                  type="button"
                  on:click={requestRemoveAIKey}
                  disabled={aiRemoving}
                  class="border border-bratrax-tomato/40 bg-bratrax-surface px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/10 disabled:opacity-50"
                >
                  {aiRemoving ? "Removing…" : "Remove key"}
                </button>
              </div>
            </div>
          {:else}
            <div class="border border-bratrax-border bg-bratrax-bg p-4">
              <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
                Status
              </div>
              <p class="mt-2 font-mono text-xs text-bratrax-text-muted">
                No key configured. Claude chat is disabled until you add one.
              </p>
            </div>
          {/if}

          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              {ai.key_set ? "Replace key" : "Add key"}
            </label>
            <div class="flex items-center gap-2">
              <input
                type="password"
                bind:value={aiKeyInput}
                placeholder="sk-ant-..."
                autocomplete="off"
                spellcheck="false"
                class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 font-mono text-xs text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
              />
              <button
                type="button"
                on:click={saveAIKey}
                disabled={aiSaving || !aiKeyInput.trim()}
                class="flex-shrink-0 bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
              >
                {aiSaving ? "Verifying…" : "Save"}
              </button>
            </div>
            <p class="mt-2 font-mono text-[10px] text-bratrax-text-muted">
              We send a single test request to Anthropic to confirm the key works before saving.
            </p>
          </div>

          {#if aiError}
            <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
              {aiError}
            </div>
          {/if}
          {#if aiSavedMessage}
            <div class="border border-bratrax-acid/30 bg-bratrax-acid/10 px-3 py-2 font-mono text-xs text-bratrax-acid">
              {aiSavedMessage}
            </div>
          {/if}
        </div>
      {:else}
        <p class="font-mono text-xs text-bratrax-text-muted">Loading…</p>
      {/if}
    {/if}

    <!-- MCP tab -->
    {#if activeTab === "mcp"}
      {#if mcpError && !mcp}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {mcpError}
        </div>
      {:else if mcp}
        <div class="flex flex-col gap-4">
          <div>
            <h2 class="text-lg font-black text-bratrax-text-headline">
              Connect Claude Desktop
            </h2>
            <p class="mt-2 text-sm font-light text-bratrax-text-body">
              Paste the config below into Claude Desktop's
              <code class="font-mono text-[11px] text-bratrax-acid">claude_desktop_config.json</code>
              to query your Bratrax data with Claude. The token authenticates as your workspace
              and gives Claude access to all 22 MCP tools (metrics queries, SQL, file operations,
              workshop tools).
            </p>
            <p class="mt-2 font-mono text-[10px] text-bratrax-text-muted">
              On macOS the config file lives at
              <code class="text-bratrax-text-body">~/Library/Application Support/Claude/claude_desktop_config.json</code>.
              Restart Claude Desktop after pasting.
            </p>
          </div>

          {#if mcp.token_set && mcp.claude_desktop_config}
            <!-- Status row + revoke button -->
            <div class="border border-bratrax-border bg-bratrax-bg p-4">
              <div class="flex items-baseline justify-between">
                <div>
                  <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
                    Token status
                  </div>
                  <div class="mt-1 font-mono text-sm text-bratrax-text-headline">
                    Active{mcp.created_at ? ` · created ${formatDate(mcp.created_at)}` : ""}
                  </div>
                </div>
                <span class="border border-bratrax-acid/40 bg-bratrax-acid/10 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-acid flex-shrink-0">
                  Connected
                </span>
              </div>
              <div class="mt-3 flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  on:click={requestRegenerateMCP}
                  disabled={mcpRegenerating}
                  class="border border-bratrax-border bg-bratrax-surface px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover disabled:opacity-50"
                >
                  {mcpRegenerating ? "Regenerating…" : "Regenerate token"}
                </button>
                <button
                  type="button"
                  on:click={requestRemoveMCP}
                  disabled={mcpDeleting}
                  class="border border-bratrax-tomato/40 bg-bratrax-surface px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/10 disabled:opacity-50"
                >
                  {mcpDeleting ? "Revoking…" : "Revoke"}
                </button>
              </div>
            </div>

            <!-- Token (separately copyable) -->
            <div>
              <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
                Token
              </label>
              <div class="flex items-center gap-2">
                <input
                  type="text"
                  readonly
                  value={mcp.token}
                  class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
                  on:click={selectAllOnClick}
                />
                <button
                  type="button"
                  on:click={copyMCPToken}
                  class="flex-shrink-0 border border-bratrax-border bg-bratrax-surface px-3 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
                >
                  {mcpTokenCopied ? "Copied!" : "Copy"}
                </button>
              </div>
            </div>

            <!-- Full config JSON (the canonical paste-into-Claude-Desktop blob) -->
            <div>
              <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
                claude_desktop_config.json
              </label>
              <pre class="border border-bratrax-border bg-bratrax-bg px-3 py-3 font-mono text-[11px] leading-relaxed text-bratrax-text-body overflow-x-auto">{JSON.stringify(mcp.claude_desktop_config, null, 2)}</pre>
              <div class="mt-2 flex items-center gap-2">
                <button
                  type="button"
                  on:click={copyMCPConfig}
                  class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
                >
                  {mcpConfigCopied ? "Copied!" : "Copy config"}
                </button>
                <p class="font-mono text-[10px] text-bratrax-text-muted">
                  Anyone with this token can query your Bratrax data — keep it private.
                </p>
              </div>
            </div>
          {:else}
            <div class="border border-bratrax-border bg-bratrax-bg p-4">
              <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
                Status
              </div>
              <p class="mt-2 font-mono text-xs text-bratrax-text-muted">
                No token yet. Generate one to connect Claude Desktop.
              </p>
              <div class="mt-3">
                <button
                  type="button"
                  on:click={requestRegenerateMCP}
                  disabled={mcpRegenerating}
                  class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
                >
                  {mcpRegenerating ? "Generating…" : "Generate token"}
                </button>
              </div>
            </div>
          {/if}

          {#if mcpError}
            <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
              {mcpError}
            </div>
          {/if}
          {#if mcpStatusMessage}
            <div class="border border-bratrax-acid/30 bg-bratrax-acid/10 px-3 py-2 font-mono text-xs text-bratrax-acid">
              {mcpStatusMessage}
            </div>
          {/if}
        </div>
      {:else}
        <p class="font-mono text-xs text-bratrax-text-muted">Loading…</p>
      {/if}
    {/if}
  </div>
</div>

<!-- Invite modal -->
{#if inviteOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (inviteOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (inviteOpen = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div class="mb-2 flex items-baseline justify-between">
        <div class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          {inviteResultUrl ? "Invite link" : "Invite teammate"}
        </div>
        <button
          on:click={() => (inviteOpen = false)}
          class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
        >
          Close
        </button>
      </div>

      {#if !inviteResultUrl}
        <h2 class="text-lg font-black text-bratrax-text-headline">Generate invite link</h2>
        <p class="mt-2 text-sm font-light text-bratrax-text-body">
          We'll create a secure link. You copy it and send it however you like.
        </p>

        <div class="mt-4 flex flex-col gap-3">
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Email
            </label>
            <input
              type="email"
              bind:value={inviteEmail}
              placeholder="teammate@company.com"
              class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
            />
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Name (optional)
            </label>
            <input
              type="text"
              bind:value={inviteName}
              placeholder="Joe Smith"
              class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
            />
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Role
            </label>
            <div class="flex gap-2">
              {#each INVITE_ROLE_OPTIONS as r}
                <button
                  type="button"
                  on:click={() => (inviteRole = r)}
                  class="flex-1 border px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {inviteRole === r
                    ? 'border-bratrax-acid bg-bratrax-acid/10 text-bratrax-acid'
                    : 'border-bratrax-border bg-bratrax-bg text-bratrax-text-muted hover:border-bratrax-text-muted'}"
                >
                  {r === "admin" ? "Admin" : "Viewer"}
                </button>
              {/each}
            </div>
          </div>
        </div>

        <div class="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            on:click={() => (inviteOpen = false)}
            class="border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={submitInvite}
            disabled={inviteSaving || !inviteEmail.includes("@")}
            class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
          >
            {inviteSaving ? "Generating…" : "Generate link"}
          </button>
        </div>
      {:else}
        <h2 class="text-lg font-black text-bratrax-text-headline">Send this link to {inviteResultEmail}</h2>
        <p class="mt-2 text-sm font-light text-bratrax-text-body">
          The link expires {formatDate(inviteResultExpiresAt)}. Anyone with the link can sign up
          using this email.
        </p>

        <div class="mt-4 flex items-center gap-2">
          <input
            type="text"
            readonly
            value={inviteResultUrl}
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            on:click={selectAllOnClick}
          />
          <button
            type="button"
            on:click={copyInviteUrl}
            class="flex-shrink-0 bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
          >
            {inviteCopied ? "Copied!" : "Copy"}
          </button>
        </div>

        <div class="mt-6 flex items-center justify-end">
          <button
            type="button"
            on:click={() => (inviteOpen = false)}
            class="border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
          >
            Done
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Remove confirmation modal -->
{#if confirmRemoveOpen && confirmTarget}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (confirmRemoveOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (confirmRemoveOpen = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-tomato"></div>

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Confirm remove
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Remove {confirmTarget.name || confirmTarget.email}?
      </h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        They will lose access to this workspace immediately. You can re-invite them later if needed.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          on:click={() => (confirmRemoveOpen = false)}
          class="border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
        >
          Cancel
        </button>
        <button
          on:click={confirmRemove}
          class="border border-bratrax-tomato/60 bg-bratrax-tomato/10 px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/20"
        >
          Remove
        </button>
      </div>
    </div>
  </div>
{/if}


<!-- AI key remove confirmation modal -->
{#if aiConfirmRemoveOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (aiConfirmRemoveOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (aiConfirmRemoveOpen = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-tomato"></div>

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Confirm remove
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">Remove your Anthropic API key?</h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Claude chat will be disabled for everyone in your workspace until a new key is added.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          on:click={() => (aiConfirmRemoveOpen = false)}
          class="border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
        >
          Cancel
        </button>
        <button
          on:click={confirmRemoveAIKey}
          class="border border-bratrax-tomato/60 bg-bratrax-tomato/10 px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/20"
        >
          Remove key
        </button>
      </div>
    </div>
  </div>
{/if}


<!-- MCP regenerate confirmation modal -->
{#if mcpConfirmRegenerateOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (mcpConfirmRegenerateOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (mcpConfirmRegenerateOpen = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        Confirm regenerate
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">Generate a new MCP token?</h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        The current token will stop working immediately. Any Claude Desktop instance still using
        it will fail until you paste the new config.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          on:click={() => (mcpConfirmRegenerateOpen = false)}
          class="border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
        >
          Cancel
        </button>
        <button
          on:click={confirmRegenerateMCP}
          class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
        >
          Regenerate
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- MCP revoke confirmation modal -->
{#if mcpConfirmRemoveOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (mcpConfirmRemoveOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (mcpConfirmRemoveOpen = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-tomato"></div>

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Confirm revoke
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">Revoke MCP token?</h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Claude Desktop instances using this token will stop working immediately. You can generate
        a new one any time.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          on:click={() => (mcpConfirmRemoveOpen = false)}
          class="border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover"
        >
          Cancel
        </button>
        <button
          on:click={confirmRemoveMCP}
          class="border border-bratrax-tomato/60 bg-bratrax-tomato/10 px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/20"
        >
          Revoke
        </button>
      </div>
    </div>
  </div>
{/if}
