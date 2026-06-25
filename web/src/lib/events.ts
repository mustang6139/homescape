// Central SSE connection: one EventSource shared by all stores, dispatched by event type.

type Handler = (data: unknown) => void;

const handlers = new Map<string, Set<Handler>>();
let es: EventSource | null = null;

// on registers a handler for an event type and returns an unsubscribe function.
export function on(type: string, h: Handler): () => void {
  let set = handlers.get(type);
  if (!set) {
    set = new Set();
    handlers.set(type, set);
  }
  set.add(h);
  return () => set!.delete(h);
}

// connect opens the shared SSE stream. Returns a disconnect function.
export function connect(): () => void {
  es = new EventSource("/api/events");
  es.onmessage = (msg) => {
    let ev: { type: string; data?: unknown };
    try {
      ev = JSON.parse(msg.data);
    } catch {
      return;
    }
    const set = handlers.get(ev.type);
    if (set) for (const h of set) h(ev.data);
  };
  return () => {
    es?.close();
    es = null;
  };
}
