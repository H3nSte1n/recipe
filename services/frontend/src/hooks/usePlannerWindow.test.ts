import { describe, expect, it } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { usePlannerWindow } from './usePlannerWindow';
import { addDays, shiftWeek, weekStart } from '../utils/plannerDates';

const TODAY = '2026-10-23'; // a Friday; its week starts Monday 2026-10-19

describe('usePlannerWindow', () => {
  it('starts two weeks before today\'s week and covers seven whole weeks', () => {
    const { result } = renderHook(() => usePlannerWindow(TODAY));

    expect(result.current.from).toBe(shiftWeek(weekStart(TODAY), -2));
    expect(result.current.to).toBe(addDays(result.current.from, 7 * 7 - 1));
    expect(result.current.dates).toHaveLength(49);
    expect(result.current.dates[0]).toBe(result.current.from);
    expect(result.current.dates[48]).toBe(result.current.to);
    expect(result.current.dates).toContain(TODAY);
  });

  it('lists consecutive calendar days', () => {
    const { dates } = renderHook(() => usePlannerWindow(TODAY)).result.current;
    dates.slice(1).forEach((date, index) => expect(date).toBe(addDays(dates[index], 1)));
  });

  it('grows four weeks upward and keeps the end', () => {
    const { result } = renderHook(() => usePlannerWindow(TODAY));
    const { from, to } = result.current;

    act(() => result.current.extendTop());

    expect(result.current.from).toBe(shiftWeek(from, -4));
    expect(result.current.to).toBe(to);
    expect(result.current.dates).toHaveLength(77);
    expect(weekStart(result.current.from)).toBe(result.current.from);
  });

  it('grows four weeks downward and keeps the start', () => {
    const { result } = renderHook(() => usePlannerWindow(TODAY));
    const { from, to } = result.current;

    act(() => result.current.extendBottom());

    expect(result.current.from).toBe(from);
    expect(result.current.to).toBe(addDays(to, 28));
    expect(result.current.dates).toHaveLength(77);
  });

  it('only ever grows, however the window is extended', () => {
    const { result } = renderHook(() => usePlannerWindow(TODAY));
    act(() => { result.current.extendTop(); result.current.extendBottom(); result.current.extendTop(); });

    expect(result.current.dates).toHaveLength(49 + 12 * 7);
    expect(result.current.dates).toContain(TODAY);
  });

  it('keeps stable extend callbacks and dates between unrelated renders', () => {
    const { result, rerender } = renderHook(() => usePlannerWindow(TODAY));
    const { extendTop, extendBottom, dates } = result.current;

    rerender();

    expect(result.current.extendTop).toBe(extendTop);
    expect(result.current.extendBottom).toBe(extendBottom);
    expect(result.current.dates).toBe(dates);
  });
});
