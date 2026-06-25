<script lang="ts">
  import type { Widget, Integration } from "../../lib/types";
  import { registry, activeIntegrations } from "../../lib/integrations.svelte";
  import { serviceStatus } from "../../lib/resources.svelte";

  let { widget }: { widget: Widget } = $props();

  // Resolve the binding: an explicit non-empty services list wins; otherwise all active
  // integrations (the beginner-bridge default).
  const services = $derived.by<Integration[]>(() => {
    const ids = widget.options?.services as string[] | undefined;
    if (ids && ids.length) {
      return ids
        .map((id) => registry.list.find((i) => i.id === id))
        .filter((i): i is Integration => !!i);
    }
    return activeIntegrations();
  });

  // Read each service's live status (may be undefined until first poll).
  const rows = $derived(
    services.map((svc) => ({ svc, status: serviceStatus(svc.id) })),
  );
  const upCount = $derived(rows.filter((r) => r.status?.up).length);
</script>

<div class="card">
  <div class="head">
    <span class="kicker">{widget.title ?? "Service health"}</span>
    {#if rows.length}
      <span class="summary">{upCount} up · {rows.length - upCount} down</span>
    {/if}
  </div>

  {#if !registry.loaded}
    <p class="muted">Loading…</p>
  {:else if rows.length === 0}
    <p class="muted">No services yet. Add one in Customize → Integrations, or enable Docker discovery.</p>
  {:else}
    <ul>
      {#each rows as { svc, status } (svc.id)}
        <li>
          <span
            class="dot"
            class:up={status?.up}
            class:down={status && !status.up}
            class:unknown={!status}
          ></span>
          <span class="nm" title={status?.message ?? ""}>{svc.name}</span>
          <span class="ping" class:bad={status && !status.up}>
            {#if !status}…{:else if status.up}{status.latencyMs}ms{:else}down{/if}
          </span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .summary {
    font-size: 11px;
    color: var(--faint);
    font-family: "IBM Plex Mono", monospace;
  }
  ul {
    list-style: none;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px 18px;
  }
  li {
    display: flex;
    align-items: center;
    gap: 9px;
    font-size: 13.5px;
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
    box-shadow: 0 0 9px rgba(91, 208, 138, 0.7);
  }
  .dot.down {
    background: #e0876f;
  }
  .dot.unknown {
    background: var(--faint);
    opacity: 0.5;
  }
  .nm {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ping {
    color: var(--muted);
    font-size: 12px;
  }
  .ping.bad {
    color: #e0876f;
  }
  .muted {
    color: var(--muted);
    font-size: 13px;
  }
</style>
