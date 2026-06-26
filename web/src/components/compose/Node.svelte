<script lang="ts">
  import Self from "./Node.svelte";
  import { getPath, lookupMap, toNumber } from "../../lib/compose/resolve";
  import { formatValue, type Fmt } from "../../lib/compose/format";
  import { styleToCss, iconGlyph, colorVar } from "../../lib/compose/tokens";

  // A single view-tree node, rendered against the current data scope. Recursive.
  interface ViewNode {
    el: string;
    children?: ViewNode[];
    bind?: string;
    text?: string;
    fmt?: Fmt;
    map?: Record<string, string>;
    colorMap?: Record<string, string>;
    of?: string;
    item?: ViewNode;
    value?: string | number;
    max?: string | number;
    href?: string;
    style?: { color?: string; size?: string; weight?: string };
  }

  let { node, data }: { node: ViewNode; data: unknown } = $props();

  const css = $derived(styleToCss(node.style));

  // colorMap overrides the static colour based on the bound value (sandboxed lookup).
  const leafCss = $derived.by(() => {
    if (node.colorMap && node.bind != null) {
      const tok = lookupMap(getPath(data, node.bind), node.colorMap);
      const c = colorVar(tok);
      if (c) return `${css};color:${c}`;
    }
    return css;
  });

  // Resolve a leaf's display string: literal text, else mapped value, else formatted value.
  function leafText(n: ViewNode, d: unknown): string {
    if (n.text != null) return n.text;
    const raw = n.bind != null ? getPath(d, n.bind) : undefined;
    if (n.map) return lookupMap(raw, n.map) ?? "";
    return formatValue(raw, n.fmt);
  }

  // bar value/max may be a literal number or a field path.
  function numAttr(v: string | number | undefined, d: unknown, fallback: number): number {
    if (v == null) return fallback;
    if (typeof v === "number") return v;
    return toNumber(getPath(d, v)) ?? fallback;
  }

  const barPct = $derived.by(() => {
    if (node.el !== "bar") return 0;
    const val = numAttr(node.value, data, 0);
    const max = numAttr(node.max, data, 100);
    return max > 0 ? Math.max(0, Math.min(100, (val / max) * 100)) : 0;
  });

  const listItems = $derived.by(() => {
    if (node.el !== "list") return [];
    const arr = getPath(data, node.of ?? "");
    return Array.isArray(arr) ? arr : [];
  });
</script>

{#if node.el === "stack" || node.el === "row"}
  <div class="hs-{node.el}" style={css}>
    {#each node.children ?? [] as child}
      <Self node={child} {data} />
    {/each}
  </div>
{:else if node.el === "text"}
  <span style={leafCss}>{leafText(node, data)}</span>
{:else if node.el === "badge"}
  <span class="hs-badge" style={leafCss}>{leafText(node, data)}</span>
{:else if node.el === "icon"}
  <span class="hs-icon" style={leafCss}>{iconGlyph(leafText(node, data))}</span>
{:else if node.el === "bar"}
  <div class="hs-bar"><i style="width:{barPct}%;{css}"></i></div>
{:else if node.el === "link"}
  <a class="hs-link" href={node.href ?? "#"} style={css}>{leafText(node, data)}</a>
{:else if node.el === "divider"}
  <hr class="hs-divider" />
{:else if node.el === "image"}
  <img class="hs-image" src={String(leafText(node, data))} alt="" />
{:else if node.el === "list"}
  <div class="hs-stack">
    {#each listItems as item}
      {#if node.item}<Self node={node.item} data={item} />{/if}
    {/each}
  </div>
{/if}

<style>
  .hs-stack {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
  }
  .hs-row {
    display: flex;
    align-items: center;
    gap: 9px;
    min-width: 0;
  }
  .hs-badge {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--muted);
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 6px;
    padding: 2px 6px;
  }
  .hs-icon {
    flex-shrink: 0;
  }
  .hs-bar {
    flex: 1;
    height: 7px;
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.08);
    overflow: hidden;
  }
  .hs-bar i {
    display: block;
    height: 100%;
    background: var(--accent);
    border-radius: 5px;
  }
  .hs-link {
    color: var(--accent);
    text-decoration: none;
  }
  .hs-divider {
    border: none;
    border-top: 1px solid var(--card-border);
    width: 100%;
  }
  .hs-image {
    max-width: 100%;
    border-radius: 8px;
  }
</style>
