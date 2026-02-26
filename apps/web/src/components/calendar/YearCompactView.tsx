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

interface YearCompactViewProps {
  year: number;
  events: EventData[];
  onMonthClick: (month: number) => void;
}

const WEEKDAYS = ["일", "월", "화", "수", "목", "금", "토"];

export function YearCompactView({ year, events, onMonthClick }: YearCompactViewProps) {
  const today = new Date();

  const eventsByDate = new Map<string, number>();
  for (const e of events) {
    const start = e.start_time.split("T")[0];
    eventsByDate.set(start, (eventsByDate.get(start) || 0) + 1);
  }

  return (
    <div className="grid grid-cols-3 md:grid-cols-4 gap-4 p-4">
      {Array.from({ length: 12 }, (_, m) => {
        const firstDay = new Date(year, m, 1);
        const daysInMonth = new Date(year, m + 1, 0).getDate();
        const startOffset = firstDay.getDay();

        const weeks: (number | null)[][] = [];
        let week: (number | null)[] = [];
        for (let i = 0; i < startOffset; i++) week.push(null);
        for (let d = 1; d <= daysInMonth; d++) {
          week.push(d);
          if (week.length === 7) { weeks.push(week); week = []; }
        }
        if (week.length > 0) {
          while (week.length < 7) week.push(null);
          weeks.push(week);
        }

        return (
          <div
            key={m}
            className="cursor-pointer rounded-lg border border-border p-2 hover:bg-surface-accent/50 transition-colors"
            onClick={() => onMonthClick(m)}
          >
            <div className="mb-1 text-sm font-medium text-text-strong">
              {m + 1}월
            </div>
            <div className="grid grid-cols-7 gap-px text-[9px]">
              {WEEKDAYS.map((d) => (
                <div key={d} className="text-center text-text-muted">{d}</div>
              ))}
              {weeks.flat().map((day, i) => {
                if (day === null) return <div key={i} />;
                const dateStr = `${year}-${String(m + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
                const count = eventsByDate.get(dateStr) || 0;
                const isToday =
                  year === today.getFullYear() && m === today.getMonth() && day === today.getDate();

                return (
                  <div key={i} className="relative text-center">
                    <span
                      className={cn(
                        "inline-flex h-4 w-4 items-center justify-center rounded-full text-[9px]",
                        isToday && "bg-primary text-white font-medium"
                      )}
                    >
                      {day}
                    </span>
                    {count > 0 && (
                      <div className="absolute -bottom-0.5 left-1/2 -translate-x-1/2 flex gap-px">
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
  );
}
