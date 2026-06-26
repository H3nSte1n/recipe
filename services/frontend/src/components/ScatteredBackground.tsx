import { useEffect, useRef } from 'react';
import '../styles/ScatteredBackground.css';

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
const BASE_SPEED = 0.8;

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
    imageIndex: id % 9,
    hovered: false,
    speedMult: 1.0,
  };
}

function recycleCard(card: Card): void {
  card.angle = Math.random() * Math.PI * 2;
  card.distance = 0;
  card.x = 0;
  card.y = 0;
  card.scale = 0.05;
  card.opacity = 0;
  card.speedMult = 1.0;
  card.hovered = false;
}

export default function ScatteredBackground() {
  const containerRef = useRef<HTMLDivElement>(null);
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
      node.style.background = '#1e1e1e';
      node.style.willChange = 'transform, opacity';
      node.style.width = `${card.size}px`;
      node.style.height = `${card.size}px`;
      // Start invisible — will be shown once spawned
      node.style.transform = 'translate(-50%, -50%) scale(0.05)';
      node.style.opacity = '0';

      container.appendChild(node);
      cards.push(card);
      nodeMap.set(i, node);
    }

    cardsRef.current = cards;
    nodeMapRef.current = nodeMap;

    // Stagger initial spawn: spawn 1 card every 100ms
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
        // Resize DOM node in case size changed
        const n = nodeMap.get(i);
        if (n) {
          n.style.width = `${c.size}px`;
          n.style.height = `${c.size}px`;
        }
      }, i * 100);
      timeoutIdsRef.current.push(tid);
    }

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

        const effectiveSpeed = BASE_SPEED * globalScrollMultRef.current * card.speedMult;

        card.distance += effectiveSpeed;
        card.x = Math.cos(card.angle) * card.distance;
        card.y = Math.sin(card.angle) * card.distance;
        card.scale = 0.05 + (card.distance / 600) * 0.95;
        card.opacity = Math.min(1, card.distance / 120);

        node.style.transform = `translate(calc(-50% + ${vpX + card.x}px), calc(-50% + ${vpY + card.y}px)) scale(${card.scale})`;
        node.style.opacity = String(card.opacity);

        // Off-screen check → recycle
        if (
          Math.abs(card.x) > vw / 2 + card.size ||
          Math.abs(card.y) > vh / 2 + card.size
        ) {
          recycleCard(card);
          // Update DOM node width/height for new size
          node.style.width = `${card.size}px`;
          node.style.height = `${card.size}px`;
          node.style.opacity = '0';
          node.style.transform = `translate(calc(-50% + ${vpX}px), calc(-50% + ${vpY}px)) scale(0.05)`;
        }
      }

      rafIdRef.current = requestAnimationFrame(tick);
    }

    rafIdRef.current = requestAnimationFrame(tick);

    // --- Scroll hijacking ---
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      scrollVelocityRef.current += e.deltaY * 0.003;
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
      targetVpXRef.current = (e.clientX - window.innerWidth / 2) / window.innerWidth * 160;
      targetVpYRef.current = (e.clientY - window.innerHeight / 2) / window.innerHeight * 160;
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

      // Remove card DOM nodes
      for (const node of nodeMap.values()) {
        if (node.parentNode === container) {
          container.removeChild(node);
        }
      }
    };
  }, []);

  return (
    <div
      ref={containerRef}
      className="scattered-bg"
      aria-hidden="true"
    />
  );
}
