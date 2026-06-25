<script lang="ts">
  import { store } from "../lib/store.svelte";

  let { onCustomize }: { onCustomize: () => void } = $props();

  let now = $state(new Date());
  $effect(() => {
    const t = setInterval(() => (now = new Date()), 1000);
    return () => clearInterval(t);
  });

  const time = $derived(
    now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
  );
  const date = $derived(
    now.toLocaleDateString([], { weekday: "long", day: "numeric", month: "long" }),
  );
</script>

<header class="card hdr">
  <div class="brand">
    <div class="logo" style="background:linear-gradient(135deg,var(--accent),var(--teal));">
      <span>⌂</span>
    </div>
    <div class="names">
      <span class="display name">HomeScape</span>
      <span class="kicker">{store.spec?.meta.name ?? "your space"}</span>
    </div>
  </div>

  <div class="search">
    <button class="engine">● DuckDuckGo ▾</button>
    <input placeholder="Search the web, services, files and bookmarks…" />
  </div>

  <div class="right">
    <div class="clock">
      <span class="mono t">{time}</span>
      <span class="d">{date}</span>
    </div>
    <button class="customize" onclick={onCustomize}>✦ Customize</button>
  </div>
</header>

<style>
  .hdr {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 14px 22px;
    border-radius: 20px;
    padding: 13px 18px;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .logo {
    width: 38px;
    height: 38px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 18px;
    color: var(--bg);
  }
  .names {
    display: flex;
    flex-direction: column;
    line-height: 1.1;
  }
  .name {
    font-weight: 700;
    font-size: 18px;
  }
  .search {
    flex: 1 1 280px;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 10px;
    background: rgba(255, 255, 255, 0.055);
    border: 1px solid var(--card-border);
    border-radius: 13px;
    height: 48px;
    padding: 0 8px;
  }
  .engine {
    background: color-mix(in srgb, var(--accent) 16%, transparent);
    border: 1px solid color-mix(in srgb, var(--accent) 32%, transparent);
    color: var(--accent);
    border-radius: 9px;
    height: 34px;
    padding: 0 11px;
    font-size: 12.5px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
  }
  .search input {
    flex: 1;
    min-width: 0;
    background: transparent;
    border: none;
    outline: none;
    color: var(--ink);
    font-size: 14.5px;
    height: 100%;
  }
  .right {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .clock {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    line-height: 1.15;
  }
  .clock .t {
    font-size: 15px;
  }
  .clock .d {
    font-size: 11px;
    color: var(--muted);
  }
  .customize {
    background: var(--accent);
    color: #241a0c;
    border: none;
    border-radius: 11px;
    height: 42px;
    padding: 0 17px;
    font-size: 13.5px;
    font-weight: 600;
    cursor: pointer;
    box-shadow: 0 6px 18px color-mix(in srgb, var(--accent) 30%, transparent);
  }
</style>
