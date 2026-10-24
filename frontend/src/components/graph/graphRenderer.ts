import type { GraphRow } from '../../stores/commits';

const CELL_W = 14;
const DOT_R = 4;

export { CELL_W };

export function drawGraphCell(ctx: CanvasRenderingContext2D, row: GraphRow, cellH: number): void {
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
  const dotX = row.column * CELL_W + CELL_W / 2;
  ctx.beginPath();
  ctx.arc(dotX, midY, DOT_R, 0, Math.PI * 2);
  ctx.fillStyle = row.color;
  ctx.fill();
  ctx.strokeStyle = dotOutlineColor();
  ctx.lineWidth = 1.5;
  ctx.stroke();
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

let cachedDotOutline = '';

// Canvas can't read CSS variables, so the dot outline resolves the graph's surface token once.
function dotOutlineColor(): string {
  if (!cachedDotOutline) {
    cachedDotOutline =
      getComputedStyle(document.documentElement).getPropertyValue('--surface-base').trim() ||
      '#1b1b1d';
  }
  return cachedDotOutline;
}
