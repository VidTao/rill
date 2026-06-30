import { describe, expect, it } from "vitest";
import { ownerMatchesNode, ownershipRosterFor } from "./ownership";

describe("metric tree ownership", () => {
  it("groups real users by ownerUserId instead of display text", () => {
    const roster = ownershipRosterFor([
      { id: "a", label: "Conversion", type: "lever", ownerUserId: 7, ownerLabel: "Sam", owner: "Sam" },
      { id: "b", label: "Checkout test", type: "experiment", ownerUserId: 7, ownerLabel: "Samuel", owner: "Samuel", status: "running" },
    ]);

    expect(roster).toHaveLength(1);
    expect(roster[0]).toMatchObject({ ownerKey: "user:7", ownerUserId: 7, leverCount: 1, experimentCount: 1, runningCount: 1 });
  });

  it("keeps custom labels and unassigned nodes distinct", () => {
    const roster = ownershipRosterFor([
      { id: "a", label: "Lifecycle", type: "lever", ownerLabel: "Lifecycle / retention" },
      { id: "b", label: "Pricing", type: "lever" },
    ]);

    expect(roster.map((owner) => owner.ownerKey)).toEqual(["unassigned", "label:lifecycle / retention"]);
    expect(roster[0].unassigned).toBe(true);
  });

  it("matches by stable key or legacy label", () => {
    const node = { id: "a", label: "Conversion", type: "lever", ownerUserId: 7, ownerLabel: "Sam" };

    expect(ownerMatchesNode("user:7", node)).toBe(true);
    expect(ownerMatchesNode("Sam", node)).toBe(true);
    expect(ownerMatchesNode("user:8", node)).toBe(false);
  });
});
