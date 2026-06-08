<script lang="ts">
  import { onMount } from "svelte";
  import EmailDetailModal from "./EmailDetailModal.svelte";
  import {
    listClientEmails,
    type EmailLogEntry,
  } from "./api";

  export let clientId: string;

  let emails: EmailLogEntry[] = [];
  let loading = true;
  let loadingMore = false;
  let error = "";
  let offset = 0;
  let hasMore = true;
  const PAGE = 50;

  let selected: EmailLogEntry | null = null;

  onMount(() => loadFirst());

  async function loadFirst() {
    error = "";
    loading = true;
    offset = 0;
    try {
      const resp = await listClientEmails(clientId, { limit: PAGE, offset: 0 });
      emails = resp.emails;
      offset = resp.emails.length;
      hasMore = resp.emails.length === PAGE;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function loadMore() {
    if (loadingMore || !hasMore) return;
    loadingMore = true;
    try {
      const resp = await listClientEmails(clientId, { limit: PAGE, offset });
      emails = [...emails, ...resp.emails];
      offset += resp.emails.length;
      hasMore = resp.emails.length === PAGE;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loadingMore = false;
    }
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

  function statusBadge(status: string): string {
    if (["delivered", "opened", "clicked"].includes(status)) return "text-bratrax-acid";
    if (["bounced", "dropped", "failed", "spamreport"].includes(status)) return "text-bratrax-tomato";
    return "text-bratrax-text-muted";
  }
</script>

<section class="mt-4 border border-bratrax-border bg-bratrax-surface p-5">
  <h2 class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
    Email history
  </h2>

  {#if error}
    <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
      {error}
    </div>
  {:else if loading}
    <p class="font-mono text-[11px] text-bratrax-text-muted">Loading…</p>
  {:else if emails.length === 0}
    <p class="font-mono text-[11px] text-bratrax-text-muted">No emails sent yet.</p>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-left font-mono text-xs">
        <thead class="text-bratrax-text-muted">
          <tr class="border-b border-bratrax-border">
            <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Sent</th>
            <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Template</th>
            <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Kind</th>
            <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Status</th>
            <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Opens</th>
            <th class="py-2 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Clicks</th>
          </tr>
        </thead>
        <tbody>
          {#each emails as e (e.id)}
            <tr
              class="cursor-pointer border-b border-bratrax-border hover:bg-bratrax-bg"
              on:click={() => (selected = e)}
            >
              <td class="py-2 pr-3 text-bratrax-text-primary">{fmt(e.sent_at)}</td>
              <td class="py-2 pr-3 text-bratrax-text-primary">{e.template_key}</td>
              <td class="py-2 pr-3 text-bratrax-text-muted">{e.kind}</td>
              <td class="py-2 pr-3 font-bold {statusBadge(e.status)}">{e.status}</td>
              <td class="py-2 pr-3 text-bratrax-text-primary">{e.open_count}</td>
              <td class="py-2 text-bratrax-text-primary">{e.click_count}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    {#if hasMore}
      <div class="mt-3 text-center">
        <button
          type="button"
          on:click={loadMore}
          disabled={loadingMore}
          class="font-mono text-[11px] text-bratrax-acid hover:underline disabled:opacity-50"
        >
          {loadingMore ? "Loading…" : "Load more"}
        </button>
      </div>
    {/if}
  {/if}
</section>

{#if selected}
  <EmailDetailModal email={selected} on:close={() => (selected = null)} />
{/if}
