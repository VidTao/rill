<script lang="ts">
  import yaml from "js-yaml";

  export let content: string;
  export let onChange: (newContent: string) => void;

  interface EventProperty {
    type: string;
    required?: boolean;
    description?: string;
    pii?: boolean;
  }

  interface EventDef {
    category: string;
    source: string;
    trigger: string;
    description?: string;
    properties: Record<string, EventProperty>;
    enrichments?: Record<string, unknown>;
  }

  interface CategoryDef {
    description: string;
    color: string;
  }

  interface TrackingPlanYaml {
    version: string;
    namespace: string;
    categories: Record<string, CategoryDef>;
    events: Record<string, EventDef>;
    [key: string]: unknown;
  }

  let parsed: TrackingPlanYaml | null = null;
  let parseError = "";
  let expandedEvent: string | null = null;
  let filterCategory = "";

  $: {
    try {
      parsed = yaml.load(content) as TrackingPlanYaml;
      parseError = "";
    } catch (e) {
      parseError = e instanceof Error ? e.message : "Invalid YAML";
      parsed = null;
    }
  }

  $: categories = parsed ? Object.keys(parsed.categories ?? {}) : [];

  $: filteredEvents = parsed
    ? Object.entries(parsed.events ?? {}).filter(
        ([, ev]) => !filterCategory || ev.category === filterCategory,
      )
    : [];

  function toggleEvent(name: string) {
    expandedEvent = expandedEvent === name ? null : name;
  }

  function removeEvent(name: string) {
    if (!parsed) return;
    const { [name]: _, ...rest } = parsed.events;
    const updated = { ...parsed, events: rest };
    onChange(yaml.dump(updated, { lineWidth: 120, noRefs: true }));
  }

  const CATEGORY_COLORS: Record<string, string> = {
    blue: "bg-blue-100 text-blue-700 border-blue-200",
    green: "bg-green-100 text-green-700 border-green-200",
    orange: "bg-orange-100 text-orange-700 border-orange-200",
    red: "bg-red-100 text-red-700 border-red-200",
    purple: "bg-purple-100 text-purple-700 border-purple-200",
  };

  function categoryBadge(catName: string): string {
    const cat = parsed?.categories?.[catName];
    const color = cat?.color ?? "gray";
    return CATEGORY_COLORS[color] ?? "bg-gray-100 text-gray-700 border-gray-200";
  }
</script>

{#if parseError}
  <div class="rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
    YAML parse error: {parseError}
  </div>
{:else if parsed}
  <div class="space-y-3">
    <!-- Categories + filter -->
    <div class="flex flex-wrap items-center gap-2 px-1">
      <span class="text-xs font-semibold uppercase tracking-wider text-gray-400">Categories:</span>
      <button
        class="rounded border px-2 py-0.5 text-xs transition
          {filterCategory === '' ? 'border-blue-300 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-500 hover:bg-gray-50'}"
        on:click={() => { filterCategory = ""; }}
      >
        All ({Object.keys(parsed.events ?? {}).length})
      </button>
      {#each categories as cat}
        {@const count = Object.values(parsed.events ?? {}).filter((e) => e.category === cat).length}
        <button
          class="rounded border px-2 py-0.5 text-xs transition
            {filterCategory === cat ? categoryBadge(cat) : 'border-gray-200 text-gray-500 hover:bg-gray-50'}"
          on:click={() => { filterCategory = filterCategory === cat ? "" : cat; }}
        >
          {cat} ({count})
        </button>
      {/each}
    </div>

    <!-- Events -->
    {#each filteredEvents as [name, ev] (name)}
      <div class="rounded border border-gray-200 bg-white">
        <!-- Event header -->
        <button
          class="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm hover:bg-gray-50"
          on:click={() => toggleEvent(name)}
        >
          <svg
            class="h-3 w-3 flex-shrink-0 text-gray-400 transition-transform {expandedEvent === name ? 'rotate-90' : ''}"
            fill="currentColor" viewBox="0 0 20 20"
          >
            <path d="M6 4l8 6-8 6V4z" />
          </svg>
          <span class="font-mono font-medium text-gray-800">{name}</span>
          <span class="rounded border px-1.5 py-0.5 text-[10px] font-medium {categoryBadge(ev.category)}">
            {ev.category}
          </span>
          <span class="ml-auto text-xs text-gray-400">
            {Object.keys(ev.properties ?? {}).length} props
          </span>
          <button
            class="ml-1 rounded p-0.5 text-gray-300 hover:bg-red-50 hover:text-red-500"
            on:click|stopPropagation={() => removeEvent(name)}
            title="Remove event"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </button>

        <!-- Event detail -->
        {#if expandedEvent === name}
          <div class="border-t border-gray-100 px-4 py-2 text-xs">
            <div class="mb-2 space-y-1">
              <p><span class="font-medium text-gray-500">Source:</span> <span class="font-mono text-gray-700">{ev.source}</span></p>
              <p><span class="font-medium text-gray-500">Trigger:</span> <span class="text-gray-700">{ev.trigger}</span></p>
              {#if ev.description}
                <p><span class="font-medium text-gray-500">Description:</span> <span class="text-gray-700">{ev.description}</span></p>
              {/if}
            </div>

            <!-- Properties table -->
            {#if Object.keys(ev.properties ?? {}).length > 0}
              <table class="w-full">
                <thead>
                  <tr class="border-b border-gray-100 bg-gray-50">
                    <th class="px-3 py-1.5 text-left font-medium text-gray-500">Property</th>
                    <th class="px-3 py-1.5 text-left font-medium text-gray-500">Type</th>
                    <th class="px-3 py-1.5 text-left font-medium text-gray-500">Required</th>
                    <th class="px-3 py-1.5 text-left font-medium text-gray-500">Description</th>
                  </tr>
                </thead>
                <tbody>
                  {#each Object.entries(ev.properties ?? {}) as [propName, prop] (propName)}
                    <tr class="border-b border-gray-50 hover:bg-gray-50">
                      <td class="px-3 py-1 font-mono text-gray-800">
                        {propName}
                        {#if prop.pii}
                          <span class="ml-1 rounded bg-red-100 px-1 py-0.5 text-[9px] font-bold text-red-600">PII</span>
                        {/if}
                      </td>
                      <td class="px-3 py-1 font-mono text-gray-500">{prop.type}</td>
                      <td class="px-3 py-1 text-gray-400">{prop.required ? "yes" : "no"}</td>
                      <td class="px-3 py-1 text-gray-400">{prop.description ?? ""}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}

            <!-- Enrichments -->
            {#if ev.enrichments && Object.keys(ev.enrichments).length > 0}
              <div class="mt-2">
                <h4 class="mb-1 font-medium text-gray-500">Enrichments</h4>
                {#each Object.entries(ev.enrichments) as [enrName, enr] (enrName)}
                  <div class="rounded bg-blue-50 px-3 py-1.5 font-mono text-gray-700">
                    {enrName}: {JSON.stringify(enr)}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  </div>
{/if}
