<script lang="ts">
  import { onMount } from "svelte";
  import {
    getSuperadmins,
    inviteSuperadmin,
    removeSuperadmin,
    revokeSuperadminInvitation,
    type SuperadminListResponse,
    type SuperadminMember,
    type SuperadminPendingInvite,
  } from "$lib/bratrax/superadmins/api";
  import { bratraxUser } from "$lib/bratrax/auth-store";

  let data: SuperadminListResponse = { members: [], pending: [] };
  let topError = "";

  // Invite modal state
  let inviteOpen = false;
  let inviteEmail = "";
  let inviteName = "";
  let inviteSaving = false;
  let inviteResultUrl = "";
  let inviteResultExpiresAt = "";
  let inviteResultEmail = "";
  let inviteCopied = false;

  // Remove confirmation
  let confirmRemoveOpen = false;
  let confirmTarget: SuperadminMember | null = null;
  let removing = false;

  $: selfId = $bratraxUser?.id ?? null;

  onMount(loadData);

  async function loadData() {
    topError = "";
    try {
      data = await getSuperadmins();
    } catch (e: any) {
      topError = e?.message ?? "Failed to load superadmins";
    }
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
