// Starter composed-widget presets. These prove the primitive model is enough to rebuild
// real widgets (service-health, system-stats) purely from data + primitives — the dogfood.

export interface Preset {
  id: string;
  name: string;
  description: string;
  title: string;
  options: { source: unknown; view: unknown };
}

export const PRESETS: Preset[] = [
  {
    id: "service-health",
    name: "Service health",
    description: "All active services with live up/down + latency (rebuilt from primitives).",
    title: "Service health",
    options: {
      source: { kind: "services" },
      view: {
        el: "stack",
        children: [
          {
            el: "row",
            children: [
              { el: "text", bind: "upCount", fmt: "round", style: { color: "up", weight: "600" } },
              { el: "text", text: "up" },
              { el: "text", bind: "downCount", fmt: "round", style: { color: "down", weight: "600" } },
              { el: "text", text: "down" },
            ],
          },
          {
            el: "list",
            of: "items",
            item: {
              el: "row",
              children: [
                {
                  el: "icon",
                  bind: "up",
                  map: { true: "check", false: "x" },
                  colorMap: { true: "up", false: "down" },
                },
                { el: "text", bind: "name" },
                { el: "text", bind: "latencyMs", fmt: "ms", style: { color: "muted" } },
              ],
            },
          },
        ],
      },
    },
  },
  {
    id: "system-stats",
    name: "System stats",
    description: "Host CPU and memory as labelled progress bars.",
    title: "System",
    options: {
      source: { kind: "host" },
      view: {
        el: "stack",
        children: [
          {
            el: "row",
            children: [
              { el: "text", text: "CPU" },
              { el: "text", bind: "cpuPercent", fmt: "percent", style: { color: "accent" } },
            ],
          },
          { el: "bar", value: "cpuPercent", max: 100 },
          {
            el: "row",
            children: [
              { el: "text", text: "Memory" },
              { el: "text", bind: "memUsed", fmt: "bytes", style: { color: "accent" } },
              { el: "text", text: "/" },
              { el: "text", bind: "memTotal", fmt: "bytes", style: { color: "muted" } },
            ],
          },
          { el: "bar", value: "memUsed", max: "memTotal" },
        ],
      },
    },
  },
  {
    id: "service-card",
    name: "Single service",
    description: "One service's status (pick the service after adding).",
    title: "Service",
    options: {
      source: { kind: "resource", resource: "" },
      view: {
        el: "row",
        children: [
          {
            el: "icon",
            bind: "up",
            map: { true: "check", false: "x" },
            colorMap: { true: "up", false: "down" },
          },
          { el: "text", bind: "version", style: { color: "muted" } },
          { el: "text", bind: "latencyMs", fmt: "ms", style: { color: "muted" } },
        ],
      },
    },
  },
  {
    id: "blank",
    name: "Blank",
    description: "An empty stack to build from scratch.",
    title: "New widget",
    options: {
      source: { kind: "host" },
      view: { el: "stack", children: [{ el: "text", text: "Hello" }] },
    },
  },
];
