// Mirrors internal/scape/scape.schema.json — the shared contract with the backend.

export type WidgetType =
  | "system-stats"
  | "service-health"
  | "clock"
  | "bookmarks"
  | "media-now-playing"
  | "composed";

export interface Widget {
  id: string;
  type: WidgetType;
  column: number;
  enabled?: boolean;
  title?: string;
  options?: Record<string, unknown>;
}

export interface Meta {
  name: string;
  theme: string;
  accent: string;
  density?: "compact" | "cozy" | "comfortable";
}

export interface Layout {
  columns: number;
}

export interface Spec {
  version: number;
  meta: Meta;
  layout: Layout;
  widgets: Widget[];
}

export interface Stats {
  cpuPercent: number;
  memUsed: number;
  memTotal: number;
  diskUsed: number;
  diskTotal: number;
  uptimeSecs: number;
  collectedAt: string;
}

// --- F3: integrations, live resources, discovery ---

export interface Integration {
  id: string;
  type: string;
  name: string;
  baseUrl: string;
  group: string;
  icon: string;
  source: "manual" | "discovery";
  status: "pending" | "active" | "hidden" | "stale";
  hasSecret: boolean;
}

export interface ServiceStatus {
  up: boolean;
  latencyMs: number;
  version?: string;
  message?: string;
}

export interface ResourceUpdate {
  integrationId: string;
  kind: string;
  data: unknown;
  at: string;
}

export interface TestResult {
  ok: boolean;
  version?: string;
  message: string;
  latencyMs: number;
}

export interface DiscoverySettings {
  enabled: boolean;
  mode: "review" | "auto";
  available?: boolean;
}
