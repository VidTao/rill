<script lang="ts">
  import { onMount } from "svelte";

  export let url: string;
  export let title: string;

  let html = "";
  let error = "";
  let loading = true;

  onMount(async () => {
    try {
      const res = await fetch(url, { cache: "no-cache" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const raw = await res.text();
      // Make links inside the srcdoc iframe navigate the top-level browser
      // window instead of staying trapped inside the iframe.
      html = raw.replace(
        /<head(\s[^>]*)?>/i,
        (match) => `${match}<base target="_top">`,
      );
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>{title}</title>
</svelte:head>

{#if loading}
  <div class="state">Loading…</div>
{:else if error}
  <div class="state error">Failed to load: {error}</div>
{:else}
  <iframe srcdoc={html} {title} class="frame"></iframe>
{/if}

<style>
  .frame {
    display: block;
    width: 100%;
    flex: 1 1 auto;
    border: 0;
  }
  .state {
    padding: 3rem 2rem;
    text-align: center;
    font-family: system-ui, sans-serif;
    color: #555;
  }
  .error {
    color: #b00020;
  }
</style>
