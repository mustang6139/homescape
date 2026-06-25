import type { Spec } from "./types";
import { fetchScape, patchScape, putScape } from "./api";
import { applyTheme } from "./themes";
import { on } from "./events";

// The frontend single source of truth, mirroring the backend Scape spec. Every component
// reads from here; mutations go through the API and the result flows back in.

interface ScapeStore {
  spec: Spec | null;
  loading: boolean;
  error: string | null;
}

export const store = $state<ScapeStore>({
  spec: null,
  loading: true,
  error: null,
});

function adopt(spec: Spec) {
  store.spec = spec;
  applyTheme(spec.meta.theme, spec.meta.accent, spec.meta.density ?? "cozy");
}

export async function load() {
  try {
    store.loading = true;
    adopt(await fetchScape());
    store.error = null;
  } catch (e) {
    store.error = String(e);
  } finally {
    store.loading = false;
  }
}

// listenScape registers for scape.updated on the shared SSE channel so changes from any
// tab/source apply live.
export function listenScape() {
  return on("scape.updated", (data) => {
    if (data) adopt(data as Spec);
  });
}

// patch sends a partial update (live appearance/layout tweaks). Optimistically applies the
// theme so the change is instant; the server response reconciles the spec.
export async function patch(partial: Record<string, unknown>) {
  if (store.spec && partial.meta && typeof partial.meta === "object") {
    const m = partial.meta as Partial<Spec["meta"]>;
    applyTheme(
      m.theme ?? store.spec.meta.theme,
      m.accent ?? store.spec.meta.accent,
      m.density ?? store.spec.meta.density ?? "cozy",
    );
  }
  adopt(await patchScape(partial));
}

// save persists a full spec (e.g. after a drag-and-drop layout change).
export async function save(spec: Spec) {
  adopt(await putScape(spec));
}
