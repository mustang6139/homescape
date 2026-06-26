import type { Stats } from "./types";
import { fetchStats } from "./api";

// Shared host-stats store for composed widgets bound to the `host` source. Polled while the
// app is open (started once in App).

export const stats = $state<{ value: Stats | null }>({ value: null });

let started = false;

export function startStats(intervalMs = 3000) {
  if (started) return () => {};
  started = true;
  const tick = async () => {
    try {
      stats.value = await fetchStats();
    } catch {
      /* keep last known */
    }
  };
  tick();
  const t = setInterval(tick, intervalMs);
  return () => {
    clearInterval(t);
    started = false;
  };
}
