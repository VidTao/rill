<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { acceptInvitation, getInvitation } from "$lib/bratrax/settings/api";
  import type { InvitationPreview } from "$lib/bratrax/settings/types";

  let invite: InvitationPreview | null = null;
  let loadError = "";
  let loading = true;

  let name = "";
  let password = "";
  let confirm = "";
  let submitError = "";
  let submitting = false;

  $: token = $page.params.token;

  $: passwordsMatch = password.length > 0 && password === confirm;
  $: canSubmit = !!invite && !invite.expired && !invite.accepted && password.length >= 8 && passwordsMatch;

  onMount(async () => {
    try {
      invite = await getInvitation(token);
    } catch (e: any) {
      loadError = e.message ?? "Invitation not found";
    } finally {
      loading = false;
    }
  });

  async function submit() {
    if (!canSubmit) return;
    submitError = "";
    submitting = true;
    try {
      await acceptInvitation(token, password, name.trim());
      await goto("/login?invited=1");
    } catch (e: any) {
      submitError = e.message ?? "Failed to accept invitation";
    } finally {
      submitting = false;
    }
  }

  function roleLabel(r: string): string {
    if (r === "admin") return "Admin";
    if (r === "viewer") return "Viewer";
    return r;
  }
</script>

<div class="flex h-screen w-screen items-center justify-center bg-bratrax-bg">
  <div class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    {#if loading}
      <p class="font-mono text-xs text-bratrax-text-muted">Loading invitation…</p>
    {:else if loadError}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Invitation
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">Link not valid</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">{loadError}</p>
      <a
        href="/login"
        class="mt-6 inline-block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-acid hover:underline"
      >
        Go to login →
      </a>
    {:else if invite && invite.accepted}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
        Already accepted
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">This invitation has already been used</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Log in with the email <strong>{invite.email}</strong>, or ask {invite.inviter_name || "your admin"}
        for a fresh invite.
      </p>
      <a
        href="/login"
        class="mt-6 inline-block bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
      >
        Go to login
      </a>
    {:else if invite && invite.expired}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Expired
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">This invitation has expired</h1>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        Ask {invite.inviter_name || "your admin"} at <strong>{invite.company_name}</strong> for a new invite link.
      </p>
    {:else if invite}
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        You're invited
      </div>
      <h1 class="text-xl font-black text-bratrax-text-headline">
        Join <span class="font-serif italic text-bratrax-acid">{invite.company_name}</span>
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Joining as <strong>{roleLabel(invite.role)}</strong>{invite.inviter_name ? `, invited by ${invite.inviter_name}` : ""}.
      </p>

      <form on:submit|preventDefault={submit} class="mt-6 flex flex-col gap-3">
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Email
          </label>
          <input
            type="email"
            value={invite.email}
            readonly
            class="w-full cursor-not-allowed border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-muted opacity-80"
          />
        </div>
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Your name
          </label>
          <input
            type="text"
            bind:value={name}
            placeholder="Joe Smith"
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          />
        </div>
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Password
          </label>
          <input
            type="password"
            bind:value={password}
            placeholder="At least 8 characters"
            minlength="8"
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          />
        </div>
        <div>
          <label class="mb-1.5 block font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted">
            Confirm password
          </label>
          <input
            type="password"
            bind:value={confirm}
            class="w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 text-sm text-bratrax-text-body placeholder-bratrax-text-muted/50 focus:border-bratrax-acid focus:outline-none"
          />
          {#if confirm.length > 0 && !passwordsMatch}
            <p class="mt-1 font-mono text-[10px] text-bratrax-tomato">Passwords don't match</p>
          {/if}
        </div>

        {#if submitError}
          <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
            {submitError}
          </div>
        {/if}

        <button
          type="submit"
          disabled={!canSubmit || submitting}
          class="mt-2 w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {submitting ? "Creating account…" : "Accept invitation"}
        </button>
      </form>
    {/if}
  </div>
</div>
