<script lang="ts">
  export let open = false;
  export let loading = false;
  export let onSubmit: (encodedAuth: string) => void = () => {};
  export let onClose: () => void = () => {};

  let username = "";
  let password = "";
  let error = "";

  function handleSubmit() {
    if (!username.trim() || !password.trim()) {
      error = "Please enter both username and password";
      return;
    }
    error = "";
    const encodedAuth = btoa(`${username}:${password}`);
    onSubmit(encodedAuth);
  }
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click={onClose}>
    <div
      class="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl"
      on:click|stopPropagation
    >
      <h2 class="mb-4 text-lg font-semibold text-gray-800">Connect Outbrain</h2>

      <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-3">
        {#if error}
          <div class="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {error}
          </div>
        {/if}

        <div class="flex flex-col gap-1">
          <label for="ob-user" class="text-sm font-medium text-gray-700">Username</label>
          <input
            id="ob-user"
            type="text"
            bind:value={username}
            placeholder="your@email.com"
            class="rounded border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            disabled={loading}
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="ob-pass" class="text-sm font-medium text-gray-700">Password</label>
          <input
            id="ob-pass"
            type="password"
            bind:value={password}
            placeholder="Enter your password"
            class="rounded border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            disabled={loading}
          />
        </div>

        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="rounded bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
            on:click={onClose}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
            disabled={loading}
          >
            {loading ? "Connecting..." : "Connect"}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
