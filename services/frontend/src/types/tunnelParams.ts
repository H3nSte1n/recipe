export interface TunnelParams {
  speed: number;             // base px/frame at 60fps — default 0.8
  portalRadius: number;      // px from center where cards are invisible — default 120
  fadeBand: number;          // px band over which cards fade in — default 60
  parallaxStrength: number;  // max vanishing point offset in px — default 160
  scrollSensitivity: number; // deltaY multiplier for scroll velocity — default 0.003
  scaleDistance: number;     // distance (px) at which cards reach full size — default 600
  hoverSpeedMult: number;    // card speed when hovered — default 0.6
  focusMode: boolean;        // when true, speed drops to 0 and cards drift to hover radius
  focusHoverRadius: number;  // px from center where cards hover in focus mode — default 200
  focusLerpRate: number;     // lerp rate for focus transition (higher = faster) — default 0.033
  focusDriftSpeed: number;   // max inward drift speed in px/frame — default 5.0
  blurPaddingX: number;      // px the blur extends left/right beyond center content — default 120
  blurPaddingY: number;      // px the blur extends above/below center content — default 80
  blurAmount: number;        // CSS blur() radius in px — default 70
}

export function createDefaultTunnelParams(): TunnelParams {
  return {
    speed: 0.60,
    portalRadius: 120,
    fadeBand: 60,
    parallaxStrength: 30,
    scrollSensitivity: 0.012,
    scaleDistance: 750,
    hoverSpeedMult: 0.30,
    focusMode: false,
    focusHoverRadius: 500,
    focusLerpRate: 0.085,
    focusDriftSpeed: 5.0,
    blurPaddingX: 120,
    blurPaddingY: 80,
    blurAmount: 70,
  };
}
