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
  // Page height before weeks were added above the viewport, so the reader's place can be restored.
  const heightBeforeExtension = useRef<number | null>(null);
  const extending = useRef(false);

  const scrollToDate = useCallback((date: string) => {
    const section = rootRef.current?.querySelector(`[data-date="${date}"]`);
    if (!section) return;
    const headerHeight = stickyRef.current?.offsetHeight ?? 0;
    window.scrollTo({ top: window.scrollY + section.getBoundingClientRect().top - headerHeight });
  }, []);

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

  // After weeks are added, keep what the reader was looking at in place.
  useLayoutEffect(() => {
    if (heightBeforeExtension.current !== null) {
      window.scrollBy(0, document.documentElement.scrollHeight - heightBeforeExtension.current);
      heightBeforeExtension.current = null;
    }
    extending.current = false;
  }, [dates]);

  useEffect(() => {
    let frame = 0;
    function update() {
      frame = 0;
      const root = rootRef.current;
      if (!root) return;
      const headerBottom = stickyRef.current?.getBoundingClientRect().bottom ?? 0;
      const sections = Array.from(root.querySelectorAll<HTMLElement>('[data-date]'));
      const date = topmostDate(sections.map((section) => ({ date: section.dataset.date ?? '', bottom: section.getBoundingClientRect().bottom })), headerBottom);
      if (date && date !== selectedDate) onSelectDate(date);

      if (extending.current) return;
      const page = document.documentElement;
      if (window.scrollY < EDGE_DISTANCE) {
        extending.current = true;
        heightBeforeExtension.current = page.scrollHeight;
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
  }, [selectedDate, onSelectDate, onExtendTop, onExtendBottom]);

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
