<script lang="ts">
  import { onMount } from "svelte";
  import { store, load, listenScape } from "./lib/store.svelte";
  import { loadResources, listenResources } from "./lib/resources.svelte";
  import { loadRegistry, listenRegistry } from "./lib/integrations.svelte";
  import { startStats } from "./lib/stats.svelte";
  import { connect } from "./lib/events";
  import Header from "./components/Header.svelte";
  import Dashboard from "./components/Dashboard.svelte";
  import Setup from "./components/Setup.svelte";

  let editing = $state(false);

  onMount(() => {
    // Register handlers before opening the stream so no early event is missed.
    const offScape = listenScape();
    const offResources = listenResources();
    const offRegistry = listenRegistry();
    const disconnect = connect();

    load();
    loadResources();
    loadRegistry();
    const stopStats = startStats();

    return () => {
      offScape();
      offResources();
      offRegistry();
      disconnect();
      stopStats();
    };
  });
</script>

<div class="shell">
  <Header onCustomize={() => (editing = true)} />

  {#if store.loading && !store.spec}
    <p class="state">Loading your space…</p>
  {:else if store.error && !store.spec}
    <p class="state error">Could not load: {store.error}</p>
  {:else if store.spec}
    <Dashboard spec={store.spec} />
  {/if}

  {#if editing && store.spec}
    <Setup onClose={() => (editing = false)} />
  {/if}
</div>

<style>
  .shell {
    max-width: 1660px;
    margin: 0 auto;
    padding: 26px 30px 64px;
  }
  .state {
    color: var(--muted);
    padding: 40px 4px;
  }
  .state.error {
    color: #e0876f;
  }
</style>
