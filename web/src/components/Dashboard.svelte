<script lang="ts">
  import type { Spec, Widget } from "../lib/types";
  import WidgetHost from "./WidgetHost.svelte";

  let { spec }: { spec: Spec } = $props();

  // Group enabled widgets by column index, respecting layout.columns.
  const columns = $derived.by(() => {
    const cols: Widget[][] = Array.from({ length: spec.layout.columns }, () => []);
    for (const w of spec.widgets) {
      if (w.enabled === false) continue;
      const c = Math.min(w.column, spec.layout.columns - 1);
      cols[c].push(w);
    }
    return cols;
  });
</script>

<main class="grid" style="--cols:{spec.layout.columns}">
  {#each columns as col}
    <section class="column">
      {#each col as widget (widget.id)}
        <WidgetHost {widget} />
      {/each}
    </section>
  {/each}
</main>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(330px, 1fr));
    gap: var(--gap);
    margin-top: 18px;
    align-items: start;
  }
  .column {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    min-width: 0;
  }
</style>
