<script lang="ts">
  import type { WorkshopResults } from "$lib/bratrax/workshop/types";

  export let results: WorkshopResults | null = null;
</script>

{#if !results}
  <div class="flex h-full items-center justify-center text-sm text-gray-400">
    Run Validate, Compile, or Deploy to see results here.
  </div>
{:else if results.phase === "validation" && results.validation}
  {@const v = results.validation}
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      {#if v.valid}
        <span class="inline-block h-2.5 w-2.5 rounded-full bg-green-500" />
        <span class="text-sm font-medium text-green-700">Valid</span>
      {:else}
        <span class="inline-block h-2.5 w-2.5 rounded-full bg-red-500" />
        <span class="text-sm font-medium text-red-700">Invalid</span>
      {/if}
      <span class="text-xs text-gray-400">
        {v.errors} error{v.errors !== 1 ? "s" : ""},
        {v.warnings} warning{v.warnings !== 1 ? "s" : ""}
      </span>
    </div>

    {#if v.issues.length > 0}
      <ul class="max-h-[60vh] space-y-1 overflow-y-auto">
        {#each v.issues as issue}
          <li
            class="rounded border px-3 py-2 text-xs {issue.severity === 'error'
              ? 'border-red-200 bg-red-50 text-red-800'
              : 'border-yellow-200 bg-yellow-50 text-yellow-800'}"
          >
            <span class="font-mono font-medium">{issue.code}</span>
            <span class="ml-1">{issue.message}</span>
            {#if issue.context}
              <span class="ml-1 text-gray-500">({issue.context})</span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{:else if results.phase === "compile" && results.compile}
  {@const c = results.compile}
  {@const rillLayers = Object.entries(c.summary).filter(([k]) => k.startsWith("rill_"))}
  {@const chLayers = Object.entries(c.summary).filter(([k]) => k.startsWith("ch_"))}
  {@const rillArtifacts = c.artifacts.filter((a) => a.layer.startsWith("rill_"))}
  {@const chArtifacts = c.artifacts.filter((a) => a.layer.startsWith("ch_"))}
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      {#if c.success}
        <span class="inline-block h-2.5 w-2.5 rounded-full bg-green-500" />
        <span class="text-sm font-medium text-green-700">Compiled</span>
      {:else}
        <span class="inline-block h-2.5 w-2.5 rounded-full bg-red-500" />
        <span class="text-sm font-medium text-red-700">Failed</span>
      {/if}
      <span class="text-xs text-gray-400">
        {c.summary.total} artifact{c.summary.total !== 1 ? "s" : ""}
      </span>
    </div>

    <div class="flex flex-wrap gap-2 text-xs">
      {#each rillLayers as [layer, count]}
        {#if count > 0}
          <span class="rounded bg-teal-100 px-2 py-0.5 text-teal-700">
            {layer}: {count}
          </span>
        {/if}
      {/each}
      {#each chLayers as [layer, count]}
        {#if count > 0}
          <span class="rounded bg-orange-100 px-2 py-0.5 text-orange-700">
            {layer}: {count}
          </span>
        {/if}
      {/each}
    </div>

    {#if rillArtifacts.length > 0}
      <details class="text-xs">
        <summary class="cursor-pointer text-teal-600 hover:text-teal-800">
          Rill Artifacts ({rillArtifacts.length})
        </summary>
        <ul class="mt-2 max-h-[30vh] space-y-1 overflow-y-auto">
          {#each rillArtifacts as artifact}
            <li class="rounded bg-gray-50 px-3 py-2">
              <div class="flex items-center gap-2">
                <span
                  class="rounded bg-teal-100 px-1.5 py-0.5 text-[10px] font-medium uppercase text-teal-700"
                >
                  {artifact.layer}
                </span>
                <span class="font-mono text-gray-700">{artifact.path}</span>
              </div>
            </li>
          {/each}
        </ul>
      </details>
    {/if}

    {#if chArtifacts.length > 0}
      <details class="text-xs">
        <summary class="cursor-pointer text-orange-600 hover:text-orange-800">
          ClickHouse DDL ({chArtifacts.length})
        </summary>
        <ul class="mt-2 max-h-[30vh] space-y-1 overflow-y-auto">
          {#each chArtifacts as artifact}
            <li class="rounded bg-gray-50 px-3 py-2">
              <div class="flex items-center gap-2">
                <span
                  class="rounded bg-orange-100 px-1.5 py-0.5 text-[10px] font-medium uppercase text-orange-700"
                >
                  {artifact.layer}
                </span>
                <span class="font-mono text-gray-700">{artifact.path}</span>
              </div>
            </li>
          {/each}
        </ul>
      </details>
    {/if}

    {#if c.validation && !c.validation.valid}
      <div class="mt-2 rounded border border-yellow-200 bg-yellow-50 px-3 py-2 text-xs text-yellow-800">
        Validation had {c.validation.errors} error{c.validation.errors !== 1
          ? "s"
          : ""}, {c.validation.warnings} warning{c.validation.warnings !== 1
          ? "s"
          : ""}
      </div>
    {/if}
  </div>
{:else if results.phase === "deploy" && results.deploy}
  {@const d = results.deploy}
  {@const chResults = d.ch_results || []}
  {@const chFailed = chResults.filter((r) => !r.ok)}
  {@const chOk = chResults.filter((r) => r.ok)}
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <span
        class="inline-block h-2.5 w-2.5 rounded-full {d.applied
          ? 'bg-green-500'
          : 'bg-blue-500'}"
      />
      <span
        class="text-sm font-medium {d.applied
          ? 'text-green-700'
          : 'text-blue-700'}"
      >
        {d.applied ? "Deployed" : "Dry Run"}
      </span>
      <span class="text-xs text-gray-400">
        {d.plan.length} action{d.plan.length !== 1 ? "s" : ""}
      </span>
    </div>

    {#if d.pr_url}
      <a
        href={d.pr_url}
        target="_blank"
        rel="noopener noreferrer"
        class="inline-block text-sm text-blue-600 underline hover:text-blue-800"
      >
        View Pull Request
      </a>
    {/if}

    {#if chResults.length > 0}
      <div class="flex items-center gap-2 text-xs">
        <span class="font-medium text-gray-600">ClickHouse DDL:</span>
        <span class="rounded bg-green-100 px-2 py-0.5 text-green-700">
          {chOk.length} OK
        </span>
        {#if chFailed.length > 0}
          <span class="rounded bg-red-100 px-2 py-0.5 text-red-700">
            {chFailed.length} failed
          </span>
        {/if}
      </div>
      {#if chFailed.length > 0}
        <details class="text-xs" open>
          <summary class="cursor-pointer text-red-600 hover:text-red-800">
            Failed statements ({chFailed.length})
          </summary>
          <ul class="mt-2 max-h-[20vh] space-y-1 overflow-y-auto">
            {#each chFailed as r}
              <li class="rounded border border-red-200 bg-red-50 px-3 py-2 text-red-800">
                <span class="font-mono font-medium">{r.file}[{r.statement}]</span>
                <span class="ml-1">({r.type})</span>
                <div class="mt-1 text-red-600">{r.error}</div>
              </li>
            {/each}
          </ul>
        </details>
      {/if}
    {/if}

    {#if d.plan.length > 0}
      <details class="text-xs">
        <summary class="cursor-pointer text-gray-600 hover:text-gray-800">
          All actions ({d.plan.length})
        </summary>
        <ul class="mt-2 max-h-[30vh] space-y-1 overflow-y-auto">
          {#each d.plan as action}
            <li class="flex items-center gap-2 rounded bg-gray-50 px-3 py-2">
              <span
                class="rounded px-1.5 py-0.5 text-[10px] font-medium uppercase {action.action ===
                'CREATE'
                  ? 'bg-green-100 text-green-700'
                  : action.action === 'EXECUTE'
                    ? 'bg-orange-100 text-orange-700'
                    : 'bg-yellow-100 text-yellow-700'}"
              >
                {action.action}
              </span>
              <span class="font-mono text-gray-700">{action.target}</span>
            </li>
          {/each}
        </ul>
      </details>
    {/if}
  </div>
{/if}
