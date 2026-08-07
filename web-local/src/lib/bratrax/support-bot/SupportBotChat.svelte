<script lang="ts">
  import { tick } from "svelte";
  import { ArrowUp } from "lucide-svelte";
  import Button from "@rilldata/web-common/components/button/Button.svelte";
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
      <!-- Same shape as the AI sidebar's empty state (`.chat-empty` in
           web-common/features/chat/core/messages/Messages.svelte): a short
           semibold line with one muted line under it, centred in the space. The
           four-sentence paragraph that was here restated the panel header and
           carried the no-access caveat that now lives in the footer. -->
      <div class="support-empty">
        <div class="support-empty-title">How can I help?</div>
        <div class="support-empty-subtitle">
          Connecting platforms, attribution, billing, dashboards.
        </div>
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

  <!-- Deliberately mirrors the AI sidebar's composer
       (web-common/features/chat/core/input/ChatInput.svelte): one bordered,
       rounded surface holding a borderless input with a control row along its
       bottom, focus ring on the container rather than the field, and the same
       square ArrowUp Button component. The two sidebars sit behind adjacent
       header buttons, so a composer of its own design read as a different
       product. Keep them in step if ChatInput's shell changes. -->
  <div class="px-4 pb-4 pt-3">
    <div class="support-composer mx-auto max-w-xl">
      <textarea
        bind:this={textarea}
        bind:value={input}
        on:keydown={onKeydown}
        rows="2"
        placeholder="Ask a question about Bratrax…"
        class="support-input max-h-32 w-full resize-none overflow-auto p-0 text-sm leading-relaxed"
      />
      <!-- Send alone on the row, right-aligned. ChatInput keeps its @ affordance
           on the left here; there is no equivalent for support, and the escalation
           links read as shouting inside the box, so they sit in the footer. -->
      <div class="flex flex-row items-center">
        <div class="grow"></div>
        <Button
          type="primary"
          label="Send message"
          disabled={!canSend}
          square
          onClick={send}
        >
          <ArrowUp size="16px" />
        </Button>
      </div>
    </div>

    <p class="support-footer">
      No access to your data ·
      <a href={`mailto:${SUPPORT_EMAIL}`}>Email support</a>{#if hasConversation && !lastNeedsEscalation}
        ·
        <button on:click={escalate}>Draft an email</button>{/if}
    </p>
  </div>
</div>

<style lang="postcss">
  /* Same shell as `.chat-input-form` in
     web-common/features/chat/core/input/ChatInput.svelte. Outer spacing is on
     the wrapper here instead of `mx-4 mb-4`, since this sidebar owns its own
     padding. `rounded-md` is currently a no-op — web-local/tailwind.config.ts
     zeroes every borderRadius for the Bratrax square-corner look — but it is
     kept so the two composers stay in step if that is ever relaxed. */
  .support-composer {
    @apply flex flex-col gap-1 p-3;
    @apply rounded-md border bg-input;
    transition: border-color 0.2s;
  }

  /* Focus lives on the container, not the textarea: that is what makes the box
     read as one control rather than a field with a button parked beside it. */
  .support-composer:focus-within {
    @apply border-ring-focus;
  }

  /* bratrax-theme.css §13 sets `input, textarea, select` background, border and
     a focus ring with `!important`, which no utility class can beat. Left alone
     it painted a second, differently-shaded box inside the composer and drew a
     focus ring around the field rather than around the container. ChatInput
     escapes this only because TipTap renders a contenteditable div rather than a
     textarea, so the override has to answer `!important` in kind. Placeholder
     colour is left to that same global rule, which already resolves to muted. */
  .support-input {
    background-color: transparent !important;
    border-color: transparent !important;
  }
  .support-input:focus {
    border-color: transparent !important;
    box-shadow: none !important;
    outline: none !important;
  }

  /* Mirrors `.chat-empty*` in the AI sidebar. Height comes from the scroll
     container, so the copy sits centred rather than pinned under the header. */
  .support-empty {
    @apply flex h-full flex-col items-center justify-center text-center;
    @apply text-fg-secondary;
  }
  .support-empty-title {
    @apply mb-1 text-base font-semibold text-fg-secondary;
  }
  .support-empty-subtitle {
    @apply text-xs text-fg-secondary;
  }

  /* Matches `.chat-footer` in the AI sidebar's ChatFooter.svelte: 0.75rem,
     muted, centred. The previous 10px uppercase mono in acid read as a heading
     rather than fine print, which is why it looked oversized next to 14px body
     copy despite being smaller. */
  .support-footer {
    @apply mx-auto max-w-xl text-fg-muted;
    padding-top: 0.5rem;
    font-size: 0.75rem;
    text-align: center;
  }
  .support-footer a,
  .support-footer button {
    color: inherit;
    text-decoration: none;
  }
  .support-footer a:hover,
  .support-footer button:hover {
    @apply text-bratrax-text-primary;
    text-decoration: underline;
  }
</style>
