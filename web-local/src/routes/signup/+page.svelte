<script lang="ts">
  import { goto } from "$app/navigation";
  import { bratraxSignup } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { onboardStart } from "$lib/bratrax/onboarding/api";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let email = "";
  let password = "";
  let companyName = "";
  let error = "";
  let loading = false;

  async function handleSubmit() {
    error = "";
    loading = true;
    try {
      // Step 1: Create user + get JWT cookie
      const { user } = await bratraxSignup(email, password, companyName);
      bratraxUser.set(user);
      queryClient.clear();

      // Step 2: Create client + CH database + template
      const result = await onboardStart(companyName);

      // Store client_id for subsequent onboarding steps
      sessionStorage.setItem("onboard_client_id", result.client_id);
      sessionStorage.setItem("onboard_client_name", result.client_name);

      // Redirect to Connect Shopify (Screen 2)
      await goto("/onboard/shopify");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="flex h-screen w-screen items-center justify-center bg-surface">
  <div class="w-full max-w-sm rounded-lg border border-border bg-white p-8 shadow-sm">
    <div class="mb-6 text-center">
      <h1 class="text-xl font-semibold text-fg-primary">Bratrax</h1>
      <p class="mt-1 text-sm text-fg-secondary">
        Get your analytics live in under 60 minutes
      </p>
    </div>

    <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
      {#if error}
        <div
          class="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
        >
          {error}
        </div>
      {/if}

      <div class="flex flex-col gap-1">
        <label for="company" class="text-sm font-medium text-fg-primary"
          >Company name</label
        >
        <input
          id="company"
          type="text"
          bind:value={companyName}
          required
          class="rounded border border-border bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
          placeholder="Your brand name"
        />
      </div>

      <div class="flex flex-col gap-1">
        <label for="email" class="text-sm font-medium text-fg-primary"
          >Email</label
        >
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          autocomplete="email"
          class="rounded border border-border bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
          placeholder="you@example.com"
        />
      </div>

      <div class="flex flex-col gap-1">
        <label for="password" class="text-sm font-medium text-fg-primary"
          >Password</label
        >
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          minlength="8"
          autocomplete="new-password"
          class="rounded border border-border bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
          placeholder="8+ characters"
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        class="w-full rounded bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
      >
        {loading ? "Setting up..." : "Get Started"}
      </button>

      <p class="text-center text-xs text-fg-secondary">
        Already have an account?
        <a href="/login" class="text-indigo-600 hover:underline">Sign in</a>
      </p>
    </form>
  </div>
</div>
