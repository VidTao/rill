<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { bratraxLogin } from "$lib/bratrax/auth";
  import { bratraxUser } from "$lib/bratrax/auth-store";

  let email = "";
  let password = "";
  let error = "";
  let loading = false;

  async function handleSubmit() {
    error = "";
    loading = true;
    try {
      const { user } = await bratraxLogin(email, password);
      bratraxUser.set(user);
      const redirect = $page.url.searchParams.get("redirect") || "/";
      await goto(redirect);
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
      <p class="mt-1 text-sm text-fg-secondary">Sign in to continue</p>
    </div>

    <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
      {#if error}
        <div class="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      {/if}

      <div class="flex flex-col gap-1">
        <label for="email" class="text-sm font-medium text-fg-primary">Email</label>
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
        <label for="password" class="text-sm font-medium text-fg-primary">Password</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          autocomplete="current-password"
          class="rounded border border-border bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
          placeholder="Enter your password"
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        class="w-full rounded bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
      >
        {loading ? "Signing in..." : "Sign in"}
      </button>
    </form>
  </div>
</div>
