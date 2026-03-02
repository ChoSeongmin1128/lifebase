"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";

interface EventData {
  id: string;
  start_time: string;
  end_time: string;
  title: string;
  color_id: string | null;
  calendar_id: string;
}

interface YearTimelineViewProps {
  year: number;
  events: EventData[];
  getEventColor: (
    colorId: string | null,
    calendar?: { id: string; color_id: string | null; google_account_id?: string | null }
  ) => string;
  calendars: { id: string; color_id: string | null; google_account_id?: string | null }[];
}

const MONTHS = ["1월", "2월", "3월", "4월", "5월", "6월", "7월", "8월", "9월", "10월", "11월", "12월"];
const HEADER_HEIGHT = 28;
const MIN_ROW_HEIGHT = 26;

export function YearTimelineView({ year, events, getEventColor, calendars }: YearTimelineViewProps) {
  const today = new Date();
  const calMap = new Map(calendars.map((c) => [c.id, c]));
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [rowHeight, setRowHeight] = useState(32);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const recalculate = () => {
      const availableHeight = element.clientHeight - HEADER_HEIGHT;
      if (availableHeight <= 0) {
        setRowHeight(32);
        return;
      }
      setRowHeight(Math.max(Math.floor(availableHeight / 12), MIN_ROW_HEIGHT));
    };

    recalculate();
    const observer = new ResizeObserver(recalculate);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const eventsByDate = new Map<string, EventData[]>();
  for (const e of events) {
    const dateStr = e.start_time.split("T")[0];
    if (!eventsByDate.has(dateStr)) eventsByDate.set(dateStr, []);
    eventsByDate.get(dateStr)!.push(e);
  }

  return (
    <div ref={containerRef} className="h-full min-h-0 overflow-auto">
      <table className="h-full w-full min-w-[860px] border-collapse text-[10px]" style={{ tableLayout: "fixed" }}>
        <thead className="sticky top-0 z-10 bg-background">
          <tr>
            <th className="w-12 border-b border-r border-border p-1 text-left text-text-muted font-normal sticky left-0 bg-background">월</th>
            {Array.from({ length: 31 }, (_, i) => (
              <th key={i} className="border-b border-border p-1 text-center text-text-muted font-normal min-w-[24px]">
                {i + 1}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {MONTHS.map((monthLabel, m) => {
            const daysInMonth = new Date(year, m + 1, 0).getDate();
            return (
              <tr key={m} className="border-b border-border/50" style={{ height: `${rowHeight}px` }}>
                <td className="border-r border-border p-1 text-text-secondary font-medium sticky left-0 bg-background">
                  {monthLabel}
                </td>
                {Array.from({ length: 31 }, (_, d) => {
                  if (d >= daysInMonth) return <td key={d} className="bg-surface-accent/30" />;
                  const dateStr = `${year}-${String(m + 1).padStart(2, "0")}-${String(d + 1).padStart(2, "0")}`;
                  const dayEvents = eventsByDate.get(dateStr) || [];
                  const isToday =
                    year === today.getFullYear() && m === today.getMonth() && d + 1 === today.getDate();
                  const isWeekend = new Date(year, m, d + 1).getDay() % 6 === 0;

                  return (
                    <td
                      key={d}
                      className={cn(
                        "h-full p-px align-top border-r border-border/30",
                        isToday && "bg-primary/10",
                        isWeekend && !isToday && "bg-surface-accent/30"
                      )}
                    >
                      {dayEvents.length > 0 && (
                        <div className="flex flex-col gap-px">
                          {dayEvents.slice(0, 2).map((e) => {
                            const cal = calMap.get(e.calendar_id);
                            return (
                              <div
                                key={e.id}
                                className="h-1.5 w-full rounded-sm"
                                style={{ backgroundColor: getEventColor(e.color_id, cal) }}
                                title={e.title}
                              />
                            );
                          })}
                          {dayEvents.length > 2 && (
                            <div className="text-[7px] text-text-muted text-center">+{dayEvents.length - 2}</div>
                          )}
                        </div>
                      )}
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
