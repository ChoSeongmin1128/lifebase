"use client";

import { cn } from "@/lib/utils";
import { buildFixedMonthGridWithWeekStart } from "@/lib/calendar/month-grid";

interface EventData {
  id: string;
  start_time: string;
  end_time: string;
  title: string;
  color_id: string | null;
  calendar_id: string;
}

interface YearCompactViewProps {
  year: number;
  events: EventData[];
  weekStartsOn: number;
  onMonthClick: (month: number) => void;
}

const WEEKDAY_LABELS = ["일", "월", "화", "수", "목", "금", "토"];

export function YearCompactView({ year, events, weekStartsOn, onMonthClick }: YearCompactViewProps) {
  const today = new Date();
  const todayKey = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
  const weekdays = Array.from({ length: 7 }, (_, index) => {
    const day = (weekStartsOn + index) % 7;
    return WEEKDAY_LABELS[day];
  });

  const eventsByDate = new Map<string, number>();
  for (const e of events) {
    const start = e.start_time.split("T")[0];
    eventsByDate.set(start, (eventsByDate.get(start) || 0) + 1);
  }

  return (
    <div className="h-full min-h-0 overflow-auto p-4">
      <div className="grid h-full min-h-full auto-rows-fr grid-cols-3 gap-4 md:grid-cols-4">
      {Array.from({ length: 12 }, (_, m) => {
        const monthDate = new Date(year, m, 1);
        const cells = buildFixedMonthGridWithWeekStart(monthDate, weekStartsOn);

        return (
          <div
            key={m}
            className="flex min-h-0 cursor-pointer flex-col rounded-lg border border-border p-2 transition-colors hover:bg-surface-accent/50"
            onClick={() => onMonthClick(m)}
          >
            <div className="mb-1 text-sm font-medium text-text-strong">
              {m + 1}월
            </div>
            <div className="grid min-h-0 flex-1 grid-cols-7 grid-rows-[auto_repeat(6,minmax(0,1fr))] gap-px text-[9px]">
              {weekdays.map((d) => (
                <div key={d} className="text-center text-text-muted">{d}</div>
              ))}
              {cells.map((cell, i) => {
                const day = cell.day;
                const dateStr = cell.dateKey;
                const count = eventsByDate.get(dateStr) || 0;
                const isToday = dateStr === todayKey;

                return (
                  <div
                    key={i}
                    className={cn(
                      "relative flex h-full items-center justify-center rounded-sm text-center",
                      !cell.inCurrentMonth && "bg-surface-accent/20"
                    )}
                  >
                    <span
                      className={cn(
                        "inline-flex h-4 w-4 items-center justify-center rounded-full text-[9px]",
                        !cell.inCurrentMonth && !isToday && "text-text-muted",
                        isToday && "bg-primary text-white font-medium"
                      )}
                    >
                      {day}
                    </span>
                    {count > 0 && (
                      <div
                        className={cn(
                          "absolute -bottom-0.5 left-1/2 flex -translate-x-1/2 gap-px",
                          !cell.inCurrentMonth && "opacity-60"
                        )}
                      >
                        {count <= 3 ? (
                          Array.from({ length: count }, (_, j) => (
                            <div key={j} className="h-1 w-1 rounded-full bg-primary" />
                          ))
                        ) : (
                          <span className="text-[7px] text-primary font-medium">+{count}</span>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
      </div>
    </div>
  );
}
