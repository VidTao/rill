<script lang="ts">
  import StaticHtmlPage from "$lib/bratrax/StaticHtmlPage.svelte";
</script>

<!-- Client-side fallback for the server-rendered page in bratrax/bratrax.go.
     Unlike /integrations/slack, the source path here mirrors the public URL. -->
<StaticHtmlPage
  url="https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/integrations/shopify/index.html"
  title="Shopify attribution that adds up — Bratrax"
/>
