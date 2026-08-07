<script lang="ts">
  import { tick } from "svelte";
  import { SUPPORT_EMAIL } from "$lib/bratrax/constants";
  import { askSupportBot, draftEscalationEmail } from "./api";
  import SupportMarkdown from "./SupportMarkdown.svelte";
  import type { ChatMessage, EscalateResponse } from "./types";

  interface DisplayMessage extends ChatMessage {
    needsEscalation?: boolean;
  }

  let messages: DisplayMessage[] = [];
  let input = "";
  let sending = false;
  let error = "";

  let escalating = false;
  let escalateError = "";
  let draft: EscalateResponse | null = null;
  let copied = false;

  let scrollBox: HTMLDivElement;
  let textarea: HTMLTextAreaElement;

  $: canSend = input.trim().length > 0 && !sending;

  async function send() {
    if (!canSend) return;
    const question = input.trim();
    input = "";
    error = "";
    draft = null;
    messages = [...messages, { role: "user", content: question }];
    sending = true;
    await scrollToBottom();
    try {
      const res = await askSupportBot(
        messages.map(({ role, content }) => ({ role, content })),
      );
      messages = [
        ...messages,
        {
          role: "assistant",
          content: res.answer,
          needsEscalation: res.needs_escalation,
        },
      ];
    } catch (e) {
      error = e.message ?? "Something went wrong";
      // Put the question back so the user can retry without retyping.
      messages = messages.slice(0, -1);
      input = question;
    } finally {
      sending = false;
      await scrollToBottom();
      textarea?.focus();
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  async function escalate() {
    if (escalating) return;
    escalating = true;
    escalateError = "";
    copied = false;
    try {
      draft = await draftEscalationEmail(
        messages.map(({ role, content }) => ({ role, content })),
      );
    } catch (e) {
      escalateError = e.message ?? "Could not draft the email";
    } finally {
      escalating = false;
      await scrollToBottom();
    }
  }

  function mailtoHref(d: EscalateResponse): string {
    return `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent(d.subject)}&body=${encodeURIComponent(d.body)}`;
  }

  async function copyDraft() {
    if (!draft) return;
    try {
      await navigator.clipboard.writeText(
        `To: ${SUPPORT_EMAIL}\nSubject: ${draft.subject}\n\n${draft.body}`,
      );
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      // Clipboard API unavailable (insecure context) — the textarea below is
      // selectable, so manual copy still works.
    }
  }

  async function scrollToBottom() {
    await tick();
    scrollBox?.scrollTo({ top: scrollBox.scrollHeight, behavior: "smooth" });
  }

  $: hasConversation = messages.length > 0;
  $: lastNeedsEscalation =
    messages.length > 0 && !!messages[messages.length - 1].needsEscalation;
</script>

<div class="flex h-full flex-col">
  <div bind:this={scrollBox} class="flex-1 overflow-y-auto px-4 py-4">
    {#if !hasConversation}
      <div class="mx-auto max-w-xl pt-8 text-center">
        <p
          class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
        >
          Ask support
        </p>
        <p class="mt-3 text-sm text-bratrax-text-muted">
          Ask anything about Bratrax — connecting platforms, attribution,
          billing, dashboards. Answers come from our verified knowledge base;
          the assistant can't see your account data. For account-specific
          issues, we'll help you reach a human.
        </p>
      </div>
    {/if}

    <div class="mx-auto flex max-w-xl flex-col gap-3">
      {#each messages as m, i (i)}
        {#if m.role === "user"}
          <div
            class="max-w-[85%] self-end whitespace-pre-wrap rounded-2xl rounded-br-sm bg-bratrax-acid/15 px-4 py-2 text-sm leading-relaxed"
          >
            {m.content}
          </div>
        {:else}
          <div
            class="max-w-[85%] self-start rounded-2xl rounded-bl-sm border border-bratrax-border bg-bratrax-surface px-4 py-2"
          >
            <SupportMarkdown content={m.content} />
          </div>
          {#if m.needsEscalation && i === messages.length - 1 && !draft}
            <button
              class="self-start bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
              disabled={escalating}
              on:click={escalate}
            >
              {escalating ? "Drafting…" : "Draft an email to support"}
            </button>
          {/if}
        {/if}
      {/each}

      {#if sending}
        <div
          class="self-start rounded-2xl rounded-bl-sm border border-bratrax-border bg-bratrax-surface px-4 py-2 font-mono text-xs text-bratrax-text-muted"
        >
          Thinking…
        </div>
      {/if}

      {#if escalateError}
        <div
          class="self-start border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
        >
          {escalateError} You can also email
          <a class="underline" href={`mailto:${SUPPORT_EMAIL}`}
            >{SUPPORT_EMAIL}</a
          > directly.
        </div>
      {/if}

      {#if draft}
        <div
          class="flex flex-col gap-2 self-stretch border border-bratrax-acid/30 bg-bratrax-acid/5 p-3"
        >
          <p
            class="font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
          >
            Email draft for {SUPPORT_EMAIL}
          </p>
          <p class="text-sm font-medium">{draft.subject}</p>
          <textarea
            class="min-h-[110px] w-full resize-y border border-bratrax-border bg-bratrax-bg p-2 text-sm leading-relaxed"
            readonly
            value={draft.body}
          />
          <div class="flex gap-2">
            <a
              class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90"
              href={mailtoHref(draft)}
            >
              Open in email app
            </a>
            <button
              class="btn-bratrax btn-neutral btn-compact"
              on:click={copyDraft}
            >
              {copied ? "Copied!" : "Copy to clipboard"}
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>

  {#if error}
    <div class="mx-auto w-full max-w-xl px-4">
      <div
        class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
      >
        {error}
      </div>
    </div>
  {/if}

  <div class="border-t border-bratrax-border px-4 py-3">
    <div class="mx-auto flex max-w-xl items-end gap-2">
      <textarea
        bind:this={textarea}
        bind:value={input}
        on:keydown={onKeydown}
        rows="2"
        placeholder="Ask a question about Bratrax…"
        class="flex-1 resize-none border border-bratrax-border bg-bratrax-bg p-2 text-sm leading-relaxed focus:border-bratrax-acid focus:outline-none"
      />
      <button
        class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
        disabled={!canSend}
        on:click={send}
      >
        Send
      </button>
    </div>
    <!-- Kept to a single dim line: the full explanation lives in the empty
         state above, so repeating it here only ate the answer area. -->
    <p
      class="mx-auto mt-1.5 max-w-xl font-mono text-[10px] leading-tight text-bratrax-text-muted"
    >
      Help-center answers · no access to your data ·
      <a class="underline" href={`mailto:${SUPPORT_EMAIL}`}>Email support</a
      >{#if hasConversation && !lastNeedsEscalation}
        ·
        <button class="underline" on:click={escalate}>Draft an email</button
        >{/if}
    </p>
  </div>
</div>
