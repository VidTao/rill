import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import type { AskResponse, ChatMessage, EscalateResponse } from "./types";

function host(): string {
  return get(runtime).host;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${host()}${path}`, {
    credentials: "include",
    ...init,
  });
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body.error) message = body.error;
    } catch {
      // ignore parse errors
    }
    throw new Error(message);
  }
  return res.json();
}

// The conversation is client-held and resent in full each turn — the backend
// keeps no chat sessions (v1 scope: no persistence across reloads).
export function askSupportBot(messages: ChatMessage[]): Promise<AskResponse> {
  return apiFetch<AskResponse>("/bratrax/support-bot/ask", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ messages }),
  });
}

export function draftEscalationEmail(
  messages: ChatMessage[],
): Promise<EscalateResponse> {
  return apiFetch<EscalateResponse>("/bratrax/support-bot/escalate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ messages }),
  });
}
