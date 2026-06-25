<script lang="ts">
  import type { Widget } from "../../lib/types";

  // Accepts a widget prop for a uniform WidgetHost interface (unused for now).
  let { widget: _widget }: { widget: Widget } = $props();

  let now = $state(new Date());
  $effect(() => {
    const t = setInterval(() => (now = new Date()), 1000);
    return () => clearInterval(t);
  });

  const time = $derived(
    now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
  );
  const date = $derived(
    now.toLocaleDateString([], {
      weekday: "long",
      year: "numeric",
      month: "long",
      day: "numeric",
    }),
  );
</script>

<div class="card clock">
  <span class="mono time">{time}</span>
  <span class="date">{date}</span>
</div>

<style>
  .clock {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 26px 18px;
  }
  .time {
    font-size: 40px;
    font-weight: 500;
    letter-spacing: 0.02em;
  }
  .date {
    color: var(--muted);
    font-size: 13px;
  }
</style>
