import { describe, expect, it } from "vitest";
import {
  consentUrl,
  isUserOnboardingDone,
  resumeIdFromRedirect,
} from "./oauth-connect";

describe("resumeIdFromRedirect", () => {
  it("reads the resume id from a consent-page redirect", () => {
    expect(resumeIdFromRedirect("/oauth/consent?resume=abc_123")).toBe("abc_123");
    expect(resumeIdFromRedirect(consentUrl("a b"))).toBe("a b");
  });

  it("ignores every other destination", () => {
    expect(resumeIdFromRedirect(null)).toBeNull();
    expect(resumeIdFromRedirect("/developer")).toBeNull();
    expect(resumeIdFromRedirect("/oauth/consentx?resume=1")).toBeNull();
    expect(resumeIdFromRedirect("https://evil.example/oauth/consent?resume=1")).toBeNull();
  });
});

describe("isUserOnboardingDone", () => {
  it("is true once activation has started", () => {
    for (const step of ["activating", "compiling", "deploying", "extracting", "ready", "error"]) {
      expect(isUserOnboardingDone(step)).toBe(true);
    }
  });

  it("is false while the merchant still has steps to do", () => {
    for (const step of ["payment_pending", "created", "embed_pending", "platforms_connected", "business_profile"]) {
      expect(isUserOnboardingDone(step)).toBe(false);
    }
    expect(isUserOnboardingDone(null)).toBe(false);
  });
});
