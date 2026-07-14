import { QueryClient } from "@tanstack/svelte-query";

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnMount: false,
        // Refetch when the browser regains network connectivity — recovers stale
        // data after wifi/VPN blips without requiring a hard refresh.
        refetchOnReconnect: true,
        refetchOnWindowFocus: false,
        retry: false,
        networkMode: "always",
      },
      mutations: {
        networkMode: "always",
      },
    },
  });
}

export const queryClient = createQueryClient();
