import type { GraphRow } from '../../stores/commits';

const CELL_W = 14;
const DOT_R = 4;
const KNOT_R = 5.5;

export { CELL_W };

// visibleWidth is set when the graph column is narrower than the lanes need: lanes beyond it
// are clipped, but a commit's dot is pulled in before the faded edge so every row keeps one.
export function drawGraphCell(
  ctx: CanvasRenderingContext2D,
  row: GraphRow,
  cellH: number,
  visibleWidth?: number
): void {
  const w = ctx.canvas.width;
  const midY = cellH / 2;

  ctx.clearRect(0, 0, w, cellH);
  ctx.lineWidth = 2;

  // Top half: incoming edges
  for (const e of row.edgesIn ?? []) {
    drawSegment(ctx, e.fromCol, 0, e.toCol, midY, e.color);
  }

  // Bottom half: outgoing edges
  for (const e of row.edges ?? []) {
    drawSegment(ctx, e.fromCol, midY, e.toCol, cellH, e.color);
  }

  // Commit dot
  const lastClearCol = visibleWidth ? Math.max(0, Math.floor(visibleWidth / CELL_W) - 2) : Infinity;
  const hidden = row.column > lastClearCol;
  const dotX = Math.min(row.column, lastClearCol) * CELL_W + CELL_W / 2;
  ctx.beginPath();
  if (hidden) {
    // A hollow ring says "this commit is in a lane further right", so it isn't read as part of
    // the edge lane it's drawn on. The surface-coloured fill hides the edge lane behind it.
    ctx.arc(dotX, midY, DOT_R - 0.5, 0, Math.PI * 2);
    ctx.fillStyle = cssVar('--surface-base');
    ctx.fill();
    ctx.strokeStyle = row.color;
    ctx.lineWidth = 1.5;
    ctx.stroke();
    return;
  }
  if (row.labels?.some((l) => l.type === 'head')) {
    drawKnot(ctx, dotX, midY);
    return;
  }
  ctx.arc(dotX, midY, DOT_R, 0, Math.PI * 2);
  ctx.fillStyle = row.color;
  ctx.fill();
  ctx.strokeStyle = cssVar('--surface-base');
  ctx.lineWidth = 1.5;
  ctx.stroke();
}

// The checked-out commit is drawn as the knot from the app icon: a dark core in a sapwood ring.
function drawKnot(ctx: CanvasRenderingContext2D, x: number, y: number): void {
  ctx.beginPath();
  ctx.arc(x, y, KNOT_R + 1.5, 0, Math.PI * 2);
  ctx.fillStyle = cssVar('--surface-base');
  ctx.fill();
  ctx.beginPath();
  ctx.arc(x, y, KNOT_R, 0, Math.PI * 2);
  ctx.fillStyle = cssVar('--brand-sapwood');
  ctx.fill();
  ctx.beginPath();
  ctx.arc(x, y, KNOT_R - 2.25, 0, Math.PI * 2);
  ctx.fillStyle = cssVar('--brand-heartwood');
  ctx.fill();
}

function drawSegment(
  ctx: CanvasRenderingContext2D,
  fromCol: number,
  fromY: number,
  toCol: number,
  toY: number,
  color: string
): void {
  const fromX = fromCol * CELL_W + CELL_W / 2;
  const toX = toCol * CELL_W + CELL_W / 2;

  ctx.beginPath();
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;

  if (fromX === toX) {
    ctx.moveTo(fromX, fromY);
    ctx.lineTo(toX, toY);
  } else {
    const midY = (fromY + toY) / 2;
    ctx.moveTo(fromX, fromY);
    ctx.bezierCurveTo(fromX, midY, toX, midY, toX, toY);
  }

  ctx.stroke();
}

const cssVarCache = new Map<string, string>();

// Canvas can't read CSS variables, so each token is resolved once from the root.
function cssVar(name: string): string {
  const cached = cssVarCache.get(name);
  if (cached) {
    return cached;
  }
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  if (value) {
    cssVarCache.set(name, value);
  }
  return value || '#000';
}
