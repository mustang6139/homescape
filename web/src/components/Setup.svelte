<script lang="ts">
  import { dndzone } from "svelte-dnd-action";
  import type { Spec, Widget, TestResult } from "../lib/types";
  import { store, patch, save } from "../lib/store.svelte";
  import { THEMES } from "../lib/themes";
  import { registry, loadRegistry } from "../lib/integrations.svelte";
  import { serviceStatus } from "../lib/resources.svelte";
  import { resolveSource, type ComposedSource } from "../lib/compose/source";
  import { fieldsFor } from "../lib/compose/introspect";
  import TreeEditor from "./compose/TreeEditor.svelte";
  import Composed from "./compose/Composed.svelte";
  import { PRESETS, type Preset } from "../lib/compose/presets";
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

  // --- Widgets / Composer (draft model: edit a widget, commit on Save) ---
  let selectedWidgetId = $state<string | null>(null);
  let draftW = $state<any>(null); // canonical widget draft (object)
  let jsonText = $state(""); // editable JSON serialisation (round-trips with draftW)
  let jsonError = $state("");
  let dirty = $state(false);

  function selectWidget(id: string) {
    selectedWidgetId = id;
    const w = spec.widgets.find((x) => x.id === id);
    draftW = w ? structuredClone($state.snapshot(w)) : null;
    jsonText = draftW ? JSON.stringify(draftW, null, 2) : "";
    jsonError = "";
    dirty = false;
  }

  // Dirty guard: when not mid-edit, refresh the draft from the spec so external/committed
  // changes re-normalise it without clobbering active editing.
  $effect(() => {
    if (!selectedWidgetId || dirty) return;
    const w = spec.widgets.find((x) => x.id === selectedWidgetId);
    if (w) {
      draftW = structuredClone($state.snapshot(w));
      jsonText = JSON.stringify(draftW, null, 2);
    }
  });

  // Visual edits mutate draftW in place → reflect into the JSON view.
  function syncJson() {
    jsonText = JSON.stringify(draftW, null, 2);
    jsonError = "";
    dirty = true;
  }

  // JSON edits parse back into draftW (on valid) → reflect into the visual editor + preview.
  function onJsonInput(v: string) {
    jsonText = v;
    dirty = true;
    try {
      const p = JSON.parse(v);
      if (selectedWidgetId) p.id = selectedWidgetId;
      draftW = p;
      jsonError = "";
    } catch (e) {
      jsonError = "Invalid JSON: " + (e as Error).message;
    }
  }

  async function saveDraft() {
    if (jsonError || !draftW) return;
    const next: Spec = structuredClone($state.snapshot(spec));
    const idx = next.widgets.findIndex((w) => w.id === selectedWidgetId);
    if (idx < 0) return;
    draftW.id = selectedWidgetId;
    next.widgets[idx] = structuredClone($state.snapshot(draftW));
    try {
      await save(next); // server validates (incl. composed schema)
      dirty = false;
      jsonError = "";
    } catch (e) {
      jsonError = String(e);
    }
  }

  function discardDraft() {
    if (selectedWidgetId) selectWidget(selectedWidgetId);
  }

  function addPreset(p: Preset) {
    const base = p.id === "blank" ? "composed" : p.id;
    const id = base + "-" + Math.random().toString(36).slice(2, 5);
    const next: Spec = structuredClone($state.snapshot(spec));
    next.widgets.push({
      id,
      type: "composed",
      column: 0,
      title: p.title,
      options: structuredClone(p.options) as Record<string, unknown>,
    });
    save(next).then(() => selectWidget(id));
  }

  function removeWidget(id: string) {
    const next: Spec = structuredClone($state.snapshot(spec));
    next.widgets = next.widgets.filter((w) => w.id !== id);
    save(next);
    if (selectedWidgetId === id) {
      selectedWidgetId = null;
      draftW = null;
      jsonText = "";
    }
  }

  // --- Composer: data source + field introspection (reads the DRAFT) ---
  const composedSource = $derived(
    draftW?.type === "composed"
      ? (draftW.options?.source as ComposedSource | undefined)
      : undefined,
  );
  const liveSample = $derived(composedSource ? resolveSource(composedSource) : undefined);
  const fields = $derived(composedSource ? fieldsFor(composedSource, liveSample) : []);

  const resourceId = $derived(
    composedSource?.kind === "resource" ? (composedSource.resource ?? "").split("|")[0] : "",
  );

  function updateSource(source: ComposedSource) {
    if (!draftW) return;
    draftW.options = { ...(draftW.options ?? {}), source };
    syncJson();
  }
  function setSourceKind(kind: string) {
    if (kind === "resource") updateSource({ kind: "resource", resource: "" });
    else if (kind === "static") updateSource({ kind: "static", data: {} });
    else updateSource({ kind: kind as ComposedSource["kind"] });
  }
  function setResource(id: string) {
    updateSource({ kind: "resource", resource: `${id}|service.status` });
  }
  function copyPath(p: string) {
    try {
      navigator.clipboard?.writeText(p);
    } catch {
      /* clipboard unavailable */
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
        <div class="presetbar">
          <span class="kicker">Add composed widget</span>
          {#each PRESETS as p (p.id)}
            <button class="ghost sm" title={p.description} onclick={() => addPreset(p)}>＋ {p.name}</button>
          {/each}
        </div>

        {#if draftW}
          {#if draftW.type === "composed"}
            <p class="hint">
              Compose from primitives: pick a data source, add elements, and bind them to its
              fields. It's all declarative and shareable — no code, no config files.
            </p>
            <h3>Data source</h3>
            <div class="seg">
              {#each ["resource", "host", "services", "static"] as k}
                <button class:sel={composedSource?.kind === k} onclick={() => setSourceKind(k)}>{k}</button>
              {/each}
            </div>
            {#if composedSource?.kind === "resource"}
              <select class="src-select" value={resourceId} onchange={(e) => setResource(e.currentTarget.value)}>
                <option value="" disabled selected={resourceId === ""}>Choose a service…</option>
                {#each registry.list as it (it.id)}
                  <option value={it.id}>{it.name} (service.status)</option>
                {/each}
              </select>
            {/if}

            <h3>Preview <span class="kicker">draft · saved on Save</span></h3>
            <div class="preview">
              <Composed widget={draftW} />
            </div>

            <h3>Build <span class="kicker">primitives</span></h3>
            {#key selectedWidgetId}
              <TreeEditor root={draftW.options.view} {fields} onChange={syncJson} />
            {/key}

            <h3>Available fields <span class="kicker">click to copy a path</span></h3>
            {#if fields.length}
              <div class="fields">
                {#each fields as f (f.path)}
                  <button class="field" onclick={() => copyPath(f.path)}>
                    <span class="fpath">{f.path}</span><span class="ftype">{f.type}</span>
                  </button>
                {/each}
              </div>
            {/if}
          {/if}

          <h3>Edit as JSON <span class="kicker">round-trips with the editor</span></h3>
          <textarea
            class="jsoned"
            value={jsonText}
            oninput={(e) => onJsonInput(e.currentTarget.value)}
            spellcheck="false"
          ></textarea>
          {#if jsonError}
            <p class="testres">{jsonError}</p>
          {:else if dirty}
            <p class="testres ok">valid · unsaved changes</p>
          {:else}
            <p class="testres">saved</p>
          {/if}
          <div class="actions">
            <button class="primary" onclick={saveDraft} disabled={!!jsonError || !dirty}>Save</button>
            <button class="ghost" onclick={discardDraft} disabled={!dirty}>Discard</button>
          </div>
        {:else}
          <p class="hint">Select a widget to edit, or add a composed one.</p>
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
  .presetbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  }
  .preview {
    border: 1px dashed var(--card-border);
    border-radius: 12px;
    padding: 10px;
    background: rgba(0, 0, 0, 0.12);
  }
  .src-select {
    width: 100%;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 9px;
    padding: 9px 11px;
    color: var(--ink);
    font-size: 13px;
    margin-top: 8px;
  }
  .fields {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .field {
    display: inline-flex;
    align-items: baseline;
    gap: 6px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--card-border);
    border-radius: 8px;
    padding: 5px 9px;
    cursor: pointer;
    font-size: 12px;
    color: var(--ink);
  }
  .field:hover {
    border-color: color-mix(in srgb, var(--accent) 50%, transparent);
  }
  .fpath {
    font-family: "IBM Plex Mono", monospace;
  }
  .ftype {
    font-size: 10px;
    color: var(--faint);
  }
</style>
