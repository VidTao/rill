<script lang="ts">
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import HelpSearch from "$lib/help/HelpSearch.svelte";
  import { pagesFor } from "$lib/help";

  $: role = $bratraxUser?.role ?? null;
  $: visiblePages = pagesFor(role);
  $: viewerCount = visiblePages.filter((p) => p.audience === "viewer").length;
  $: adminCount = visiblePages.filter((p) => p.audience === "admin").length;
  $: isAdmin = role === "admin" || role === "super_admin";

  let searchQuery = "";
  $: searching = searchQuery.trim().length > 0;
  $: searchPlaceholder = `Search ${visiblePages.length} help pages…`;
</script>

<article class="help-landing">
  <header>
    <h1>Bratrax Help</h1>
    <p class="lede">
      Start with the question you are trying to answer. Each guide maps a common
      ecommerce question to the dashboard, metrics, and drilldowns that matter.
    </p>
  </header>

  <div class="search-row">
    <HelpSearch
      pages={visiblePages}
      variant="landing"
      placeholder={searchPlaceholder}
      bind:query={searchQuery}
    />
  </div>

  {#if !searching}
  <section class="question-grid" aria-label="Dashboard help">
    <a class="question-card primary" href="/help/viewer/store-performance">
      <div class="card-tag">Daily health</div>
      <h2>Is the business up or down?</h2>
      <p>Use Store Performance for Total Sales, MER, Paid ROAS, returns, customer mix, and subscriptions.</p>
    </a>

    <a class="question-card" href="/help/viewer/attribution">
      <div class="card-tag">Channels</div>
      <h2>Where did sales come from?</h2>
      <p>Use Attribution to inspect channel, campaign, ad set, ad, Direct, ROAS, CPA, and NC ROAS.</p>
    </a>

    <a class="question-card" href="/help/viewer/products">
      <div class="card-tag">Products</div>
      <h2>What is selling?</h2>
      <p>Use Products for top products, SKUs, units sold, product net sales, and discount pressure.</p>
    </a>

    <a class="question-card" href="/help/viewer/customer-analytics">
      <div class="card-tag">Customers</div>
      <h2>Are customers coming back?</h2>
      <p>Use Customer Analytics for MRR, active subscribers, LTV, churn, cohorts, and subscriber acquisition.</p>
    </a>

    <a class="question-card" href="/help/viewer/email-sms">
      <div class="card-tag">Retention</div>
      <h2>Are campaigns and flows working?</h2>
      <p>Use Email &amp; SMS for sends, opens, clicks, unsubs, flows, campaigns, and attributed sales.</p>
    </a>

    <a class="question-card" href="/help/viewer/new-customer-source-report">
      <div class="card-tag">Acquisition</div>
      <h2>Which sources bring new customers?</h2>
      <p>Use New Customer Source Report for NCV, NCP, NC ROAS, NC CPA, and source-level acquisition quality.</p>
    </a>
  </section>

  <section class="help-cards">
    <a class="help-card" href="/help/viewer/welcome">
      <div class="card-tag">For everyone</div>
      <h2>Learn the basics</h2>
      <p>
        Reading dashboards, picking time ranges, filtering, asking Claude, and
        understanding the glossary. {viewerCount} pages.
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
      Looking for a specific metric? Open the Glossary under Reference.
    </p>
  </footer>
  {/if}
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
    margin-bottom: 24px;
  }
  .search-row {
    margin-bottom: 32px;
  }
  .question-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 12px;
    margin-bottom: 32px;
  }
  .question-card {
    display: block;
    border: 1px solid var(--color-bratrax-border, #e5e7eb);
    border-radius: 8px;
    padding: 16px;
    text-decoration: none;
    color: inherit;
    background: var(--color-bratrax-surface);
    transition: border-color 0.15s, background-color 0.15s;
    min-height: 150px;
  }
  .question-card.primary {
    border-color: var(--color-acid);
    background: var(--color-acid-dim);
  }
  .question-card:hover {
    border-color: var(--color-acid);
    background: var(--color-acid-dim);
  }
  .question-card h2 {
    font-size: 1rem;
    font-weight: 650;
    line-height: 1.3;
    margin-bottom: 6px;
  }
  .question-card p {
    color: var(--color-text-muted);
    font-size: 0.8125rem;
    line-height: 1.45;
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
</style>
