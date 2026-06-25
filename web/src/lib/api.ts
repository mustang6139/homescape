import type { Spec, Stats } from "./types";

// Thin API client over the Go backend.

export async function fetchScape(): Promise<Spec> {
  const res = await fetch("/api/scape");
  if (!res.ok) throw new Error(`GET /api/scape: ${res.status}`);
  return res.json();
}

export async function putScape(spec: Spec): Promise<Spec> {
  const res = await fetch("/api/scape", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(spec),
  });
  if (!res.ok) throw new Error(`PUT /api/scape: ${res.status}`);
  return res.json();
}

// patchScape deep-merges a partial spec server-side (live, granular updates).
export async function patchScape(patch: Record<string, unknown>): Promise<Spec> {
  const res = await fetch("/api/scape", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw new Error(`PATCH /api/scape: ${res.status}`);
  return res.json();
}

export async function fetchStats(): Promise<Stats> {
  const res = await fetch("/api/stats");
  if (!res.ok) throw new Error(`GET /api/stats: ${res.status}`);
  return res.json();
}
