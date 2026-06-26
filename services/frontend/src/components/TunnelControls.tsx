import { useState } from 'react';
import { TunnelParams, createDefaultTunnelParams } from '../types/tunnelParams';
import '../styles/TunnelControls.css';

interface TunnelControlsProps {
  paramsRef: { current: TunnelParams };
}

// Only numeric fields of TunnelParams can be driven by range sliders
type NumericTunnelParam = { [K in keyof TunnelParams]: TunnelParams[K] extends number ? K : never }[keyof TunnelParams];

interface SliderConfig {
  label: string;
  field: NumericTunnelParam;
  min: number;
  max: number;
  step: number;
}

const TUNNEL_SLIDERS: SliderConfig[] = [
  { label: 'Speed',         field: 'speed',            min: 0.1,  max: 3.0,   step: 0.05   },
  { label: 'Portal radius', field: 'portalRadius',     min: 50,   max: 450,   step: 10     },
  { label: 'Fade band',     field: 'fadeBand',         min: 10,   max: 150,   step: 5      },
  { label: 'Parallax',      field: 'parallaxStrength', min: 0,    max: 320,   step: 10     },
  { label: 'Scroll power',  field: 'scrollSensitivity',min: 0,    max: 0.012, step: 0.0005 },
  { label: 'Scale distance',field: 'scaleDistance',    min: 100,  max: 1200,  step: 50     },
  { label: 'Hover slow',    field: 'hoverSpeedMult',   min: 0.1,  max: 1.0,   step: 0.05   },
];

const FOCUS_SLIDERS: SliderConfig[] = [
  { label: 'Hover radius',  field: 'focusHoverRadius', min: 50,   max: 500,   step: 10     },
  { label: 'Transition',    field: 'focusLerpRate',    min: 0.005,max: 0.1,   step: 0.005  },
];

function formatValue(field: NumericTunnelParam, value: number): string {
  if (field === 'scrollSensitivity') return value.toFixed(4);
  if (field === 'focusLerpRate') return value.toFixed(3);
  if (field === 'portalRadius' || field === 'fadeBand' || field === 'parallaxStrength' || field === 'scaleDistance' || field === 'focusHoverRadius') {
    return String(Math.round(value));
  }
  return value.toFixed(2);
}

type Tab = 'tunnel' | 'focus';

export default function TunnelControls({ paramsRef }: TunnelControlsProps) {
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<Tab>('tunnel');
  const [values, setValues] = useState<TunnelParams>(() => ({ ...paramsRef.current }));

  const sliders = tab === 'tunnel' ? TUNNEL_SLIDERS : FOCUS_SLIDERS;

  function handleChange(field: NumericTunnelParam, raw: string) {
    const newValue = parseFloat(raw);
    paramsRef.current[field] = newValue;
    setValues(prev => ({ ...prev, [field]: newValue }));
  }

  function handleReset() {
    const defaults = createDefaultTunnelParams();
    Object.assign(paramsRef.current, defaults);
    setValues({ ...defaults });
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
          <button
            className={`tunnel-controls__tab${tab === 'tunnel' ? ' tunnel-controls__tab--active' : ''}`}
            onClick={() => setTab('tunnel')}
            type="button"
          >
            Tunnel
          </button>
          <button
            className={`tunnel-controls__tab${tab === 'focus' ? ' tunnel-controls__tab--active' : ''}`}
            onClick={() => setTab('focus')}
            type="button"
          >
            Focus
          </button>
        </div>

        {sliders.map(({ label, field, min, max, step }) => (
          <div key={field} className="tunnel-controls__row">
            <div className="tunnel-controls__row-header">
              <span className="tunnel-controls__label">{label}</span>
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
