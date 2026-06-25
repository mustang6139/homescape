<script lang="ts">
  import type { Widget } from "../../lib/types";

  let { widget }: { widget: Widget } = $props();

  // F3 will populate these from real connectors; for now read placeholders from options or
  // fall back to a representative sample so the dashboard looks alive.
  type Service = { name: string; ping?: number; up: boolean };
  const services = $derived(
    (widget.options?.services as Service[] | undefined)?.length
      ? (widget.options!.services as Service[])
      : [
          { name: "Jellyfin", ping: 9, up: true },
          { name: "Home Assistant", ping: 7, up: true },
          { name: "Vaultwarden", ping: 14, up: true },
          { name: "Nextcloud", ping: 212, up: true },
          { name: "Grafana", ping: 11, up: true },
          { name: "Immich", up: false },
        ],
  );

  const upCount = $derived(services.filter((s) => s.up).length);
</script>

<div class="card">
  <div class="head">
    <span class="kicker">Service health</span>
    <span class="summary">{upCount} up · {services.length - upCount} down</span>
  </div>
  <ul>
    {#each services as s}
      <li>
        <span class="dot" class:down={!s.up}></span>
        <span class="nm">{s.name}</span>
        <span class="ping" class:bad={!s.up}>{s.up ? `${s.ping}ms` : "down"}</span>
      </li>
    {/each}
  </ul>
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
    background: #5bd08a;
    box-shadow: 0 0 9px rgba(91, 208, 138, 0.7);
    flex-shrink: 0;
  }
  .dot.down {
    background: #e0876f;
    box-shadow: none;
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
</style>
