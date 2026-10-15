import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import TimeSlotEventCard from './TimeSlotEventCard';
import type { Event } from '@/types/calendar';

const LONG_TITLE = 'Mudança Petrópolis/RJ → Osasco/SP (dia todo OFF)';

function makeEvent(startTime: string, endTime: string): Event {
  return {
    id: 'evt-1',
    calendarId: 'cal-1',
    title: LONG_TITLE,
    startTime,
    endTime,
    startDate: '2026-10-16T00:00:00.000Z',
    isRecurring: false,
    isActive: true,
  } as Event;
}

function renderCard(daysCount: number, startTime: string, endTime: string, totalColumns: number) {
  vi.spyOn(console, 'log').mockImplementation(() => {});
  render(
    <TimeSlotEventCard
      event={makeEvent(startTime, endTime)}
      date={new Date(2026, 9, 16)}
      daysCount={daysCount}
      positionInfo={{ column: 0, totalColumns }}
      onDragStart={() => {}}
      onDragEnd={() => {}}
      onResizeStart={() => {}}
      onToggleExecution={() => {}}
      onDeleteClick={() => {}}
    />,
  );
}

const classesOf = (el: Element) => (el.getAttribute('class') ?? '').split(/\s+/);

/**
 * A flex item defaults to `min-width: auto`, so it grows to fit its content and
 * `truncate` never kicks in: the text spills past the card. Every element on the
 * path from the title up to the card that sits in a row flex container must opt
 * out with `min-w-0`, and the title itself must truncate or wrap.
 */
function expectTitleContained() {
  const title = screen.getByText(LONG_TITLE);
  const titleClasses = classesOf(title);
  expect(
    titleClasses.some((c) => c === 'truncate' || c.startsWith('line-clamp') || c === 'break-words'),
  ).toBe(true);

  const card = title.closest('[data-event-id]')!;
  for (let el: Element = title; el !== card; el = el.parentElement!) {
    const parentClasses = classesOf(el.parentElement!);
    const parentIsRowFlex = parentClasses.includes('flex') && !parentClasses.includes('flex-col');
    if (parentIsRowFlex) {
      expect(classesOf(el), `<${el.tagName.toLowerCase()} class="${el.getAttribute('class')}">`).toContain('min-w-0');
    }
  }
}

describe('TimeSlotEventCard title overflow', () => {
  const cases: Array<[string, number, string, string, number]> = [
    ['3 days, long event, single column', 3, '08:00', '18:00', 1],
    ['3 days, long event, overlapping', 3, '08:00', '18:00', 2],
    ['3 days, 30-minute event', 3, '09:00', '09:30', 1],
    ['3 days, 15-minute event', 3, '09:00', '09:15', 1],
    ['week, long event', 7, '08:00', '18:00', 1],
    ['week, 30-minute event', 7, '09:00', '09:30', 1],
    ['week, long event, overlapping', 7, '08:00', '18:00', 2],
    ['day, long event, single column', 1, '08:00', '18:00', 1],
    ['day, long event, overlapping', 1, '08:00', '18:00', 2],
    ['day, 30-minute event, overlapping', 1, '09:00', '09:30', 2],
    ['day, 30-minute event, single column', 1, '09:00', '09:30', 1],
  ];

  it.each(cases)('keeps the title inside the card (%s)', (_label, days, start, end, columns) => {
    renderCard(days, start, end, columns);
    expectTitleContained();
  });
});
