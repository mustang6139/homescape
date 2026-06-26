<script lang="ts">
  import type { Widget } from "../../lib/types";
  import { resolveSource, type ComposedSource } from "../../lib/compose/source";
  import Node from "./Node.svelte";

  let { widget }: { widget: Widget } = $props();

  const source = $derived((widget.options?.source as ComposedSource) ?? { kind: "static" });
  // Reading the stores inside this $derived keeps the widget live (SSE → re-render).
  const data = $derived(resolveSource(source));
  const view = $derived(widget.options?.view as any);
</script>

<div class="card">
  {#if widget.title}<div class="kicker" style="margin-bottom:12px">{widget.title}</div>{/if}
  {#if view}
    <Node node={view} {data} />
  {:else}
    <p class="muted">Empty composition.</p>
  {/if}
</div>

<style>
  .muted {
    color: var(--muted);
    font-size: 13px;
  }
</style>
