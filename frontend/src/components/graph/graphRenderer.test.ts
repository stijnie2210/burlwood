// @vitest-environment jsdom
import { describe, it, expect } from 'vitest';
import { drawGraphCell, CELL_W } from './graphRenderer';
import type { GraphRow } from '../../stores/commits';

function fakeCtx() {
  const arcs: number[] = [];
  const fills: string[] = [];
  const strokes: string[] = [];
  const ctx = {
    fillStyle: '',
    strokeStyle: '',
    canvas: { width: 1000 },
    clearRect() {},
    beginPath() {},
    moveTo() {},
    lineTo() {},
    bezierCurveTo() {},
    stroke() {
      strokes.push(ctx.strokeStyle as string);
    },
    fill() {
      fills.push(ctx.fillStyle as string);
    },
    arc(x: number) {
      arcs.push(x);
    },
  } as unknown as CanvasRenderingContext2D;
  return { ctx, arcs, fills, strokes };
}

const LANE = '#e3a35c';
const row = (column: number) =>
  ({ column, color: LANE, edges: [], edgesIn: [] }) as unknown as GraphRow;

describe('drawGraphCell', () => {
  it('draws the dot in its own lane when the column is wide enough', () => {
    const { ctx, arcs } = fakeCtx();
    drawGraphCell(ctx, row(12), 28);
    expect(arcs).toEqual([12 * CELL_W + CELL_W / 2]);
  });

  it('pulls a far-right dot in before the faded edge of a capped column', () => {
    const { ctx, arcs } = fakeCtx();
    const visible = 10 * CELL_W;
    drawGraphCell(ctx, row(25), 28, visible);
    expect(arcs[0]).toBe(8 * CELL_W + CELL_W / 2);
    expect(arcs[0]).toBeLessThan(visible - CELL_W);
  });

  it('leaves dots inside the cap where they are', () => {
    const { ctx, arcs } = fakeCtx();
    drawGraphCell(ctx, row(3), 28, 10 * CELL_W);
    expect(arcs).toEqual([3 * CELL_W + CELL_W / 2]);
  });

  it('draws a pulled-in dot as a hollow ring in its lane colour', () => {
    const { ctx, fills, strokes } = fakeCtx();
    drawGraphCell(ctx, row(25), 28, 10 * CELL_W);
    expect(strokes.at(-1)).toBe(LANE);
    expect(fills.at(-1)).not.toBe(LANE);
  });

  it('keeps visible dots filled with their lane colour', () => {
    const { ctx, fills } = fakeCtx();
    drawGraphCell(ctx, row(3), 28, 10 * CELL_W);
    expect(fills.at(-1)).toBe(LANE);
  });
});
