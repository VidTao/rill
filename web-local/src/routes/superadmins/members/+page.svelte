<script lang="ts">
  import { onMount } from "svelte";
  import {
    listDirectoryMembers,
    type DirectoryMember,
  } from "$lib/bratrax/superadmins/api";

  // Directory members: free Bratrax accounts made on bratrax.com/stores (and
  // demo visitors who signed in there). Viewers on the shared demo workspace
  // until they press Start attribution; then `converted` is set and their
  // workspace shows up in Overview like any other client.

  const PAGE = 100;
  let members: DirectoryMember[] = [];
  let total = 0;
  let offset = 0;
  let hasMore = false;
  let loading = true;
  let error = "";

  let q = "";
  let onlyConverted = false;
  let debounce: ReturnType<typeof setTimeout> | null = null;

  onMount(reload);

  async function fetchPage(from: number) {
    return listDirectoryMembers({ q: q.trim() || undefined, converted: onlyConverted, limit: PAGE, offset: from });
  }

  async function reload() {
    error = "";
    loading = true;
    try {
      const resp = await fetchPage(0);
      members = resp.members;
      total = resp.total;
      offset = resp.members.length;
      hasMore = offset < total;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function loadMore() {
    if (!hasMore || loading) return;
    loading = true;
    try {
      const resp = await fetchPage(offset);
      members = [...members, ...resp.members];
      offset += resp.members.length;
      hasMore = resp.members.length > 0 && offset < resp.total;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function onSearch() {
    if (debounce) clearTimeout(debounce);
    debounce = setTimeout(reload, 300);
  }

  function fmt(iso: string | null): string {
    if (!iso) return "—";
    return new Date(iso).toLocaleString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function storeHref(url: string): string {
    const bare = url.trim().replace(/^https?:\/\//i, "");
    return bare ? `https://${bare}` : "";
  }
</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="w-full max-w-[1400px] px-4">
    <a href="/superadmins" class="font-mono text-[11px] text-bratrax-acid hover:underline">
      ← Super-admin
    </a>

    <h1 class="mt-4 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
      Directory members
    </h1>
    <p class="mt-1 text-sm font-light text-bratrax-text-body">
      Free accounts from Bratrax Stores. They see the demo workspace until they press Start attribution.
    </p>

    <section class="mt-3 border border-bratrax-border bg-bratrax-surface p-5">
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <input
          type="text"
          bind:value={q}
          on:input={onSearch}
          placeholder="Email contains…"
          aria-label="Search members by email"
          class="filter-input min-w-[260px]"
        />
        <label class="flex items-center gap-2 font-mono text-[11px] text-bratrax-text-body">
          <input type="checkbox" bind:checked={onlyConverted} on:change={reload} class="accent-bratrax-acid" />
          Started attribution only
        </label>
      </div>

      {#if error}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {error}
        </div>
      {/if}

      <div class="mb-3 font-mono text-[11px] text-bratrax-text-muted">
        {total.toLocaleString()} total · showing {members.length}
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left font-mono text-xs">
          <thead class="text-bratrax-text-muted">
            <tr class="border-b border-bratrax-border">
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Email</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Signed up on</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Store URL</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Country</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Newsletter</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Joined</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Last sign-in</th>
              <th class="py-2 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Started attribution</th>
            </tr>
          </thead>
          <tbody>
            {#each members as m (m.user_id)}
              <tr class="border-b border-bratrax-border">
                <td class="py-2 pr-3 text-bratrax-text-primary">{m.email}</td>
                <td class="py-2 pr-3 text-bratrax-text-muted">{m.signup_page || "—"}</td>
                <td class="py-2 pr-3">
                  {#if m.store_url}
                    <a href={storeHref(m.store_url)} target="_blank" rel="noopener noreferrer nofollow" class="text-bratrax-acid hover:underline">{m.store_url}</a>
                  {:else}
                    <span class="text-bratrax-text-muted">—</span>
                  {/if}
                </td>
                <td class="py-2 pr-3 text-bratrax-text-muted">{m.country || "—"}</td>
                <td class="py-2 pr-3 text-bratrax-text-muted">{m.marketing_consent ? "yes" : "no"}</td>
                <td class="py-2 pr-3 text-bratrax-text-primary">{fmt(m.created_at)}</td>
                <td class="py-2 pr-3 text-bratrax-text-primary">{fmt(m.last_login_at)}</td>
                <td class="py-2 {m.converted_at ? 'font-bold text-bratrax-acid' : 'text-bratrax-text-muted'}">
                  {m.converted_at ? fmt(m.converted_at) : "—"}
                </td>
              </tr>
            {:else}
              {#if !loading}
                <tr><td colspan="8" class="py-6 text-center text-bratrax-text-muted">No members yet.</td></tr>
              {/if}
            {/each}
          </tbody>
        </table>
      </div>

      {#if hasMore}
        <div class="mt-3 text-center">
          <button
            type="button"
            on:click={loadMore}
            disabled={loading}
            class="font-mono text-[11px] text-bratrax-acid hover:underline disabled:opacity-50"
          >
            {loading ? "Loading…" : "Load more"}
          </button>
        </div>
      {/if}
    </section>
  </div>
</div>

<style>
  .filter-input {
    font-family: "Outfit", sans-serif;
    border: 1px solid var(--bratrax-border, #2a2a2a);
    background: var(--bratrax-bg, #0a0a0a);
    color: var(--bratrax-text-primary, #f5f5f7);
    padding: 8px 10px;
    font-size: 12px;
  }
  .filter-input:focus {
    outline: none;
    border-color: #d4ff00;
  }
</style>
