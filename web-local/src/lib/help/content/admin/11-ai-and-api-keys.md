---
title: AI chat & Anthropic API key
audience: admin
order: 6
group: workspace
status: ready
---

# AI chat & Anthropic API key

## Where do I add my Anthropic API key in Bratrax?

Go to [bratrax.com/settings](https://bratrax.com/settings) and open the [**AI**](https://bratrax.com/settings/ai) tab. Paste your key there and save — the in-app chat will work immediately.

To use Bratrax data from Claude itself instead, connect Bratrax from Claude's connector settings (see [how to connect](https://bratrax.com/integrations/claude)); connected assistants are listed one tab over in the **MCP** tab on the same Settings page.

## How do I create an Anthropic API key from scratch?

1. Go to **console.anthropic.com** and sign up (or log in).
2. Add a payment method under **Billing**. Anthropic charges per token used — no monthly minimum.
3. Go to **API Keys**, click **Create Key**, give it a name like "Bratrax," and copy the key shown.
4. Paste it into Bratrax at [bratrax.com/settings](https://bratrax.com/settings) under the **AI** tab.

Your Anthropic usage costs go directly to Anthropic, not to us. We don't mark up or take a cut.

## Which Claude model does the in-app chat use?

By default, **Claude Sonnet 5**. Admins can change it under the [**AI**](https://bratrax.com/settings/ai) tab with the **Chat model** dropdown:

- **Claude Sonnet 5** (default): strong analysis at a moderate cost.
- **Claude Opus 5**: the most capable, for complex analysis. Roughly 2.5× the cost of Sonnet 5.
- **Claude Haiku 4.5**: the fastest and cheapest, for simple lookups. Roughly half the cost of Sonnet 5.

The change applies to the whole workspace. When you pick a model, Bratrax checks that your Anthropic key can use it before saving. Usage is billed to your Anthropic key at Anthropic's prices.
