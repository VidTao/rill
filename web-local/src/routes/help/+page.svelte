<script lang="ts">
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { pagesFor } from "$lib/help";

  $: role = $bratraxUser?.role ?? null;
  $: visiblePages = pagesFor(role);
  $: viewerCount = visiblePages.filter((p) => p.audience === "viewer").length;
  $: adminCount = visiblePages.filter((p) => p.audience === "admin").length;
  $: isAdmin = role === "admin" || role === "super_admin";
</script>

<article class="help-landing">
  <header>
    <h1>Bratrax Help</h1>
    <p class="lede">
      Everything you need to read your dashboards, ask Claude, and
      {#if isAdmin}build new metrics and dashboards{:else}understand the data you see{/if}.
    </p>
  </header>

  <section class="help-cards">
    <a class="help-card" href="/help/viewer/welcome">
      <div class="card-tag">For everyone</div>
      <h2>Using dashboards</h2>
      <p>
        Reading KPIs and charts, picking time ranges, filtering, asking Claude,
        and a glossary of every metric on screen. {viewerCount} pages.
      </p>
    </a>

    {#if isAdmin}
      <a class="help-card" href="/help/admin/overview">
        <div class="card-tag">For admins</div>
        <h2>Building metrics &amp; dashboards</h2>
        <p>
          Connect platforms, write SQL models, define metrics views, and lay
          out canvas dashboards. {adminCount} pages.
        </p>
      </a>
    {/if}
  </section>

  <footer>
    <p class="hint">
      Looking for something specific? Pick a page from the sidebar.
    </p>
  </footer>
</article>

<style lang="postcss">
  .help-landing {
    max-width: 720px;
    margin: 0 auto;
    padding: 48px 32px;
  }
  header h1 {
    font-family: "Space Mono", monospace;
    font-size: 1.875rem;
    font-weight: 700;
    letter-spacing: -0.02em;
    margin-bottom: 8px;
  }
  .lede {
    color: var(--color-text-muted);
    font-size: 1rem;
    line-height: 1.6;
    margin-bottom: 32px;
  }
  .help-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
    margin-bottom: 32px;
  }
  .help-card {
    display: block;
    border: 1px solid var(--color-bratrax-border, #e5e7eb);
    border-radius: 8px;
    padding: 20px;
    text-decoration: none;
    color: inherit;
    background: var(--color-bratrax-surface);
    transition: border-color 0.15s, background-color 0.15s;
  }
  .help-card:hover {
    border-color: var(--color-acid);
    background: var(--color-acid-dim);
  }
  .card-tag {
    font-family: "Space Mono", monospace;
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 1.5px;
    color: var(--color-text-muted);
    margin-bottom: 8px;
  }
  .help-card h2 {
    font-size: 1.125rem;
    font-weight: 600;
    margin-bottom: 6px;
  }
  .help-card p {
    color: var(--color-text-muted);
    font-size: 0.875rem;
    line-height: 1.5;
  }
  footer .hint {
    color: var(--color-text-muted);
    font-size: 0.875rem;
  }
  .inline-tag {
    font-family: "Space Mono", monospace;
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 1px;
    background: var(--color-acid-dim);
    color: var(--color-text-muted);
    padding: 1px 6px;
    border-radius: 3px;
  }
</style>
