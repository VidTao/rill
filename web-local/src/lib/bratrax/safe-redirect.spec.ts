import { describe, expect, it } from "vitest";
import { safeRedirect } from "./safe-redirect";

describe("safeRedirect", () => {
  it("keeps same-site paths, query and hash included", () => {
    expect(safeRedirect("/canvas/campaign_deep_dive")).toBe("/canvas/campaign_deep_dive");
    expect(safeRedirect("/oauth/consent?resume=abc_123")).toBe("/oauth/consent?resume=abc_123");
    expect(safeRedirect("/settings#team")).toBe("/settings#team");
    expect(safeRedirect("/")).toBe("/");
  });

  it("falls back for anything that could leave the site", () => {
    for (const bad of [
      "https://evil.example/",
      "//evil.example/path",
      "/\\evil.example",
      "\\\\evil.example",
      "javascript:alert(1)",
      "evil.example",
      "/\t/evil.example",
      "/\n/evil.example",
      " /canvas",
      "",
      null,
      undefined,
    ]) {
      expect(safeRedirect(bad), String(bad)).toBe("/developer");
    }
  });

  it("uses the caller's fallback", () => {
    expect(safeRedirect("https://evil.example", "/")).toBe("/");
  });
});
