import { afterEach, describe, expect, it } from "vitest";
import { clientCurrency, formatMoney } from "./client-currency";

const fmt = (n: number, currency: string) =>
  n.toLocaleString(undefined, {
    style: "currency",
    currency,
    maximumFractionDigits: 2,
  });

describe("formatMoney", () => {
  afterEach(() => clientCurrency.set("USD"));

  it("formats in the active store's currency", () => {
    clientCurrency.set("CZK");
    expect(formatMoney(365, { maximumFractionDigits: 2 })).toBe(fmt(365, "CZK"));
  });

  it("lets an explicit currency win, e.g. an order's own", () => {
    clientCurrency.set("EUR");
    expect(formatMoney(5700, { maximumFractionDigits: 2 }, "HUF")).toBe(
      fmt(5700, "HUF"),
    );
  });

  it("falls back to USD instead of throwing on an unknown code", () => {
    clientCurrency.set("not-a-currency");
    expect(formatMoney(10, { maximumFractionDigits: 2 })).toBe(fmt(10, "USD"));
  });
});
