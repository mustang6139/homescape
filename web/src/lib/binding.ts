import type { Widget, Integration } from "./types";

// resolveServices implements the service-health binding convention (pure, so it is unit
// testable): an explicit non-empty options.services list wins; otherwise all active
// integrations are shown (the beginner-bridge default).
export function resolveServices(widget: Widget, list: Integration[]): Integration[] {
  const ids = widget.options?.services as string[] | undefined;
  if (ids && ids.length) {
    return ids
      .map((id) => list.find((i) => i.id === id))
      .filter((i): i is Integration => !!i);
  }
  return list.filter((i) => i.status === "active");
}
