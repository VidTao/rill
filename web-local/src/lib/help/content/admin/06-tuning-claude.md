---
title: Tuning Claude
audience: admin
order: 4
group: building
status: ready
---

# Tuning Claude

Claude (the AI analyst) reads a system prompt to know what your data means and how your business works. You can edit this prompt to make Claude smarter about your specific business.

The file lives at `/knowledge/claude_system_prompt.txt`.

## What's in there by default

The auto-generated section lists every metric, dimension, and dashboard available, plus how they relate. This section is regenerated automatically whenever you change the data model — **don't edit it directly**, your edits will get overwritten.

Below the auto-generated section is space for your own additions.

## What to add

Anything that helps Claude answer your specific questions correctly.

**Business rules**

> Subscription renewals (Shopify-native subscriptions) renew automatically and should be excluded from new-customer metrics.

**Naming conventions**

> Our "core" channels are paid social and paid search. Everything else (organic, email, direct) is "long tail."

**Known data quirks**

> Black Friday week 2024 had a tracking outage on Nov 27 — figures for that day undercount by ~15%.

**Tone preferences**

> Be concise. When asked "why," explain in one sentence and back it up with numbers — don't lecture.

**Definitions specific to you**

> By "profit" we mean revenue minus ad spend minus 35% COGS. Don't include shipping costs — those are tracked separately.

## What not to add

- Passwords, API keys, customer email lists, or any PII
- Raw SQL — Claude has tools for that, don't pre-write queries
- Long historical narratives. Keep additions under ~2000 words; focused context works better than a wall of text.

## Testing your changes

After editing the prompt:

1. Save the file.
2. Start a new Claude conversation — the existing one keeps the old prompt.
3. Ask a question that should be affected by your edit.
4. Compare the answer to what you expected.

If the answer doesn't reflect your edit, tighten the wording. Claude follows specific instructions better than vague ones — *"exclude subscription orders from new-customer counts"* works better than *"be careful about subscriptions."*
