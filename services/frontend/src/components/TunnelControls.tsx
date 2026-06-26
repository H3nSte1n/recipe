import { useState } from 'react';
import { TunnelParams, createDefaultTunnelParams } from '../types/tunnelParams';
import '../styles/TunnelControls.css';

interface TunnelControlsProps {
  paramsRef: { current: TunnelParams };
  blurTargetRef?: React.RefObject<HTMLElement>;
}

type NumericTunnelParam = { [K in keyof TunnelParams]: TunnelParams[K] extends number ? K : never }[keyof TunnelParams];

interface SliderConfig {
  label: string;
  field: NumericTunnelParam;
  min: number;
  max: number;
  step: number;
  description: string;
}

const TUNNEL_SLIDERS: SliderConfig[] = [
  { label: 'Speed',          field: 'speed',             min: 0.1,  max: 3.0,   step: 0.05,   description: 'Base outward speed of cards in px/frame at 60 fps' },
  { label: 'Portal radius',  field: 'portalRadius',      min: 50,   max: 450,   step: 10,     description: 'Radius of the center zone where cards are invisible' },
  { label: 'Fade band',      field: 'fadeBand',          min: 10,   max: 150,   step: 5,      description: 'Width of the fade-in zone as cards emerge from the portal' },
  { label: 'Parallax',       field: 'parallaxStrength',  min: 0,    max: 320,   step: 10,     description: 'Max vanishing-point shift in px when moving the mouse to the edge' },
  { label: 'Scroll power',   field: 'scrollSensitivity', min: 0,    max: 0.5,   step: 0.005,  description: 'How much a scroll or swipe boosts card speed' },
  { label: 'Scale distance', field: 'scaleDistance',     min: 100,  max: 1200,  step: 50,     description: 'Distance in px at which cards grow to their full size' },
  { label: 'Hover slow',     field: 'hoverSpeedMult',    min: 0.1,  max: 1.0,   step: 0.05,   description: 'Speed multiplier applied to a card while the cursor is over it' },
];

const FOCUS_SLIDERS: SliderConfig[] = [
  { label: 'Hover radius',   field: 'focusHoverRadius',  min: 50,   max: 500,   step: 10,     description: 'Distance from center where cards drift to and hover in focus mode' },
  { label: 'Transition',     field: 'focusLerpRate',     min: 0.005,max: 0.1,   step: 0.005,  description: 'How fast speed drops when entering focus mode — higher is snappier' },
  { label: 'Drift speed',    field: 'focusDriftSpeed',   min: 0,    max: 100,   step: 1,      description: 'Max px/frame cards move inward toward the hover radius in focus mode' },
  { label: 'Drift pull',     field: 'focusDriftPull',    min: 0,    max: 0.2,   step: 0.005,  description: 'Proportional pull strength — fraction of remaining distance covered per frame' },
];

const BLUR_SLIDERS: SliderConfig[] = [
  { label: 'W padding',      field: 'blurPaddingX',      min: 0,    max: 300,   step: 10,     description: 'How far the blur extends left and right beyond the center content' },
  { label: 'H padding',      field: 'blurPaddingY',      min: 0,    max: 200,   step: 10,     description: 'How far the blur extends above and below the center content' },
  { label: 'Softness',       field: 'blurAmount',        min: 0,    max: 150,   step: 5,      description: 'CSS blur radius — higher values create a softer, wider fade edge' },
];

const BLUR_FIELDS = new Set<NumericTunnelParam>(['blurPaddingX', 'blurPaddingY', 'blurAmount']);

function applyBlurVars(el: HTMLElement, p: TunnelParams) {
  el.style.setProperty('--blur-padding-x', `${p.blurPaddingX}px`);
  el.style.setProperty('--blur-padding-y', `${p.blurPaddingY}px`);
  el.style.setProperty('--blur-amount', `${p.blurAmount}px`);
}

function formatValue(field: NumericTunnelParam, value: number): string {
  if (field === 'scrollSensitivity') return value.toFixed(4);
  if (field === 'focusLerpRate' || field === 'focusDriftPull') return value.toFixed(3);
  if (field === 'focusDriftSpeed') return value.toFixed(1);
  if (field === 'portalRadius' || field === 'fadeBand' || field === 'parallaxStrength' ||
      field === 'scaleDistance' || field === 'focusHoverRadius' ||
      field === 'blurPaddingX' || field === 'blurPaddingY' || field === 'blurAmount') {
    return String(Math.round(value));
  }
  return value.toFixed(2);
}

type Tab = 'tunnel' | 'focus' | 'blur';

export default function TunnelControls({ paramsRef, blurTargetRef }: TunnelControlsProps) {
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<Tab>('tunnel');
  const [values, setValues] = useState<TunnelParams>(() => ({ ...paramsRef.current }));

  const sliders = tab === 'tunnel' ? TUNNEL_SLIDERS : tab === 'focus' ? FOCUS_SLIDERS : BLUR_SLIDERS;

  function handleChange(field: NumericTunnelParam, raw: string) {
    const newValue = parseFloat(raw);
    paramsRef.current[field] = newValue;
    setValues(prev => ({ ...prev, [field]: newValue }));
    if (blurTargetRef?.current && BLUR_FIELDS.has(field)) {
      applyBlurVars(blurTargetRef.current, paramsRef.current);
    }
  }

  function handleReset() {
    const defaults = createDefaultTunnelParams();
    Object.assign(paramsRef.current, defaults);
    setValues({ ...defaults });
    if (blurTargetRef?.current) {
      applyBlurVars(blurTargetRef.current, defaults);
    }
  }

  return (
    <>
      <button
        className="tunnel-controls__toggle"
        onClick={() => setOpen(prev => !prev)}
        aria-label="Toggle animation controls"
      >
        ⚙
      </button>

      <div className={`tunnel-controls__panel${open ? '' : ' tunnel-controls__panel--hidden'}`}>
        <div className="tunnel-controls__tabs">
          <button className={`tunnel-controls__tab${tab === 'tunnel' ? ' tunnel-controls__tab--active' : ''}`} onClick={() => setTab('tunnel')} type="button">Tunnel</button>
          <button className={`tunnel-controls__tab${tab === 'focus'  ? ' tunnel-controls__tab--active' : ''}`} onClick={() => setTab('focus')}  type="button">Focus</button>
          <button className={`tunnel-controls__tab${tab === 'blur'   ? ' tunnel-controls__tab--active' : ''}`} onClick={() => setTab('blur')}   type="button">Blur</button>
        </div>

        {sliders.map(({ label, field, min, max, step, description }) => (
          <div key={field} className="tunnel-controls__row">
            <div className="tunnel-controls__row-header">
              <span className="tunnel-controls__label">
                {label}
                <span className="tunnel-controls__tip" data-tip={description}>ℹ</span>
              </span>
              <span className="tunnel-controls__value">{formatValue(field, values[field])}</span>
            </div>
            <input
              type="range"
              className="tunnel-controls__slider"
              min={min}
              max={max}
              step={step}
              value={values[field]}
              onChange={e => handleChange(field, e.target.value)}
            />
          </div>
        ))}

        <button className="tunnel-controls__reset" onClick={handleReset}>
          Reset defaults
        </button>
      </div>
    </>
  );
}
