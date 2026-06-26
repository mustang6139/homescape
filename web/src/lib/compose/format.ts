// Format presets for composed-widget bindings. Pure, sandboxed — a fixed set of functions,
// never user code. Each gracefully returns "" for missing/invalid input.

export type Fmt = "ms" | "percent" | "bytes" | "duration" | "round" | "relTime" | "text";

// formatValue renders a bound value using a preset. `now` is injectable for testing relTime.
export function formatValue(value: unknown, fmt?: Fmt, now: number = Date.now()): string {
  switch (fmt) {
    case "ms":
      return num(value, (n) => `${Math.round(n)}ms`);
    case "percent":
      return num(value, (n) => `${Math.round(n)}%`);
    case "round":
      return num(value, (n) => `${Math.round(n)}`);
    case "bytes":
      return num(value, bytes);
    case "duration":
      return num(value, duration);
    case "relTime":
      return relTime(value, now);
    case "text":
    default:
      return value == null ? "" : String(value);
  }
}

// num coerces to a finite number and applies fn, else "".
function num(value: unknown, fn: (n: number) => string): string {
  if (value == null || value === "") return "";
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? fn(n) : "";
}

function bytes(n: number): string {
  if (n < 0) return "";
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const s = i === 0 || v >= 100 ? Math.round(v).toString() : v.toFixed(1);
  return `${s} ${units[i]}`;
}

function duration(secs: number): string {
  if (secs < 0) return "";
  const s = Math.floor(secs);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}

// relTime renders an ISO string or epoch (seconds or ms) as "Xs/m/h/d ago".
export function relTime(value: unknown, now: number): string {
  let ms: number;
  if (typeof value === "number") {
    ms = value < 1e12 ? value * 1000 : value; // heuristic: seconds vs milliseconds
  } else {
    ms = Date.parse(String(value));
  }
  if (!Number.isFinite(ms)) return "";

  const sec = Math.round((now - ms) / 1000);
  if (sec < 0) return "just now";
  if (sec < 60) return `${sec}s ago`;
  const min = Math.round(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.round(min / 60);
  if (hr < 24) return `${hr}h ago`;
  return `${Math.round(hr / 24)}d ago`;
}
