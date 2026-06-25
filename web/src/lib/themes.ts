// Theme definitions. A theme maps to a set of CSS custom properties; switching theme is a
// variable swap, not a re-render. Accent is overridden separately from meta.accent.

export interface Theme {
  id: string;
  label: string;
  vars: Record<string, string>;
}

export const THEMES: Theme[] = [
  {
    id: "midnight-glass",
    label: "Midnight Glass",
    vars: {
      "--bg": "#141310",
      "--ink": "#f4f1ea",
      "--muted": "rgba(244,241,234,0.5)",
      "--faint": "rgba(244,241,234,0.34)",
      "--card": "rgba(255,255,255,0.045)",
      "--card-border": "rgba(255,255,255,0.09)",
      "--teal": "#57C4C0",
    },
  },
  {
    id: "warm-hearth",
    label: "Warm Hearth",
    vars: {
      "--bg": "#1c1510",
      "--ink": "#f6efe6",
      "--muted": "rgba(246,239,230,0.55)",
      "--faint": "rgba(246,239,230,0.36)",
      "--card": "rgba(255,236,210,0.05)",
      "--card-border": "rgba(255,220,180,0.12)",
      "--teal": "#d98a4a",
    },
  },
  {
    id: "neon-cove",
    label: "Neon Cove",
    vars: {
      "--bg": "#0d1020",
      "--ink": "#eaf0ff",
      "--muted": "rgba(234,240,255,0.55)",
      "--faint": "rgba(234,240,255,0.36)",
      "--card": "rgba(120,140,255,0.06)",
      "--card-border": "rgba(140,160,255,0.14)",
      "--teal": "#5ad1ff",
    },
  },
];

export function themeById(id: string): Theme {
  return THEMES.find((t) => t.id === id) ?? THEMES[0];
}

const DENSITY_GAP: Record<string, string> = {
  compact: "12px",
  cozy: "18px",
  comfortable: "26px",
};

// applyTheme writes the theme + accent + density onto the document root.
export function applyTheme(themeId: string, accent: string, density = "cozy") {
  const theme = themeById(themeId);
  const root = document.documentElement;
  for (const [k, v] of Object.entries(theme.vars)) root.style.setProperty(k, v);
  root.style.setProperty("--accent", accent);
  root.style.setProperty("--gap", DENSITY_GAP[density] ?? DENSITY_GAP.cozy);
}
