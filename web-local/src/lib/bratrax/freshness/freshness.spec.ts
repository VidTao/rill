import { describe, expect, it } from "vitest";
import { formatEta, freshnessView, type DataFreshness } from "./freshness";

const NOW = Date.parse("2026-09-28T14:20:00Z");

function freshness(overrides: Partial<DataFreshness>): DataFreshness {
  return {
    status: "current",
    data_as_of: "2026-09-28T13:57:00Z",
    last_extract_at: "2026-09-28T13:57:00Z",
    last_processed_at: "2026-09-28T14:05:00Z",
    next_update_at: "2026-09-28T14:37:00Z",
    ...overrides,
  };
}

describe("formatEta", () => {
  it("counts whole minutes up", () => {
    expect(formatEta("2026-09-28T14:37:00Z", NOW)).toBe("in 17 min");
    expect(formatEta("2026-09-28T14:20:30Z", NOW)).toBe("in 1 min");
  });

  it("switches to hours at 60 minutes", () => {
    expect(formatEta("2026-09-28T15:20:00Z", NOW)).toBe("in 1 hr");
  });

  it("says any minute once the tick has passed but no refetch has landed", () => {
    expect(formatEta("2026-09-28T14:20:00Z", NOW)).toBe("any minute");
    expect(formatEta("2026-09-28T14:10:00Z", NOW)).toBe("any minute");
  });

  it("returns empty for garbage", () => {
    expect(formatEta("not a date", NOW)).toBe("");
  });
});

describe("freshnessView", () => {
  it("current: shows age and the next update", () => {
    const view = freshnessView(freshness({}), NOW);
    expect(view).not.toBeNull();
    expect(view!.tone).toBe("current");
    expect(view!.updated).toBe("Updated 23 min ago");
    expect(view!.next).toBe("Next update in 17 min");
    expect(view!.tooltip).toHaveLength(2);
  });

  it("current without an ETA (client config has no offset)", () => {
    const view = freshnessView(freshness({ next_update_at: null }), NOW);
    expect(view!.next).toBeNull();
    expect(view!.tooltip).toHaveLength(1);
  });

  it("delayed: the stalled-client case never promises an update", () => {
    const view = freshnessView(
      freshness({
        status: "delayed",
        data_as_of: "2026-09-11T08:19:00Z",
        // Even if a stale payload carried one, delayed must not show it.
        next_update_at: "2026-09-28T14:37:00Z",
      }),
      NOW,
    );
    expect(view!.tone).toBe("delayed");
    expect(view!.updated).toBe("Updated 17 days ago");
    expect(view!.next).toBeNull();
    expect(view!.tooltip[0]).toMatch(/^No new data since /);
  });

  it("lower-cases 'Yesterday' mid-sentence", () => {
    const view = freshnessView(
      freshness({ status: "delayed", data_as_of: "2026-09-27T10:00:00Z" }),
      NOW,
    );
    expect(view!.updated).toBe("Updated yesterday");
  });

  it("renders nothing for unknown, null, or a missing timestamp", () => {
    expect(freshnessView(null, NOW)).toBeNull();
    expect(
      freshnessView(freshness({ status: "unknown", data_as_of: null }), NOW),
    ).toBeNull();
    expect(freshnessView(freshness({ data_as_of: null }), NOW)).toBeNull();
  });
});
