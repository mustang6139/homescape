<script lang="ts">
  import type { Widget } from "../../lib/types";

  let { widget }: { widget: Widget } = $props();

  type Link = { label: string; url: string };
  const links = $derived(
    (widget.options?.links as Link[] | undefined)?.length
      ? (widget.options!.links as Link[])
      : [
          { label: "Proxmox", url: "#" },
          { label: "Router", url: "#" },
          { label: "Grafana", url: "#" },
          { label: "Portainer", url: "#" },
        ],
  );
</script>

<div class="card">
  <div class="kicker" style="margin-bottom:12px">{widget.title ?? "Bookmarks"}</div>
  <div class="links">
    {#each links as l}
      <a href={l.url} class="chip">{l.label}</a>
    {/each}
  </div>
</div>

<style>
  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .chip {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--card-border);
    border-radius: 10px;
    padding: 8px 12px;
    font-size: 13px;
    color: var(--ink);
    text-decoration: none;
  }
  .chip:hover {
    border-color: color-mix(in srgb, var(--accent) 50%, transparent);
  }
</style>
