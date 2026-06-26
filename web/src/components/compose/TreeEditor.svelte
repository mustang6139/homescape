<script lang="ts">
  import type { Field } from "../../lib/compose/introspect";

  interface ViewNode {
    el: string;
    children?: ViewNode[];
    item?: ViewNode;
    bind?: string;
    text?: string;
    fmt?: string;
    of?: string;
    value?: string | number;
    max?: string | number;
    href?: string;
    map?: Record<string, string>;
    colorMap?: Record<string, string>;
    style?: Record<string, string>;
  }

  let { root, fields, onChange }: { root: ViewNode; fields: Field[]; onChange: () => void } =
    $props();

  let selected = $state<ViewNode>(root);
  // Reading `root` tracks the prop reference; this re-runs only when the draft tree is
  // replaced wholesale (e.g. a JSON edit), resetting selection — not on in-place edits.
  $effect(() => {
    selected = root;
  });

  const PALETTE = ["text", "row", "stack", "icon", "badge", "bar", "list", "link", "divider", "image"];
  const COLORS = ["", "accent", "muted", "faint", "ink", "teal", "up", "down"];
  const SIZES = ["", "sm", "md", "lg"];
  const WEIGHTS = ["", "400", "500", "600", "700"];
  const FMTS = ["", "text", "ms", "percent", "bytes", "duration", "round", "relTime"];

  // Binding suggestions: absolute paths + the relative item fields (after "[].").
  const bindSuggestions = $derived([
    ...new Set(
      fields.flatMap((f) => {
        const out = [f.path];
        const i = f.path.indexOf("[].");
        if (i >= 0) out.push(f.path.slice(i + 3));
        return out;
      }),
    ),
  ]);
  const arrayPaths = $derived([
    ...new Set(fields.filter((f) => f.path.includes("[]")).map((f) => f.path.split("[]")[0])),
  ]);

  const isContainer = (n: ViewNode) => n.el === "stack" || n.el === "row";
  const isLeaf = (n: ViewNode) => ["text", "badge", "icon", "link"].includes(n.el);

  function newNode(el: string): ViewNode {
    switch (el) {
      case "stack":
      case "row":
        return { el, children: [] };
      case "list":
        return { el, of: "", item: { el: "text", text: "item" } };
      case "bar":
        return { el, value: 0, max: 100 };
      case "icon":
        return { el, text: "dot" };
      case "badge":
        return { el, text: "badge" };
      case "link":
        return { el, href: "#", text: "link" };
      case "divider":
        return { el };
      case "image":
        return { el, text: "" };
      default:
        return { el: "text", text: "text" };
    }
  }

  function addChild(el: string) {
    const n = newNode(el);
    if (selected.el === "list") {
      selected.item = n;
    } else if (isContainer(selected)) {
      (selected.children ??= []).push(n);
    }
    selected = n;
    onChange();
  }

  function removeNode(node: ViewNode, parent: ViewNode) {
    if (parent.el === "list") delete parent.item;
    else if (parent.children) parent.children = parent.children.filter((c) => c !== node);
    if (selected === node) selected = parent;
    onChange();
  }

  function moveNode(node: ViewNode, parent: ViewNode, dir: number) {
    const a = parent.children;
    if (!a) return;
    const i = a.indexOf(node);
    const j = i + dir;
    if (j < 0 || j >= a.length) return;
    [a[i], a[j]] = [a[j], a[i]];
    onChange();
  }

  function setField(node: ViewNode, key: string, val: string) {
    const rec = node as unknown as Record<string, unknown>;
    if (val === "") delete rec[key];
    else rec[key] = val;
    onChange();
  }

  function setStyle(key: string, val: string) {
    const s = { ...(selected.style ?? {}) };
    if (val === "") delete s[key];
    else s[key] = val;
    if (Object.keys(s).length) selected.style = s;
    else delete selected.style;
    onChange();
  }

  function setMapEntry(mapKey: "map" | "colorMap", k: string, val: string) {
    const m = { ...(selected[mapKey] ?? {}) };
    if (val === "") delete m[k];
    else m[k] = val;
    if (Object.keys(m).length) selected[mapKey] = m;
    else delete selected[mapKey];
    onChange();
  }
</script>

{#snippet tree(node: ViewNode, parent: ViewNode | null)}
  <div class="tnode">
    <div class="trow" class:sel={selected === node}>
      <button class="tlabel" onclick={() => (selected = node)}>{node.el}</button>
      {#if parent}
        {#if parent.children}
          <button class="tbtn" onclick={() => moveNode(node, parent, -1)} aria-label="up">↑</button>
          <button class="tbtn" onclick={() => moveNode(node, parent, 1)} aria-label="down">↓</button>
        {/if}
        <button class="tbtn" onclick={() => removeNode(node, parent)} aria-label="remove">✕</button>
      {/if}
    </div>
    {#if node.el === "stack" || node.el === "row"}
      <div class="tchildren">
        {#each node.children ?? [] as child (child)}{@render tree(child, node)}{/each}
      </div>
    {:else if node.el === "list" && node.item}
      <div class="tchildren">{@render tree(node.item, node)}</div>
    {/if}
  </div>
{/snippet}

<div class="editor">
  <div class="tree">
    {@render tree(root, null)}
  </div>

  {#if selected}
    <div class="prop">
      <div class="prophdr">{selected.el}</div>

      {#if isContainer(selected) || selected.el === "list"}
        <label>
          {selected.el === "list" ? "Set item" : "Add child"}
          <select
            value=""
            onchange={(e) => {
              if (e.currentTarget.value) addChild(e.currentTarget.value);
              e.currentTarget.value = "";
            }}
          >
            <option value="">+ primitive…</option>
            {#each PALETTE as el}<option value={el}>{el}</option>{/each}
          </select>
        </label>
      {/if}

      {#if selected.el === "list"}
        <label>
          of (collection)
          <input
            list="arrpaths"
            value={selected.of ?? ""}
            oninput={(e) => setField(selected, "of", e.currentTarget.value)}
          />
        </label>
      {/if}

      {#if isLeaf(selected)}
        <label>
          Text (literal)
          <input value={selected.text ?? ""} oninput={(e) => setField(selected, "text", e.currentTarget.value)} />
        </label>
        <label>
          Bind (field)
          <input
            list="binds"
            value={selected.bind ?? ""}
            oninput={(e) => setField(selected, "bind", e.currentTarget.value)}
          />
        </label>
        <label>
          Format
          <select value={selected.fmt ?? ""} onchange={(e) => setField(selected, "fmt", e.currentTarget.value)}>
            {#each FMTS as f}<option value={f}>{f || "—"}</option>{/each}
          </select>
        </label>
      {/if}

      {#if selected.el === "bar"}
        <label>
          Value
          <input
            list="binds"
            value={String(selected.value ?? "")}
            oninput={(e) => setField(selected, "value", e.currentTarget.value)}
          />
        </label>
        <label>
          Max
          <input
            value={String(selected.max ?? "")}
            oninput={(e) => setField(selected, "max", e.currentTarget.value)}
          />
        </label>
      {/if}

      {#if selected.el === "link"}
        <label>
          Href
          <input value={selected.href ?? ""} oninput={(e) => setField(selected, "href", e.currentTarget.value)} />
        </label>
      {/if}

      <div class="proprow">
        <label>
          Color
          <select value={selected.style?.color ?? ""} onchange={(e) => setStyle("color", e.currentTarget.value)}>
            {#each COLORS as c}<option value={c}>{c || "—"}</option>{/each}
          </select>
        </label>
        <label>
          Size
          <select value={selected.style?.size ?? ""} onchange={(e) => setStyle("size", e.currentTarget.value)}>
            {#each SIZES as s}<option value={s}>{s || "—"}</option>{/each}
          </select>
        </label>
        <label>
          Weight
          <select value={selected.style?.weight ?? ""} onchange={(e) => setStyle("weight", e.currentTarget.value)}>
            {#each WEIGHTS as w}<option value={w}>{w || "—"}</option>{/each}
          </select>
        </label>
      </div>

      {#if selected.el === "icon" || isLeaf(selected)}
        <div class="maps">
          <span class="maplbl">map (value → text/icon)</span>
          <div class="proprow">
            <input
              placeholder="true →"
              value={selected.map?.true ?? ""}
              oninput={(e) => setMapEntry("map", "true", e.currentTarget.value)}
            />
            <input
              placeholder="false →"
              value={selected.map?.false ?? ""}
              oninput={(e) => setMapEntry("map", "false", e.currentTarget.value)}
            />
          </div>
          <span class="maplbl">colorMap (value → color)</span>
          <div class="proprow">
            <select value={selected.colorMap?.true ?? ""} onchange={(e) => setMapEntry("colorMap", "true", e.currentTarget.value)}>
              {#each COLORS as c}<option value={c}>{c ? `true → ${c}` : "true → —"}</option>{/each}
            </select>
            <select value={selected.colorMap?.false ?? ""} onchange={(e) => setMapEntry("colorMap", "false", e.currentTarget.value)}>
              {#each COLORS as c}<option value={c}>{c ? `false → ${c}` : "false → —"}</option>{/each}
            </select>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<datalist id="binds">{#each bindSuggestions as s}<option value={s}></option>{/each}</datalist>
<datalist id="arrpaths">{#each arrayPaths as s}<option value={s}></option>{/each}</datalist>

<style>
  .editor {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .tree {
    border: 1px solid var(--card-border);
    border-radius: 10px;
    padding: 8px;
    background: rgba(0, 0, 0, 0.15);
  }
  .tchildren {
    margin-left: 14px;
    border-left: 1px solid var(--card-border);
    padding-left: 8px;
  }
  .trow {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 2px 0;
  }
  .tlabel {
    flex: 1;
    text-align: left;
    background: transparent;
    border: none;
    color: var(--ink);
    cursor: pointer;
    font-family: "IBM Plex Mono", monospace;
    font-size: 12px;
    padding: 3px 6px;
    border-radius: 6px;
  }
  .trow.sel .tlabel {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    color: var(--accent);
  }
  .tbtn {
    background: transparent;
    border: none;
    color: var(--faint);
    cursor: pointer;
    font-size: 11px;
    padding: 2px 5px;
  }
  .tbtn:hover {
    color: var(--ink);
  }
  .prop {
    display: flex;
    flex-direction: column;
    gap: 8px;
    border: 1px solid var(--card-border);
    border-radius: 10px;
    padding: 10px;
  }
  .prophdr {
    font-family: "IBM Plex Mono", monospace;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--accent);
  }
  .prop label {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 11px;
    color: var(--muted);
  }
  .proprow {
    display: flex;
    gap: 8px;
  }
  .proprow label {
    flex: 1;
  }
  .prop input,
  .prop select {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 7px;
    padding: 6px 8px;
    color: var(--ink);
    font-size: 12px;
    outline: none;
    min-width: 0;
  }
  .maps {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .maplbl {
    font-size: 10px;
    color: var(--faint);
  }
</style>
