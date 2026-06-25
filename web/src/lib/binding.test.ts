import { describe, it, expect } from "vitest";
import { resolveServices } from "./binding";
import type { Integration, Widget } from "./types";

function it_(id: string, status: Integration["status"]): Integration {
  return {
    id,
    type: "http-health",
    name: id,
    baseUrl: "",
    group: "",
    icon: "",
    source: "manual",
    status,
    hasSecret: false,
  };
}

const list: Integration[] = [
  it_("a", "active"),
  it_("b", "active"),
  it_("c", "hidden"),
  it_("d", "pending"),
];

function widget(options: Record<string, unknown>): Widget {
  return { id: "w", type: "service-health", column: 0, options };
}

describe("resolveServices", () => {
  it("returns all active integrations by default", () => {
    const got = resolveServices(widget({ source: "all" }), list).map((i) => i.id);
    expect(got).toEqual(["a", "b"]);
  });

  it("treats an empty services list as the default (all active)", () => {
    const got = resolveServices(widget({ services: [] }), list).map((i) => i.id);
    expect(got).toEqual(["a", "b"]);
  });

  it("uses an explicit services list regardless of status", () => {
    const got = resolveServices(widget({ services: ["c", "a"] }), list).map((i) => i.id);
    expect(got).toEqual(["c", "a"]);
  });

  it("drops handles that don't resolve", () => {
    const got = resolveServices(widget({ services: ["a", "missing"] }), list).map((i) => i.id);
    expect(got).toEqual(["a"]);
  });
});
