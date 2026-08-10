<script lang="ts">
  import { page } from "$app/stores";
  import StaticHtmlPage from "$lib/bratrax/StaticHtmlPage.svelte";

  $: slug = $page.params.slug ?? "";
</script>

<!--
  StaticHtmlPage fetches in onMount only, so navigating between two entries
  client-side would keep showing the first one's HTML. Keying on the slug
  forces a remount, which re-runs the fetch.
-->
{#key slug}
  <StaticHtmlPage
    url="https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/changelog/{slug}/index.html"
    title="Changelog — Bratrax"
  />
{/key}
