// Mirrors internal/scape/scape.schema.json — the shared contract with the backend.

export type WidgetType =
  | "system-stats"
  | "service-health"
  | "clock"
  | "bookmarks"
  | "media-now-playing";

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
