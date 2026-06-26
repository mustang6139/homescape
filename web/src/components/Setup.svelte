<script lang="ts">
  import { dndzone } from "svelte-dnd-action";
  import type { Spec, Widget, TestResult } from "../lib/types";
  import { store, patch, save } from "../lib/store.svelte";
  import { THEMES } from "../lib/themes";
  import { registry, loadRegistry } from "../lib/integrations.svelte";
  import { serviceStatus } from "../lib/resources.svelte";
  import {
    createIntegration,
    deleteIntegration,
    testIntegration,
    saveDiscoverySettings,
    acceptDiscovered,
    hideDiscovered,
  } from "../lib/api";

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

  // --- Integrations ---
  let form = $state({ name: "", baseUrl: "" });
  let testState = $state<{ pending: boolean; result?: TestResult; error?: string }>({
    pending: false,
  });
  let formError = $state("");

  async function runTest() {
    testState = { pending: true };
    try {
      const result = await testIntegration({ type: "http-health", baseUrl: form.baseUrl });
      testState = { pending: false, result };
    } catch (e) {
      testState = { pending: false, error: String(e) };
    }
  }

  async function addIntegration() {
    formError = "";
    if (!form.name.trim()) {
      formError = "Name is required";
      return;
    }
    try {
      await createIntegration({ type: "http-health", name: form.name, baseUrl: form.baseUrl });
      form = { name: "", baseUrl: "" };
      testState = { pending: false };
      await loadRegistry();
    } catch (e) {
      formError = String(e);
    }
  }

  async function removeIntegration(id: string) {
    await deleteIntegration(id);
    await loadRegistry();
  }

  // --- Auto-discovery ---
  async function setDiscovery(enabled: boolean, mode: string) {
    await saveDiscoverySettings({ enabled, mode: mode as "review" | "auto" });
    await loadRegistry();
  }
  async function accept(id: string) {
    await acceptDiscovered(id);
    await loadRegistry();
  }
  async function hide(id: string) {
    await hideDiscovered(id);
    await loadRegistry();
  }

  // --- Widgets / L3 escape hatch (edit a widget as JSON) ---
  let selectedWidgetId = $state<string | null>(null);
  let draft = $state("");
  let draftDirty = $state(false);
  let draftError = $state("");

  const selectedWidget = $derived(spec.widgets.find((w) => w.id === selectedWidgetId) ?? null);

  function selectWidget(id: string) {
    selectedWidgetId = id;
    const w = spec.widgets.find((x) => x.id === id);
    draft = w ? JSON.stringify($state.snapshot(w), null, 2) : "";
    draftDirty = false;
    draftError = "";
  }

  // Dirty guard: refresh the draft from the spec only when the user isn't mid-edit, so an
  // incoming SSE update (or our own commit) re-normalises the JSON without clobbering typing.
  $effect(() => {
    if (!selectedWidgetId || draftDirty) return;
    const w = spec.widgets.find((x) => x.id === selectedWidgetId);
    if (w) draft = JSON.stringify($state.snapshot(w), null, 2);
  });

  function onDraftInput(v: string) {
    draft = v;
    draftDirty = true;
    try {
      JSON.parse(v);
      draftError = "";
    } catch (e) {
      draftError = "Invalid JSON: " + (e as Error).message;
    }
  }

  async function applyDraft() {
    let parsed: any;
    try {
      parsed = JSON.parse(draft);
    } catch (e) {
      draftError = String(e);
      return;
    }
    const next: Spec = structuredClone($state.snapshot(spec));
    const idx = next.widgets.findIndex((w) => w.id === selectedWidgetId);
    if (idx < 0) {
      draftError = "widget not found";
      return;
    }
    parsed.id = selectedWidgetId; // keep the handle stable
    next.widgets[idx] = parsed;
    try {
      await save(next); // server validates (incl. composed schema); error surfaces below
      draftDirty = false;
      draftError = "";
    } catch (e) {
      draftError = String(e);
    }
  }

  function revertDraft() {
    if (selectedWidgetId) selectWidget(selectedWidgetId);
  }

  function addComposed() {
    const id = "composed-" + Math.random().toString(36).slice(2, 7);
    const next: Spec = structuredClone($state.snapshot(spec));
    next.widgets.push({
      id,
      type: "composed",
      column: 0,
      title: "New widget",
      options: {
        source: { kind: "host" },
        view: { el: "stack", children: [{ el: "text", text: "Hello" }] },
      },
    });
    save(next).then(() => selectWidget(id));
  }

  function removeWidget(id: string) {
    const next: Spec = structuredClone($state.snapshot(spec));
    next.widgets = next.widgets.filter((w) => w.id !== id);
    save(next);
    if (selectedWidgetId === id) {
      selectedWidgetId = null;
      draft = "";
    }
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
      {:else if active === "Widgets"}
        <h3>Widgets</h3>
        <ul class="ilist">
          {#each spec.widgets as w (w.id)}
            <li>
              <button class="wsel" class:sel={selectedWidgetId === w.id} onclick={() => selectWidget(w.id)}>
                <span class="iname">{w.title ?? w.type}</span>
                <span class="ibadge">{w.type}</span>
              </button>
              <button class="del" onclick={() => removeWidget(w.id)} aria-label="Delete">✕</button>
            </li>
          {/each}
        </ul>
        <button class="ghost" onclick={addComposed}>＋ Add composed widget</button>

        {#if selectedWidget}
          <h3>Edit as JSON <span class="kicker">live · no config files</span></h3>
          <textarea
            class="jsoned"
            value={draft}
            oninput={(e) => onDraftInput(e.currentTarget.value)}
            spellcheck="false"
          ></textarea>
          {#if draftError}
            <p class="testres">{draftError}</p>
          {:else}
            <p class="testres ok">valid JSON</p>
          {/if}
          <div class="actions">
            <button class="primary" onclick={applyDraft} disabled={!!draftError}>Apply</button>
            <button class="ghost" onclick={revertDraft}>Revert</button>
          </div>
          <p class="hint">
            Edit the widget's declarative spec directly — the dashboard follows on Apply. The
            GUI and this JSON are the same state.
          </p>
        {:else}
          <p class="hint">Select a widget to edit its JSON, or add a composed one.</p>
        {/if}
      {:else if active === "Integrations"}
        <h3>Your services</h3>
        {#if registry.list.length === 0}
          <p class="hint">No integrations yet. Add one below or enable Docker discovery.</p>
        {:else}
          <ul class="ilist">
            {#each registry.list as it (it.id)}
              {@const st = serviceStatus(it.id)}
              <li>
                <span class="dot" class:up={st?.up} class:down={st && !st.up}></span>
                <span class="iname">{it.name}</span>
                <span class="ibadge">{it.status}</span>
                {#if it.source === "discovery"}<span class="ibadge disc">auto</span>{/if}
                <button class="del" onclick={() => removeIntegration(it.id)} aria-label="Delete">✕</button>
              </li>
            {/each}
          </ul>
        {/if}

        <h3>Add a service (HTTP health)</h3>
        <div class="addform">
          <input placeholder="Name (e.g. Jellyfin)" bind:value={form.name} />
          <input placeholder="URL (e.g. http://jellyfin:8096)" bind:value={form.baseUrl} />
          <div class="actions">
            <button class="ghost" onclick={runTest} disabled={testState.pending || !form.baseUrl}>
              {testState.pending ? "Testing…" : "Test connection"}
            </button>
            <button class="primary" onclick={addIntegration}>Add</button>
          </div>
          {#if testState.result}
            <p class="testres" class:ok={testState.result.ok}>
              {testState.result.ok ? "✓" : "✕"}
              {testState.result.message}{testState.result.latencyMs ? ` · ${testState.result.latencyMs}ms` : ""}
            </p>
          {:else if testState.error}
            <p class="testres">✕ {testState.error}</p>
          {/if}
          {#if formError}<p class="testres">{formError}</p>{/if}
        </div>
        <p class="hint">Keys stay on your server. Encrypted at rest when HS_SECRET_KEY is set.</p>
      {:else if active === "Auto-discovery"}
        {#if !registry.discovery.available}
          <p class="hint">
            The Docker socket isn't mounted, so discovery is unavailable. Mount it read-only:
            <code>/var/run/docker.sock:/var/run/docker.sock:ro</code>
          </p>
        {/if}
        <h3>Docker discovery</h3>
        <label class="toggle">
          <input
            type="checkbox"
            checked={registry.discovery.enabled}
            disabled={!registry.discovery.available}
            onchange={(e) => setDiscovery(e.currentTarget.checked, registry.discovery.mode)}
          />
          Watch the Docker socket for labelled containers
        </label>

        <h3>When a new container is found</h3>
        <div class="seg">
          {#each ["review", "auto"] as m}
            <button
              class:sel={registry.discovery.mode === m}
              disabled={!registry.discovery.available}
              onclick={() => setDiscovery(registry.discovery.enabled, m)}
            >
              {m === "review" ? "Review first" : "Add automatically"}
            </button>
          {/each}
        </div>

        {#if registry.pending.length}
          <h3>Pending review ({registry.pending.length})</h3>
          <ul class="ilist">
            {#each registry.pending as it (it.id)}
              <li>
                <span class="iname">{it.name}</span>
                {#if it.group}<span class="ibadge">{it.group}</span>{/if}
                <button class="ghost sm" onclick={() => accept(it.id)}>Add</button>
                <button class="del" onclick={() => hide(it.id)} aria-label="Hide">✕</button>
              </li>
            {/each}
          </ul>
        {/if}

        <h3>Labels</h3>
        <p class="hint">
          Label a container with <code>homescape.enable=true</code> to surface it. Optional:
          <code>homescape.name</code>, <code>homescape.group</code>, <code>homescape.url</code>
          (enables HTTP health checks), <code>homescape.icon</code>.
        </p>
      {:else}
        <div class="placeholder">
          <p>“{active}” comes in a later phase.</p>
          <span class="kicker">F5 profiles · F6 scapes</span>
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

  /* Integrations + discovery */
  .ilist {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 8px;
  }
  .ilist li {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--card);
    border: 1px solid var(--card-border);
    border-radius: 8px;
    padding: 8px 10px;
    font-size: 13px;
  }
  .iname {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ibadge {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--muted);
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 6px;
    padding: 2px 6px;
  }
  .ibadge.disc {
    color: var(--teal);
    border-color: color-mix(in srgb, var(--teal) 30%, transparent);
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--faint);
    flex-shrink: 0;
  }
  .dot.up {
    background: #5bd08a;
    box-shadow: 0 0 8px rgba(91, 208, 138, 0.7);
  }
  .dot.down {
    background: #e0876f;
  }
  .del {
    background: transparent;
    border: none;
    color: var(--faint);
    cursor: pointer;
    font-size: 13px;
    padding: 2px 6px;
  }
  .del:hover {
    color: #e0876f;
  }
  .addform {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .addform input {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 9px;
    padding: 9px 11px;
    color: var(--ink);
    font-size: 13px;
    outline: none;
  }
  .addform input:focus {
    border-color: color-mix(in srgb, var(--accent) 50%, transparent);
  }
  .actions {
    display: flex;
    gap: 8px;
  }
  .ghost {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    color: var(--ink);
    border-radius: 9px;
    padding: 8px 12px;
    cursor: pointer;
    font-size: 13px;
  }
  .ghost.sm {
    padding: 4px 10px;
    font-size: 12px;
  }
  .ghost:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .primary {
    background: var(--accent);
    color: #241a0c;
    border: none;
    border-radius: 9px;
    padding: 8px 14px;
    font-weight: 600;
    cursor: pointer;
    font-size: 13px;
  }
  .testres {
    font-size: 12.5px;
    color: var(--muted);
    margin-top: 2px;
  }
  .testres.ok {
    color: #5bd08a;
  }
  .toggle {
    display: flex;
    align-items: center;
    gap: 9px;
    font-size: 13.5px;
    cursor: pointer;
  }
  code {
    font-family: "IBM Plex Mono", monospace;
    font-size: 11.5px;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 5px;
    padding: 1px 5px;
  }
  .wsel {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    background: transparent;
    border: none;
    color: var(--ink);
    cursor: pointer;
    text-align: left;
    padding: 0;
  }
  .wsel.sel .iname {
    color: var(--accent);
    font-weight: 600;
  }
  .jsoned {
    width: 100%;
    min-height: 240px;
    background: rgba(0, 0, 0, 0.25);
    border: 1px solid var(--card-border);
    border-radius: 10px;
    padding: 12px;
    color: var(--ink);
    font-family: "IBM Plex Mono", monospace;
    font-size: 12px;
    line-height: 1.5;
    resize: vertical;
    outline: none;
    white-space: pre;
    tab-size: 2;
  }
  .jsoned:focus {
    border-color: color-mix(in srgb, var(--accent) 50%, transparent);
  }
</style>
