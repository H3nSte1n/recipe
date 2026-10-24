import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useMediaQuery } from './useMediaQuery';

interface FakeList {
  matches: boolean;
  media: string;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
}

function stubMatchMedia(matches: boolean) {
  const lists: FakeList[] = [];
  vi.stubGlobal('matchMedia', (query: string) => {
    const list: FakeList = { matches, media: query, addEventListener: vi.fn(), removeEventListener: vi.fn() };
    lists.push(list);
    return list;
  });
  return lists;
}

const fireChange = (list: FakeList, matches: boolean) => {
  list.matches = matches;
  list.addEventListener.mock.calls.filter(([type]) => type === 'change').forEach(([, listener]) => (listener as () => void)());
};

afterEach(() => vi.unstubAllGlobals());

describe('useMediaQuery', () => {
  it('reports false where matchMedia does not exist', () => {
    vi.stubGlobal('matchMedia', undefined);
    expect(renderHook(() => useMediaQuery('(min-width: 1024px)')).result.current).toBe(false);
  });

  it('starts with the current match', () => {
    stubMatchMedia(true);
    expect(renderHook(() => useMediaQuery('(min-width: 1024px)')).result.current).toBe(true);
  });

  it('follows change events', () => {
    const lists = stubMatchMedia(false);
    const { result } = renderHook(() => useMediaQuery('(min-width: 1024px)'));
    expect(result.current).toBe(false);

    act(() => fireChange(lists[lists.length - 1], true));
    expect(result.current).toBe(true);
    act(() => fireChange(lists[lists.length - 1], false));
    expect(result.current).toBe(false);
  });

  it('stops listening on unmount', () => {
    const lists = stubMatchMedia(false);
    const { unmount } = renderHook(() => useMediaQuery('(min-width: 1024px)'));
    const list = lists[lists.length - 1];
    const [, listener] = list.addEventListener.mock.calls[0];

    unmount();

    expect(list.removeEventListener).toHaveBeenCalledWith('change', listener);
  });

  it('subscribes again when the query changes', () => {
    const lists = stubMatchMedia(false);
    const { rerender } = renderHook(({ query }) => useMediaQuery(query), { initialProps: { query: '(min-width: 600px)' } });
    rerender({ query: '(min-width: 1024px)' });
    expect(lists.some((list) => list.media === '(min-width: 1024px)' && list.addEventListener.mock.calls.length > 0)).toBe(true);
  });
});
