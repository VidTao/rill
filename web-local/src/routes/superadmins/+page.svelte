<script lang="ts">
  import { onMount } from "svelte";
  import {
    getSuperadmins,
    inviteSuperadmin,
    removeSuperadmin,
    revokeSuperadminInvitation,
    listSignupInvitations,
    createSignupInvite,
    revokeSignupInvitation,
    listStoreSignupLinks,
    createStoreSignupLink,
    revokeStoreSignupLink,
    listAccessRequests,
    approveAccessRequest,
    dismissAccessRequest,
    listInvitationEnrollments,
    type SuperadminListResponse,
    type SuperadminMember,
    type SuperadminPendingInvite,
    type SignupPendingInvite,
    type StoreSignupLink,
    type StorePlatform,
    type AccessRequest,
    type SequenceEnrollment,
  } from "$lib/bratrax/superadmins/api";
  import { bratraxUser } from "$lib/bratrax/auth-store";

  let data: SuperadminListResponse = { members: [], pending: [] };
  let signupPending: SignupPendingInvite[] = [];
  let accessRequests: AccessRequest[] = [];
  // token → first matching invitation_expiry enrollment (or null when
  // no enrollment yet — most invites are >48h from expiry, ticker hasn't
  // fired for them). Populated alongside signupPending; the loop is
  // bounded by the typical pending-invites count (<50).
  let inviteExpiryByToken = new Map<string, SequenceEnrollment | null>();
  let topError = "";

  // Invite modal state (super_admin)
  let inviteOpen = false;
  let inviteEmail = "";
  let inviteName = "";
  let inviteSaving = false;
  let inviteResultUrl = "";
  let inviteResultExpiresAt = "";
  let inviteResultEmail = "";
  let inviteCopied = false;

  // Signup-invite modal state (customer signup link).
  // signupInviteRequiresPayment: TRUE = real paying customer (LS checkout
  // required after onboard_start); FALSE = inceptly internal team (skip
  // payment). Default TRUE for safety — accidentally inviting a paying
  // customer with the toggle in "free" position would skip billing.
  // signupInviteMultiStore: TRUE = agency-style parent account that owns
  // several sub-stores (rill_multi_clients). Orthogonal to requires_payment.
  // Default FALSE — opt-in only.
  let signupInviteOpen = false;
  let signupInviteEmail = "";
  let signupInviteRequiresPayment = true;
  let signupInviteMultiStore = false;
  let signupInviteSaving = false;
  let signupInviteResultUrl = "";
  let signupInviteResultExpiresAt = "";
  let signupInviteResultEmail = "";
  let signupInviteResultRequiresPayment = true;
  let signupInviteResultMultiStore = false;
  let signupInviteCopied = false;

  // Store signup link modal (multi-use, store-locked, capped). Every signup from
  // the link defaults to a paying, single-store account with the store preselected.
  let storeLinkOpen = false;
  let storeLinkStore: StorePlatform = "shopify";
  let storeLinkMaxUsers = 10;
  let storeLinkSaving = false;
  let storeLinkResultUrl = "";
  let storeLinkResultStore: StorePlatform = "shopify";
  let storeLinkResultMaxUsers = 0;
  let storeLinkResultExpiresAt = "";
  let storeLinkCopied = false;
  let storePending: StoreSignupLink[] = [];

  // Access-request actions in flight per id, so the right row's button shows
  // a loading state without blocking the others.
  let accessActionId: number | null = null;

  // Access-request dismiss confirmation
  let dismissOpen = false;
  let dismissTarget: AccessRequest | null = null;
  let dismissing = false;

  // Remove confirmation
  let confirmRemoveOpen = false;
  let confirmTarget: SuperadminMember | null = null;
  let removing = false;

  $: selfId = $bratraxUser?.id ?? null;

  onMount(loadData);

  async function loadData() {
    topError = "";
    try {
      const [superadmins, signups, requests, storeLinks] = await Promise.all([
        getSuperadmins(),
        listSignupInvitations(),
        listAccessRequests(),
        listStoreSignupLinks(),
      ]);
      data = superadmins;
      signupPending = signups.pending;
      accessRequests = requests.pending;
      storePending = storeLinks.pending;
      void loadInviteExpiryEnrollments();
    } catch (e: any) {
      topError = e?.message ?? "Failed to load superadmins";
    }
  }

  // Fan out one /superadmins/invitations/<token>/enrollments call per pending
  // invite to surface the invitation_expiry sequence status in the table.
  // Non-blocking — the column populates after the main render. Individual
  // failures fall back to "—".
  async function loadInviteExpiryEnrollments() {
    const next = new Map<string, SequenceEnrollment | null>();
    await Promise.all(
      signupPending.map(async (p) => {
        try {
          const { enrollments } = await listInvitationEnrollments(p.token);
          const expiry = enrollments.find(
            (e) => e.sequence_id === "invitation_expiry",
          );
          next.set(p.token, expiry ?? null);
        } catch {
          next.set(p.token, null);
        }
      }),
    );
    inviteExpiryByToken = next;
  }

  function expiryBadgeLabel(status: string | undefined | null): string {
    if (!status) return "—";
    return status;
  }

  function expiryBadgeClass(status: string | undefined | null): string {
    if (status === "active") return "text-bratrax-acid";
    if (status === "paused") return "text-yellow-400";
    if (status === "stopped") return "text-bratrax-tomato";
    return "text-bratrax-text-muted";
  }

  function openInvite() {
    inviteEmail = "";
    inviteName = "";
    inviteResultUrl = "";
    inviteResultExpiresAt = "";
    inviteResultEmail = "";
    inviteCopied = false;
    inviteOpen = true;
  }

  async function submitInvite() {
    const email = inviteEmail.trim().toLowerCase();
    if (!email || !email.includes("@")) {
      topError = "Valid email required";
      return;
    }
    inviteSaving = true;
    topError = "";
    try {
      const result = await inviteSuperadmin(email, inviteName.trim());
      inviteResultUrl = result.accept_url;
      inviteResultExpiresAt = result.expires_at ?? "";
      inviteResultEmail = result.email;
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to invite";
    } finally {
      inviteSaving = false;
    }
  }

  async function copyInviteUrl() {
    if (!inviteResultUrl) return;
    try {
      await navigator.clipboard.writeText(inviteResultUrl);
      inviteCopied = true;
      setTimeout(() => (inviteCopied = false), 2000);
    } catch {
      // ignore — user can manually copy from the visible field
    }
  }

  async function copyPendingUrl(p: SuperadminPendingInvite) {
    try {
      await navigator.clipboard.writeText(p.accept_url);
    } catch {
      // ignore
    }
  }

  function requestRemove(m: SuperadminMember) {
    confirmTarget = m;
    confirmRemoveOpen = true;
  }

  async function confirmRemove() {
    if (!confirmTarget) return;
    removing = true;
    topError = "";
    try {
      await removeSuperadmin(confirmTarget.id);
      confirmRemoveOpen = false;
      confirmTarget = null;
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to remove";
    } finally {
      removing = false;
    }
  }

  async function revokePending(p: SuperadminPendingInvite) {
    try {
      await revokeSuperadminInvitation(p.token);
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to revoke";
    }
  }

  // ---- Customer signup invitations -----------------------------------------

  function openSignupInvite() {
    signupInviteEmail = "";
    signupInviteRequiresPayment = true; // safer default
    signupInviteMultiStore = false; // safer default
    signupInviteResultUrl = "";
    signupInviteResultExpiresAt = "";
    signupInviteResultEmail = "";
    signupInviteResultRequiresPayment = true;
    signupInviteResultMultiStore = false;
    signupInviteCopied = false;
    signupInviteOpen = true;
  }

  async function submitSignupInvite() {
    const email = signupInviteEmail.trim().toLowerCase();
    if (!email || !email.includes("@")) {
      topError = "Valid email required";
      return;
    }
    signupInviteSaving = true;
    topError = "";
    try {
      const result = await createSignupInvite(
        email,
        signupInviteRequiresPayment,
        signupInviteMultiStore,
      );
      signupInviteResultUrl = result.accept_url;
      signupInviteResultExpiresAt = result.expires_at ?? "";
      signupInviteResultEmail = result.email;
      signupInviteResultRequiresPayment = result.requires_payment;
      signupInviteResultMultiStore = result.is_multi_store;
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to create signup link";
    } finally {
      signupInviteSaving = false;
    }
  }

  async function copySignupInviteUrl() {
    if (!signupInviteResultUrl) return;
    try {
      await navigator.clipboard.writeText(signupInviteResultUrl);
      signupInviteCopied = true;
      setTimeout(() => (signupInviteCopied = false), 2000);
    } catch {
      // ignore — user can manually copy
    }
  }

  async function copySignupPendingUrl(p: SignupPendingInvite) {
    try {
      await navigator.clipboard.writeText(p.accept_url);
    } catch {
      // ignore
    }
  }

  async function revokeSignupPending(p: SignupPendingInvite) {
    try {
      await revokeSignupInvitation(p.token);
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to revoke";
    }
  }

  // ---- Store signup links --------------------------------------------------

  function openStoreLink() {
    storeLinkStore = "shopify";
    storeLinkMaxUsers = 10;
    storeLinkResultUrl = "";
    storeLinkResultStore = "shopify";
    storeLinkResultMaxUsers = 0;
    storeLinkResultExpiresAt = "";
    storeLinkCopied = false;
    storeLinkOpen = true;
  }

  async function submitStoreLink() {
    const max = Number(storeLinkMaxUsers);
    if (!Number.isInteger(max) || max < 1) {
      topError = "Max users must be a positive whole number";
      return;
    }
    storeLinkSaving = true;
    topError = "";
    try {
      const result = await createStoreSignupLink(storeLinkStore, max);
      storeLinkResultUrl = result.join_url;
      storeLinkResultStore = result.store;
      storeLinkResultMaxUsers = result.max_users;
      storeLinkResultExpiresAt = result.expires_at ?? "";
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to create store signup link";
    } finally {
      storeLinkSaving = false;
    }
  }

  async function copyStoreLinkUrl() {
    if (!storeLinkResultUrl) return;
    try {
      await navigator.clipboard.writeText(storeLinkResultUrl);
      storeLinkCopied = true;
      setTimeout(() => (storeLinkCopied = false), 2000);
    } catch {
      // ignore — user can manually copy
    }
  }

  async function copyStorePendingUrl(p: StoreSignupLink) {
    try {
      await navigator.clipboard.writeText(p.join_url);
    } catch {
      // ignore
    }
  }

  async function revokeStorePending(p: StoreSignupLink) {
    try {
      await revokeStoreSignupLink(p.token);
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to revoke";
    }
  }

  // ---- Access requests -----------------------------------------------------

  async function approveRequest(r: AccessRequest) {
    if (accessActionId !== null) return;
    accessActionId = r.id;
    topError = "";
    try {
      const result = await approveAccessRequest(r.id);
      // Reuse the signup-invite result modal so the super_admin can copy
      // the freshly-generated link in the same UX they use for direct
      // invites. The list itself refreshes via loadData() so the request
      // disappears from this section and re-appears under "Customer Signup
      // Invitations". Access-request approvals always create a paying
      // invite (no inceptly toggle on the public access-request flow).
      signupInviteResultUrl = result.accept_url;
      signupInviteResultExpiresAt = result.expires_at ?? "";
      signupInviteResultEmail = result.email;
      signupInviteResultRequiresPayment = result.requires_payment ?? true;
      signupInviteResultMultiStore = result.is_multi_store ?? false;
      signupInviteCopied = false;
      signupInviteOpen = true;
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to approve request";
    } finally {
      accessActionId = null;
    }
  }

  function requestDismiss(r: AccessRequest) {
    dismissTarget = r;
    dismissOpen = true;
  }

  async function confirmDismiss() {
    if (!dismissTarget) return;
    dismissing = true;
    topError = "";
    try {
      await dismissAccessRequest(dismissTarget.id);
      dismissOpen = false;
      dismissTarget = null;
      await loadData();
    } catch (e: any) {
      topError = e?.message ?? "Failed to dismiss request";
    } finally {
      dismissing = false;
    }
  }

  function initialsOf(name: string, email: string): string {
    const src = (name || email || "").trim();
    if (!src) return "??";
    const parts = src.split(/[\s@.]+/).filter(Boolean);
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  }

  function formatDate(iso: string | null): string {
    if (!iso) return "";
    return new Date(iso).toLocaleDateString();
  }
</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="relative w-full max-w-3xl border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        SUPERADMINS
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Manage Bratrax superadmins
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Superadmins have access to every client. Invite a new one to share full control.
      </p>
    </div>

    {#if topError}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {topError}
      </div>
    {/if}

    <div class="flex items-baseline justify-between">
      <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
        Superadmins ({data.members.length})
      </h2>
      <button
        type="button"
        on:click={openInvite}
        class="btn-bratrax btn-primary btn-compact"
      >
        Invite superadmin
      </button>
    </div>

    <ul class="mt-3 flex flex-col gap-2">
      {#each data.members as m (m.id)}
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

          <span class="bratrax-status-pill">Superadmin</span>

          {#if m.id !== selfId}
            <button
              type="button"
              on:click={() => requestRemove(m)}
              class="btn-bratrax btn-destructive btn-compact"
            >
              Remove
            </button>
          {/if}
        </li>
      {/each}
    </ul>

    {#if data.pending.length > 0}
      <h2 class="mt-6 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
        Pending invitations ({data.pending.length})
      </h2>
      <ul class="mt-3 flex flex-col gap-2">
        {#each data.pending as p (p.token)}
          <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
            <div class="min-w-0 flex-1">
              <div class="truncate font-mono text-xs font-bold text-bratrax-text-headline">{p.email}</div>
              <div class="truncate font-mono text-[10px] text-bratrax-text-muted">
                Superadmin · expires {formatDate(p.expires_at)}
              </div>
            </div>
            <button
              type="button"
              on:click={() => copyPendingUrl(p)}
              class="btn-bratrax btn-neutral btn-compact"
            >
              Copy link
            </button>
            <button
              type="button"
              on:click={() => revokePending(p)}
              class="btn-bratrax btn-destructive btn-compact"
            >
              Revoke
            </button>
          </li>
        {/each}
      </ul>
    {/if}

    <!-- Access requests: visitors who hit the invite-only /signup screen and
         submitted their email. Approve generates a kind='signup' invite (and
         opens the existing signup-invite-result modal so the super_admin can
         copy the link); Dismiss closes without generating. -->
    <div class="mt-10 border-t border-bratrax-border pt-6">
      <div class="flex items-baseline justify-between">
        <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
          Access requests ({accessRequests.length})
        </h2>
      </div>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Pending requests from the public signup screen. Approving generates a single-use signup link to share manually.
      </p>

      {#if accessRequests.length === 0}
        <p class="mt-4 font-mono text-[10px] text-bratrax-text-muted">
          No pending requests.
        </p>
      {:else}
        <ul class="mt-3 flex flex-col gap-2">
          {#each accessRequests as r (r.id)}
            <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
              <div class="min-w-0 flex-1">
                <div class="truncate font-mono text-xs font-bold text-bratrax-text-headline">{r.email}</div>
                <div class="truncate font-mono text-[10px] text-bratrax-text-muted">
                  Requested {formatDate(r.created_at)}
                </div>
              </div>
              <button
                type="button"
                on:click={() => approveRequest(r)}
                disabled={accessActionId === r.id}
                class="btn-bratrax btn-primary btn-compact"
              >
                {accessActionId === r.id ? "Approving…" : "Approve"}
              </button>
              <button
                type="button"
                on:click={() => requestDismiss(r)}
                disabled={accessActionId === r.id}
                class="btn-bratrax btn-destructive btn-compact"
              >
                Dismiss
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>

    <!-- Customer signup invitations: pre-authorize someone to create a new
         Bratrax account. Used while ONLY_INVITATION_LINK=true gates public
         signup. The link expires in 7 days and is single-use. -->
    <div class="mt-10 border-t border-bratrax-border pt-6">
      <div class="flex items-baseline justify-between">
        <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
          Customer signup invitations
        </h2>
        <button
          type="button"
          on:click={openSignupInvite}
          class="btn-bratrax btn-primary btn-compact"
        >
          Generate signup link
        </button>
      </div>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Pre-authorize a new customer to create their own Bratrax account. They'll set their password and company name, then onboard normally.
      </p>

      {#if signupPending.length === 0}
        <p class="mt-4 font-mono text-[10px] text-bratrax-text-muted">
          No pending signup invitations.
        </p>
      {:else}
        <ul class="mt-3 flex flex-col gap-2">
          {#each signupPending as p (p.token)}
            <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="truncate font-mono text-xs font-bold text-bratrax-text-headline">{p.email}</span>
                  <span class="shrink-0 border px-1.5 py-0.5 font-mono text-[9px] {p.requires_payment ? 'border-bratrax-acid/40 text-bratrax-acid' : 'border-bratrax-border text-bratrax-text-muted'}">
                    {p.requires_payment ? "REQUIRES PAYMENT" : "FREE (INCEPTLY)"}
                  </span>
                  {#if p.is_multi_store}
                    <span class="shrink-0 border border-bratrax-acid/40 px-1.5 py-0.5 font-mono text-[9px] text-bratrax-acid">
                      MULTI-STORE
                    </span>
                  {/if}
                </div>
                <div class="truncate font-mono text-[10px] text-bratrax-text-muted">
                  Signup link · expires {formatDate(p.expires_at)}
                  · expiry-reminder
                  <span class="font-bold {expiryBadgeClass(inviteExpiryByToken.get(p.token)?.status)}">
                    {expiryBadgeLabel(inviteExpiryByToken.get(p.token)?.status)}
                  </span>
                </div>
              </div>
              <button
                type="button"
                on:click={() => copySignupPendingUrl(p)}
                class="btn-bratrax btn-neutral btn-compact"
              >
                Copy link
              </button>
              <button
                type="button"
                on:click={() => revokeSignupPending(p)}
                class="btn-bratrax btn-destructive btn-compact"
              >
                Revoke
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>

    <!-- Store signup links: multi-use, store-locked, capped links for a batch of
         new customers (distributed via external email listings). No email is sent
         — copy the URL out. Each use = one client; the link dies at the cap or
         after 90 days. -->
    <div class="mt-10 border-t border-bratrax-border pt-6">
      <div class="flex items-baseline justify-between">
        <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
          Store signup links
        </h2>
        <button
          type="button"
          on:click={openStoreLink}
          class="btn-bratrax btn-primary btn-compact"
        >
          Generate store signup link
        </button>
      </div>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        A reusable link for a batch of new customers. Pick a store platform and a max number of signups. Anyone can use it with their own email until the cap is hit (or 90 days pass); the chosen store is preselected during their onboarding.
      </p>

      {#if storePending.length === 0}
        <p class="mt-4 font-mono text-[10px] text-bratrax-text-muted">
          No active store signup links.
        </p>
      {:else}
        <ul class="mt-3 flex flex-col gap-2">
          {#each storePending as p (p.token)}
            <li class="flex items-center gap-3 border border-bratrax-border bg-bratrax-bg px-4 py-3">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="shrink-0 border border-bratrax-acid/40 px-1.5 py-0.5 font-mono text-[9px] uppercase text-bratrax-acid">
                    {p.store_platform}
                  </span>
                  <span class="font-mono text-xs font-bold text-bratrax-text-headline">
                    {p.used_count} / {p.max_uses} used
                  </span>
                </div>
                <div class="truncate font-mono text-[10px] text-bratrax-text-muted">
                  {p.join_url} · expires {formatDate(p.expires_at)}
                </div>
              </div>
              <button
                type="button"
                on:click={() => copyStorePendingUrl(p)}
                class="btn-bratrax btn-neutral btn-compact"
              >
                Copy link
              </button>
              <button
                type="button"
                on:click={() => revokeStorePending(p)}
                class="btn-bratrax btn-destructive btn-compact"
              >
                Revoke
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
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

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        INVITE SUPERADMIN
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Invite a new superadmin
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        They'll get access to every client. Share the link with them privately.
      </p>

      {#if !inviteResultUrl}
        <div class="mt-4 flex flex-col gap-3">
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Email
            </label>
            <input
              type="email"
              bind:value={inviteEmail}
              class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
              placeholder="superadmin@bratrax.com"
            />
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Name (optional)
            </label>
            <input
              type="text"
              bind:value={inviteName}
              class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            />
          </div>
        </div>

        <div class="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            on:click={() => (inviteOpen = false)}
            class="btn-bratrax btn-neutral btn-compact"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={submitInvite}
            disabled={inviteSaving}
            class="btn-bratrax btn-primary btn-compact"
          >
            {inviteSaving ? "Creating…" : "Create invite link"}
          </button>
        </div>
      {:else}
        <div class="mt-4 flex flex-col gap-3">
          <div class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Link for {inviteResultEmail}
          </div>
          <div class="flex items-center gap-2">
            <input
              type="text"
              readonly
              value={inviteResultUrl}
              class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            />
            <button
              type="button"
              on:click={copyInviteUrl}
              class="btn-bratrax btn-primary btn-compact"
            >
              {inviteCopied ? "Copied" : "Copy"}
            </button>
          </div>
          <p class="font-mono text-[10px] text-bratrax-text-muted">
            Expires {formatDate(inviteResultExpiresAt)}.
          </p>
        </div>

        <div class="mt-6 flex items-center justify-end">
          <button
            type="button"
            on:click={() => (inviteOpen = false)}
            class="btn-bratrax btn-neutral btn-compact"
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
        They'll lose access to every client immediately. You can re-invite them later.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          type="button"
          on:click={() => (confirmRemoveOpen = false)}
          class="btn-bratrax btn-neutral btn-compact"
        >
          Cancel
        </button>
        <button
          type="button"
          on:click={confirmRemove}
          disabled={removing}
          class="btn-bratrax btn-destructive btn-compact"
        >
          {removing ? "Removing…" : "Remove"}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Signup-invite modal -->
{#if signupInviteOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (signupInviteOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (signupInviteOpen = false)}
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
        SIGNUP LINK
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Generate a customer signup link
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Single-use, expires in 7 days. Share with the customer privately — they set their password and company name and start onboarding.
      </p>

      {#if !signupInviteResultUrl}
        <div class="mt-4 flex flex-col gap-4">
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              User type
            </label>
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                on:click={() => (signupInviteRequiresPayment = true)}
                class="border px-3 py-3 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {signupInviteRequiresPayment ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-acid/40'}"
              >
                Real user
                <span class="block text-[9px] font-normal normal-case tracking-normal text-bratrax-text-muted/70">Requires payment</span>
              </button>
              <button
                type="button"
                on:click={() => (signupInviteRequiresPayment = false)}
                class="border px-3 py-3 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {!signupInviteRequiresPayment ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-acid/40'}"
              >
                Inceptly user
                <span class="block text-[9px] font-normal normal-case tracking-normal text-bratrax-text-muted/70">Free, internal team</span>
              </button>
            </div>
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Account type
            </label>
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                on:click={() => (signupInviteMultiStore = false)}
                class="border px-3 py-3 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {!signupInviteMultiStore ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-acid/40'}"
              >
                Single store
                <span class="block text-[9px] font-normal normal-case tracking-normal text-bratrax-text-muted/70">One Shopify brand</span>
              </button>
              <button
                type="button"
                on:click={() => (signupInviteMultiStore = true)}
                class="border px-3 py-3 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {signupInviteMultiStore ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-acid/40'}"
              >
                Multi-store
                <span class="block text-[9px] font-normal normal-case tracking-normal text-bratrax-text-muted/70">Agency w/ multiple brands</span>
              </button>
            </div>
          </div>
          <div>
            <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Customer email
            </label>
            <input
              type="email"
              bind:value={signupInviteEmail}
              class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
              placeholder="customer@example.com"
            />
          </div>
        </div>

        <div class="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            on:click={() => (signupInviteOpen = false)}
            class="btn-bratrax btn-neutral btn-compact"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={submitSignupInvite}
            disabled={signupInviteSaving}
            class="btn-bratrax btn-primary btn-compact"
          >
            {signupInviteSaving ? "Creating…" : "Create signup link"}
          </button>
        </div>
      {:else}
        <div class="mt-4 flex flex-col gap-3">
          <div class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Link for {signupInviteResultEmail}
            <span class="ml-2 inline-block border px-1.5 py-0.5 font-mono text-[9px] {signupInviteResultRequiresPayment ? 'border-bratrax-acid/40 text-bratrax-acid' : 'border-bratrax-border text-bratrax-text-muted'}">
              {signupInviteResultRequiresPayment ? "REQUIRES PAYMENT" : "FREE (INCEPTLY)"}
            </span>
            {#if signupInviteResultMultiStore}
              <span class="ml-1 inline-block border border-bratrax-acid/40 px-1.5 py-0.5 font-mono text-[9px] text-bratrax-acid">
                MULTI-STORE
              </span>
            {/if}
          </div>
          <div class="flex items-center gap-2">
            <input
              type="text"
              readonly
              value={signupInviteResultUrl}
              class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            />
            <button
              type="button"
              on:click={copySignupInviteUrl}
              class="btn-bratrax btn-primary btn-compact"
            >
              {signupInviteCopied ? "Copied" : "Copy"}
            </button>
          </div>
          <p class="font-mono text-[10px] text-bratrax-text-muted">
            Expires {formatDate(signupInviteResultExpiresAt)}.
          </p>
        </div>

        <div class="mt-6 flex items-center justify-end">
          <button
            type="button"
            on:click={() => (signupInviteOpen = false)}
            class="btn-bratrax btn-neutral btn-compact"
          >
            Done
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Store-signup-link modal -->
{#if storeLinkOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (storeLinkOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (storeLinkOpen = false)}
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
        STORE SIGNUP LINK
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Generate a store signup link
      </h2>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Multi-use, expires in 90 days or when the cap is hit. Distribute via your email listings — every signup defaults to a paying, single-store account with this store preselected.
      </p>

      {#if !storeLinkResultUrl}
        <div class="mt-4 flex flex-col gap-4">
          <div>
            <div class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
              Store platform
            </div>
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                on:click={() => (storeLinkStore = "shopify")}
                class="border px-3 py-3 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {storeLinkStore === 'shopify' ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-acid/40'}"
              >
                Shopify
              </button>
              <button
                type="button"
                on:click={() => (storeLinkStore = "woocommerce")}
                class="border px-3 py-3 font-mono text-[11px] font-bold uppercase tracking-wider transition-colors {storeLinkStore === 'woocommerce' ? 'border-bratrax-acid text-bratrax-acid bg-bratrax-acid bg-opacity-10' : 'border-bratrax-border text-bratrax-text-muted hover:border-bratrax-acid/40'}"
              >
                WooCommerce
              </button>
            </div>
          </div>
          <div>
            <label
              for="store-link-max"
              class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
            >
              Max users (clients)
            </label>
            <input
              id="store-link-max"
              type="number"
              min="1"
              step="1"
              bind:value={storeLinkMaxUsers}
              class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
              placeholder="10"
            />
          </div>
        </div>

        <div class="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            on:click={() => (storeLinkOpen = false)}
            class="btn-bratrax btn-neutral btn-compact"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={submitStoreLink}
            disabled={storeLinkSaving}
            class="btn-bratrax btn-primary btn-compact"
          >
            {storeLinkSaving ? "Creating…" : "Create store link"}
          </button>
        </div>
      {:else}
        <div class="mt-4 flex flex-col gap-3">
          <div class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            {storeLinkResultStore} · up to {storeLinkResultMaxUsers} users
          </div>
          <div class="flex items-center gap-2">
            <input
              type="text"
              readonly
              value={storeLinkResultUrl}
              class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-body focus:border-bratrax-acid focus:outline-none"
            />
            <button
              type="button"
              on:click={copyStoreLinkUrl}
              class="btn-bratrax btn-primary btn-compact"
            >
              {storeLinkCopied ? "Copied" : "Copy"}
            </button>
          </div>
          <p class="font-mono text-[10px] text-bratrax-text-muted">
            Expires {formatDate(storeLinkResultExpiresAt)}.
          </p>
        </div>

        <div class="mt-6 flex items-center justify-end">
          <button
            type="button"
            on:click={() => (storeLinkOpen = false)}
            class="btn-bratrax btn-neutral btn-compact"
          >
            Done
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Dismiss access-request confirmation modal -->
{#if dismissOpen && dismissTarget}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (dismissOpen = false)}
    on:keydown={(e) => e.key === "Escape" && (dismissOpen = false)}
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
        Confirm dismiss
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Dismiss request from {dismissTarget.email}?
      </h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        The request will be marked dismissed and disappear from this list. They can submit a new request later if needed.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          type="button"
          on:click={() => (dismissOpen = false)}
          class="btn-bratrax btn-neutral btn-compact"
        >
          Cancel
        </button>
        <button
          type="button"
          on:click={confirmDismiss}
          disabled={dismissing}
          class="btn-bratrax btn-destructive btn-compact"
        >
          {dismissing ? "Dismissing…" : "Dismiss"}
        </button>
      </div>
    </div>
  </div>
{/if}
