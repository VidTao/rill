<script lang="ts">
  import { onMount } from "svelte";
  import EmailDetailModal from "$lib/bratrax/superadmins/EmailDetailModal.svelte";
  import {
    listEmailLog,
    listSequences,
    type EmailKind,
    type EmailLogEntry,
    type EmailStatus,
    type SequenceMeta,
  } from "$lib/bratrax/superadmins/api";

  let emails: EmailLogEntry[] = [];
  let total = 0;
  let sequences: SequenceMeta[] = [];
  let loading = true;
  let error = "";
  let offset = 0;
  const PAGE = 100;
  let hasMore = true;

  // Filter state
  let kind = "";
  let sequenceId = "";
  let status = "";
  let recipientSearch = "";
  let recipientDebounce: ReturnType<typeof setTimeout> | null = null;

  let selected: EmailLogEntry | null = null;

  onMount(async () => {
    try {
      sequences = (await listSequences()).sequences;
    } catch {
      // ignore — sequence dropdown just shows the default item
    }
    await reload();
  });

  async function reload() {
    error = "";
    loading = true;
    offset = 0;
    try {
      const resp = await listEmailLog({
        kind: (kind || undefined) as EmailKind | undefined,
        sequence_id: sequenceId || undefined,
        status: (status || undefined) as EmailStatus | undefined,
        recipient_search: recipientSearch || undefined,
        limit: PAGE,
        offset: 0,
      });
      emails = resp.emails;
      total = resp.total ?? emails.length;
      offset = resp.emails.length;
      hasMore = resp.emails.length === PAGE && offset < total;
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
      const resp = await listEmailLog({
        kind: (kind || undefined) as EmailKind | undefined,
        sequence_id: sequenceId || undefined,
        status: (status || undefined) as EmailStatus | undefined,
        recipient_search: recipientSearch || undefined,
        limit: PAGE,
        offset,
      });
      emails = [...emails, ...resp.emails];
      offset += resp.emails.length;
      hasMore = resp.emails.length === PAGE && offset < (resp.total ?? offset);
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function debouncedRecipientChange() {
    if (recipientDebounce) clearTimeout(recipientDebounce);
    recipientDebounce = setTimeout(reload, 300);
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

  function statusClass(s: string): string {
    if (["delivered", "opened", "clicked"].includes(s)) return "text-bratrax-acid";
    if (["bounced", "dropped", "failed", "spamreport"].includes(s)) return "text-bratrax-tomato";
    return "text-bratrax-text-muted";
  }

  const KINDS: { value: string; label: string }[] = [
    { value: "", label: "All kinds" },
    { value: "transactional", label: "Transactional" },
    { value: "lifecycle", label: "Lifecycle" },
  ];

  const STATUSES: { value: string; label: string }[] = [
    { value: "", label: "All statuses" },
    { value: "queued", label: "queued" },
    { value: "sent", label: "sent" },
    { value: "delivered", label: "delivered" },
    { value: "opened", label: "opened" },
    { value: "clicked", label: "clicked" },
    { value: "bounced", label: "bounced" },
    { value: "dropped", label: "dropped" },
    { value: "failed", label: "failed" },
  ];
</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="w-full max-w-6xl px-4">
    <a href="/superadmins" class="font-mono text-[11px] text-bratrax-acid hover:underline">
      ← Super-admin
    </a>

    <h1 class="mt-4 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
      System-wide email log
    </h1>

    <section class="mt-3 border border-bratrax-border bg-bratrax-surface p-5">
      <!-- Filters -->
      <div class="mb-4 grid grid-cols-1 gap-3 md:grid-cols-4">
        <select bind:value={kind} on:change={reload} class="filter-input">
          {#each KINDS as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <select bind:value={sequenceId} on:change={reload} class="filter-input">
          <option value="">All sequences</option>
          {#each sequences as s}
            <option value={s.sequence_id}>{s.display_name}</option>
          {/each}
        </select>

        <select bind:value={status} on:change={reload} class="filter-input">
          {#each STATUSES as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <input
          type="text"
          bind:value={recipientSearch}
          on:input={debouncedRecipientChange}
          placeholder="Recipient email (LIKE)"
          class="filter-input"
        />
      </div>

      {#if error}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {error}
        </div>
      {/if}

      <div class="mb-3 font-mono text-[11px] text-bratrax-text-muted">
        {total.toLocaleString()} total · showing {emails.length}
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left font-mono text-xs">
          <thead class="text-bratrax-text-muted">
            <tr class="border-b border-bratrax-border">
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Sent</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Recipient</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Template</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Kind</th>
              <th class="py-2 pr-3 font-mono text-[10px] font-bold uppercase tracking-[1.5px]">Sequence</th>
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
                <td class="py-2 pr-3 text-bratrax-text-primary">{e.recipient_email}</td>
                <td class="py-2 pr-3 text-bratrax-text-primary">{e.template_key}</td>
                <td class="py-2 pr-3 text-bratrax-text-muted">{e.kind}</td>
                <td class="py-2 pr-3 text-bratrax-text-muted">{e.sequence_id ?? "—"}</td>
                <td class="py-2 pr-3 font-bold {statusClass(e.status)}">{e.status}</td>
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

{#if selected}
  <EmailDetailModal email={selected} on:close={() => (selected = null)} />
{/if}

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
