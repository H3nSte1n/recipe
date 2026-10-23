import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import PlannerDayScroller from './PlannerDayScroller';
import { addDays } from '../utils/plannerDates';

const TODAY = '2026-10-23';
const HEADER_BOTTOM = 100;
const DAY_HEIGHT = 100;

const days = (start: string, count: number) => Array.from({ length: count }, (_, index) => addDays(start, index));

// Lays the day sections out one under the other, DAY_HEIGHT apart. `firstTop` is where the section at
// DOM position `anchorIndex` sits, so moving the anchor index simulates content added above it.
let anchorIndex = 0;
let anchorTop = HEADER_BOTTOM;

function rect(top: number, height: number): DOMRect {
  return { top, bottom: top + height, left: 0, right: 0, width: 0, height, x: 0, y: top, toJSON: () => ({}) };
}

function mockLayout() {
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    if (this.classList.contains('planner-scroller__sticky')) return rect(0, HEADER_BOTTOM);
    const date = (this as HTMLElement).dataset?.date;
    if (!date) return rect(0, 0);
    const position = Array.from(document.querySelectorAll('[data-date]')).findIndex((section) => (section as HTMLElement).dataset.date === date);
    return rect(anchorTop + (position - anchorIndex) * DAY_HEIGHT, DAY_HEIGHT);
  });
}

function setPage(scrollY: number, scrollHeight: number) {
  vi.spyOn(window, 'scrollY', 'get').mockReturnValue(scrollY);
  vi.spyOn(document.documentElement, 'scrollHeight', 'get').mockReturnValue(scrollHeight);
}

function scroller(overrides: Partial<React.ComponentProps<typeof PlannerDayScroller>> = {}) {
  const props = {
    topbar: <div>bar</div>,
    dates: days('2026-10-05', 28),
    items: [],
    recipesById: new Map(),
    selectedDate: TODAY,
    today: TODAY,
    onSelectDate: vi.fn(),
    onEdit: vi.fn(),
    onExtendTop: vi.fn(),
    onExtendBottom: vi.fn(),
    ...overrides,
  };
  return props;
}

beforeEach(() => {
  anchorIndex = 0;
  anchorTop = HEADER_BOTTOM;
  mockLayout();
  setPage(2000, 9000); // far from both ends
});
afterEach(() => vi.restoreAllMocks());

describe('PlannerDayScroller', () => {
  it('scrolls to the selected day once when it opens, not on later renders', () => {
    anchorIndex = days('2026-10-05', 28).indexOf(TODAY);
    anchorTop = HEADER_BOTTOM + 300; // today starts 300px below the header
    const props = scroller();
    const { rerender } = render(<PlannerDayScroller {...props} />);

    expect(window.scrollTo).toHaveBeenCalledTimes(1);
    expect(window.scrollTo).toHaveBeenCalledWith({ top: window.scrollY + 300 });

    rerender(<PlannerDayScroller {...props} items={[]} />);
    expect(window.scrollTo).toHaveBeenCalledTimes(1);

    // The selection follows the reader while scrolling; that must not drag the page back.
    rerender(<PlannerDayScroller {...props} selectedDate="2026-10-12" />);
    expect(window.scrollTo).toHaveBeenCalledTimes(1);
  });

  it('selects the day at the top of the list while scrolling', async () => {
    const props = scroller();
    render(<PlannerDayScroller {...props} />);
    const target = '2026-10-12';
    anchorIndex = props.dates.indexOf(target); // target sits right under the header
    anchorTop = HEADER_BOTTOM;

    fireEvent.scroll(window);

    await waitFor(() => expect(props.onSelectDate).toHaveBeenCalledWith(target));
  });

  it('does not re-select the day that is already selected', async () => {
    const props = scroller();
    render(<PlannerDayScroller {...props} />);
    anchorIndex = props.dates.indexOf(TODAY);
    anchorTop = HEADER_BOTTOM;

    fireEvent.scroll(window);
    await new Promise((resolve) => requestAnimationFrame(() => resolve(null)));

    expect(props.onSelectDate).not.toHaveBeenCalled();
  });

  it('puts the reader back on their day when weeks are added above', async () => {
    const props = scroller();
    const { rerender } = render(<PlannerDayScroller {...props} />);
    const target = '2026-10-12';
    anchorIndex = props.dates.indexOf(target);
    anchorTop = HEADER_BOTTOM;
    fireEvent.scroll(window); // remembers `target` as the anchor, 0px under the header
    await waitFor(() => expect(props.onSelectDate).toHaveBeenCalledWith(target));
    vi.mocked(window.scrollBy).mockClear();

    // Four weeks arrive above: every section moves 28 positions down, so the target is 2800px lower.
    rerender(<PlannerDayScroller {...props} dates={days(addDays('2026-10-05', -28), 56)} />);

    await waitFor(() => expect(window.scrollBy).toHaveBeenCalledWith(0, 28 * DAY_HEIGHT));
  });

  it('leaves the scroll position alone when nothing moved', async () => {
    const props = scroller();
    const { rerender } = render(<PlannerDayScroller {...props} />);
    anchorIndex = props.dates.indexOf('2026-10-12');
    fireEvent.scroll(window);
    await waitFor(() => expect(props.onSelectDate).toHaveBeenCalled());
    vi.mocked(window.scrollBy).mockClear();

    rerender(<PlannerDayScroller {...props} items={[]} />);

    expect(window.scrollBy).not.toHaveBeenCalled();
  });

  it('asks for more weeks above near the top, but only once until the dates change', async () => {
    setPage(100, 9000);
    const props = scroller();
    const { rerender } = render(<PlannerDayScroller {...props} />);

    fireEvent.scroll(window);
    await waitFor(() => expect(props.onExtendTop).toHaveBeenCalledTimes(1));
    fireEvent.scroll(window);
    await new Promise((resolve) => requestAnimationFrame(() => resolve(null)));
    expect(props.onExtendTop).toHaveBeenCalledTimes(1);

    rerender(<PlannerDayScroller {...props} dates={days('2026-09-07', 56)} />);
    fireEvent.scroll(window);
    await waitFor(() => expect(props.onExtendTop).toHaveBeenCalledTimes(2));
    expect(props.onExtendBottom).not.toHaveBeenCalled();
  });

  it('asks for more weeks below near the bottom', async () => {
    setPage(5000, 5000 + window.innerHeight + 100);
    const props = scroller();
    render(<PlannerDayScroller {...props} />);

    fireEvent.scroll(window);

    await waitFor(() => expect(props.onExtendBottom).toHaveBeenCalledTimes(1));
    expect(props.onExtendTop).not.toHaveBeenCalled();
  });

  it('asks for nothing in the middle of the list', async () => {
    const props = scroller();
    render(<PlannerDayScroller {...props} />);

    fireEvent.scroll(window);
    await new Promise((resolve) => requestAnimationFrame(() => resolve(null)));

    expect(props.onExtendTop).not.toHaveBeenCalled();
    expect(props.onExtendBottom).not.toHaveBeenCalled();
  });

  it('jumps to today from the Today button', () => {
    const props = scroller({ selectedDate: '2026-10-12' });
    render(<PlannerDayScroller {...props} />);
    vi.mocked(window.scrollTo).mockClear();

    fireEvent.click(screen.getByRole('button', { name: 'Today' }));

    expect(props.onSelectDate).toHaveBeenCalledWith(TODAY);
    expect(window.scrollTo).toHaveBeenCalledTimes(1);
  });

  it('stops listening to the page scroll when it unmounts', async () => {
    setPage(100, 9000);
    const props = scroller();
    const { unmount } = render(<PlannerDayScroller {...props} />);
    unmount();

    fireEvent.scroll(window);
    await new Promise((resolve) => requestAnimationFrame(() => resolve(null)));

    expect(props.onExtendTop).not.toHaveBeenCalled();
  });
});
