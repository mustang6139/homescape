import type {
  Spec,
  Stats,
  Integration,
  ResourceUpdate,
  TestResult,
  DiscoverySettings,
} from "./types";

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
  if (!res.ok) throw new Error(await errorMessage(res));
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

// --- integrations ---

export interface IntegrationInput {
  type: string;
  name?: string;
  baseUrl?: string;
  group?: string;
  icon?: string;
  secret?: string;
}

export async function fetchIntegrations(): Promise<Integration[]> {
  const res = await fetch("/api/integrations");
  if (!res.ok) throw new Error(`GET /api/integrations: ${res.status}`);
  return res.json();
}

export async function createIntegration(input: IntegrationInput): Promise<Integration> {
  const res = await fetch("/api/integrations", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) throw new Error(await errorMessage(res));
  return res.json();
}

export async function deleteIntegration(id: string): Promise<void> {
  const res = await fetch(`/api/integrations/${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) throw new Error(`DELETE: ${res.status}`);
}

export async function testIntegration(input: IntegrationInput): Promise<TestResult> {
  const res = await fetch("/api/integrations/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) throw new Error(await errorMessage(res));
  return res.json();
}

// --- live resources ---

export async function fetchResources(): Promise<ResourceUpdate[]> {
  const res = await fetch("/api/resources");
  if (!res.ok) throw new Error(`GET /api/resources: ${res.status}`);
  return res.json();
}

// --- discovery ---

export async function fetchDiscoverySettings(): Promise<DiscoverySettings> {
  const res = await fetch("/api/discovery/settings");
  if (!res.ok) throw new Error(`GET /api/discovery/settings: ${res.status}`);
  return res.json();
}

export async function saveDiscoverySettings(
  s: Pick<DiscoverySettings, "enabled" | "mode">,
): Promise<void> {
  const res = await fetch("/api/discovery/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(s),
  });
  if (!res.ok) throw new Error(`PUT /api/discovery/settings: ${res.status}`);
}

export async function fetchPending(): Promise<Integration[]> {
  const res = await fetch("/api/discovery/pending");
  if (!res.ok) throw new Error(`GET /api/discovery/pending: ${res.status}`);
  return res.json();
}

export async function acceptDiscovered(id: string): Promise<void> {
  const res = await fetch(`/api/discovery/${encodeURIComponent(id)}/accept`, { method: "POST" });
  if (!res.ok && res.status !== 204) throw new Error(`accept: ${res.status}`);
}

export async function hideDiscovered(id: string): Promise<void> {
  const res = await fetch(`/api/discovery/${encodeURIComponent(id)}/hide`, { method: "POST" });
  if (!res.ok && res.status !== 204) throw new Error(`hide: ${res.status}`);
}

async function errorMessage(res: Response): Promise<string> {
  try {
    const body = await res.json();
    if (body && typeof body.error === "string") return body.error;
  } catch {
    /* not JSON */
  }
  return `request failed: ${res.status}`;
}
