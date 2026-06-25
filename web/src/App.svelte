<script lang="ts">
  import { onMount } from "svelte";
  import { store, load, connectEvents } from "./lib/store.svelte";
  import Header from "./components/Header.svelte";
  import Dashboard from "./components/Dashboard.svelte";
  import Setup from "./components/Setup.svelte";

  let editing = $state(false);

  onMount(() => {
    load();
    const disconnect = connectEvents();
    return disconnect;
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
