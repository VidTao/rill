import type {
  BannerEvent,
  PageContentResized,
  NotificationMessage,
} from "./events";
import { EventEmitter } from "@rilldata/web-common/lib/event-emitter.ts";

export interface Events {
  notification: NotificationMessage;
  "clear-all-notifications": void;
  "add-banner": BannerEvent;
  "remove-banner": string;
  "shift-click": void;
  "command-click": void;
  click: void;
  "shift-command-click": void;
  "page-content-resized": PageContentResized;
  "start-chat": string;
  "rill-yaml-updated": void;
  // The AI endpoint refused the prompt because the caller's quota is spent
  // (HTTP 402). Emitted regardless of which agent was in use, so a wrapper can
  // swap the chat for an upsell without knowing which conversation manager
  // produced it. See BratraxChatGate.svelte.
  "ai-quota-exhausted": void;
}

export const eventBus = new EventEmitter<Events>();
