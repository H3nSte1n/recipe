import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import PlannerList from './PlannerList';
import PlannerWeekStrip from './PlannerWeekStrip';
import type { PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { topmostDate } from '../utils/plannerScroll';

// How close (px) to either end of the list before more weeks are added.
const EDGE_DISTANCE = 600;

interface PlannerDayScrollerProps {
  dates: string[];
  items: PlannedMeal[];
  recipesById: Map<string, Recipe>;
  selectedDate: string;
  today: string;
  onSelectDate: (date: string) => void;
  onEdit: (item: PlannedMeal) => void;
  onExtendTop: () => void;
  onExtendBottom: () => void;
}

// Mobile planner: the week strip and Today button stay on screen while the day list scrolls beneath.
// The strip follows the day at the top of the list; tapping a day in the strip scrolls to it.
export default function PlannerDayScroller({ dates, items, recipesById, selectedDate, today, onSelectDate, onEdit, onExtendTop, onExtendBottom }: PlannerDayScrollerProps) {
  const rootRef = useRef<HTMLDivElement>(null);
  const stickyRef = useRef<HTMLDivElement>(null);
  const didInitialScroll = useRef(false);
  // The day at the top of the list and how far below the sticky header it sits, so the reader's
  // place can be restored when meals load or weeks are added above them.
  const anchor = useRef<{ date: string; offset: number } | null>(null);
  const extending = useRef(false);

  const sectionFor = useCallback((date: string) => rootRef.current?.querySelector<HTMLElement>(`[data-date="${date}"]`) ?? null, []);
  const headerBottom = useCallback(() => stickyRef.current?.getBoundingClientRect().bottom ?? 0, []);

  const scrollToDate = useCallback((date: string) => {
    const section = sectionFor(date);
    if (!section) return;
    window.scrollTo({ top: window.scrollY + section.getBoundingClientRect().top - headerBottom() });
    anchor.current = { date, offset: 0 };
  }, [sectionFor, headerBottom]);

  function jumpTo(date: string) {
    onSelectDate(date);
    scrollToDate(date);
  }

  // Open on the selected day (today).
  useLayoutEffect(() => {
    if (didInitialScroll.current) return;
    didInitialScroll.current = true;
    scrollToDate(selectedDate);
  }, [scrollToDate, selectedDate]);

  // Meals loading or weeks being added above the reader would push the page; put the anchor day back.
  useLayoutEffect(() => {
    const section = anchor.current && sectionFor(anchor.current.date);
    if (anchor.current && section) {
      const shift = section.getBoundingClientRect().top - headerBottom() - anchor.current.offset;
      if (Math.abs(shift) >= 1) window.scrollBy(0, shift);
    }
    extending.current = false;
  }, [dates, items, sectionFor, headerBottom]);

  useEffect(() => {
    let frame = 0;
    function update() {
      frame = 0;
      const root = rootRef.current;
      if (!root) return;
      const headerEdge = headerBottom();
      const sections = Array.from(root.querySelectorAll<HTMLElement>('[data-date]'));
      const date = topmostDate(sections.map((section) => ({ date: section.dataset.date ?? '', bottom: section.getBoundingClientRect().bottom })), headerEdge);
      if (date) {
        const top = sectionFor(date)?.getBoundingClientRect().top ?? headerEdge;
        anchor.current = { date, offset: top - headerEdge };
        if (date !== selectedDate) onSelectDate(date);
      }

      if (extending.current) return;
      const page = document.documentElement;
      if (window.scrollY < EDGE_DISTANCE) {
        extending.current = true;
        onExtendTop();
      } else if (page.scrollHeight - (window.scrollY + window.innerHeight) < EDGE_DISTANCE) {
        extending.current = true;
        onExtendBottom();
      }
    }
    function onScroll() {
      if (!frame) frame = window.requestAnimationFrame(update);
    }
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      window.removeEventListener('scroll', onScroll);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [selectedDate, onSelectDate, onExtendTop, onExtendBottom, sectionFor, headerBottom]);

  return (
    <div className="planner-scroller" ref={rootRef}>
      <div className="planner-scroller__sticky" ref={stickyRef}>
        <div className="planner-scroller__today">
          <button type="button" className="planner__pill-btn" onClick={() => jumpTo(today)}>Today</button>
        </div>
        <PlannerWeekStrip selectedDate={selectedDate} today={today} onSelect={jumpTo} />
      </div>
      <PlannerList dates={dates} items={items} recipesById={recipesById} onEdit={onEdit} onSelectDate={jumpTo} />
    </div>
  );
}
