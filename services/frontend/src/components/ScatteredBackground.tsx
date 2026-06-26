import { useEffect, useRef } from 'react';
import { type TunnelParams, createDefaultTunnelParams } from '../types/tunnelParams';
import '../styles/ScatteredBackground.css';

const FOOD_IMAGES = [
  'https://images.unsplash.com/photo-1504674900247-0877df9cc836?w=400&q=80',
  'https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=400&q=80',
  'https://images.unsplash.com/photo-1476224203421-9ac39bcb3df1?w=400&q=80',
  'https://images.unsplash.com/photo-1467003909585-2f8a72700288?w=400&q=80',
  'https://images.unsplash.com/photo-1490645935967-10de6ba17061?w=400&q=80',
  'https://images.unsplash.com/photo-1498837167922-ddd27525d352?w=400&q=80',
  'https://images.unsplash.com/photo-1546069901-ba9599a7e63c?w=400&q=80',
  'https://images.unsplash.com/photo-1414235077428-338989a2e8c0?w=400&q=80',
  'https://images.unsplash.com/photo-1565299624946-b28f40a0ae38?w=400&q=80',
  'https://images.unsplash.com/photo-1504754524776-8f4f37790ca0?w=400&q=80',
  'https://images.unsplash.com/photo-1484723091739-30a097e8f929?w=400&q=80',
  'https://images.unsplash.com/photo-1473093226795-af9932fe5856?w=400&q=80',
];

interface Card {
  id: number;
  x: number;        // current x position (px from center)
  y: number;        // current y position (px from center)
  angle: number;    // direction in radians
  distance: number; // how far from origin (grows each frame)
  size: number;     // base size in px (120–200, randomized at spawn)
  scale: number;    // current rendered scale (grows with distance)
  opacity: number;  // 0 at spawn, lerps to 1 as it crosses the portal edge
  imageIndex: number;
  hovered: boolean;
  speedMult: number; // per-card speed multiplier (1.0 default, 0.6 when hovered)
}

const CARD_COUNT = 9;

function makeCard(id: number): Card {
  return {
    id,
    x: 0,
    y: 0,
    angle: Math.random() * Math.PI * 2,
    distance: 0,
    size: 120 + Math.random() * 80,
    scale: 0.05,
    opacity: 0,
    imageIndex: id % FOOD_IMAGES.length,
    hovered: false,
    speedMult: 1.0,
  };
}

function recycleCard(card: Card): void {
  card.angle = Math.random() * Math.PI * 2;
  card.distance = 0;
  card.x = 0;
  card.y = 0;
  card.size = 120 + Math.random() * 80;
  card.scale = 0.05;
  card.opacity = 0;
  card.speedMult = 1.0;
  card.hovered = false;
  card.imageIndex = (card.imageIndex + 1) % FOOD_IMAGES.length;
}

function applyCardBackground(node: HTMLDivElement, imageIndex: number): void {
  // Clear the shorthand first so longhands are not wiped
  node.style.background = '';
  node.style.backgroundImage = `url(${FOOD_IMAGES[imageIndex]})`;
  node.style.backgroundSize = 'cover';
  node.style.backgroundPosition = 'center';
}

interface ScatteredBackgroundProps {
  paramsRef?: { current: TunnelParams };
}

export default function ScatteredBackground({ paramsRef }: ScatteredBackgroundProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const defaultParamsRef = useRef<TunnelParams>(createDefaultTunnelParams());
  const params = paramsRef ?? defaultParamsRef;
  const cardsRef = useRef<Card[]>([]);
  const nodeMapRef = useRef<Map<number, HTMLDivElement>>(new Map());

  const vwRef = useRef(window.innerWidth);
  const vhRef = useRef(window.innerHeight);

  // Vanishing point (lerped each frame toward target)
  const vpXRef = useRef(0);
  const vpYRef = useRef(0);
  const targetVpXRef = useRef(0);
  const targetVpYRef = useRef(0);

  // Scroll velocity and global speed multiplier
  const scrollVelocityRef = useRef(0);
  const globalScrollMultRef = useRef(1);
  const touchStartYRef = useRef(0);

  const rafIdRef = useRef<number | null>(null);
  const timeoutIdsRef = useRef<ReturnType<typeof setTimeout>[]>([]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // Preload all food images
    FOOD_IMAGES.forEach(url => {
      const img = new Image();
      img.src = url;
    });

    // Track viewport size
    const onResize = () => {
      vwRef.current = window.innerWidth;
      vhRef.current = window.innerHeight;
    };
    window.addEventListener('resize', onResize);

    // --- Create card DOM nodes ---
    const cards: Card[] = [];
    const nodeMap = new Map<number, HTMLDivElement>();

    for (let i = 0; i < CARD_COUNT; i++) {
      const card = makeCard(i);
      card.distance = -Infinity; // mark as not yet spawned

      const node = document.createElement('div');
      node.style.position = 'absolute';
      node.style.left = '50%';
      node.style.top = '50%';
      node.style.borderRadius = '10px';
      node.style.willChange = 'transform, opacity';
      node.style.width = `${card.size}px`;
      node.style.height = `${card.size}px`;
      // Start invisible — will be shown once spawned
      node.style.transform = 'translate(-50%, -50%) scale(0.05)';
      node.style.opacity = '0';
      applyCardBackground(node, card.imageIndex);

      container.appendChild(node);
      cards.push(card);
      nodeMap.set(i, node);
    }

    cardsRef.current = cards;
    nodeMapRef.current = nodeMap;

    // Stagger initial spawn: 800ms initial delay, then 150ms between each card
    for (let i = 0; i < CARD_COUNT; i++) {
      const tid = setTimeout(() => {
        const c = cards[i];
        // Reset to a proper spawned state
        c.distance = 0;
        c.angle = Math.random() * Math.PI * 2;
        c.size = 120 + Math.random() * 80;
        c.scale = 0.05;
        c.opacity = 0;
        c.speedMult = 1.0;
        // Resize DOM node and set image
        const n = nodeMap.get(i);
        if (n) {
          n.style.width = `${c.size}px`;
          n.style.height = `${c.size}px`;
          applyCardBackground(n, c.imageIndex);
        }
      }, 800 + i * 150);
      timeoutIdsRef.current.push(tid);
    }

    // --- Hover detection via container mousemove ---
    const onContainerMouseMove = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      const mx = e.clientX - rect.left - rect.width / 2;
      const my = e.clientY - rect.top - rect.height / 2;

      for (const card of cards) {
        const cx = vpXRef.current + card.x;
        const cy = vpYRef.current + card.y;
        const halfSize = (card.size * card.scale) / 2;
        card.hovered = (
          mx >= cx - halfSize && mx <= cx + halfSize &&
          my >= cy - halfSize && my <= cy + halfSize
        );
      }
    };
    container.addEventListener('mousemove', onContainerMouseMove);

    // --- rAF loop ---
    function tick() {
      // Lerp vanishing point
      vpXRef.current += (targetVpXRef.current - vpXRef.current) * 0.08;
      vpYRef.current += (targetVpYRef.current - vpYRef.current) * 0.08;

      // Decay scroll velocity and update global mult
      scrollVelocityRef.current *= 0.92;
      globalScrollMultRef.current = 1 + scrollVelocityRef.current;
      globalScrollMultRef.current = Math.max(0.05, Math.min(3.5, globalScrollMultRef.current));

      const vw = vwRef.current;
      const vh = vhRef.current;
      const vpX = vpXRef.current;
      const vpY = vpYRef.current;

      for (let i = 0; i < cards.length; i++) {
        const card = cards[i];
        const node = nodeMap.get(i);
        if (!node) continue;

        // Not yet spawned (distance === -Infinity)
        if (card.distance === -Infinity) continue;

        const effectiveSpeed = params.current.speed * globalScrollMultRef.current * card.speedMult;

        card.distance += effectiveSpeed;
        card.x = Math.cos(card.angle) * card.distance;
        card.y = Math.sin(card.angle) * card.distance;

        // Smooth scale with hover boost
        const targetSpeed = card.hovered ? params.current.hoverSpeedMult : 1.0;
        card.speedMult += (targetSpeed - card.speedMult) * 0.05;

        const targetScale = 0.05 + (card.distance / params.current.scaleDistance) * 0.95;
        const hoverBoost = card.hovered ? 1.08 : 1.0;
        card.scale += (targetScale * hoverBoost - card.scale) * 0.1;

        // Portal fade: cards fade in as they emerge from the blur radius
        if (card.distance < params.current.portalRadius) {
          card.opacity = 0;
        } else if (card.distance < params.current.portalRadius + params.current.fadeBand) {
          card.opacity = (card.distance - params.current.portalRadius) / params.current.fadeBand;
        } else {
          card.opacity = 1;
        }

        node.style.transform = `translate(calc(-50% + ${vpX + card.x}px), calc(-50% + ${vpY + card.y}px)) scale(${card.scale})`;
        node.style.opacity = String(card.opacity);

        // Off-screen check → recycle
        if (
          Math.abs(card.x) > vw / 2 + card.size ||
          Math.abs(card.y) > vh / 2 + card.size
        ) {
          recycleCard(card);
          // Update DOM node for new size and image
          node.style.width = `${card.size}px`;
          node.style.height = `${card.size}px`;
          node.style.opacity = '0';
          node.style.transform = `translate(calc(-50% + ${vpX}px), calc(-50% + ${vpY}px)) scale(0.05)`;
          applyCardBackground(node, card.imageIndex);
        }
      }

      rafIdRef.current = requestAnimationFrame(tick);
    }

    rafIdRef.current = requestAnimationFrame(tick);

    // --- Scroll hijacking ---
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      scrollVelocityRef.current += e.deltaY * params.current.scrollSensitivity;
      scrollVelocityRef.current = Math.max(-0.5, Math.min(2.5, scrollVelocityRef.current));
    };

    const onTouchStart = (e: TouchEvent) => {
      touchStartYRef.current = e.touches[0].clientY;
    };

    const onTouchMove = (e: TouchEvent) => {
      e.preventDefault();
      const dy = touchStartYRef.current - e.touches[0].clientY;
      scrollVelocityRef.current += dy * 0.004;
      scrollVelocityRef.current = Math.max(-0.5, Math.min(2.5, scrollVelocityRef.current));
      touchStartYRef.current = e.touches[0].clientY;
    };

    container.addEventListener('wheel', onWheel, { passive: false });
    container.addEventListener('touchstart', onTouchStart);
    container.addEventListener('touchmove', onTouchMove, { passive: false });

    // --- Vanishing point mouse tracking ---
    const onMouseMove = (e: MouseEvent) => {
      targetVpXRef.current = (e.clientX - window.innerWidth / 2) / window.innerWidth * params.current.parallaxStrength;
      targetVpYRef.current = (e.clientY - window.innerHeight / 2) / window.innerHeight * params.current.parallaxStrength;
    };

    window.addEventListener('mousemove', onMouseMove);

    // --- Cleanup ---
    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
        rafIdRef.current = null;
      }
      for (const tid of timeoutIdsRef.current) {
        clearTimeout(tid);
      }
      timeoutIdsRef.current = [];

      window.removeEventListener('resize', onResize);
      window.removeEventListener('mousemove', onMouseMove);
      container.removeEventListener('wheel', onWheel);
      container.removeEventListener('touchstart', onTouchStart);
      container.removeEventListener('touchmove', onTouchMove);
      container.removeEventListener('mousemove', onContainerMouseMove);

      // Remove card DOM nodes
      for (const node of nodeMap.values()) {
        if (node.parentNode === container) {
          container.removeChild(node);
        }
      }
    };
  }, [params]);

  return (
    <div
      ref={containerRef}
      className="scattered-bg"
      aria-hidden="true"
    >
      <div className="scattered-bg__blur" />
    </div>
  );
}
