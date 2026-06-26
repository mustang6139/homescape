import { describe, it, expect, beforeEach } from "vitest";
import { resolveSource } from "./source";
import { resources } from "../resources.svelte";
import { registry } from "../integrations.svelte";
import { stats } from "../stats.svelte";
import type { Integration } from "../types";

function integ(id: string, status: Integration["status"]): Integration {
  return {
    id,
    type: "http-health",
    name: id.toUpperCase(),
    baseUrl: "",
    group: "g",
    icon: "",
    source: "manual",
    status,
    hasSecret: false,
  };
}

beforeEach(() => {
  resources.byKey = {};
  registry.list = [];
  registry.pending = [];
  stats.value = null;
});

describe("resolveSource", () => {
  it("resource → the resource data by key", () => {
    resources.byKey = { "x|service.status": { up: true, latencyMs: 9 } };
    expect(resolveSource({ kind: "resource", resource: "x|service.status" })).toEqual({
      up: true,
      latencyMs: 9,
    });
  });

  it("host → the stats store value", () => {
    stats.value = { cpuPercent: 10 } as never;
    expect(resolveSource({ kind: "host" })).toEqual({ cpuPercent: 10 });
  });

  it("static → embedded data", () => {
    expect(resolveSource({ kind: "static", data: { hi: 1 } })).toEqual({ hi: 1 });
  });

  it("services → join of active integrations with live status + aggregates", () => {
    registry.list = [integ("a", "active"), integ("b", "active"), integ("c", "hidden")];
    resources.byKey = {
      "a|service.status": { up: true, latencyMs: 5 },
      "b|service.status": { up: false },
    };
    const out = resolveSource({ kind: "services" }) as {
      items: { id: string; up: boolean; latencyMs: number | null }[];
      upCount: number;
      downCount: number;
      total: number;
    };
    expect(out.total).toBe(2); // hidden excluded
    expect(out.upCount).toBe(1);
    expect(out.downCount).toBe(1);
    const a = out.items.find((i) => i.id === "a")!;
    expect(a.up).toBe(true);
    expect(a.latencyMs).toBe(5);
    const b = out.items.find((i) => i.id === "b")!;
    expect(b.up).toBe(false);
    expect(b.latencyMs).toBeNull();
  });
});
