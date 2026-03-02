<script lang="ts">
  import yaml from "js-yaml";

  export let content: string;
  export let onChange: (newContent: string) => void;

  interface StreamConfig {
    fields: string[];
  }

  interface SourceDef {
    type: string;
    tap: string | null;
    raw_table: string | null;
    streams: Record<string, StreamConfig>;
  }

  interface SourcesYaml {
    version: string;
    namespace: string;
    sources: Record<string, SourceDef>;
    [key: string]: unknown;
  }

  let parsed: SourcesYaml | null = null;
  let parseError = "";
  let expandedSource: string | null = null;

  $: {
    try {
      parsed = yaml.load(content) as SourcesYaml;
      parseError = "";
    } catch (e) {
      parseError = e instanceof Error ? e.message : "Invalid YAML";
      parsed = null;
    }
  }

  function toggleSource(name: string) {
    expandedSource = expandedSource === name ? null : name;
  }

  function removeSource(name: string) {
    if (!parsed) return;
    const { [name]: _, ...rest } = parsed.sources;
    const updated = { ...parsed, sources: rest };
    onChange(yaml.dump(updated, { lineWidth: 120, noRefs: true }));
  }

  function removeStream(sourceName: string, streamName: string) {
    if (!parsed) return;
    const src = parsed.sources[sourceName];
    if (!src) return;
    const { [streamName]: _, ...rest } = src.streams;
    const updated = {
      ...parsed,
      sources: {
        ...parsed.sources,
        [sourceName]: { ...src, streams: rest },
      },
    };
    onChange(yaml.dump(updated, { lineWidth: 120, noRefs: true }));
  }
</script>

{#if parseError}
  <div class="rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
    YAML parse error: {parseError}
  </div>
{:else if parsed}
  <div class="space-y-2">
    <div class="flex items-center justify-between px-1">
      <span class="text-xs font-semibold uppercase tracking-wider text-gray-400">
        {Object.keys(parsed.sources ?? {}).length} Sources
      </span>
      <span class="text-xs text-gray-400">namespace: {parsed.namespace}</span>
    </div>

    {#each Object.entries(parsed.sources ?? {}) as [name, src] (name)}
      <div class="rounded border border-gray-200 bg-white">
        <!-- Source header -->
        <button
          class="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm hover:bg-gray-50"
          on:click={() => toggleSource(name)}
        >
          <svg
            class="h-3 w-3 flex-shrink-0 text-gray-400 transition-transform {expandedSource === name ? 'rotate-90' : ''}"
            fill="currentColor" viewBox="0 0 20 20"
          >
            <path d="M6 4l8 6-8 6V4z" />
          </svg>
          <span class="font-medium text-gray-800">{name}</span>
          <span class="rounded bg-blue-100 px-1.5 py-0.5 text-[10px] font-medium text-blue-700">
            {src.type}
          </span>
          {#if src.tap}
            <span class="text-xs text-gray-400">{src.tap}</span>
          {/if}
          <span class="ml-auto text-xs text-gray-400">
            {Object.keys(src.streams ?? {}).length} stream{Object.keys(src.streams ?? {}).length !== 1 ? "s" : ""}
          </span>
          <button
            class="ml-1 rounded p-0.5 text-gray-300 hover:bg-red-50 hover:text-red-500"
            on:click|stopPropagation={() => removeSource(name)}
            title="Remove source"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </button>

        <!-- Streams -->
        {#if expandedSource === name}
          <div class="border-t border-gray-100 px-4 py-2">
            {#if src.raw_table}
              <p class="mb-2 text-xs text-gray-400">raw_table: <span class="font-mono">{src.raw_table}</span></p>
            {/if}
            {#each Object.entries(src.streams ?? {}) as [streamName, stream] (streamName)}
              <div class="mb-1 flex items-center justify-between rounded bg-gray-50 px-3 py-1.5 text-xs">
                <div class="flex items-center gap-2">
                  <span class="font-mono text-gray-700">{streamName}</span>
                  <span class="text-gray-400">
                    {stream.fields?.length ?? 0} field{(stream.fields?.length ?? 0) !== 1 ? "s" : ""}
                  </span>
                </div>
                <button
                  class="rounded p-0.5 text-gray-300 hover:text-red-500"
                  on:click={() => removeStream(name, streamName)}
                  title="Remove stream"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
  </div>
{/if}
