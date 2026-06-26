import { useRef, useState, useCallback, useEffect } from 'react';
import { Recipe } from '../types/recipe';
import '../styles/RecipeGraph.css';

interface RecipeGraphProps {
  recipes: Recipe[];
  usedIn: Record<string, Recipe[]>;
  onRecipeClick: (recipe: Recipe) => void;
}

const NODE_W = 220;
const NODE_H = 62;
const COL_GAP = 300;
const ROW_GAP = 84;
const PAD = 80;

function computeColumn(id: string, recipes: Recipe[], memo: Map<string, number>): number {
  if (memo.has(id)) return memo.get(id)!;
  const r = recipes.find(p => p.id === id);
  const childIds = (r?.sub_recipes ?? [])
    .map(sr => sr.child_id)
    .filter(cid => recipes.some(p => p.id === cid));
  if (childIds.length === 0) {
    memo.set(id, 0);
    return 0;
  }
  const col = Math.max(...childIds.map(cid => computeColumn(cid, recipes, memo))) + 1;
  memo.set(id, col);
  return col;
}

interface NodePos {
  id: string;
  x: number;
  y: number;
  recipe: Recipe;
}

function computeLayout(recipes: Recipe[]): NodePos[] {
  const isChild = new Set<string>();
  const isParent = new Set<string>();
  for (const r of recipes) {
    for (const sr of r.sub_recipes ?? []) {
      isChild.add(sr.child_id);
      isParent.add(r.id);
    }
  }

  const memo = new Map<string, number>();
  const connected = recipes.filter(r => isChild.has(r.id) || isParent.has(r.id));
  const isolated = recipes.filter(r => !isChild.has(r.id) && !isParent.has(r.id));

  const byCol = new Map<number, Recipe[]>();
  for (const r of connected) {
    const col = computeColumn(r.id, recipes, memo);
    if (!byCol.has(col)) byCol.set(col, []);
    byCol.get(col)!.push(r);
  }

  const nodes: NodePos[] = [];
  let maxY = 0;

  for (const [col, colRecipes] of Array.from(byCol.entries()).sort((a, b) => a[0] - b[0])) {
    const x = PAD + col * COL_GAP;
    colRecipes.forEach((r, i) => {
      const y = PAD + i * ROW_GAP;
      nodes.push({ id: r.id, x, y, recipe: r });
      maxY = Math.max(maxY, y + NODE_H);
    });
  }

  const isoY = maxY > 0 ? maxY + 80 : PAD;
  isolated.forEach((r, i) => {
    nodes.push({
      id: r.id,
      x: PAD + (i % 4) * COL_GAP,
      y: isoY + Math.floor(i / 4) * ROW_GAP,
      recipe: r,
    });
  });

  return nodes;
}

interface Edge { x1: number; y1: number; x2: number; y2: number; }

export default function RecipeGraph({ recipes, usedIn, onRecipeClick }: RecipeGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [transform, setTransform] = useState({ x: 40, y: 40, scale: 1 });
  const dragRef = useRef<{ startX: number; startY: number; tx: number; ty: number } | null>(null);
  const hasDraggedRef = useRef(false);

  const nodes = computeLayout(recipes);
  const nodeMap = new Map(nodes.map(n => [n.id, n]));

  const canvasW = nodes.reduce((m, n) => Math.max(m, n.x + NODE_W + PAD), 400);
  const canvasH = nodes.reduce((m, n) => Math.max(m, n.y + NODE_H + PAD), 400);

  const edges: Edge[] = recipes.flatMap(r =>
    (r.sub_recipes ?? []).flatMap(sr => {
      const parent = nodeMap.get(r.id);
      const child = nodeMap.get(sr.child_id);
      if (!parent || !child) return [];
      return [{ x1: child.x + NODE_W, y1: child.y + NODE_H / 2, x2: parent.x, y2: parent.y + NODE_H / 2 }];
    })
  );

  const handleWheel = useCallback((e: WheelEvent) => {
    e.preventDefault();
    if (e.ctrlKey) {
      // pinch-to-zoom or Ctrl+scroll → zoom around cursor
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;
      setTransform(prev => {
        const factor = Math.exp(-e.deltaY / 300);
        const scale = Math.min(3, Math.max(0.2, prev.scale * factor));
        return { scale, x: mx - (mx - prev.x) * (scale / prev.scale), y: my - (my - prev.y) * (scale / prev.scale) };
      });
    } else {
      // two-finger scroll → pan
      setTransform(prev => ({ ...prev, x: prev.x - e.deltaX, y: prev.y - e.deltaY }));
    }
  }, []);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    el.addEventListener('wheel', handleWheel, { passive: false });
    return () => el.removeEventListener('wheel', handleWheel);
  }, [handleWheel]);

  const onMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    hasDraggedRef.current = false;
    dragRef.current = { startX: e.clientX, startY: e.clientY, tx: transform.x, ty: transform.y };
  };

  const onMouseMove = (e: React.MouseEvent) => {
    const drag = dragRef.current;
    if (!drag) return;
    const dx = e.clientX - drag.startX;
    const dy = e.clientY - drag.startY;
    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) hasDraggedRef.current = true;
    setTransform(prev => ({ ...prev, x: drag.tx + dx, y: drag.ty + dy }));
  };

  const onMouseUp = () => { dragRef.current = null; };

  return (
    <div
      ref={containerRef}
      className="recipe-graph"
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={onMouseUp}
      onMouseLeave={onMouseUp}
    >
      <div
        className="recipe-graph__canvas"
        style={{ width: canvasW, height: canvasH, transform: `translate(${transform.x}px,${transform.y}px) scale(${transform.scale})` }}
      >
        <svg className="recipe-graph__svg" width={canvasW} height={canvasH}>
          <defs>
            <marker id="graph-arrow" markerWidth="6" markerHeight="6" refX="6" refY="3" orient="auto">
              <path d="M0,0 L6,3 L0,6 Z" fill="rgba(0,0,0,0.18)" />
            </marker>
          </defs>
          {edges.map((e, i) => {
            const mx = (e.x2 - e.x1) * 0.5;
            return (
              <path
                key={i}
                d={`M${e.x1},${e.y1} C${e.x1 + mx},${e.y1} ${e.x2 - mx},${e.y2} ${e.x2},${e.y2}`}
                fill="none"
                stroke="rgba(0,0,0,0.12)"
                strokeWidth={1.5}
                markerEnd="url(#graph-arrow)"
              />
            );
          })}
        </svg>

        {nodes.map(n => {
          const parentCount = usedIn[n.id]?.length ?? 0;
          return (
            <button
              key={n.id}
              type="button"
              className="recipe-graph__node"
              style={{ left: n.x, top: n.y, width: NODE_W, height: NODE_H }}
              onClick={() => { if (!hasDraggedRef.current) onRecipeClick(n.recipe); }}
            >
              {n.recipe.image_url ? (
                <img className="recipe-graph__node-img" src={n.recipe.image_url} alt="" />
              ) : (
                <div className="recipe-graph__node-img recipe-graph__node-img--placeholder" />
              )}
              <div className="recipe-graph__node-text">
                <span className="recipe-graph__node-title">{n.recipe.title}</span>
                {parentCount > 0 && (
                  <span className="recipe-graph__node-meta">In {parentCount} recipe{parentCount !== 1 ? 's' : ''}</span>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
