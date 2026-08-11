<script lang="ts">
  import { onMount } from "svelte";
  import CalculationPreferences from "$lib/bratrax/costs/CalculationPreferences.svelte";
  import type { CostTab } from "$lib/bratrax/costs/types";
  import type {
    ProductCogs,
    MarketplaceProductCogs,
    GatewayFee,
    ExpenseRule,
    MediaSpendScopeGuidance,
    MediaSpendScopeAccountOption,
    MediaSpendScopeRule,
    MediaSpendScopeRuleData,
    MediaSpendScopeAction,
    MediaSpendScopeChannel,
    MediaSpendScopeMatchField,
    MediaSpendScopeOperator,
    StoreSettings,
  } from "$lib/bratrax/costs/types";
  import {
    getStoreSettings,
    saveStoreSettings,
    getProductsCogs,
    saveProductsCogs,
    getMarketplaceProductsCogs,
    saveMarketplaceProductsCogs,
    getGatewayFees,
    saveGatewayFee,
    getExpenseRules,
    createExpenseRule,
    updateExpenseRule,
    deleteExpenseRule,
    getMediaSpendScopeRules,
    createMediaSpendScopeRule,
    updateMediaSpendScopeRule,
    deleteMediaSpendScopeRule,
    createOneTimeExpense,
  } from "$lib/bratrax/costs/api";

  const tabs: { id: CostTab; label: string; icon: string }[] = [
    { id: "cogs", label: "Cost of Goods", icon: "📦" },
    { id: "amazon_cogs", label: "Amazon COGS", icon: "A" },
    { id: "shipping", label: "Shipping", icon: "🚚" },
    { id: "gateway", label: "Gateway Costs", icon: "💳" },
    { id: "expenses", label: "Custom Expenses", icon: "📋" },
    { id: "media_scope", label: "Media Scope", icon: "🎯" },
    { id: "calculation", label: "Profit Rules", icon: "∑" },
  ];

  let activeTab: CostTab = "gateway";
  let loading = true;
  let saving = false;
  let errorMessage = "";
  let savedMessage = "";

  // Data
  let storeSettings: StoreSettings = {};
  let products: ProductCogs[] = [];
  let marketplaceProducts: MarketplaceProductCogs[] = [];
  let gateways: GatewayFee[] = [];
  let expenseRules: ExpenseRule[] = [];
  let mediaScopeRules: MediaSpendScopeRule[] = [];
  let mediaScopeAvailableAccounts: MediaSpendScopeAccountOption[] = [];
  $: filteredMediaScopeAccounts = mediaScopeAvailableAccounts.filter(
    (account) =>
      !mediaScopeForm.channel || account.channel === mediaScopeForm.channel,
  );
  let mediaScopeGuidance: MediaSpendScopeGuidance | null = null;

  // COGS state
  let cogsMode: "per_product" | "global_percent" = "per_product";
  let globalCogsPercent = 0;
  let enableHandlingFee = false;
  let showCogsModal = false;

  // Shipping state
  let shippingCostMode: "customer_charges" | "flat_rate" = "customer_charges";
  let defaultShippingCost = 0;

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

  // Media scope modal state
  let showMediaScopeModal = false;
  let editingMediaScopeRuleId: string | null = null;
  let mediaScopeForm: MediaSpendScopeRuleData = {
    name: "",
    channel: "",
    account_id: "",
    match_field: "campaign_name",
    operator: "regex",
    match_value: "",
    action: "include",
    priority: 100,
    start_date: "",
    end_date: "",
    is_active: false,
  };

  function showError(msg: string) {
    errorMessage = msg;
    setTimeout(() => {
      errorMessage = "";
    }, 8000);
  }

  function showSaved(msg: string) {
    savedMessage = msg;
    setTimeout(() => {
      savedMessage = "";
    }, 8000);
  }

  async function loadAll() {
    loading = true;
    try {
      const [settings, prods, marketplaceProds, gws, rules, mediaRules] =
        await Promise.all([
          getStoreSettings().catch((e) => {
            console.error("Settings:", e);
            return {};
          }),
          getProductsCogs().catch((e) => {
            console.error("COGS:", e);
            return [];
          }),
          getMarketplaceProductsCogs().catch((e) => {
            console.error("Amazon COGS:", e);
            return [];
          }),
          getGatewayFees().catch((e) => {
            console.error("Gateways:", e);
            return [];
          }),
          getExpenseRules().catch((e) => {
            console.error("Expenses:", e);
            return [];
          }),
          getMediaSpendScopeRules().catch((e) => {
            console.error("Media scope:", e);
            return { rules: [], guidance: null, availableAccounts: [] };
          }),
        ]);
      storeSettings = settings;
      products = prods;
      marketplaceProducts = marketplaceProds;
      gateways = gws;
      expenseRules = rules;
      mediaScopeRules = mediaRules.rules;
      mediaScopeGuidance = mediaRules.guidance;
      mediaScopeAvailableAccounts = mediaRules.availableAccounts;

      // Hydrate COGS settings
      const mode = storeSettings?.cogs_mode as
        | Record<string, string>
        | undefined;
      if (mode?.value) cogsMode = mode.value as typeof cogsMode;
      const pct = storeSettings?.global_cogs_percent as
        | Record<string, string>
        | undefined;
      if (pct?.value) globalCogsPercent = parseFloat(pct.value) || 0;
      const hf = storeSettings?.enable_handling_fee as
        | Record<string, string>
        | undefined;
      if (hf?.value !== undefined) {
        enableHandlingFee = hf.value === true || hf.value === "true";
      }

      const shippingMode = storeSettings?.shipping_cost_mode as
        | { value?: unknown }
        | undefined;
      if (
        shippingMode?.value === "customer_charges" ||
        shippingMode?.value === "flat_rate"
      ) {
        shippingCostMode = shippingMode.value;
      }
      const shippingCost = storeSettings?.default_shipping_cost as
        | { value?: unknown }
        | undefined;
      const parsedShippingCost = Number(shippingCost?.value);
      if (Number.isFinite(parsedShippingCost) && parsedShippingCost >= 0) {
        defaultShippingCost = parsedShippingCost;
      }
    } catch (e) {
      showError(e instanceof Error ? e.message : "Failed to load settings");
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    // Honor ?tab= deep-links (the onboarding checklist links here with
    // ?tab=cogs / ?tab=shipping / ?tab=expenses).
    const requested = new URLSearchParams(window.location.search).get("tab");
    if (requested && tabs.some((t) => t.id === requested)) {
      activeTab = requested as CostTab;
    }
    void loadAll();
  });

  // --- COGS handlers ---

  async function handleSaveCogsSettings() {
    saving = true;
    try {
      await saveStoreSettings({
        cogs_mode: cogsMode,
        global_cogs_percent: globalCogsPercent,
        enable_handling_fee: enableHandlingFee,
      });
      showCogsModal = false;
    } catch (e) {
      showError(
        e instanceof Error ? e.message : "Failed to save COGS settings",
      );
    } finally {
      saving = false;
    }
  }

  // --- Shipping handlers ---

  async function handleSaveShippingSettings() {
    if (!Number.isFinite(defaultShippingCost) || defaultShippingCost < 0) {
      showError("Default shipping cost must be a non-negative number");
      return;
    }

    saving = true;
    try {
      await saveStoreSettings({
        shipping_cost_mode: shippingCostMode,
        default_shipping_cost: defaultShippingCost,
      });
      showSaved("Shipping settings saved. Dashboard refresh queued.");
    } catch (e) {
      showError(
        e instanceof Error ? e.message : "Failed to save shipping settings",
      );
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

  async function handleSaveMarketplaceProductCogs(
    product: MarketplaceProductCogs,
  ) {
    saving = true;
    try {
      await saveMarketplaceProductsCogs([
        {
          platform: "amazon",
          marketplace_id: product.marketplace_id,
          asin: product.asin,
          seller_sku: product.seller_sku,
          title: product.title,
          unit_cost: product.unit_cost,
          handling_cost: product.handling_cost,
        },
      ]);
      product.is_configured = product.unit_cost !== null;
      showSaved("Amazon product cost saved. Dashboard refresh queued.");
    } catch (e) {
      showError(
        e instanceof Error ? e.message : "Failed to save Amazon product cost",
      );
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

  // --- Media scope handlers ---

  const mediaScopeChannels: Array<{
    value: MediaSpendScopeChannel;
    label: string;
  }> = [
    { value: "", label: "All channels" },
    { value: "meta", label: "Meta" },
    { value: "google", label: "Google" },
    { value: "tiktok", label: "TikTok" },
    { value: "bing", label: "Bing" },
    { value: "taboola", label: "Taboola" },
    { value: "outbrain", label: "Outbrain" },
    { value: "pinterest", label: "Pinterest" },
  ];

  const mediaScopeActions: Array<{
    value: MediaSpendScopeAction;
    label: string;
  }> = [
    { value: "include", label: "Include matching spend" },
    { value: "exclude", label: "Exclude matching spend" },
  ];

  const mediaScopeMatchFields: Array<{
    value: MediaSpendScopeMatchField;
    label: string;
  }> = [
    { value: "campaign_name", label: "Campaign name" },
    { value: "campaign_id", label: "Campaign ID" },
    { value: "ad_set_name", label: "Ad set / ad group name" },
    { value: "ad_set_id", label: "Ad set ID" },
    { value: "ad_name", label: "Ad name" },
    { value: "ad_id", label: "Ad ID" },
    { value: "account_id", label: "Account ID" },
  ];

  const mediaScopeOperators: Array<{
    value: MediaSpendScopeOperator;
    label: string;
    advanced?: boolean;
  }> = [
    { value: "equals", label: "Equals" },
    { value: "prefix", label: "Starts with" },
    { value: "contains", label: "Contains" },
    { value: "regex", label: "Regex", advanced: true },
  ];

  function mediaScopeLabel<T extends string>(
    items: Array<{ value: T; label: string }>,
    value: T,
  ) {
    return items.find((item) => item.value === value)?.label ?? value;
  }

  function resetMediaScopeForm() {
    editingMediaScopeRuleId = null;
    mediaScopeForm = {
      name: "",
      channel: "",
      account_id: "",
      match_field: "campaign_name",
      operator: "regex",
      match_value: "",
      action: "include",
      priority: 100,
      start_date: "",
      end_date: "",
      is_active: false,
    };
  }

  function openMediaScopeCreate() {
    resetMediaScopeForm();
    showMediaScopeModal = true;
  }

  function openMediaScopeEdit(rule: MediaSpendScopeRule) {
    editingMediaScopeRuleId = rule.entity_id;
    mediaScopeForm = {
      name: rule.data.name || "",
      channel: rule.data.channel || "",
      account_id: rule.data.account_id || "",
      match_field: rule.data.match_field || "campaign_name",
      operator: rule.data.operator || "regex",
      match_value: rule.data.match_value || "",
      action: rule.data.action || "include",
      priority: rule.data.priority ?? 100,
      start_date: rule.data.start_date || "",
      end_date: rule.data.end_date || "",
      is_active: rule.data.is_active ?? true,
    };
    showMediaScopeModal = true;
  }

  function normalizedMediaScopeForm(): MediaSpendScopeRuleData {
    return {
      ...mediaScopeForm,
      name: mediaScopeForm.name.trim(),
      account_id: mediaScopeForm.account_id.trim(),
      match_value: mediaScopeForm.match_value.trim(),
      priority: Number(mediaScopeForm.priority) || 100,
    };
  }

  async function reloadMediaScopeRules() {
    const result = await getMediaSpendScopeRules();
    mediaScopeRules = result.rules;
    mediaScopeGuidance = result.guidance;
    mediaScopeAvailableAccounts = result.availableAccounts;
  }

  async function handleSaveMediaScopeRule() {
    saving = true;
    try {
      const payload = normalizedMediaScopeForm();
      if (editingMediaScopeRuleId) {
        await updateMediaSpendScopeRule(editingMediaScopeRuleId, payload);
      } else {
        await createMediaSpendScopeRule(payload);
      }
      showMediaScopeModal = false;
      resetMediaScopeForm();
      await reloadMediaScopeRules();
      showSaved("Media scope rule saved. Dashboard refresh queued.");
    } catch (e) {
      showError(
        e instanceof Error ? e.message : "Failed to save media scope rule",
      );
    } finally {
      saving = false;
    }
  }

  async function handleToggleMediaScopeRule(rule: MediaSpendScopeRule) {
    saving = true;
    try {
      await updateMediaSpendScopeRule(rule.entity_id, {
        ...rule.data,
        is_active: !rule.data.is_active,
      });
      await reloadMediaScopeRules();
      showSaved("Media scope rule updated. Dashboard refresh queued.");
    } catch (e) {
      showError(
        e instanceof Error ? e.message : "Failed to update media scope rule",
      );
    } finally {
      saving = false;
    }
  }

  async function handleDeleteMediaScopeRule(ruleId: string) {
    saving = true;
    try {
      await deleteMediaSpendScopeRule(ruleId);
      mediaScopeRules = mediaScopeRules.filter(
        (rule) => rule.entity_id !== ruleId,
      );
      showSaved("Media scope rule deleted. Dashboard refresh queued.");
    } catch (e) {
      showError(
        e instanceof Error ? e.message : "Failed to delete media scope rule",
      );
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

    <!-- Saved banner -->
    {#if savedMessage}
      <div
        class="mb-4 border border-bratrax-acid/40 bg-bratrax-acid/10 px-4 py-3 font-mono text-xs text-bratrax-acid-text"
      >
        {savedMessage}
        <button
          class="ml-2 text-bratrax-text-muted hover:text-bratrax-text-headline"
          on:click={() => {
            savedMessage = "";
          }}>&times;</button
        >
      </div>
    {/if}

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
                <label
                  class="flex items-center gap-2 text-sm text-bratrax-text-body"
                >
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
                <label
                  class="flex items-center gap-2 text-sm text-bratrax-text-body"
                >
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
                    This will apply to all orders as a percentage of gross
                    sales.
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
          {:else if activeTab === "amazon_cogs"}
            <div class="space-y-4">
              <div class="border border-bratrax-border bg-bratrax-hover p-4">
                <div class="font-medium text-bratrax-text-headline">
                  Amazon US SKU Costs
                </div>
                <div class="mt-1 text-xs text-bratrax-text-muted">
                  Product and handling costs are matched by Amazon seller SKU.
                  Profit stays provisional until all shipped units have a cost.
                </div>
              </div>
              <div class="overflow-x-auto">
                <table class="w-full text-sm">
                  <thead>
                    <tr
                      class="border-b border-bratrax-border font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
                    >
                      <th class="px-3 py-2 text-left">Product</th>
                      <th class="px-3 py-2 text-left">Seller SKU</th>
                      <th class="px-3 py-2 text-left">ASIN</th>
                      <th class="px-3 py-2 text-right">Unit Cost</th>
                      <th class="px-3 py-2 text-right">Handling</th>
                      <th class="px-3 py-2 text-right">Units Shipped</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each marketplaceProducts as product (product.cost_id)}
                      <tr
                        class="border-b border-bratrax-border/50 hover:bg-bratrax-hover"
                      >
                        <td class="px-3 py-2 text-bratrax-text-body">
                          {product.title || product.asin}
                        </td>
                        <td
                          class="px-3 py-2 font-mono text-xs text-bratrax-text-muted"
                        >
                          {product.seller_sku || "—"}
                        </td>
                        <td
                          class="px-3 py-2 font-mono text-xs text-bratrax-text-muted"
                        >
                          {product.asin || "—"}
                        </td>
                        <td class="px-3 py-2 text-right">
                          <input
                            type="number"
                            step="0.01"
                            min="0"
                            class="w-28 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-right text-sm"
                            placeholder="$ 0"
                            bind:value={product.unit_cost}
                            on:blur={() =>
                              handleSaveMarketplaceProductCogs(product)}
                          />
                        </td>
                        <td class="px-3 py-2 text-right">
                          <input
                            type="number"
                            step="0.01"
                            min="0"
                            class="w-28 border border-bratrax-border bg-bratrax-surface px-2 py-1 text-right text-sm"
                            placeholder="$ 0"
                            bind:value={product.handling_cost}
                            on:blur={() =>
                              handleSaveMarketplaceProductCogs(product)}
                          />
                        </td>
                        <td class="px-3 py-2 text-right text-bratrax-text-body">
                          {product.units_shipped || 0}
                        </td>
                      </tr>
                    {/each}
                    {#if marketplaceProducts.length === 0}
                      <tr>
                        <td
                          colspan="6"
                          class="px-3 py-10 text-center text-bratrax-text-muted"
                        >
                          Amazon products will appear after the first Seller
                          sync.
                        </td>
                      </tr>
                    {/if}
                  </tbody>
                </table>
              </div>
            </div>
          {:else if activeTab === "shipping"}
            <div class="space-y-6">
              <div class="space-y-3">
                <label
                  class="flex items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover cursor-pointer"
                >
                  <input
                    type="radio"
                    name="shipping_mode"
                    value="customer_charges"
                    bind:group={shippingCostMode}
                  />
                  <div>
                    <div class="font-medium text-bratrax-text-headline">
                      Use Shipping Charges for Shipping Costs
                    </div>
                    <div class="text-xs text-bratrax-text-muted">
                      Shipping cost equals what customers were charged.
                    </div>
                  </div>
                </label>

                <label
                  class="flex items-start gap-3 border border-bratrax-border p-4 hover:bg-bratrax-hover cursor-pointer"
                >
                  <input
                    type="radio"
                    name="shipping_mode"
                    value="flat_rate"
                    bind:group={shippingCostMode}
                  />
                  <div>
                    <div class="font-medium text-bratrax-text-headline">
                      Default Shipping Costs
                    </div>
                    <div class="text-xs text-bratrax-text-muted">
                      Apply one operational shipping cost to every
                      sales-eligible order, including free-shipping orders.
                    </div>
                  </div>
                </label>

                {#if shippingCostMode === "flat_rate"}
                  <div
                    class="border border-bratrax-border bg-bratrax-surface p-4"
                  >
                    <label
                      class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                    >
                      Cost per order
                    </label>
                    <div class="flex items-center gap-2">
                      <span class="text-sm text-bratrax-text-muted">€</span>
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        class="w-32 border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                        bind:value={defaultShippingCost}
                      />
                    </div>
                  </div>
                {/if}

                <label
                  class="flex items-start gap-3 border border-bratrax-border p-4 opacity-50"
                >
                  <input
                    type="radio"
                    name="shipping_mode"
                    value="integration"
                    disabled
                  />
                  <div>
                    <div class="font-medium text-bratrax-text-headline">
                      Use Shipping Integration
                    </div>
                    <div class="text-xs text-bratrax-text-muted">
                      Connect shipping providers directly. Coming soon.
                    </div>
                  </div>
                </label>
              </div>

              <div class="flex justify-end">
                <button
                  class="btn-bratrax btn-primary btn-compact"
                  disabled={saving}
                  on:click={handleSaveShippingSettings}
                >
                  {saving ? "Saving…" : "Save Shipping Settings"}
                </button>
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
                    <tr
                      class="border-b border-bratrax-border/50 hover:bg-bratrax-hover"
                    >
                      <td class="px-3 py-2 text-bratrax-text-body"
                        >{gw.gateway_name}</td
                      >

                      {#if gw.is_shopify_imported}
                        <td
                          colspan="2"
                          class="px-3 py-2 text-xs text-bratrax-text-muted"
                        >
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
                  No payment gateways found. Connect Shopify to see your
                  gateways.
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
                  Add fixed or variable expenses so we can calculate accurate
                  profit, MER, and blended ROAS for your store.
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
                          {rule.data.variable_percentage}% of {rule.data
                            .variable_metric}
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
                  <div
                    class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"
                  ></div>
                  <h3 class="mb-4 text-lg font-bold text-bratrax-text-headline">
                    Add {expenseForm.expense_type === "fixed"
                      ? "Fixed"
                      : "Variable"} Expense
                  </h3>

                  <div class="space-y-3">
                    <div>
                      <label
                        class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                        >Title</label
                      >
                      <input
                        type="text"
                        class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                        placeholder="e.g., Agency retainer"
                        bind:value={expenseForm.title}
                      />
                    </div>

                    <div>
                      <label
                        class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                        >Category</label
                      >
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
                          <label
                            class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                            >Amount</label
                          >
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
                          <label
                            class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                            >Period</label
                          >
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
                          <label
                            class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                            >Percentage</label
                          >
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
                          <label
                            class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                            >Of Metric</label
                          >
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
                      <label
                        class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                        >Start Date</label
                      >
                      <input
                        type="date"
                        class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                        bind:value={expenseForm.start_date}
                      />
                    </div>

                    <label
                      class="flex items-center gap-2 text-sm text-bratrax-text-body"
                    >
                      <input
                        type="checkbox"
                        bind:checked={expenseForm.is_ad_spend}
                      />
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
            <!-- ================================================================ -->
            <!-- TAB: MEDIA SCOPE -->
            <!-- ================================================================ -->
          {:else if activeTab === "calculation"}
            <CalculationPreferences />

            <!-- ================================================================ -->
            <!-- TAB: MEDIA SCOPE -->
            <!-- ================================================================ -->
          {:else if activeTab === "media_scope"}
            <div class="space-y-5">
              <div class="media-scope-guidance">
                <div>
                  <div
                    class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted"
                  >
                    Shared ad account controls
                  </div>
                  <h2 class="mt-1 text-lg font-bold text-bratrax-text-headline">
                    {mediaScopeGuidance?.title ||
                      "Media spend scope rules are exact, not fuzzy."}
                  </h2>
                  <p
                    class="mt-2 max-w-3xl text-sm leading-6 text-bratrax-text-body"
                  >
                    {mediaScopeGuidance?.body ||
                      "Use these rules when one ad account sends traffic to multiple stores. Matching spend is kept for this store; unmatched spend is excluded when include rules exist."}
                  </p>
                  {#if mediaScopeGuidance?.unmatched_policy}
                    <p
                      class="mt-2 font-mono text-[11px] uppercase tracking-wide text-bratrax-text-muted"
                    >
                      {mediaScopeGuidance.unmatched_policy}
                    </p>
                  {/if}
                </div>
                <button
                  class="btn-bratrax btn-primary btn-compact"
                  on:click={openMediaScopeCreate}
                >
                  + Add rule
                </button>
              </div>

              {#if mediaScopeGuidance?.examples?.length}
                <div class="grid gap-3 md:grid-cols-2">
                  {#each mediaScopeGuidance.examples as example}
                    <div
                      class="border border-bratrax-border bg-bratrax-surface p-3"
                    >
                      <div
                        class="font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                      >
                        Example
                      </div>
                      <div
                        class="mt-1 font-mono text-xs text-bratrax-text-headline"
                      >
                        {example.rule}
                      </div>
                      <div class="mt-2 text-xs text-bratrax-text-body">
                        Matches: {example.matches}
                      </div>
                      <div class="text-xs text-bratrax-text-muted">
                        Does not match: {example.does_not_match}
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}

              {#if mediaScopeRules.length === 0}
                <div class="cost-empty-state">
                  <div class="cost-empty-icon" aria-hidden="true"></div>
                  <div class="cost-empty-headline">
                    No media scope rules yet
                  </div>
                  <p class="cost-empty-text">
                    Add rules only when this store shares ad accounts with
                    another store. With no rules, all ad spend stays included.
                  </p>
                  <div class="cost-empty-actions">
                    <button
                      class="btn-bratrax btn-primary"
                      on:click={openMediaScopeCreate}
                    >
                      + Add media scope rule
                    </button>
                  </div>
                </div>
              {:else}
                <div class="space-y-3">
                  {#each mediaScopeRules as rule (rule.entity_id)}
                    <div
                      class="media-scope-rule"
                      class:opacity-50={!rule.data.is_active}
                    >
                      <div class="min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                          <span class="font-medium text-bratrax-text-headline">
                            {rule.data.name || "Untitled rule"}
                          </span>
                          <span
                            class="media-scope-badge"
                            class:exclude={rule.data.action === "exclude"}
                          >
                            {rule.data.action === "include"
                              ? "Include"
                              : "Exclude"}
                          </span>
                          {#if !rule.data.is_active}
                            <span class="media-scope-badge muted">Paused</span>
                          {/if}
                        </div>
                        <div
                          class="mt-2 text-xs leading-5 text-bratrax-text-muted"
                        >
                          {mediaScopeLabel(
                            mediaScopeChannels,
                            rule.data.channel || "",
                          )} · {mediaScopeLabel(
                            mediaScopeMatchFields,
                            rule.data.match_field,
                          )}
                          {mediaScopeLabel(
                            mediaScopeOperators,
                            rule.data.operator,
                          ).toLowerCase()}
                          <span class="font-mono text-bratrax-text-body"
                            >{rule.data.match_value}</span
                          >
                          {#if rule.data.account_id}
                            · account <span
                              class="font-mono text-bratrax-text-body"
                              >{rule.data.account_id}</span
                            >
                          {/if}
                          · priority {rule.data.priority}
                          {#if rule.data.start_date || rule.data.end_date}
                            · {rule.data.start_date || "any"} to {rule.data
                              .end_date || "any"}
                          {/if}
                        </div>
                      </div>
                      <div class="flex shrink-0 items-center gap-2">
                        <button
                          class="font-mono text-[10px] uppercase text-bratrax-text-muted hover:text-bratrax-acid"
                          on:click={() => handleToggleMediaScopeRule(rule)}
                        >
                          {rule.data.is_active ? "Pause" : "Resume"}
                        </button>
                        <button
                          class="font-mono text-[10px] uppercase text-bratrax-text-muted hover:text-bratrax-acid"
                          on:click={() => openMediaScopeEdit(rule)}
                        >
                          Edit
                        </button>
                        <button
                          class="font-mono text-[10px] uppercase text-bratrax-text-muted hover:text-bratrax-tomato"
                          on:click={() =>
                            handleDeleteMediaScopeRule(rule.entity_id)}
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>

            <!-- Media scope modal -->
            {#if showMediaScopeModal}
              <div
                class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4"
                on:click|self={() => {
                  showMediaScopeModal = false;
                }}
                on:keydown={(e) => {
                  if (e.key === "Escape") showMediaScopeModal = false;
                }}
                role="dialog"
                tabindex="-1"
              >
                <div
                  class="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto border border-bratrax-border bg-bratrax-surface p-6"
                >
                  <div
                    class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"
                  ></div>
                  <h3 class="mb-1 text-lg font-bold text-bratrax-text-headline">
                    {editingMediaScopeRuleId
                      ? "Edit media scope rule"
                      : "Add media scope rule"}
                  </h3>
                  <p class="mb-5 text-xs leading-5 text-bratrax-text-muted">
                    Rules are exact. Use another explicit rule or fix the
                    campaign name if buyers use malformed campaign prefixes.
                  </p>

                  <div class="space-y-4">
                    <div>
                      <label
                        for="media-scope-name"
                        class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                        >Rule name</label
                      >
                      <input
                        id="media-scope-name"
                        type="text"
                        class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                        placeholder="e.g., Vibit AU campaigns"
                        bind:value={mediaScopeForm.name}
                      />
                    </div>

                    <div class="grid gap-3 md:grid-cols-2">
                      <div>
                        <label
                          for="media-scope-action"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Action</label
                        >
                        <select
                          id="media-scope-action"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.action}
                        >
                          {#each mediaScopeActions as item}
                            <option value={item.value}>{item.label}</option>
                          {/each}
                        </select>
                      </div>
                      <div>
                        <label
                          for="media-scope-channel"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Channel</label
                        >
                        <select
                          id="media-scope-channel"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.channel}
                        >
                          {#each mediaScopeChannels as item}
                            <option value={item.value}>{item.label}</option>
                          {/each}
                        </select>
                      </div>
                    </div>

                    <div class="grid gap-3 md:grid-cols-3">
                      <div>
                        <label
                          for="media-scope-match-field"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Match field</label
                        >
                        <select
                          id="media-scope-match-field"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.match_field}
                        >
                          {#each mediaScopeMatchFields as item}
                            <option value={item.value}>{item.label}</option>
                          {/each}
                        </select>
                      </div>
                      <div>
                        <label
                          for="media-scope-operator"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Operator</label
                        >
                        <select
                          id="media-scope-operator"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.operator}
                        >
                          {#each mediaScopeOperators as item}
                            <option value={item.value}
                              >{item.label}{item.advanced
                                ? " - advanced"
                                : ""}</option
                            >
                          {/each}
                        </select>
                      </div>
                      <div>
                        <label
                          for="media-scope-priority"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Priority</label
                        >
                        <input
                          id="media-scope-priority"
                          type="number"
                          step="1"
                          min="0"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.priority}
                        />
                      </div>
                    </div>

                    <div>
                      <label
                        for="media-scope-match-value"
                        class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                        >Match value</label
                      >
                      <input
                        id="media-scope-match-value"
                        type="text"
                        class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 font-mono text-sm"
                        placeholder={mediaScopeForm.operator === "regex"
                          ? "^\\s*(\\[(AUS|AU)\\]|(AUS|AU))(\\s|[-|])"
                          : "[AU]"}
                        bind:value={mediaScopeForm.match_value}
                      />
                    </div>

                    <div class="grid gap-3 md:grid-cols-3">
                      <div>
                        <label
                          for="media-scope-account-id"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Account ID</label
                        >
                        <input
                          id="media-scope-account-id"
                          type="text"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          placeholder="Optional; choose a known account or type an ID"
                          list="media-scope-account-options"
                          bind:value={mediaScopeForm.account_id}
                        />
                        <datalist id="media-scope-account-options">
                          {#each filteredMediaScopeAccounts as account}
                            <option value={account.account_id}>
                              {account.channel} · {account.account_id}
                            </option>
                          {/each}
                        </datalist>
                      </div>
                      <div>
                        <label
                          for="media-scope-start-date"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >Start date</label
                        >
                        <input
                          id="media-scope-start-date"
                          type="date"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.start_date}
                        />
                      </div>
                      <div>
                        <label
                          for="media-scope-end-date"
                          class="mb-1 block font-mono text-[11px] font-bold uppercase text-bratrax-text-muted"
                          >End date</label
                        >
                        <input
                          id="media-scope-end-date"
                          type="date"
                          class="w-full border border-bratrax-border bg-bratrax-surface px-3 py-2 text-sm"
                          bind:value={mediaScopeForm.end_date}
                        />
                      </div>
                    </div>

                    <label
                      class="flex items-center gap-2 text-sm text-bratrax-text-body"
                    >
                      <input
                        type="checkbox"
                        bind:checked={mediaScopeForm.is_active}
                      />
                      Rule is active
                    </label>
                  </div>

                  <div class="mt-6 flex justify-end gap-2">
                    <button
                      class="btn-bratrax btn-neutral btn-compact"
                      on:click={() => {
                        showMediaScopeModal = false;
                      }}>Cancel</button
                    >
                    <button
                      class="btn-bratrax btn-primary btn-compact"
                      disabled={saving ||
                        !mediaScopeForm.match_value ||
                        !mediaScopeForm.name}
                      on:click={handleSaveMediaScopeRule}>Save rule</button
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
    top: 0;
    left: 0;
    right: 0;
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
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .cost-subtab:hover:not(.active) {
    color: var(--color-text);
    border-bottom: 2px solid var(--color-acid);
    padding-bottom: 4px;
  }
  .cost-subtab.active {
    background: var(--color-acid);
    color: #0a0a0a;
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
  .media-scope-guidance {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 24px;
    border: 0.5px solid var(--color-border);
    background: rgba(0, 0, 0, 0.015);
    padding: 18px;
  }
  .media-scope-rule {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 18px;
    border: 0.5px solid var(--color-border);
    padding: 16px;
  }
  .media-scope-badge {
    font-family: "Space Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 1px;
    text-transform: uppercase;
    padding: 3px 6px;
    background: var(--color-acid);
    color: #0a0a0a;
  }
  .media-scope-badge.exclude {
    background: rgba(214, 80, 63, 0.14);
    color: var(--color-tomato);
  }
  .media-scope-badge.muted {
    background: transparent;
    border: 0.5px solid var(--color-border);
    color: var(--color-text-secondary);
  }
  @media (max-width: 720px) {
    .media-scope-guidance,
    .media-scope-rule {
      flex-direction: column;
    }
  }
</style>
