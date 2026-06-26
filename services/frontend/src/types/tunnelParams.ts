export interface TunnelParams {
  speed: number;             // base px/frame at 60fps — default 0.8
  portalRadius: number;      // px from center where cards are invisible — default 220
  fadeBand: number;          // px band over which cards fade in — default 60
  parallaxStrength: number;  // max vanishing point offset in px — default 160
  scrollSensitivity: number; // deltaY multiplier for scroll velocity — default 0.003
  scaleDistance: number;     // distance (px) at which cards reach full size — default 600
  hoverSpeedMult: number;    // card speed when hovered — default 0.6
}

export function createDefaultTunnelParams(): TunnelParams {
  return {
    speed: 0.60,
    portalRadius: 200,
    fadeBand: 60,
    parallaxStrength: 30,
    scrollSensitivity: 0.012,
    scaleDistance: 750,
    hoverSpeedMult: 0.30,
  };
}
