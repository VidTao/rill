// Types for the in-app support bot (bratrax docs/SUPPORT_BOT_PROPOSAL.md).
// Field names are snake_case to match the Flask JSON payloads verbatim.

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

export interface AskResponse {
  answer: string;
  // True when the KB didn't cover the question (or the daily cap was hit) —
  // the panel should highlight the email-support assist.
  needs_escalation: boolean;
  rate_limited: boolean;
}

export interface EscalateResponse {
  subject: string;
  body: string;
}
