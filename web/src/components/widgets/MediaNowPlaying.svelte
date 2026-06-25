<script lang="ts">
  import type { Widget } from "../../lib/types";

  let { widget }: { widget: Widget } = $props();

  // Placeholder until the Plex connector lands in F3.
  const sample = { title: "Dune: Part Two", where: "Living Room", at: "1:04:22", of: "2:46:00", progress: 39 };
</script>

<div class="card">
  <div class="head">
    <span class="dot"></span>
    <span class="kicker">{widget.title ?? "Plex"} · now playing</span>
  </div>
  <div class="body">
    <div class="poster"><span>poster</span></div>
    <div class="info">
      <div class="title">{sample.title}</div>
      <div class="sub">{sample.where} · {sample.at} of {sample.of}</div>
      <div class="bar"><i style="width:{sample.progress}%"></i></div>
    </div>
  </div>
</div>

<style>
  .head {
    display: flex;
    align-items: center;
    gap: 9px;
    margin-bottom: 14px;
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #5bd08a;
    box-shadow: 0 0 9px rgba(91, 208, 138, 0.8);
    animation: hs-pulse 2.4s ease-in-out infinite;
  }
  .body {
    display: flex;
    gap: 14px;
  }
  .poster {
    width: 62px;
    height: 92px;
    border-radius: 9px;
    flex-shrink: 0;
    border: 1px solid var(--card-border);
    background: repeating-linear-gradient(
      135deg,
      rgba(255, 255, 255, 0.07) 0 6px,
      rgba(255, 255, 255, 0.02) 6px 12px
    );
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding-bottom: 6px;
  }
  .poster span {
    font-family: "IBM Plex Mono", monospace;
    font-size: 8px;
    color: var(--faint);
  }
  .info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 8px;
  }
  .title {
    font-weight: 600;
    font-size: 15px;
  }
  .sub {
    font-size: 12px;
    color: var(--muted);
  }
  .bar {
    height: 6px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.09);
    overflow: hidden;
  }
  .bar i {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent), color-mix(in srgb, var(--accent) 60%, #fff));
  }
</style>
