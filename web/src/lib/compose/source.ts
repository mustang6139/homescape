import { resources, serviceStatus } from "../resources.svelte";
import { activeIntegrations } from "../integrations.svelte";
import { stats } from "../stats.svelte";

// A composed widget's data source. Curated kinds with fixed semantics — the power comes
// from these built-in sources + primitives, not from user logic.
export interface ComposedSource {
  kind: "resource" | "host" | "services" | "static";
  resource?: string; // "<integration-handle>|<resource.kind>"
  data?: unknown;
}

// resolveSource returns the data object a view tree binds against. Called inside a reactive
// context (the renderer's $derived), so reads of the stores track updates → live.
export function resolveSource(src: ComposedSource | undefined): unknown {
  if (!src) return undefined;
  switch (src.kind) {
    case "resource": {
      const keyStr = src.resource ?? "";
      return resources.byKey[keyStr];
    }
    case "host":
      return stats.value;
    case "services": {
      const items = activeIntegrations().map((i) => {
        const st = serviceStatus(i.id);
        return {
          id: i.id,
          name: i.name,
          group: i.group,
          up: !!st?.up,
          latencyMs: st?.latencyMs ?? null,
        };
      });
      const upCount = items.filter((x) => x.up).length;
      return { items, upCount, downCount: items.length - upCount, total: items.length };
    }
    case "static":
      return src.data;
    default:
      return undefined;
  }
}
