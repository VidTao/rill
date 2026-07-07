<script lang="ts">
  import { goto } from "$app/navigation";
  import StaticHtmlPage from "$lib/bratrax/StaticHtmlPage.svelte";
  import DemoModal from "$lib/bratrax/DemoModal.svelte";

  // Public self-serve "Try demo" entry. The static homepage (served by the Go
  // proxy on the apex) links here with <a href="/try-demo">, which loads the
  // SPA on a route it owns. We render the same marketing homepage as a backdrop
  // — via the trusted StaticHtmlPage-reads-from-GitHub pattern, identical to the
  // apex — with the demo modal open on top, so it reads as "the modal opened
  // over the homepage". All the logic (email capture → /bratrax/demo-request →
  // invite → auto-login) lives here in the SPA. Closing returns to the homepage.
  let open = true;
</script>

<StaticHtmlPage
  url="https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/index.html"
  title="Try Bratrax — live demo"
/>

<DemoModal bind:open onClose={() => goto("/")} />
