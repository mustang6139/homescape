// Presentational token maps for composed primitives: style tokens → CSS, icon names →
// glyphs. Fixed sets — no arbitrary CSS or code.

const COLORS: Record<string, string> = {
  accent: "var(--accent)",
  muted: "var(--muted)",
  faint: "var(--faint)",
  ink: "var(--ink)",
  teal: "var(--teal)",
  up: "#5bd08a",
  down: "#e0876f",
};

const SIZES: Record<string, string> = { sm: "12px", md: "14px", lg: "18px" };

export interface StyleTokens {
  color?: string;
  size?: string;
  weight?: string;
}

// colorVar resolves a color token to its CSS value (or undefined if unknown).
export function colorVar(token: string | undefined): string | undefined {
  return token ? COLORS[token] : undefined;
}

// styleToCss turns style tokens into an inline style string.
export function styleToCss(style: StyleTokens | undefined): string {
  if (!style) return "";
  const parts: string[] = [];
  if (style.color && COLORS[style.color]) parts.push(`color:${COLORS[style.color]}`);
  if (style.size && SIZES[style.size]) parts.push(`font-size:${SIZES[style.size]}`);
  if (style.weight) parts.push(`font-weight:${style.weight}`);
  return parts.join(";");
}

const GLYPHS: Record<string, string> = {
  check: "✓",
  x: "✕",
  dot: "●",
  up: "▲",
  down: "▼",
  warn: "⚠",
  play: "▶",
  pause: "⏸",
};

// iconGlyph maps an icon name to a glyph, falling back to the name itself.
export function iconGlyph(name: string): string {
  return GLYPHS[name] ?? name;
}
