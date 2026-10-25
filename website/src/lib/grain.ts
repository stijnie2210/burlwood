export interface Knot {
  x: number;
  y: number;
  rx: number;
  ry: number;
}

export interface FigureRing {
  rx: number;
  ry: number;
  fill: string;
}

export interface Figure {
  x: number;
  y: number;
  rings: FigureRing[];
}

export interface GrainOptions {
  width: number;
  height: number;
  knots: Knot[];
  gap?: number;
  step?: number;
}

export interface Grain {
  lines: string[];
  rings: Knot[];
}

export function figureRing(cx: number, cy: number, ring: FigureRing, index: number): string {
  let d = '';
  for (let i = 0; i <= 96; i++) {
    const t = (i / 96) * Math.PI * 2;
    const x = cx + ring.rx * Math.cos(t) * (1 + 0.025 * Math.sin(3 * t + index));
    const y = cy + ring.ry * Math.sin(t) * (1 + 0.06 * Math.sin(2 * t + index * 1.7));
    d += `${i === 0 ? 'M' : 'L'}${x.toFixed(1)} ${y.toFixed(1)}`;
  }
  return `${d}Z`;
}

export function bookmatch(knots: Knot[], width: number): Knot[] {
  const half = width / 2;
  return knots.flatMap((k) => (Math.abs(k.x - half) < 1 ? [k] : [k, { ...k, x: width - k.x }]));
}

export function grainLine(y0: number, width: number, knots: Knot[], step = 10): string {
  const half = width / 2;
  let d = '';
  for (let x = 0; x <= width; x += step) {
    // Mirroring the wave as well as the knots is what makes the two leaves read as bookmatched.
    const m = x <= half ? x : width - x;
    let y = y0 + Math.sin(m / 210 + y0 / 70) * 5;
    for (const k of knots) {
      const dy = y0 - k.y;
      const side = dy >= 0 ? 1 : -1;
      const along = Math.exp(-(((x - k.x) / (k.rx * 2.1)) ** 2));
      y += side * along * k.ry * 1.7 * Math.exp(-Math.abs(dy) / (k.ry * 2.4));
    }
    d += `${x === 0 ? 'M' : 'L'}${x} ${y.toFixed(1)}`;
  }
  return d;
}

export function grainRows(height: number, gap: number): number[] {
  const rows: number[] = [];
  for (let y = -gap * 3; y < height + gap * 3; y += gap) {
    rows.push(y);
  }
  return rows;
}

export function grain({ width, height, knots, gap = 60, step = 10 }: GrainOptions): Grain {
  const all = bookmatch(knots, width);
  return {
    lines: grainRows(height, gap).map((y0) => grainLine(y0, width, all, step)),
    rings: all,
  };
}
