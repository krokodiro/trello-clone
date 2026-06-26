import { describe, expect, it } from "vitest";
import { getRedirectPath, withRedirect } from "./redirect";

describe("getRedirectPath", () => {
  it("returns safe internal paths", () => {
    expect(getRedirectPath("redirect=%2Fboards")).toBe("/boards");
  });

  it("rejects external and protocol-relative URLs", () => {
    expect(getRedirectPath("redirect=https://evil.com")).toBeNull();
    expect(getRedirectPath("redirect=//evil.com")).toBeNull();
    expect(getRedirectPath("")).toBeNull();
  });
});

describe("withRedirect", () => {
  it("appends redirect query param", () => {
    expect(withRedirect("/login", "/boards")).toBe(
      "/login?redirect=%2Fboards"
    );
  });

  it("uses ampersand when path already has query", () => {
    expect(withRedirect("/login?foo=1", "/x")).toBe(
      "/login?foo=1&redirect=%2Fx"
    );
  });

  it("returns path unchanged when redirect is null", () => {
    expect(withRedirect("/register", null)).toBe("/register");
  });
});
