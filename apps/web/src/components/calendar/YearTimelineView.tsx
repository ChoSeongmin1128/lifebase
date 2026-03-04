"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { getEventEndDateKey, getEventStartDateKey } from "@/lib/calendar/event-date";

interface EventData {
  id: string;
  start_time: string;
  end_time: string;
  timezone: string;
  is_all_day: boolean;
  title: string;
  color_id: string | null;
  calendar_id: string;
}

interface TimelineEvent extends EventData {
  startKey: string;
  endKey: string;
  durationDays: number;
}

interface YearTimelineViewProps {
  year: number;
  events: EventData[];
  holidaysByDate: Map<string, string[]>;
  selectedDateKey?: string | null;
  getEventColor: (
    colorId: string | null,
    calendar?: { id: string; color_id: string | null; google_account_id?: string | null }
  ) => string;
  calendars: { id: string; color_id: string | null; google_account_id?: string | null }[];
  onDateClick?: (date: Date, dateKey: string) => void;
}

const MONTHS = ["1월", "2월", "3월", "4월", "5월", "6월", "7월", "8월", "9월", "10월", "11월", "12월"];
const MIN_ROW_HEIGHT = 12;
const HEADER_FALLBACK = 36;
const MAX_VISIBLE_LANES = 3;

function toDateKey(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

function eachDateKey(startKey: string, endKey: string, callback: (dateKey: string) => void) {
  const cursor = new Date(`${startKey}T00:00:00`);
  const end = new Date(`${endKey}T00:00:00`);
  while (cursor <= end) {
    callback(toDateKey(cursor));
    cursor.setDate(cursor.getDate() + 1);
  }
}

function maxDateKey(a: string, b: string): string {
  return a >= b ? a : b;
}

function minDateKey(a: string, b: string): string {
  return a <= b ? a : b;
}

function calculateDurationDays(startKey: string, endKey: string): number {
  const start = new Date(`${startKey}T00:00:00`);
  const end = new Date(`${endKey}T00:00:00`);
  const diff = Math.floor((end.getTime() - start.getTime()) / 86400000) + 1;
  return Math.max(diff, 1);
}

function assignLanes(events: TimelineEvent[]): Map<string, number> {
  const sorted = [...events].sort((a, b) => {
    if (a.startKey !== b.startKey) return a.startKey < b.startKey ? -1 : 1;
    if (a.durationDays !== b.durationDays) return b.durationDays - a.durationDays;
    return (a.title || "").localeCompare(b.title || "");
  });

  const laneEndByIndex: string[] = [];
  const laneByEventID = new Map<string, number>();
  for (const event of sorted) {
    let laneIndex = 0;
    while (laneIndex < laneEndByIndex.length && laneEndByIndex[laneIndex] >= event.startKey) {
      laneIndex += 1;
    }
    laneEndByIndex[laneIndex] = event.endKey;
    laneByEventID.set(event.id, laneIndex);
  }
  return laneByEventID;
}

export function YearTimelineView({
  year,
  events,
  holidaysByDate,
  selectedDateKey,
  getEventColor,
  calendars,
  onDateClick,
}: YearTimelineViewProps) {
  const today = new Date();
  const todayKey = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
  const calMap = useMemo(() => new Map(calendars.map((calendar) => [calendar.id, calendar])), [calendars]);

  const containerRef = useRef<HTMLDivElement | null>(null);
  const [rowHeight, setRowHeight] = useState(20);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const recalculate = () => {
      const head = element.querySelector("thead");
      const headHeight = head instanceof HTMLElement ? head.getBoundingClientRect().height : HEADER_FALLBACK;
      const available = element.clientHeight - headHeight - 2;
      if (available <= 0) return;
      setRowHeight(Math.max(MIN_ROW_HEIGHT, available / 31));
    };

    recalculate();
    const observer = new ResizeObserver(recalculate);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const timelineEvents = useMemo(() => {
    return events
      .map((event) => {
        const startKey = getEventStartDateKey(event);
        const endKey = getEventEndDateKey(event);
        if (!startKey || !endKey) return null;
        return {
          ...event,
          startKey,
          endKey,
          durationDays: calculateDurationDays(startKey, endKey),
        } as TimelineEvent;
      })
      .filter((event): event is TimelineEvent => !!event);
  }, [events]);

  const laneByEventID = useMemo(() => assignLanes(timelineEvents), [timelineEvents]);

  const eventsByDate = useMemo(() => {
    const map = new Map<string, TimelineEvent[]>();
    const yearStartKey = `${year}-01-01`;
    const yearEndKey = `${year}-12-31`;

    for (const event of timelineEvents) {
      const rangeStart = maxDateKey(event.startKey, yearStartKey);
      const rangeEnd = minDateKey(event.endKey, yearEndKey);
      if (rangeStart > rangeEnd) continue;

      eachDateKey(rangeStart, rangeEnd, (dateKey) => {
        const list = map.get(dateKey) || [];
        list.push(event);
        map.set(dateKey, list);
      });
    }

    for (const [dateKey, items] of map.entries()) {
      items.sort((a, b) => {
        const laneA = laneByEventID.get(a.id) ?? 0;
        const laneB = laneByEventID.get(b.id) ?? 0;
        if (laneA !== laneB) return laneA - laneB;
        if (a.startKey !== b.startKey) return a.startKey < b.startKey ? -1 : 1;
        return (a.title || "").localeCompare(b.title || "");
      });
      map.set(dateKey, items);
    }
    return map;
  }, [laneByEventID, timelineEvents, year]);

  return (
    <div ref={containerRef} className="h-full min-h-0 overflow-auto">
      <table className="h-full w-full min-w-[980px] border-collapse text-[10px]" style={{ tableLayout: "fixed" }}>
        <thead className="sticky top-0 z-10 bg-background">
          <tr>
            <th className="sticky left-0 z-20 w-10 border-b border-r border-border bg-background px-1 py-1.5 text-center text-text-muted font-normal">
              일
            </th>
            {MONTHS.map((month) => (
              <th key={month} className="border-b border-r border-border px-1 py-1.5 text-center text-text-muted font-normal">
                {month}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: 31 }, (_, index) => {
            const day = index + 1;
            return (
              <tr key={day} style={{ height: `${rowHeight}px` }}>
                <td className="sticky left-0 z-10 border-b border-r border-border bg-background px-1 py-0.5 text-center text-text-secondary">
                  {day}
                </td>
                {Array.from({ length: 12 }, (_, monthIndex) => {
                  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
                  if (day > daysInMonth) {
                    return <td key={monthIndex} className="border-b border-r border-border/40 bg-surface-accent/30" />;
                  }

                  const dateKey = `${year}-${String(monthIndex + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
                  const dayEvents = eventsByDate.get(dateKey) || [];
                  const visibleEvents = dayEvents.filter((event) => (laneByEventID.get(event.id) ?? 0) < MAX_VISIBLE_LANES);
                  const hiddenCount = Math.max(0, dayEvents.length - visibleEvents.length);
                  const holidayLabel = holidaysByDate.get(dateKey)?.[0] || "";
                  const displayEvent = visibleEvents.find((event) => event.startKey === dateKey) || visibleEvents[0] || null;
                  const displayText = holidayLabel || displayEvent?.title || "";
                  const isToday = dateKey === todayKey;
                  const isWeekend = new Date(year, monthIndex, day).getDay() % 6 === 0;
                  const isSelected = selectedDateKey === dateKey;

                  return (
                    <td key={monthIndex} className="h-full border-b border-r border-border/30 p-0 align-top">
                      <button
                        type="button"
                        onClick={() => onDateClick?.(new Date(year, monthIndex, day), dateKey)}
                        className={cn(
                          "relative flex h-full w-full cursor-pointer items-center overflow-hidden px-1 pl-[14px] text-left transition-colors",
                          isWeekend && !isToday && "bg-surface-accent/25",
                          isToday && "bg-primary/10",
                          isSelected && "ring-1 ring-inset ring-primary/60"
                        )}
                      >
                        {visibleEvents.map((event) => {
                          const lane = laneByEventID.get(event.id) ?? 0;
                          const isSingle = event.startKey === dateKey && event.endKey === dateKey;
                          const isStart = event.startKey === dateKey;
                          const isEnd = event.endKey === dateKey;
                          return (
                            <span
                              key={event.id}
                              className={cn(
                                "pointer-events-none absolute w-[3px]",
                                isSingle && "top-[2px] bottom-[2px] rounded-[3px]",
                                !isSingle && isStart && "top-[1px] bottom-0 rounded-t-[3px]",
                                !isSingle && isEnd && "top-0 bottom-[1px] rounded-b-[3px]",
                                !isSingle && !isStart && !isEnd && "top-0 bottom-0"
                              )}
                              style={{
                                left: `${1 + lane * 4}px`,
                                backgroundColor: getEventColor(event.color_id, calMap.get(event.calendar_id)),
                              }}
                            />
                          );
                        })}
                        <span
                          className={cn(
                            "truncate text-[10px] leading-none",
                            holidayLabel ? "font-semibold text-error" : "text-text-secondary"
                          )}
                          title={displayText}
                        >
                          {displayText}
                        </span>
                        {hiddenCount > 0 ? (
                          <span className="ml-1 shrink-0 text-[8px] font-semibold leading-none text-primary">+{hiddenCount}</span>
                        ) : null}
                      </button>
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
