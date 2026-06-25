import type { ResourceUpdate, ServiceStatus } from "./types";
import { fetchResources } from "./api";
import { on } from "./events";

// Live data store, separate from the spec store. Keyed by `${integrationId}|${kind}`,
// seeded from /api/resources and kept fresh by resource.updated SSE events.

export const resources = $state<{ byKey: Record<string, unknown> }>({ byKey: {} });

const key = (id: string, kind: string) => `${id}|${kind}`;

export async function loadResources() {
  try {
    const list = await fetchResources();
    const next: Record<string, unknown> = {};
    for (const u of list) next[key(u.integrationId, u.kind)] = u.data;
    resources.byKey = next;
  } catch {
    /* keep whatever we have */
  }
}

export function listenResources() {
  return on("resource.updated", (data) => {
    const u = data as ResourceUpdate;
    if (!u || !u.integrationId) return;
    resources.byKey = { ...resources.byKey, [key(u.integrationId, u.kind)]: u.data };
  });
}

// serviceStatus returns the latest service.status for an integration, if known.
export function serviceStatus(id: string): ServiceStatus | undefined {
  return resources.byKey[key(id, "service.status")] as ServiceStatus | undefined;
}
