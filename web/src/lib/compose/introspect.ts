import type { ComposedSource } from "./source";

// Field introspection for the composer: discover bindable paths from a live data sample,
// falling back to a per-kind known-field registry when there's no live data yet.

export interface Field {
  path: string; // dot-path; arrays use "[]" (e.g. "items[].name")
  type: string; // boolean | number | string | array | object | null
}

function typeOf(v: unknown): string {
  if (v === null) return "null";
  if (Array.isArray(v)) return "array";
  return typeof v;
}

// flatten walks a sample value into dot-path fields, descending one level into arrays
// (using "[]"), so collection item fields are discoverable for list bindings.
export function flatten(data: unknown, prefix = ""): Field[] {
  if (data == null) return [];

  if (Array.isArray(data)) {
    const p = `${prefix}[]`;
    if (data.length === 0) return [{ path: p, type: "array" }];
    const el = data[0];
    if (el !== null && typeof el === "object") return flatten(el, p);
    return [{ path: p, type: typeOf(el) }];
  }

  if (typeof data === "object") {
    const out: Field[] = [];
    for (const [k, v] of Object.entries(data as Record<string, unknown>)) {
      const p = prefix ? `${prefix}.${k}` : k;
      if (v !== null && typeof v === "object") out.push(...flatten(v, p));
      else out.push({ path: p, type: typeOf(v) });
    }
    return out;
  }

  return [{ path: prefix || "(value)", type: typeOf(data) }];
}

// Known fields per source/resource kind — the fallback when no live sample exists.
const KNOWN: Record<string, Field[]> = {
  "service.status": [
    { path: "up", type: "boolean" },
    { path: "latencyMs", type: "number" },
    { path: "version", type: "string" },
    { path: "message", type: "string" },
  ],
  host: [
    { path: "cpuPercent", type: "number" },
    { path: "memUsed", type: "number" },
    { path: "memTotal", type: "number" },
    { path: "diskUsed", type: "number" },
    { path: "diskTotal", type: "number" },
    { path: "uptimeSecs", type: "number" },
    { path: "collectedAt", type: "string" },
  ],
  services: [
    { path: "upCount", type: "number" },
    { path: "downCount", type: "number" },
    { path: "total", type: "number" },
    { path: "items[].id", type: "string" },
    { path: "items[].name", type: "string" },
    { path: "items[].group", type: "string" },
    { path: "items[].up", type: "boolean" },
    { path: "items[].latencyMs", type: "number" },
  ],
};

// knownFields returns the registered fields for a source (resource kind, host, or services).
export function knownFields(src: ComposedSource): Field[] {
  switch (src.kind) {
    case "resource": {
      const kind = (src.resource ?? "").split("|")[1] ?? "";
      return KNOWN[kind] ?? [];
    }
    case "host":
      return KNOWN.host;
    case "services":
      return KNOWN.services;
    default:
      return [];
  }
}

// fieldsFor prefers a live sample, falling back to the known registry.
export function fieldsFor(src: ComposedSource, live: unknown): Field[] {
  if (live != null && typeof live === "object") {
    const flat = flatten(live);
    if (flat.length) return flat;
  }
  return knownFields(src);
}
