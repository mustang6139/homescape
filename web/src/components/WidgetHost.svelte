<script lang="ts">
  import type { Widget } from "../lib/types";
  import SystemStats from "./widgets/SystemStats.svelte";
  import ServiceHealth from "./widgets/ServiceHealth.svelte";
  import Clock from "./widgets/Clock.svelte";
  import Bookmarks from "./widgets/Bookmarks.svelte";
  import MediaNowPlaying from "./widgets/MediaNowPlaying.svelte";
  import Composed from "./compose/Composed.svelte";

  let { widget }: { widget: Widget } = $props();

  const registry = {
    "system-stats": SystemStats,
    "service-health": ServiceHealth,
    clock: Clock,
    bookmarks: Bookmarks,
    "media-now-playing": MediaNowPlaying,
    composed: Composed,
  } as const;

  const Comp = $derived(registry[widget.type]);
</script>

{#if Comp}
  <Comp {widget} />
{:else}
  <div class="card">Unknown widget: {widget.type}</div>
{/if}
