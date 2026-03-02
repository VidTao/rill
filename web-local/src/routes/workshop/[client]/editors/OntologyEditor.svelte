<script lang="ts">
  import yaml from "js-yaml";

  export let content: string;
  export let onChange: (newContent: string) => void;

  interface PropertyDef {
    type: string;
    description?: string;
    pii?: boolean;
    backing?: { source: string; field: string };
    derivation?: string;
    formula?: string;
    [key: string]: unknown;
  }

  interface ObjectDef {
    description?: string;
    primary_key: string;
    table: string;
    properties: Record<string, PropertyDef>;
  }

  interface LinkDef {
    from: { object: string; property: string };
    to: { object: string; property: string };
    relationship: string;
  }

  interface OntologyYaml {
    version: string;
    namespace: string;
    objects: Record<string, ObjectDef>;
    links: Record<string, LinkDef>;
    [key: string]: unknown;
  }

  let parsed: OntologyYaml | null = null;
  let parseError = "";
  let expandedObject: string | null = null;

  $: {
    try {
      parsed = yaml.load(content) as OntologyYaml;
      parseError = "";
    } catch (e) {
      parseError = e instanceof Error ? e.message : "Invalid YAML";
      parsed = null;
    }
  }

  function toggleObject(name: string) {
    expandedObject = expandedObject === name ? null : name;
  }

  function removeObject(name: string) {
    if (!parsed) return;
    const { [name]: _, ...rest } = parsed.objects;
    const updated = { ...parsed, objects: rest };
    onChange(yaml.dump(updated, { lineWidth: 120, noRefs: true }));
  }

  function removeProperty(objName: string, propName: string) {
    if (!parsed) return;
    const obj = parsed.objects[objName];
    if (!obj) return;
    const { [propName]: _, ...rest } = obj.properties;
    const updated = {
      ...parsed,
      objects: {
        ...parsed.objects,
        [objName]: { ...obj, properties: rest },
      },
    };
    onChange(yaml.dump(updated, { lineWidth: 120, noRefs: true }));
  }

  function backingDisplay(prop: PropertyDef): string {
    if (prop.backing) {
      const ref = `${prop.backing.source}.${prop.backing.field}`;
      return ref.replace("$sources.", "");
    }
    if (prop.derivation) return `derivation: ${prop.derivation}`;
    if (prop.formula) return `formula`;
    return "—";
  }

  const TYPE_COLORS: Record<string, string> = {
    string: "bg-green-100 text-green-700",
    integer: "bg-blue-100 text-blue-700",
    float: "bg-amber-100 text-amber-700",
    boolean: "bg-purple-100 text-purple-700",
    timestamp: "bg-red-100 text-red-700",
    date: "bg-red-100 text-red-600",
  };
</script>

{#if parseError}
  <div class="rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
    YAML parse error: {parseError}
  </div>
{:else if parsed}
  <div class="space-y-2">
    <!-- Summary -->
    <div class="flex items-center justify-between px-1">
      <span class="text-xs font-semibold uppercase tracking-wider text-gray-400">
        {Object.keys(parsed.objects ?? {}).length} Objects &middot;
        {Object.keys(parsed.links ?? {}).length} Links
      </span>
      <span class="text-xs text-gray-400">namespace: {parsed.namespace}</span>
    </div>

    <!-- Objects -->
    {#each Object.entries(parsed.objects ?? {}) as [name, obj] (name)}
      <div class="rounded border border-gray-200 bg-white">
        <!-- Object header -->
        <button
          class="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm hover:bg-gray-50"
          on:click={() => toggleObject(name)}
        >
          <svg
            class="h-3 w-3 flex-shrink-0 text-gray-400 transition-transform {expandedObject === name ? 'rotate-90' : ''}"
            fill="currentColor" viewBox="0 0 20 20"
          >
            <path d="M6 4l8 6-8 6V4z" />
          </svg>
          <span class="font-semibold text-gray-800">{name}</span>
          <span class="text-xs text-gray-400">pk: {obj.primary_key}</span>
          <span class="font-mono text-xs text-gray-400">{obj.table}</span>
          <span class="ml-auto text-xs text-gray-400">
            {Object.keys(obj.properties ?? {}).length} properties
          </span>
          <button
            class="ml-1 rounded p-0.5 text-gray-300 hover:bg-red-50 hover:text-red-500"
            on:click|stopPropagation={() => removeObject(name)}
            title="Remove object"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </button>

        <!-- Properties table -->
        {#if expandedObject === name}
          <div class="border-t border-gray-100">
            {#if obj.description}
              <p class="px-4 pt-2 text-xs text-gray-400">{obj.description}</p>
            {/if}
            <table class="w-full text-xs">
              <thead>
                <tr class="border-b border-gray-100 bg-gray-50">
                  <th class="px-4 py-1.5 text-left font-medium text-gray-500">Property</th>
                  <th class="px-3 py-1.5 text-left font-medium text-gray-500">Type</th>
                  <th class="px-3 py-1.5 text-left font-medium text-gray-500">Backing</th>
                  <th class="w-8 px-2 py-1.5"></th>
                </tr>
              </thead>
              <tbody>
                {#each Object.entries(obj.properties ?? {}) as [propName, prop] (propName)}
                  <tr class="border-b border-gray-50 hover:bg-gray-50">
                    <td class="px-4 py-1.5">
                      <span class="font-mono font-medium text-gray-800">{propName}</span>
                      {#if propName === obj.primary_key}
                        <span class="ml-1 rounded bg-yellow-100 px-1 py-0.5 text-[9px] font-bold text-yellow-700">PK</span>
                      {/if}
                      {#if prop.pii}
                        <span class="ml-1 rounded bg-red-100 px-1 py-0.5 text-[9px] font-bold text-red-600">PII</span>
                      {/if}
                    </td>
                    <td class="px-3 py-1.5">
                      <span class="rounded px-1.5 py-0.5 text-[10px] font-medium {TYPE_COLORS[prop.type] ?? 'bg-gray-100 text-gray-600'}">
                        {prop.type}
                      </span>
                    </td>
                    <td class="px-3 py-1.5 font-mono text-gray-500">
                      {backingDisplay(prop)}
                    </td>
                    <td class="px-2 py-1.5 text-right">
                      <button
                        class="rounded p-0.5 text-gray-300 hover:text-red-500"
                        on:click={() => removeProperty(name, propName)}
                        title="Remove property"
                      >
                        <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/each}

    <!-- Links -->
    {#if Object.keys(parsed.links ?? {}).length > 0}
      <div class="mt-4">
        <h3 class="mb-2 px-1 text-xs font-semibold uppercase tracking-wider text-gray-400">Links</h3>
        <div class="space-y-1">
          {#each Object.entries(parsed.links ?? {}) as [linkName, link] (linkName)}
            <div class="flex items-center gap-2 rounded border border-gray-100 bg-white px-4 py-2 text-xs">
              <span class="font-mono font-medium text-gray-700">{link.from.object}</span>
              <span class="text-gray-400">.{link.from.property}</span>
              <span class="text-gray-300">&rarr;</span>
              <span class="font-mono font-medium text-gray-700">{link.to.object}</span>
              <span class="text-gray-400">.{link.to.property}</span>
              <span class="ml-auto rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-500">
                {link.relationship}
              </span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
{/if}
