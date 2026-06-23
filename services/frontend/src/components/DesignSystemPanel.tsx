import { useState } from 'react';
import addRecipeModalCss from '../styles/AddRecipeModal.css?raw';
import recipeModalCss from '../styles/RecipeModal.css?raw';
import recipeCardCss from '../styles/RecipeCard.css?raw';
import homePageCss from '../styles/HomePage.css?raw';
import loginPageCss from '../styles/LoginPage.css?raw';
import searchBarCss from '../styles/SearchBar.css?raw';
import recipeModalTsx from './RecipeModal.tsx?raw';
import addRecipeModalTsx from './AddRecipeModal.tsx?raw';
import recipeCardTsx from './RecipeCard.tsx?raw';
import homePageTsx from '../pages/HomePage.tsx?raw';
import '../styles/DesignSystemPanel.css';

const COLOR_TOKEN_NAMES = [
  '--bg', '--ink', '--accent', '--ph', '--ph-dark',
  '--meta', '--label', '--line', '--surface', '--error',
  '--status-draft-bg', '--status-draft-text',
  '--status-archived-bg', '--status-archived-text',
];

const FONT_TOKENS = ['--font-serif', '--font-sans'];

const TYPE_SCALE_NAMES = [
  '--text-xs', '--text-sm', '--text-base', '--text-lg',
  '--text-xl', '--text-2xl', '--text-display',
];

const TYPE_UTILITY_CLASSES = [
  'type-display', 'type-h1', 'type-h2', 'type-h3',
  'type-body', 'type-body-sm', 'type-caption', 'type-label',
];

function deriveTokenUsages(cssFiles: Record<string, string>): Record<string, string[]> {
  const usages: Record<string, string[]> = {};
  const re = /var\(--([\w-]+)\)/g;
  for (const [filename, css] of Object.entries(cssFiles)) {
    let m: RegExpExecArray | null;
    re.lastIndex = 0;
    while ((m = re.exec(css)) !== null) {
      const token = `--${m[1]}`;
      (usages[token] ??= []).push(filename);
    }
  }
  for (const k of Object.keys(usages)) {
    usages[k] = [...new Set(usages[k])];
  }
  return usages;
}

function deriveClassUsages(tsxFiles: Record<string, string>): Record<string, string[]> {
  const usages: Record<string, string[]> = {};
  const re = /className=["']([^"']+)["']/g;
  for (const [filename, src] of Object.entries(tsxFiles)) {
    let m: RegExpExecArray | null;
    re.lastIndex = 0;
    while ((m = re.exec(src)) !== null) {
      for (const cls of m[1].split(/\s+/)) {
        if (cls.startsWith('type-')) {
          (usages[cls] ??= []).push(filename);
        }
      }
    }
  }
  for (const k of Object.keys(usages)) {
    usages[k] = [...new Set(usages[k])];
  }
  return usages;
}

function DesignSystemPanel() {
  const [open, setOpen] = useState(false);

  const cssFiles: Record<string, string> = {
    'AddRecipeModal.css': addRecipeModalCss,
    'RecipeModal.css': recipeModalCss,
    'RecipeCard.css': recipeCardCss,
    'HomePage.css': homePageCss,
    'LoginPage.css': loginPageCss,
    'SearchBar.css': searchBarCss,
  };

  const tsxFiles: Record<string, string> = {
    'RecipeModal.tsx': recipeModalTsx,
    'AddRecipeModal.tsx': addRecipeModalTsx,
    'RecipeCard.tsx': recipeCardTsx,
    'HomePage.tsx': homePageTsx,
  };

  const tokenUsages = deriveTokenUsages(cssFiles);
  const classUsages = deriveClassUsages(tsxFiles);

  const computedStyle = getComputedStyle(document.documentElement);
  const val = (name: string) => computedStyle.getPropertyValue(name).trim();

  return (
    <>
      <button className="dsp-fab" onClick={() => setOpen(true)} aria-label="Open design system">
        <svg width={22} height={22} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
          <circle cx="13.5" cy="6.5" r="1.5" />
          <circle cx="17.5" cy="10.5" r="1.5" />
          <circle cx="8.5" cy="7.5" r="1.5" />
          <circle cx="6.5" cy="12.5" r="1.5" />
          <path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.926 0 1.648-.746 1.648-1.688 0-.437-.18-.835-.437-1.125-.29-.289-.438-.652-.438-1.125a1.64 1.64 0 0 1 1.668-1.668h1.996c3.051 0 5.555-2.503 5.555-5.554C21.965 6.012 17.461 2 12 2z" />
        </svg>
      </button>

      {open && (
        <div className="dsp-panel">
          <div className="dsp-header">
            <span className="dsp-title">Design System</span>
            <button className="dsp-close" onClick={() => setOpen(false)} aria-label="Close design system">✕</button>
          </div>

          <div className="dsp-body">

            {/* Color Tokens */}
            <section className="dsp-section">
              <h2 className="dsp-section-title">Color Tokens</h2>
              {COLOR_TOKEN_NAMES.map((name) => (
                <div key={name} className="dsp-token-row">
                  <span className="dsp-swatch" style={{ background: val(name) }} />
                  <code className="dsp-token-name">{name}</code>
                  <span className="dsp-token-value">{val(name)}</span>
                  <span className="dsp-token-usage">{(tokenUsages[name] ?? []).join(', ') || '—'}</span>
                </div>
              ))}
            </section>

            {/* Font Families */}
            <section className="dsp-section">
              <h2 className="dsp-section-title">Font Families</h2>
              {FONT_TOKENS.map((name) => (
                <div key={name} className="dsp-font-row">
                  <code className="dsp-token-name">{name}</code>
                  <span className="dsp-font-specimen" style={{ fontFamily: val(name) }}>
                    The quick brown fox
                  </span>
                  <span className="dsp-token-usage">{(tokenUsages[name] ?? []).join(', ') || '—'}</span>
                </div>
              ))}
            </section>

            {/* Type Scale */}
            <section className="dsp-section">
              <h2 className="dsp-section-title">Type Scale</h2>
              {TYPE_SCALE_NAMES.map((name) => (
                <div key={name} className="dsp-scale-row">
                  <code className="dsp-token-name">{name}</code>
                  <span className="dsp-token-value">{val(name)}</span>
                  <span style={{ fontSize: val(name), lineHeight: 1.2 }}>Recipe</span>
                  <span className="dsp-token-usage">{(tokenUsages[name] ?? []).join(', ') || '—'}</span>
                </div>
              ))}
            </section>

            {/* Type Utilities */}
            <section className="dsp-section">
              <h2 className="dsp-section-title">Type Utilities</h2>
              {TYPE_UTILITY_CLASSES.map((cls) => (
                <div key={cls} className="dsp-utility-row">
                  <code className="dsp-token-name">.{cls}</code>
                  <span className={cls}>The quick brown fox</span>
                  <span className="dsp-token-usage">{(classUsages[cls] ?? []).join(', ') || '—'}</span>
                </div>
              ))}
            </section>

            {/* Shadows */}
            <section className="dsp-section">
              <h2 className="dsp-section-title">Shadows</h2>
              <div className="dsp-shadow-row">
                <span className="dsp-shadow-box" style={{ boxShadow: '0 6px 20px -6px rgba(20,22,28,0.5)' }} />
                <code className="dsp-token-name">profile-btn shadow</code>
                <span className="dsp-token-usage">HomePage.css (.home-page__profile)</span>
              </div>
              <div className="dsp-shadow-row">
                <span className="dsp-shadow-box" style={{ boxShadow: '0 8px 24px rgba(0,0,0,0.08)' }} />
                <code className="dsp-token-name">dropdown shadow</code>
                <span className="dsp-token-usage">AddRecipeModal.css (.add-recipe-modal__section-dropdown)</span>
              </div>
            </section>

          </div>
        </div>
      )}
    </>
  );
}

export default DesignSystemPanel;
