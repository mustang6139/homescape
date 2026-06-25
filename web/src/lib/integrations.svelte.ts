import type { Integration, DiscoverySettings } from "./types";
import { fetchIntegrations, fetchDiscoverySettings, fetchPending } from "./api";
import { on } from "./events";

// Registry store: the integration list, discovery settings, and the pending review queue.
// Refreshed on integrations.changed / discovery.changed SSE events.

export const registry = $state<{
  list: Integration[];
  pending: Integration[];
  discovery: DiscoverySettings;
  loaded: boolean;
}>({
  list: [],
  pending: [],
  discovery: { enabled: false, mode: "review", available: false },
  loaded: false,
});

export async function loadRegistry() {
  try {
    const [list, pending, discovery] = await Promise.all([
      fetchIntegrations(),
      fetchPending(),
      fetchDiscoverySettings(),
    ]);
    registry.list = list;
    registry.pending = pending;
    registry.discovery = discovery;
    registry.loaded = true;
  } catch {
    /* keep previous state */
  }
}

export function listenRegistry() {
  const offA = on("integrations.changed", () => void loadRegistry());
  const offB = on("discovery.changed", () => void loadRegistry());
  return () => {
    offA();
    offB();
  };
}

// activeIntegrations returns integrations shown on the dashboard (status active).
export function activeIntegrations(): Integration[] {
  return registry.list.filter((i) => i.status === "active");
}
