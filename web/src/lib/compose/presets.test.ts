import { describe, it, expect } from "vitest";
import { PRESETS } from "./presets";

describe("presets", () => {
  it("each preset has a source and a view", () => {
    for (const p of PRESETS) {
      expect(p.options.source, p.id).toBeTruthy();
      expect(p.options.view, p.id).toBeTruthy();
    }
  });

  it("the service-health dogfood uses the services source + a list (the proof)", () => {
    const sh = PRESETS.find((p) => p.id === "service-health")!;
    expect((sh.options.source as { kind: string }).kind).toBe("services");
    const view = sh.options.view as { children: { el: string }[] };
    expect(view.children.some((c) => c.el === "list")).toBe(true);
  });
});
