<script lang="ts">
  import { onMount } from "svelte";
  import type { CostTab } from "$lib/bratrax/costs/types";
  import type {
    ProductCogs,
    GatewayFee,
    ExpenseRule,
    StoreSettings,
  } from "$lib/bratrax/costs/types";
  import {
    getStoreSettings,
    saveStoreSettings,
    getProductsCogs,
    saveProductsCogs,
    getGatewayFees,
    saveGatewayFee,
    getExpenseRules,
    createExpenseRule,
    updateExpenseRule,
    deleteExpenseRule,
    createOneTimeExpense,
  } from "$lib/bratrax/costs/api";

  const tabs: { id: CostTab; label: string; icon: string }[] = [
    { id: "cogs", label: "Cost of Goods", icon: "📦" },
    { id: "shipping", label: "Shipping", icon: "🚚" },
    { id: "gateway", label: "Gateway Costs", icon: "💳" },
    { id: "expenses", label: "Custom Expenses", icon: "📋" },
  ];

  let activeTab: CostTab = "gateway";
  let loading = true;
  let saving = false;
  let errorMessage = "";

  // Data
  let storeSettings: StoreSettings = {};
  let products: ProductCogs[] = [];
  let gateways: GatewayFee[] = [];
  let expenseRules: ExpenseRule[] = [];

  // COGS state
  let cogsMode: "per_product" | "global_percent" = "per_product";
  let globalCogsPercent = 0;
  let enableHandlingFee = false;
  let showCogsModal = false;

  // Gateway edit state
  let editingGateway: string | null = null;
  let editPercentage = 0;
  let editFixed = 0;

  // Expense modal state
  let showExpenseModal = false;
  let expenseForm = {
    expense_type: "fixed" as "fixed" | "variable",
    title: "",
    category: "other",
    fixed_amount: 0,
    period: "monthly" as "daily" | "monthly" | "yearly",
    variable_metric: "" as "" | "revenue" | "gross_sales" | "ad_spend",
    variable_percentage: 0,
    is_ad_spend: false,
    start_date: new Date().toISOString().split("T")[0],
    end_date: null as string | null,
    is_active: true,
  };

  function showError(msg: string) {
    errorMessage = msg;
    setTimeout(() => {
      errorMessage = "";
    }, 8000);
  }

  async function loadAll() {
    loading = true;
    try {
      const [settings, prods, gws, rules] = await Promise.all([
        getStoreSettings().catch((e) => { console.error("Settings:", e); return {}; }),
        getProductsCogs().catch((e) => { console.error("COGS:", e); return []; }),
        getGatewayFees().catch((e) => { console.error("Gateways:", e); return []; }),
        getExpenseRules().catch((e) => { console.error("Expenses:", e); return []; }),
      ]);
      storeSettings = settings;
      products = prods;
      gateways = gws;
      expenseRules = rules;

      // Hydrate COGS settings
      const mode = storeSettings?.cogs_mode as Record<string, string> | undefined;
      if (mode?.value) cogsMode = mode.value as typeof cogsMode;
      const pct = storeSettings?.global_cogs_percent as Record<string, string> | undefined;
      if (pct?.value) globalCogsPercent = parseFloat(pct.value) || 0;
      const hf = storeSettings?.enable_handling_fee as Record<string, string> | undefined;
      if (hf?.value) enableHandlingFee = hf.value === "true";
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to load settings");
    } finally {
      loading = false;
    }
  }

  onMount(loadAll);

  // --- COGS handlers ---

  async function handleSaveCogsSettings() {
    saving = true;
    try {
      await saveStoreSettings({
        cogs_mode: cogsMode,
        global_cogs_percent: String(globalCogsPercent),
        enable_handling_fee: String(enableHandlingFee),
      });
      showCogsModal = false;
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to save COGS settings");
    } finally {
      saving = false;
    }
  }

  async function handleSaveProductCogs(product: ProductCogs) {
    saving = true;
    try {
      await saveProductsCogs([
        {
          product_id: product.product_id,
          cogs_amount: product.cogs_amount,
          handling_fee: product.handling_fee,
          title: product.title,
        },
      ]);
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to save product cost");
    } finally {
      saving = false;
    }
  }

  // --- Gateway handlers ---

  function startEditGateway(gw: GatewayFee) {
    editingGateway = gw.gateway_name;
    editPercentage = gw.percentage_fee || 0;
    editFixed = gw.fixed_fee || 0;
  }

  async function handleSaveGateway() {
    if (!editingGateway) return;
    saving = true;
    try {
      await saveGatewayFee(editingGateway, editPercentage, editFixed);
      const idx = gateways.findIndex((g) => g.gateway_name === editingGateway);
      if (idx >= 0) {
        gateways[idx] = {
          ...gateways[idx],
          percentage_fee: editPercentage,
          fixed_fee: editFixed,
        };
        gateways = gateways;
      }
      editingGateway = null;
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to save gateway fee");
    } finally {
      saving = false;
    }
  }

  // --- Expense handlers ---

  function resetExpenseForm() {
    expenseForm = {
      expense_type: "fixed",
      title: "",
      category: "other",
      fixed_amount: 0,
      period: "monthly",
      variable_metric: "",
      variable_percentage: 0,
      is_ad_spend: false,
      start_date: new Date().toISOString().split("T")[0],
      end_date: null,
      is_active: true,
    };
  }

  async function handleCreateExpense() {
    saving = true;
    try {
      await createExpenseRule(expenseForm);
      showExpenseModal = false;
      resetExpenseForm();
      expenseRules = await getExpenseRules();
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to create expense");
    } finally {
      saving = false;
    }
  }

  async function handleDeleteExpense(ruleId: string) {
    saving = true;
    try {
      await deleteExpenseRule(ruleId);
      expenseRules = expenseRules.filter((r) => r.entity_id !== ruleId);
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to delete expense");
    } finally {
      saving = false;
    }
  }

  async function handleToggleExpense(rule: ExpenseRule) {
    saving = true;
    try {
      const updated = { ...rule.data, is_active: !rule.data.is_active };
      await updateExpenseRule(rule.entity_id, updated);
      const idx = expenseRules.findIndex((r) => r.entity_id === rule.entity_id);
      if (idx >= 0) {
        expenseRules[idx] = { ...rule, data: updated };
        expenseRules = expenseRules;
      }
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to update expense");
    } finally {
      saving = false;
    }
  }

  const categories = [
    "agency",
    "software",
    "influencer",
    "fulfillment",
    "shipping",
    "handling",
    "other",
  ];
</script>

<div class="h-full overflow-y-auto">
<div class="mx-auto w-full max-w-6xl px-6 py-8">
  <!-- Header -->
  <div class="mb-8">
    <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
      Cost Settings
    </h1>
    <p class="mt-1 text-sm text-bratrax-text-muted">
      Configure costs to calculate true profit, MER, and blended ROAS.
    </p>
  </div>

  <!-- Error banner -->
  {#if errorMessage}
    <div
      class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato"
    >
      {errorMessage}
      <button
        class="ml-2 text-bratrax-tomato/60 hover:text-bratrax-tomato"
        on:click={() => {
          errorMessage = "";
        }}>&times;</button
      >
    </div>
  {/if}

  <!-- White card wrapper with 4px acid top bar -->
  <div class="cost-card">
    <!-- Sub-tabs as highlighter pills (Round 3 §3) -->
    <div class="cost-subtab-bar">
      {#each tabs as tab}
        <button
          class="cost-subtab"
          class:active={activeTab === tab.id}
          on:click={() => {
            activeTab = tab.id;
          }}
        >
          {tab.label}
        </button>
      {/each}
    </div>

  {#if loading}
    <div class="flex items-center gap-2 px-6 py-12 text-bratrax-text-muted">
      <svg
        class="h-4 w-4 animate-spin"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
      >
        <circle cx="12" cy="12" r="10" stroke-width="2" opacity="0.3" />
        <path d="M12 2a10 10 0 0 1 10 10" stroke-width="2" />
      </svg>
      <span class="font-mono text-xs">Loading cost settings...</span>
    </div>
  {:else}
    <div class="cost-card-body">
    <!-- ================================================================ -->
    <!-- TAB: COGS -->
    <!-- ================================================================ -->
    {#if activeTab === "cogs"}
      <div class="space-y-4">
        <!-- COGS mode toggles -->
        <div class="flex items-center gap-8">
          <label class="flex items-center gap-2 text-sm text-bratrax-text-body">
            <input
              type="checkbox"
              checked={cogsMode === "global_percent"}
              on:change={() => {
                cogsMode =
                  cogsMode === "global_percent"
                    ? "per_product"
                    : "global_percent";
                showCogsModal = cogsMode === "global_percent";
              }}
            />
            Enable COGS as % of Gross Sales
          </label>
          <label class="flex items-center gap-2 text-sm text-bratrax-text-body">
            <input type="checkbox" bind:checked={enableHandlingFee} />
            Enable Fixed Handling Fee
          </label>
        </div>

        <!-- Product table -->
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr
                class="border-b border-bratrax-border font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
              >
                <th class="px-3 py-2 text-left">Title</th>
                <th class="px-3 py-2 text-left">SKU</th>
                <th class="px-3 py-2 text-right">Price</th>
                <th class="px-3 py-2 text-right">Product Cost</th>
                <th class="px-3 py-2 text-right">Handling Fees</th>
                <th class="px-3 py-2 text-right">Qty</th>
              </tr>
            </thead>
            <tbody>
              {#each products as product (product.product_id)}
                <tr
                  class="border-b border-bratrax-border/50 hover:bg-bratrax-hover"
                >
                  <td class="px-3 py-2 text-bratrax-text-body">
                    {product.title || product.product_id}
                  </td>
                  <td class="px-3 py-2 text-bratrax-text-muted text-sm">
                    {product.sku || "—"}
                  </td>
                  <td class="px-3 py-2 text-right text-bratrax-text-body">
                    {product.price ? `$ ${product.price}` : "$ 0"}
                  </td>
                  <td class="px-3 py-2 text-right">
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      class="w-28 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-right text-sm"
                      placeholder="$ 0"
                      bind:value={product.cogs_amount}
                      on:blur={() => handleSaveProductCogs(product)}
                    />
                  </td>
                  <td class="px-3 py-2 text-right">
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      class="w-28 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-right text-sm"
                        placeholder="$ 0"
                        bind:value={product.handling_fee}
                        on:blur={() => handleSaveProductCogs(product)}
                      />
                    </td>
                  <td class="px-3 py-2 text-right text-bratrax-text-body">
                    {product.qty || 0}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <!-- COGS % Modal -->
      {#if showCogsModal}
        <div
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
          on:click|self={() => {
            showCogsModal = false;
          }}
          on:keydown={(e) => {
            if (e.key === "Escape") showCogsModal = false;
          }}
          role="dialog"
          tabindex="-1"
        >
          <div
            class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
          >
            <div
              class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"
            ></div>
            <h3 class="mb-4 text-lg font-bold text-bratrax-text-headline">
              Enable Custom COGS Percent
            </h3>
            <label class="mb-2 block text-sm text-bratrax-text-body">
              Enter Custom COGS Percent
              <div class="mt-1 flex items-center gap-1">
                <span class="text-bratrax-text-muted">%</span>
                <input
                  type="number"
                  step="1"
                  min="0"
                  max="100"
                  class="w-20 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-sm"
                  bind:value={globalCogsPercent}
                />
              </div>
            </label>
            <p class="mb-4 text-xs text-bratrax-text-muted">
              This will apply to all orders as a percentage of gross sales.
            </p>
            <div class="flex justify-end gap-2">
              <button
                class="btn-bratrax btn-neutral btn-compact"
                on:click={() => {
                  showCogsModal = false;
                  cogsMode = "per_product";
                }}>Cancel</button
              >
              <button
                class="btn-bratrax btn-primary btn-compact"
                disabled={saving}
                on:click={handleSaveCogsSettings}>Save</button
              >
            </div>
          </div>
        </div>
      {/if}

      <!-- ================================================================ -->
      <!-- TAB: SHIPPING -->
      <!-- ================================================================ -->
    {:else if activeTab === "shipping"}
      <div class="space-y-6">
        <div class="space-y-3">
          <label class="flex items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover cursor-pointer">
            <input type="radio" name="shipping_mode" value="shopify_charges" checked />
            <div>
              <div class="font-medium text-bratrax-text-headline">Use Shipping Charges for Shipping Costs</div>
              <div class="text-xs text-bratrax-text-muted">Shipping cost equals what your customers were charged.</div>
            </div>
          </label>
          <label class="flex items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover cursor-pointer opacity-50">
            <input type="radio" name="shipping_mode" value="integration" disabled />
            <div>
              <div class="font-medium text-bratrax-text-headline">Use Shipping Integration</div>
              <div class="text-xs text-bratrax-text-muted">Connect shipping providers directly. Coming soon.</div>
            </div>
          </label>
          <label class="flex items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover cursor-pointer opacity-50">
            <input type="radio" name="shipping_mode" value="default_profile" disabled />
            <div>
              <div class="font-medium text-bratrax-text-headline">Default Shipping Costs</div>
              <div class="text-xs text-bratrax-text-muted">Create fulfillment profiles with custom rates. Coming soon.</div>
            </div>
          </label>
        </div>
      </div>

      <!-- ================================================================ -->
      <!-- TAB: GATEWAY COSTS -->
      <!-- ================================================================ -->
    {:else if activeTab === "gateway"}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr
              class="border-b border-bratrax-border font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
            >
              <th class="px-3 py-2 text-left">Payment Gateway Name</th>
              <th class="px-3 py-2 text-left">Cost</th>
              <th class="px-3 py-2 text-left">Fee</th>
              <th class="px-3 py-2 text-right w-24"></th>
            </tr>
          </thead>
          <tbody>
            {#each gateways as gw (gw.gateway_name)}
              <tr class="border-b border-bratrax-border/50 hover:bg-bratrax-hover">
                <td class="px-3 py-2 text-bratrax-text-body">{gw.gateway_name}</td>

                {#if gw.is_shopify_imported}
                  <td colspan="2" class="px-3 py-2 text-xs text-bratrax-text-muted">
                    Fees are imported directly from Shopify
                  </td>
                  <td></td>
                {:else if editingGateway === gw.gateway_name}
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1">
                      <span class="text-bratrax-text-muted">%</span>
                      <input
                        type="number"
                        step="0.1"
                        min="0"
                        class="w-20 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-sm"
                        bind:value={editPercentage}
                      />
                    </div>
                  </td>
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1">
                      <span class="text-bratrax-text-muted">$</span>
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        class="w-20 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-sm"
                        bind:value={editFixed}
                      />
                    </div>
                  </td>
                  <td class="px-3 py-2 text-right">
                    <button
                      class="btn-bratrax btn-primary btn-compact"
                      disabled={saving}
                      on:click={handleSaveGateway}>Save</button
                    >
                  </td>
                {:else}
                  <td class="px-3 py-2 text-bratrax-text-body">
                    {gw.percentage_fee ? `${gw.percentage_fee}%` : "—"}
                  </td>
                  <td class="px-3 py-2 text-bratrax-text-body">
                    {gw.fixed_fee ? `$${gw.fixed_fee}` : "—"}
                  </td>
                  <td class="px-3 py-2 text-right">
                    <button
                      class="btn-bratrax btn-neutral btn-compact"
                      on:click={() => startEditGateway(gw)}>Edit</button
                    >
                  </td>
                {/if}
              </tr>
            {/each}
          </tbody>
        </table>

        {#if gateways.length === 0}
          <p class="py-8 text-center text-sm text-bratrax-text-muted">
            No payment gateways found. Connect Shopify to see your gateways.
          </p>
        {/if}
      </div>

      <!-- ================================================================ -->
      <!-- TAB: CUSTOM EXPENSES -->
      <!-- ================================================================ -->
    {:else if activeTab === "expenses"}
      {#if expenseRules.length === 0}
        <!-- Structured empty state (Round 3 §3) -->
        <div class="cost-empty-state">
          <div class="cost-empty-icon" aria-hidden="true"></div>
          <div class="cost-empty-headline">No custom expenses yet</div>
          <p class="cost-empty-text">
            Add fixed or variable expenses so we can calculate accurate profit, MER, and blended ROAS for your store.
          </p>
          <div class="cost-empty-actions">
            <button
              class="btn-bratrax btn-primary"
              on:click={() => {
                resetExpenseForm();
                expenseForm.expense_type = "fixed";
                showExpenseModal = true;
              }}
            >
              + Add fixed expense
            </button>
            <button
              class="btn-bratrax btn-neutral"
              on:click={() => {
                resetExpenseForm();
                expenseForm.expense_type = "variable";
                showExpenseModal = true;
              }}
            >
              + Add variable expense
            </button>
          </div>
        </div>
      {:else}
        <div class="space-y-4 px-6 py-6">
          <!-- Actions bar -->
          <div class="flex items-center justify-end gap-3">
            <button
              class="btn-bratrax btn-primary btn-compact"
              on:click={() => {
                resetExpenseForm();
                expenseForm.expense_type = "fixed";
                showExpenseModal = true;
              }}
            >
              + Add fixed expense
            </button>
            <button
              class="btn-bratrax btn-neutral btn-compact"
              on:click={() => {
                resetExpenseForm();
                expenseForm.expense_type = "variable";
                showExpenseModal = true;
              }}
            >
              + Add variable expense
            </button>
          </div>

          <!-- Expense rules list -->
          {#each expenseRules as rule (rule.entity_id)}
            <div
              class="flex items-center justify-between border border-bratrax-border p-4"
              class:opacity-50={!rule.data.is_active}
            >
              <div>
                <div class="font-medium text-bratrax-text-headline">
                  {rule.data.title || "Untitled"}
                </div>
                <div class="mt-1 text-xs text-bratrax-text-muted">
                  {#if rule.data.expense_type === "fixed"}
                    ${rule.data.fixed_amount}/{rule.data.period}
                  {:else}
                    {rule.data.variable_percentage}% of {rule.data.variable_metric}
                  {/if}
                  &middot; {rule.data.category}
                  {#if rule.data.is_ad_spend}
                    &middot; Included in ad spend
                  {/if}
                </div>
              </div>
              <div class="flex items-center gap-2">
                <button
                  class="font-mono text-[10px] uppercase text-bratrax-text-muted hover:text-bratrax-acid"
                  on:click={() => handleToggleExpense(rule)}
                >
                  {rule.data.is_active ? "Pause" : "Resume"}
                </button>
                <button
                  class="font-mono text-[10px] uppercase text-bratrax-text-muted hover:text-bratrax-tomato"
                  on:click={() => handleDeleteExpense(rule.entity_id)}
                >
                  Delete
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}

      <!-- Expense modal -->
      {#if showExpenseModal}
        <div
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
          on:click|self={() => {
            showExpenseModal = false;
          }}
          on:keydown={(e) => {
            if (e.key === "Escape") showExpenseModal = false;
          }}
          role="dialog"
          tabindex="-1"
        >
          <div
            class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
          >
            <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>
            <h3 class="mb-4 text-lg font-bold text-bratrax-text-headline">
              Add {expenseForm.expense_type === "fixed" ? "Fixed" : "Variable"} Expense
            </h3>

            <div class="space-y-3">
              <div>
                <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Title</label>
                <input
                  type="text"
                  class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                  placeholder="e.g., Agency retainer"
                  bind:value={expenseForm.title}
                />
              </div>

              <div>
                <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Category</label>
                <select
                  class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                  bind:value={expenseForm.category}
                >
                  {#each categories as cat}
                    <option value={cat}>{cat}</option>
                  {/each}
                </select>
              </div>

              {#if expenseForm.expense_type === "fixed"}
                <div class="flex gap-3">
                  <div class="flex-1">
                    <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Amount</label>
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                      placeholder="$ 0"
                      bind:value={expenseForm.fixed_amount}
                    />
                  </div>
                  <div class="flex-1">
                    <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Period</label>
                    <select
                      class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                      bind:value={expenseForm.period}
                    >
                      <option value="daily">Daily</option>
                      <option value="monthly">Monthly</option>
                      <option value="yearly">Yearly</option>
                    </select>
                  </div>
                </div>
              {:else}
                <div class="flex gap-3">
                  <div class="flex-1">
                    <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Percentage</label>
                    <input
                      type="number"
                      step="0.1"
                      min="0"
                      max="100"
                      class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                      placeholder="% 0"
                      bind:value={expenseForm.variable_percentage}
                    />
                  </div>
                  <div class="flex-1">
                    <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Of Metric</label>
                    <select
                      class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                      bind:value={expenseForm.variable_metric}
                    >
                      <option value="revenue">Revenue</option>
                      <option value="gross_sales">Gross Sales</option>
                      <option value="ad_spend">Ad Spend</option>
                    </select>
                  </div>
                </div>
              {/if}

              <div>
                <label class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted">Start Date</label>
                <input
                  type="date"
                  class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                  bind:value={expenseForm.start_date}
                />
              </div>

              <label class="flex items-center gap-2 text-sm text-bratrax-text-body">
                <input type="checkbox" bind:checked={expenseForm.is_ad_spend} />
                Include in ad spend (affects MER calculation)
              </label>
            </div>

            <div class="mt-6 flex justify-end gap-2">
              <button
                class="btn-bratrax btn-neutral btn-compact"
                on:click={() => {
                  showExpenseModal = false;
                }}>Cancel</button
              >
              <button
                class="btn-bratrax btn-primary btn-compact"
                disabled={saving || !expenseForm.title}
                on:click={handleCreateExpense}>Save</button
              >
            </div>
          </div>
        </div>
      {/if}
    {/if}
    </div>
  {/if}
  </div>
</div>
</div>

<style lang="postcss">
  /* Round 3 §3 — page wraps content in a white card with the 4px acid bar.
     Outer page bg + title block sit on the cream canvas. */
  .cost-card {
    position: relative;
    background: var(--color-elevated);
    border: 0.5px solid var(--color-border);
  }
  .cost-card::before {
    content: "";
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 4px;
    background: var(--color-acid);
    pointer-events: none;
    z-index: 1;
  }
  .cost-card-body {
    padding: 24px;
  }
  /* Empty state owns its own padding (full-bleed center column). */
  .cost-card-body :global(.cost-empty-state) {
    padding: 64px 32px;
  }
  /* Custom-expenses populated state — already has px-6/py-6 from inline class;
     cancel the card-body double-padding for that branch. */
  .cost-card-body :global(.space-y-4.px-6.py-6) {
    padding: 0;
  }

  /* Sub-tab row inside the card. Inactive = dark olive caps; active = solid
     acid pill with black text. Hover (when not active) underlines with acid. */
  .cost-subtab-bar {
    display: flex;
    align-items: center;
    padding: 22px 24px 14px;
    border-bottom: 0.5px solid rgba(0, 0, 0, 0.06);
    flex-wrap: wrap;
    gap: 14px;
  }
  .cost-subtab {
    font-family: "Space Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    padding: 6px 10px;
    color: var(--color-acid-text);
    background: transparent;
    border: none;
    cursor: pointer;
    transition: background-color 120ms ease, color 120ms ease;
  }
  .cost-subtab:hover:not(.active) {
    color: var(--color-text);
    border-bottom: 2px solid var(--color-acid);
    padding-bottom: 4px;
  }
  .cost-subtab.active {
    background: var(--color-acid);
    color: #0A0A0A;
  }

  /* Structured empty state for Custom Expenses tab. */
  .cost-empty-state {
    padding: 64px 32px;
    text-align: center;
  }
  .cost-empty-icon {
    width: 56px;
    height: 56px;
    margin: 0 auto 20px;
    border: 1.5px dashed rgba(0, 0, 0, 0.2);
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  .cost-empty-icon::before {
    content: "";
    position: absolute;
    width: 18px;
    height: 2px;
    background: var(--color-acid-text);
  }
  .cost-empty-icon::after {
    content: "";
    position: absolute;
    width: 2px;
    height: 18px;
    background: var(--color-acid-text);
  }
  .cost-empty-headline {
    font-family: "Outfit", sans-serif;
    font-size: 20px;
    font-weight: 700;
    color: var(--color-text);
    margin-bottom: 8px;
  }
  .cost-empty-text {
    font-family: "Outfit", sans-serif;
    font-size: 14px;
    color: var(--color-text-secondary);
    max-width: 380px;
    margin: 0 auto 24px;
    line-height: 1.6;
  }
  .cost-empty-actions {
    display: flex;
    gap: 10px;
    justify-content: center;
    flex-wrap: wrap;
  }
</style>
