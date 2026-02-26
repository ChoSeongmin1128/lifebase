"use client";

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
  getEventColor: (colorId: string | null, calColorId: string | null) => string;
  calendars: { id: string; color_id: string | null }[];
}

const MONTHS = ["1월", "2월", "3월", "4월", "5월", "6월", "7월", "8월", "9월", "10월", "11월", "12월"];

export function YearTimelineView({ year, events, getEventColor, calendars }: YearTimelineViewProps) {
  const today = new Date();
  const calMap = new Map(calendars.map((c) => [c.id, c]));

  const eventsByDate = new Map<string, EventData[]>();
  for (const e of events) {
    const dateStr = e.start_time.split("T")[0];
    if (!eventsByDate.has(dateStr)) eventsByDate.set(dateStr, []);
    eventsByDate.get(dateStr)!.push(e);
  }

  return (
    <div className="overflow-auto">
      <table className="w-full text-[10px] border-collapse" style={{ tableLayout: "fixed" }}>
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
              <tr key={m} className="border-b border-border/50">
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
                        "p-px border-r border-border/30 align-top h-6",
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
                                style={{ backgroundColor: getEventColor(e.color_id, cal?.color_id ?? null) }}
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
