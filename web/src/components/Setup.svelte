<script lang="ts">
  import { dndzone } from "svelte-dnd-action";
  import type { Spec, Widget } from "../lib/types";
  import { store, patch, save } from "../lib/store.svelte";
  import { THEMES } from "../lib/themes";

  let { onClose }: { onClose: () => void } = $props();

  const sections = [
    "Appearance",
    "Layout",
    "Widgets",
    "Integrations",
    "Auto-discovery",
    "Search",
    "Profiles",
    "General",
  ];
  let active = $state("Appearance");

  const spec = $derived(store.spec as Spec);

  // --- Appearance (live PATCH) ---
  const accents = ["#E6A95C", "#57C4C0", "#E0876F", "#8C9BFF", "#9BD16B", "#D98AC4"];
  function setTheme(theme: string) {
    patch({ meta: { theme } });
  }
  function setAccent(accent: string) {
    patch({ meta: { accent } });
  }
  function setDensity(density: string) {
    patch({ meta: { density } });
  }

  // --- Layout (full PUT after structural edits) ---
  function setColumns(n: number) {
    const next: Spec = structuredClone($state.snapshot(spec));
    next.layout.columns = n;
    // Clamp any widget that now points past the last column.
    for (const w of next.widgets) if (w.column > n - 1) w.column = n - 1;
    save(next);
  }

  // Per-column working lists for drag-and-drop.
  let columns = $state<Widget[][]>([]);
  $effect(() => {
    const cols: Widget[][] = Array.from({ length: spec.layout.columns }, () => []);
    for (const w of spec.widgets) {
      const c = Math.min(w.column, spec.layout.columns - 1);
      cols[c].push(structuredClone($state.snapshot(w)));
    }
    columns = cols;
  });

  function onConsider(ci: number, e: CustomEvent<{ items: Widget[] }>) {
    columns[ci] = e.detail.items;
  }
  function onFinalize(ci: number, e: CustomEvent<{ items: Widget[] }>) {
    columns[ci] = e.detail.items;
    persistLayout();
  }

  function persistLayout() {
    const widgets: Widget[] = [];
    columns.forEach((col, ci) => {
      for (const w of col) widgets.push({ ...w, column: ci });
    });
    const next: Spec = structuredClone($state.snapshot(spec));
    next.widgets = widgets;
    save(next);
  }

  function toggleWidget(id: string) {
    const next: Spec = structuredClone($state.snapshot(spec));
    const w = next.widgets.find((x) => x.id === id);
    if (w) w.enabled = w.enabled === false ? true : false;
    save(next);
  }
</script>

<aside class="drawer">
  <header class="top">
    <div>
      <h2 class="display">Customize your space</h2>
      <span class="kicker">live · applies instantly · no config files</span>
    </div>
    <button class="done" onclick={onClose}>Done</button>
  </header>

  <div class="body">
    <nav class="sections">
      {#each sections as s}
        <button class:active={active === s} onclick={() => (active = s)}>{s}</button>
      {/each}
    </nav>

    <div class="editor">
      {#if active === "Appearance"}
        <h3>Theme</h3>
        <div class="themes">
          {#each THEMES as t}
            <button
              class="theme"
              class:sel={spec.meta.theme === t.id}
              style="background:{t.vars['--bg']}"
              onclick={() => setTheme(t.id)}
            >
              <span class="swatch" style="background:{t.vars['--teal']}"></span>
              <span class="tlabel">{t.label}</span>
            </button>
          {/each}
        </div>

        <h3>Accent</h3>
        <div class="accents">
          {#each accents as a}
            <button
              class="accent"
              class:sel={spec.meta.accent.toLowerCase() === a.toLowerCase()}
              style="background:{a}"
              onclick={() => setAccent(a)}
              aria-label={a}
            ></button>
          {/each}
        </div>

        <h3>Density</h3>
        <div class="seg">
          {#each ["compact", "cozy", "comfortable"] as d}
            <button class:sel={(spec.meta.density ?? "cozy") === d} onclick={() => setDensity(d)}>{d}</button>
          {/each}
        </div>
      {:else if active === "Layout"}
        <h3>Columns</h3>
        <div class="seg">
          {#each [1, 2, 3, 4] as n}
            <button class:sel={spec.layout.columns === n} onclick={() => setColumns(n)}>{n}</button>
          {/each}
        </div>

        <h3>Arrange widgets</h3>
        <p class="hint">Drag between columns to rearrange. Toggle to show/hide.</p>
        <div class="cols">
          {#each columns as col, ci (ci)}
            <div
              class="col"
              use:dndzone={{ items: col, flipDurationMs: 150 }}
              onconsider={(e) => onConsider(ci, e)}
              onfinalize={(e) => onFinalize(ci, e)}
            >
              {#each col as w (w.id)}
                <div class="wchip" class:off={w.enabled === false}>
                  <span class="grip">⋮⋮</span>
                  <span class="wname">{w.title ?? w.type}</span>
                  <button class="toggle" onclick={() => toggleWidget(w.id)}>
                    {w.enabled === false ? "off" : "on"}
                  </button>
                </div>
              {/each}
            </div>
          {/each}
        </div>
      {:else}
        <div class="placeholder">
          <p>“{active}” comes in a later phase.</p>
          <span class="kicker">F3 integrations · F5 profiles · F6 scapes</span>
        </div>
      {/if}
    </div>
  </div>
</aside>

<style>
  .drawer {
    position: fixed;
    top: 0;
    right: 0;
    height: 100vh;
    width: min(440px, 96vw);
    background: color-mix(in srgb, var(--bg) 86%, #000);
    border-left: 1px solid var(--card-border);
    box-shadow: -20px 0 60px rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(20px);
    display: flex;
    flex-direction: column;
    z-index: 50;
  }
  .top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 18px 20px;
    border-bottom: 1px solid var(--card-border);
  }
  h2 {
    font-size: 18px;
    margin-bottom: 3px;
  }
  .done {
    background: var(--accent);
    color: #241a0c;
    border: none;
    border-radius: 10px;
    padding: 9px 16px;
    font-weight: 600;
    cursor: pointer;
  }
  .body {
    flex: 1;
    display: flex;
    min-height: 0;
  }
  .sections {
    width: 130px;
    flex-shrink: 0;
    border-right: 1px solid var(--card-border);
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow-y: auto;
  }
  .sections button {
    text-align: left;
    background: transparent;
    border: none;
    color: var(--muted);
    padding: 9px 10px;
    border-radius: 8px;
    font-size: 13px;
    cursor: pointer;
  }
  .sections button.active {
    background: color-mix(in srgb, var(--accent) 16%, transparent);
    color: var(--accent);
    font-weight: 600;
  }
  .editor {
    flex: 1;
    padding: 16px 18px;
    overflow-y: auto;
  }
  h3 {
    font-family: "Space Grotesk", sans-serif;
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--faint);
    margin: 18px 0 10px;
  }
  h3:first-child {
    margin-top: 0;
  }
  .themes {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .theme {
    border: 2px solid var(--card-border);
    border-radius: 12px;
    padding: 14px 12px;
    cursor: pointer;
    text-align: left;
    color: var(--ink);
    min-height: 70px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }
  .theme.sel {
    border-color: var(--accent);
  }
  .swatch {
    width: 14px;
    height: 14px;
    border-radius: 50%;
  }
  .tlabel {
    font-size: 13px;
    font-weight: 600;
  }
  .accents {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }
  .accent {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
  }
  .accent.sel {
    border-color: var(--ink);
    outline: 2px solid var(--accent);
  }
  .seg {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
  .seg button {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    color: var(--ink);
    border-radius: 9px;
    padding: 8px 14px;
    cursor: pointer;
    font-size: 13px;
    text-transform: capitalize;
  }
  .seg button.sel {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    border-color: color-mix(in srgb, var(--accent) 40%, transparent);
    color: var(--accent);
    font-weight: 600;
  }
  .hint {
    font-size: 12px;
    color: var(--muted);
    margin-bottom: 10px;
  }
  .cols {
    display: flex;
    gap: 8px;
  }
  .col {
    flex: 1;
    min-height: 60px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px dashed var(--card-border);
    border-radius: 10px;
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .wchip {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--card);
    border: 1px solid var(--card-border);
    border-radius: 8px;
    padding: 7px 8px;
    font-size: 12px;
    cursor: grab;
  }
  .wchip.off {
    opacity: 0.45;
  }
  .grip {
    color: var(--faint);
    font-size: 10px;
  }
  .wname {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .toggle {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid var(--card-border);
    color: var(--muted);
    border-radius: 6px;
    padding: 2px 7px;
    font-size: 10px;
    cursor: pointer;
  }
  .placeholder {
    color: var(--muted);
    padding: 30px 4px;
  }
</style>
