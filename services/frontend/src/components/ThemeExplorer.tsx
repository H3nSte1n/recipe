// TEMPORARY — color exploration tool for the landing page.
// To remove: delete this file, ThemeExplorer.css, and the import + <ThemeExplorer /> in LandingPage.tsx.
import { useState, useCallback } from 'react';
import '../styles/ThemeExplorer.css';

interface ColorSlot {
  label: string;
  cssVar: string;
  defaultHex: string;
}

const SLOTS: ColorSlot[] = [
  { label: 'Text & ink',       cssVar: '--ink',     defaultHex: '#0a0a0a' },
  { label: 'Surface & blur',   cssVar: '--surface', defaultHex: '#ffffff' },
  { label: 'Background tint',  cssVar: '--bg',      defaultHex: '#f8f8f8' },
  { label: 'Secondary text',   cssVar: '--meta',    defaultHex: '#808080' },
];

export default function ThemeExplorer() {
  const [open, setOpen] = useState(false);
  const [colors, setColors] = useState<Record<string, string>>(
    () => Object.fromEntries(SLOTS.map(s => [s.cssVar, s.defaultHex]))
  );

  const apply = useCallback((cssVar: string, hex: string) => {
    document.documentElement.style.setProperty(cssVar, hex);
    setColors(prev => ({ ...prev, [cssVar]: hex }));
  }, []);

  const reset = useCallback(() => {
    SLOTS.forEach(({ cssVar }) => document.documentElement.style.removeProperty(cssVar));
    setColors(Object.fromEntries(SLOTS.map(s => [s.cssVar, s.defaultHex])));
  }, []);

  return (
    <>
      <button
        className="theme-explorer__toggle"
        onClick={() => setOpen(p => !p)}
        aria-label="Toggle theme explorer"
        title="Theme explorer"
      >
        ◑
      </button>

      {open && (
        <div className="theme-explorer__panel">
          <div className="theme-explorer__header">
            <span className="theme-explorer__title">Theme</span>
            <button className="theme-explorer__close" onClick={() => setOpen(false)} aria-label="Close">×</button>
          </div>

          {SLOTS.map(({ label, cssVar }) => (
            <div key={cssVar} className="theme-explorer__row">
              <label className="theme-explorer__label">{label}</label>
              <div className="theme-explorer__color-wrap">
                <input
                  type="color"
                  className="theme-explorer__color-input"
                  value={colors[cssVar]}
                  onChange={e => apply(cssVar, e.target.value)}
                />
                <span className="theme-explorer__hex">{colors[cssVar].toUpperCase()}</span>
              </div>
            </div>
          ))}

          <button className="theme-explorer__reset" onClick={reset}>Reset defaults</button>
        </div>
      )}
    </>
  );
}
