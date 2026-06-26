import { describe, it, expect } from "vitest";
import { getPath, lookupMap, toNumber } from "./resolve";

describe("getPath", () => {
  const data = {
    up: true,
    meta: { name: "Sonarr", nested: { deep: 7 } },
    items: [{ name: "a" }, { name: "b" }],
  };

  it("reads top-level and nested fields", () => {
    expect(getPath(data, "up")).toBe(true);
    expect(getPath(data, "meta.name")).toBe("Sonarr");
    expect(getPath(data, "meta.nested.deep")).toBe(7);
  });

  it("indexes into arrays", () => {
    expect(getPath(data, "items.1.name")).toBe("b");
  });

  it("returns undefined for missing paths", () => {
    expect(getPath(data, "nope")).toBeUndefined();
    expect(getPath(data, "meta.missing.x")).toBeUndefined();
    expect(getPath(data, "up.notObject")).toBeUndefined();
  });

  it("empty path returns the data itself", () => {
    expect(getPath(data, "")).toBe(data);
  });
});

describe("lookupMap", () => {
  const map = { true: "check", false: "x" };
  it("maps stringified values", () => {
    expect(lookupMap(true, map)).toBe("check");
    expect(lookupMap(false, map)).toBe("x");
  });
  it("undefined when no map or no key", () => {
    expect(lookupMap(true, undefined)).toBeUndefined();
    expect(lookupMap("other", map)).toBeUndefined();
  });
});

describe("toNumber", () => {
  it("coerces finite numbers, else undefined", () => {
    expect(toNumber(5)).toBe(5);
    expect(toNumber("42")).toBe(42);
    expect(toNumber("nope")).toBeUndefined();
    expect(toNumber(null)).toBeUndefined();
  });
});
