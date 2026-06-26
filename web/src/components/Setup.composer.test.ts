// @vitest-environment jsdom
import { render, fireEvent } from "@testing-library/svelte";
import { describe, it, expect, vi } from "vitest";
import Setup from "./Setup.svelte";
import { store } from "../lib/store.svelte";

const spec = {
  version: 1,
  meta: { name: "x", theme: "midnight-glass", accent: "#E6A95C" },
  layout: { columns: 3 },
  widgets: [
    { id: "w-stats", type: "system-stats", column: 0, title: "System", options: {} },
    {
      id: "sh",
      type: "composed",
      column: 1,
      title: "Health",
      options: {
        source: { kind: "services" },
        view: {
          el: "stack",
          children: [
            { el: "text", bind: "upCount", fmt: "round" },
            {
              el: "list",
              of: "items",
              item: {
                el: "row",
                children: [
                  { el: "icon", bind: "up", map: { true: "check", false: "x" } },
                  { el: "text", bind: "name" },
                ],
              },
            },
          ],
        },
      },
    },
  ],
};

describe("Setup composer (repro)", () => {
  it("selecting a composed widget does not hang or throw", async () => {
    const errs: unknown[] = [];
    const orig = console.error;
    console.error = (...a: unknown[]) => {
      errs.push(a);
      orig(...a);
    };

    store.spec = spec as never;
    store.loading = false;

    const { getByRole } = render(Setup, { props: { onClose: () => {} } });
    await fireEvent.click(getByRole("button", { name: "Widgets" }));
    await fireEvent.click(getByRole("button", { name: /Health/ }));

    console.error = orig;
    expect(errs, "console errors during composed select: " + JSON.stringify(errs)).toEqual([]);
  });
});
