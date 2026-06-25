<script lang="ts">
  import type { Widget, Stats } from "../../lib/types";
  import { fetchStats } from "../../lib/api";

  let { widget }: { widget: Widget } = $props();

  let stats = $state<Stats | null>(null);

  $effect(() => {
    let alive = true;
    const tick = async () => {
      try {
        const s = await fetchStats();
        if (alive) stats = s;
      } catch {
        /* keep last known */
      }
    };
    tick();
    const t = setInterval(tick, 3000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  });

  const gb = (b: number) => (b / 1024 ** 3).toFixed(1);
  const tb = (b: number) => (b / 1024 ** 4).toFixed(1);
  const pct = (used: number, total: number) =>
    total > 0 ? Math.round((used / total) * 100) : 0;
  const uptime = (s: number) => {
    const d = Math.floor(s / 86400);
    const h = Math.floor((s % 86400) / 3600);
    return `${d}d ${h}h`;
  };
</script>

<div class="card">
  <div class="head">
    <span class="kicker">System · {String(widget.options?.host ?? "host")}</span>
    {#if stats}<span class="up">uptime {uptime(stats.uptimeSecs)}</span>{/if}
  </div>

  {#if stats}
    <div class="row">
      <span>CPU</span><span class="val">{stats.cpuPercent}%</span>
    </div>
    <div class="bar"><i style="width:{stats.cpuPercent}%"></i></div>

    <div class="row">
      <span>Memory</span>
      <span class="val">{gb(stats.memUsed)} / {gb(stats.memTotal)} GB</span>
    </div>
    <div class="bar"><i style="width:{pct(stats.memUsed, stats.memTotal)}%"></i></div>

    <div class="row">
      <span>Storage · /</span>
      <span class="val">{tb(stats.diskUsed)} / {tb(stats.diskTotal)} TB</span>
    </div>
    <div class="bar teal"><i style="width:{pct(stats.diskUsed, stats.diskTotal)}%"></i></div>
  {:else}
    <p class="muted">Reading host metrics…</p>
  {/if}
</div>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 14px;
  }
  .up {
    font-size: 11px;
    color: var(--faint);
    font-family: "IBM Plex Mono", monospace;
  }
  .row {
    display: flex;
    justify-content: space-between;
    font-size: 13.5px;
    margin: 14px 0 6px;
  }
  .val {
    color: var(--teal);
    font-weight: 600;
  }
  .bar {
    height: 7px;
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.08);
    overflow: hidden;
  }
  .bar i {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent), color-mix(in srgb, var(--accent) 60%, #fff));
    border-radius: 5px;
  }
  .bar.teal i {
    background: linear-gradient(90deg, var(--teal), color-mix(in srgb, var(--teal) 60%, #fff));
  }
  .muted {
    color: var(--muted);
    font-size: 13px;
  }
</style>
