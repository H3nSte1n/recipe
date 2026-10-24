import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';

// Node's experimental localStorage getter may shadow jsdom's storage object.
const values = new Map<string, string>();
Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => { values.set(key, value); },
    removeItem: (key: string) => { values.delete(key); },
    clear: () => values.clear(),
  },
});

// jsdom does not implement window scrolling.
window.scrollTo = vi.fn();
window.scrollBy = vi.fn();

afterEach(() => { cleanup(); values.clear(); });
