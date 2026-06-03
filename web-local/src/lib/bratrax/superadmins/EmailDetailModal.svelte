<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { EmailLogEntry } from "./api";

  export let email: EmailLogEntry;

  const dispatch = createEventDispatcher<{ close: void }>();

  function fmt(iso: string | null): string {
    if (!iso) return "—";
    return new Date(iso).toLocaleString();
  }
</script>

<div class="backdrop" on:click|self={() => dispatch("close")}>
  <div class="card">
    <div class="flex items-start justify-between gap-4">
      <div>
        <div class="font-mono text-[10px] uppercase tracking-[2px] text-bratrax-text-muted">
          {email.kind}{email.sequence_id ? ` · ${email.sequence_id}` : ""}
        </div>
        <h3 class="mt-1 text-base font-bold text-bratrax-text-primary">
          {email.template_key}
        </h3>
        <p class="mt-1 font-mono text-xs text-bratrax-text-muted">
          to {email.recipient_email}
        </p>
      </div>
      <button type="button" on:click={() => dispatch("close")} class="close-btn">×</button>
    </div>

    <dl class="mt-4 grid grid-cols-2 gap-x-4 gap-y-2 font-mono text-[11px]">
      <dt class="text-bratrax-text-muted">Status</dt>
      <dd class="text-bratrax-text-primary">{email.status}</dd>

      <dt class="text-bratrax-text-muted">Subject</dt>
      <dd class="text-bratrax-text-primary">{email.subject || "—"}</dd>

      <dt class="text-bratrax-text-muted">Sent</dt>
      <dd class="text-bratrax-text-primary">{fmt(email.sent_at)}</dd>

      <dt class="text-bratrax-text-muted">Delivered</dt>
      <dd class="text-bratrax-text-primary">{fmt(email.delivered_at)}</dd>

      <dt class="text-bratrax-text-muted">First opened</dt>
      <dd class="text-bratrax-text-primary">{fmt(email.first_opened_at)}</dd>

      <dt class="text-bratrax-text-muted">Open count</dt>
      <dd class="text-bratrax-text-primary">{email.open_count}</dd>

      <dt class="text-bratrax-text-muted">First clicked</dt>
      <dd class="text-bratrax-text-primary">{fmt(email.first_clicked_at)}</dd>

      <dt class="text-bratrax-text-muted">Click count</dt>
      <dd class="text-bratrax-text-primary">{email.click_count}</dd>

      <dt class="text-bratrax-text-muted">SendGrid template</dt>
      <dd class="break-all text-bratrax-text-primary">{email.sendgrid_template_id || "—"}</dd>

      {#if email.bounce_reason}
        <dt class="text-bratrax-text-muted">Bounce reason</dt>
        <dd class="text-bratrax-tomato">{email.bounce_reason}</dd>
      {/if}

      {#if email.error}
        <dt class="text-bratrax-text-muted">Error</dt>
        <dd class="text-bratrax-tomato">{email.error}</dd>
      {/if}
    </dl>

    <h4 class="mt-5 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-text-muted">
      Template variables
    </h4>
    <pre class="mt-2 max-h-60 overflow-auto border border-bratrax-border bg-bratrax-bg p-3 font-mono text-[11px] text-bratrax-text-primary">{JSON.stringify(email.template_variables, null, 2)}</pre>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
    padding: 24px;
  }
  .card {
    width: 100%;
    max-width: 640px;
    max-height: 90vh;
    overflow-y: auto;
    background: var(--bratrax-surface, #18181a);
    border: 1px solid var(--bratrax-border, #2a2a2a);
    padding: 24px;
  }
  .close-btn {
    background: transparent;
    border: 0;
    color: var(--bratrax-text-muted, #858585);
    font-size: 28px;
    line-height: 1;
    cursor: pointer;
  }
  .close-btn:hover {
    color: var(--bratrax-text-primary, #f5f5f7);
  }
</style>
