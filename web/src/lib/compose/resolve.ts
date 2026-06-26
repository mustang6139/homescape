// Binding resolution for composed widgets: dot-path field access and static map lookups.
// Sandboxed — paths only, never expressions.

// getPath walks a dot-path (e.g. "a.b.0.c") into data, returning undefined if any step is
// missing. Numeric segments index into arrays.
export function getPath(data: unknown, path: string): unknown {
  if (!path) return data;
  let cur: unknown = data;
  for (const key of path.split(".")) {
    if (cur == null) return undefined;
    if (Array.isArray(cur)) {
      const i = Number(key);
      cur = Number.isInteger(i) ? cur[i] : undefined;
    } else if (typeof cur === "object") {
      cur = (cur as Record<string, unknown>)[key];
    } else {
      return undefined;
    }
  }
  return cur;
}

// lookupMap returns the mapped string for a value (stringified key), or undefined.
export function lookupMap(
  value: unknown,
  map: Record<string, string> | undefined,
): string | undefined {
  if (!map) return undefined;
  return map[String(value)];
}

// toNumber coerces a bound value to a finite number, or undefined (used by the bar primitive).
export function toNumber(value: unknown): number | undefined {
  if (value == null || value === "") return undefined;
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? n : undefined;
}
