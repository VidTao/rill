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

      // Rewrite link behavior for srcdoc iframe rendering:
      //   1. <base target="_top"> — absolute / external links escape the
      //      iframe and navigate the parent window (e.g. /signup, https://...).
      //   2. <a href="#..."> hash anchors are handled by an injected click
      //      listener that calls scrollIntoView and prevents the default
      //      navigation. We can't use target="_self" here because srcdoc
      //      iframes treat about:srcdoc#xyz as a URL change and re-render
      //      the entire iframe (looks like a page reload). The click
      //      interceptor avoids that by never letting navigation happen.
      const parser = new DOMParser();
      const doc = parser.parseFromString(raw, "text/html");

      const base = doc.createElement("base");
      base.setAttribute("target", "_top");
      doc.head?.prepend(base);

      const tocScript = doc.createElement("script");
      tocScript.textContent = `
        document.addEventListener("click", function (e) {
          var a = e.target && e.target.closest && e.target.closest('a[href^="#"]');
          if (!a) return;
          var id = a.getAttribute("href").slice(1);
          if (!id) return;
          var target = document.getElementById(id);
          if (!target) return;
          e.preventDefault();
          target.scrollIntoView({ behavior: "smooth", block: "start" });
        });
      `;
      doc.body?.appendChild(tocScript);

      html = "<!DOCTYPE html>" + doc.documentElement.outerHTML;
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
