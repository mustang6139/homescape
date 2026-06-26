import { describe, it, expect } from "vitest";
import { flatten, knownFields, fieldsFor } from "./introspect";

describe("flatten", () => {
  it("flattens nested objects to dot-paths with types", () => {
    const f = flatten({ up: true, meta: { name: "x", n: 3 } });
    expect(f).toContainEqual({ path: "up", type: "boolean" });
    expect(f).toContainEqual({ path: "meta.name", type: "string" });
    expect(f).toContainEqual({ path: "meta.n", type: "number" });
  });

  it("descends one level into arrays of objects with []", () => {
    const f = flatten({ items: [{ name: "a", up: true }], upCount: 1 });
    expect(f).toContainEqual({ path: "items[].name", type: "string" });
    expect(f).toContainEqual({ path: "items[].up", type: "boolean" });
    expect(f).toContainEqual({ path: "upCount", type: "number" });
  });

  it("handles empty arrays and scalars", () => {
    expect(flatten({ items: [] })).toContainEqual({ path: "items[]", type: "array" });
    expect(flatten(null)).toEqual([]);
  });
});

describe("knownFields", () => {
  it("resolves resource kind from the handle string", () => {
    const f = knownFields({ kind: "resource", resource: "sonarr|service.status" });
    expect(f.map((x) => x.path)).toContain("latencyMs");
  });
  it("host and services have registries", () => {
    expect(knownFields({ kind: "host" }).map((x) => x.path)).toContain("cpuPercent");
    expect(knownFields({ kind: "services" }).map((x) => x.path)).toContain("items[].name");
  });
});

describe("fieldsFor", () => {
  it("prefers a live sample", () => {
    const f = fieldsFor({ kind: "host" }, { customField: 1 });
    expect(f).toContainEqual({ path: "customField", type: "number" });
  });
  it("falls back to known fields without a sample", () => {
    const f = fieldsFor({ kind: "host" }, undefined);
    expect(f.map((x) => x.path)).toContain("cpuPercent");
  });
});
