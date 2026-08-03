import { sessionStorageStore } from "../../../../lib/store-utils/session-storage";
import { sidebarActions } from "./sidebar-store";

// Open/closed state for the Bratrax support-bot sidebar (the "?" header
// button). Lives in web-common so ApplicationHeader can toggle it; the panel
// itself is bratrax-specific and mounts from web-local's root layout.
export const supportChatOpen = sessionStorageStore<boolean>(
  "support-chat-open",
  false,
);

export const supportChatActions = {
  toggle(): void {
    supportChatOpen.update((isOpen) => {
      // Only one right-hand sidebar at a time: opening support closes the AI
      // chat (the reverse direction is handled by the panel watching chatOpen).
      if (!isOpen) sidebarActions.closeChat();
      return !isOpen;
    });
  },

  open(): void {
    sidebarActions.closeChat();
    supportChatOpen.set(true);
  },

  close(): void {
    supportChatOpen.set(false);
  },
};
