<script lang="ts">
  import { onMount } from "svelte";
  import {
    listClientEnrollments,
    listSequences,
    enrollClient,
    patchEnrollment,
    type EnrollmentAction,
    type SequenceEnrollment,
    type SequenceMeta,
  } from "./api";

  export let clientId: string;

  let enrollments: SequenceEnrollment[] = [];
  let sequences: SequenceMeta[] = [];
  let loading = true;
  let error = "";

  // Manual-enroll modal state
  let showEnrollModal = false;
  let pickedSequenceId = "";
  let forceEnroll = false;
  let enrollError = "";
  let enrollLoading = false;

  onMount(load);

  async function load() {
    error = "";
    loading = true;
    try {
      const [enr, seq] = await Promise.all([
        listClientEnrollments(clientId),
        listSequences(),
      ]);
      enrollments = enr.enrollments;
      sequences = seq.sequences.filter((s) => s.subject_type === "client");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function act(enrollmentId: number, action: EnrollmentAction) {
    try {
      const { enrollment } = await patchEnrollment(enrollmentId, action);
      enrollments = enrollments.map((e) => (e.id === enrollment.id ? enrollment : e));
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e));
    }
  }

  async function submitEnroll() {
    enrollError = "";
    if (!pickedSequenceId) {
      enrollError = "Pick a sequence first.";
      return;
    }
    enrollLoading = true;
    try {
      await enrollClient(clientId, pickedSequenceId, forceEnroll);
      showEnrollModal = false;
      pickedSequenceId = "";
      forceEnroll = false;
      await load();
    } catch (e) {
      enrollError = e instanceof Error ? e.message : String(e);
    } finally {
      enrollLoading = false;
    }
  }

  function fmtDelta(iso: string | null): string {
    if (!iso) return "—";
    const ms = new Date(iso).getTime() - Date.now();
    const h = Math.round(ms / 3_600_000);
    if (Math.abs(h) < 1) return ms > 0 ? "due soon" : "due now";
    if (h > 0) return `due in ${h}h`;
    return `${-h}h ago`;
  }

  function fmtAgo(iso: string | null): string {
    if (!iso) return "—";
    const h = Math.round((Date.now() - new Date(iso).getTime()) / 3_600_000);
    if (h < 1) return "just now";
    if (h < 48) return `${h}h ago`;
    return `${Math.round(h / 24)}d ago`;
  }

  function statusDot(status: string): string {
    if (status === "active") return "bg-bratrax-acid";
    if (status === "paused") return "bg-yellow-400";
    if (status === "completed") return "bg-bratrax-text-muted";
    return "bg-bratrax-tomato"; // stopped
  }
</script>

<section class="mt-4 border border-bratrax-border bg-bratrax-surface p-5">
  <div class="mb-3 flex items-baseline justify-between">
    <h2 class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
      Email automations
    </h2>
    <button
      type="button"
      on:click={() => (showEnrollModal = true)}
      class="font-mono text-[11px] text-bratrax-acid hover:underline"
    >
      + Enroll in sequence
    </button>
  </div>

  {#if error}
    <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
      {error}
    </div>
  {:else if loading}
    <p class="font-mono text-[11px] text-bratrax-text-muted">Loading…</p>
  {:else if enrollments.length === 0}
    <p class="font-mono text-[11px] text-bratrax-text-muted">
      No active or past enrollments.
    </p>
  {:else}
    <ul class="space-y-3">
      {#each enrollments as e (e.id)}
        <li class="border border-bratrax-border bg-bratrax-bg p-3">
          <div class="flex items-start gap-3">
            <span class="mt-1 h-2 w-2 rounded-full {statusDot(e.status)}"></span>
            <div class="flex-1">
              <div class="flex items-baseline justify-between">
                <span class="font-mono text-xs font-bold text-bratrax-text-primary">
                  {e.sequence_display_name}
                </span>
                <span class="font-mono text-[10px] uppercase text-bratrax-text-muted">
                  {e.status}{e.stopped_reason ? ` · ${e.stopped_reason}` : ""}
                </span>
              </div>
              <div class="mt-1 font-mono text-[11px] text-bratrax-text-muted">
                email {e.next_email_index}/{e.total_emails}
                {#if e.next_email_due_at && e.status === "active"}
                  · {fmtDelta(e.next_email_due_at)}
                {/if}
                · {e.emails_sent_count} sent
                · last event {fmtAgo(e.last_event_at)}
              </div>
              <div class="mt-2 flex flex-wrap gap-2">
                {#if e.status === "active"}
                  <button type="button" on:click={() => act(e.id, "pause")} class="action-btn">Pause</button>
                  <button type="button" on:click={() => act(e.id, "send_next_now")} class="action-btn">Send next now</button>
                  <button type="button" on:click={() => act(e.id, "cancel")} class="action-btn">Cancel sequence</button>
                {:else if e.status === "paused"}
                  <button type="button" on:click={() => act(e.id, "resume")} class="action-btn">Resume</button>
                  <button type="button" on:click={() => act(e.id, "cancel")} class="action-btn">Cancel sequence</button>
                {/if}
              </div>
            </div>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</section>

{#if showEnrollModal}
  <div class="modal-backdrop" on:click|self={() => (showEnrollModal = false)}>
    <div class="modal-card">
      <h3 class="mb-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-text-primary">
        Enroll in sequence
      </h3>

      {#if enrollError}
        <div class="mb-3 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {enrollError}
        </div>
      {/if}

      <label class="mb-3 block">
        <span class="font-mono text-[11px] uppercase text-bratrax-text-muted">Sequence</span>
        <select
          bind:value={pickedSequenceId}
          class="mt-1 w-full border border-bratrax-border bg-bratrax-bg px-3 py-2 font-mono text-xs text-bratrax-text-primary"
        >
          <option value="">— pick one —</option>
          {#each sequences as s}
            <option value={s.sequence_id}>{s.display_name}</option>
          {/each}
        </select>
      </label>

      <label class="mb-4 flex items-start gap-2">
        <input type="checkbox" bind:checked={forceEnroll} class="mt-1" />
        <span class="font-mono text-[11px] text-bratrax-text-muted">
          Force-enroll even if entry condition isn't met
        </span>
      </label>

      <div class="flex justify-end gap-2">
        <button
          type="button"
          on:click={() => (showEnrollModal = false)}
          class="px-3 py-2 font-mono text-[11px] uppercase tracking-[1.5px] text-bratrax-text-muted hover:text-bratrax-text-primary"
        >Cancel</button>
        <button
          type="button"
          on:click={submitEnroll}
          disabled={enrollLoading}
          class="bg-bratrax-acid px-3 py-2 font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-bg disabled:opacity-50"
        >{enrollLoading ? "Enrolling…" : "Enroll →"}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .action-btn {
    font-family: "Outfit", sans-serif;
    border: 1px solid var(--bratrax-border, #2a2a2a);
    background: transparent;
    color: var(--bratrax-text-primary, #f5f5f7);
    padding: 4px 10px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }
  .action-btn:hover {
    border-color: #d4ff00;
    color: #d4ff00;
  }
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }
  .modal-card {
    width: 100%;
    max-width: 420px;
    background: var(--bratrax-surface, #18181a);
    border: 1px solid var(--bratrax-border, #2a2a2a);
    padding: 24px;
  }
</style>
