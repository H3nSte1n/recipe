export interface DaySection {
  date: string;
  bottom: number;
}

// The day the reader is looking at: the first section that still reaches below the sticky header.
export function topmostDate(sections: DaySection[], headerBottom: number): string | null {
  return sections.find((section) => section.bottom > headerBottom)?.date ?? null;
}
