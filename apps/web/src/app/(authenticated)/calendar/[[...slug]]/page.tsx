"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { ViewDropdown, type CalendarViewMode } from "@/components/calendar/ViewDropdown";
import { YearCompactView } from "@/components/calendar/YearCompactView";
import { YearTimelineView } from "@/components/calendar/YearTimelineView";
import { QuickCreatePopover } from "@/components/calendar/QuickCreatePopover";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface CalendarData {
  id: string;
  name: string;
  color_id: string | null;
  is_primary: boolean;
  is_visible: boolean;
}

interface EventData {
  id: string;
  calendar_id: string;
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  timezone: string;
  is_all_day: boolean;
  color_id: string | null;
  recurrence_rule: string | null;
}

const COLORS = [
  "#4285f4", "#7986cb", "#33b679", "#8e24aa", "#e67c73",
  "#f6bf26", "#f4511e", "#039be5", "#616161", "#3f51b5",
  "#0b8043", "#d50000",
];

function getEventColor(colorId: string | null, calColorId: string | null): string {
  const id = colorId || calColorId;
  if (!id) return COLORS[0];
  const num = parseInt(id, 10);
  return COLORS[(num - 1) % COLORS.length] || COLORS[0];
}

function parseSlug(slug: string[] | undefined): { view: CalendarViewMode; date: Date } {
  if (!slug || slug.length === 0) return { view: "month", date: new Date() };

  const viewStr = slug[0];
  const dateStr = slug[1];
  let view: CalendarViewMode = "month";
  let date = new Date();

  const viewMap: Record<string, CalendarViewMode> = {
    "year-compact": "year-compact",
    "year-timeline": "year-timeline",
    month: "month",
    week: "week",
    "3day": "3day",
    agenda: "agenda",
  };
  if (viewMap[viewStr]) view = viewMap[viewStr];

  if (dateStr) {
    if (/^\d{4}$/.test(dateStr)) {
      date = new Date(parseInt(dateStr), 0, 1);
    } else if (/^\d{4}-\d{2}$/.test(dateStr)) {
      const [y, m] = dateStr.split("-").map(Number);
      date = new Date(y, m - 1, 1);
    } else if (/^\d{4}-W\d{2}$/.test(dateStr)) {
      const [y, w] = dateStr.split("-W").map(Number);
      date = getDateOfISOWeek(w, y);
    } else if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
      date = new Date(dateStr + "T00:00:00");
    }
  }

  return { view, date };
}

function getDateOfISOWeek(week: number, year: number): Date {
  const jan4 = new Date(year, 0, 4);
  const dayOfWeek = jan4.getDay() || 7;
  const firstMonday = new Date(jan4);
  firstMonday.setDate(jan4.getDate() - dayOfWeek + 1);
  const result = new Date(firstMonday);
  result.setDate(firstMonday.getDate() + (week - 1) * 7);
  return result;
}

function buildCalendarUrl(view: CalendarViewMode, date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");

  switch (view) {
    case "year-compact":
    case "year-timeline":
      return `/calendar/${view}/${y}`;
    case "month":
      return `/calendar/month/${y}-${m}`;
    case "week": {
      const jan4 = new Date(y, 0, 4);
      const dayOfYear = Math.floor((date.getTime() - new Date(y, 0, 1).getTime()) / 86400000) + 1;
      const weekNum = Math.ceil((dayOfYear + jan4.getDay()) / 7);
      return `/calendar/week/${y}-W${String(weekNum).padStart(2, "0")}`;
    }
    case "3day":
      return `/calendar/3day/${y}-${m}-${d}`;
    case "agenda":
      return "/calendar/agenda";
    default:
      return "/calendar";
  }
}

export default function CalendarPage() {
  const router = useRouter();
  const params = useParams();
  const slug = params.slug as string[] | undefined;
  const { view: initialView, date: initialDate } = parseSlug(slug);

  const [view, setView] = useState<CalendarViewMode>(initialView);
  const [currentDate, setCurrentDate] = useState(initialDate);
  const [calendars, setCalendars] = useState<CalendarData[]>([]);
  const [events, setEvents] = useState<EventData[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedEvent, setSelectedEvent] = useState<EventData | null>(null);

  const token = getAccessToken();

  const loadCalendars = useCallback(async () => {
    if (!token) return;
    try {
      const data = await api<{ calendars: CalendarData[] }>("/calendars", { token });
      setCalendars(data.calendars || []);
    } catch {
      setCalendars([]);
    }
  }, [token]);

  const loadEvents = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    const { start, end } = getDateRange(currentDate, view);
    try {
      const data = await api<{ events: EventData[] }>(
        `/events?start=${start}&end=${end}`,
        { token }
      );
      setEvents(data.events || []);
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [token, currentDate, view]);

  useEffect(() => { loadCalendars(); }, [loadCalendars]);
  useEffect(() => { loadEvents(); }, [loadEvents]);

  const updateUrl = useCallback((v: CalendarViewMode, d: Date) => {
    router.replace(buildCalendarUrl(v, d), { scroll: false });
  }, [router]);

  const handleViewChange = (newView: CalendarViewMode) => {
    setView(newView);
    updateUrl(newView, currentDate);
  };

  const navigate = (dir: number) => {
    const d = new Date(currentDate);
    if (view === "year-compact" || view === "year-timeline") d.setFullYear(d.getFullYear() + dir);
    else if (view === "month") d.setMonth(d.getMonth() + dir);
    else if (view === "week") d.setDate(d.getDate() + 7 * dir);
    else if (view === "3day") d.setDate(d.getDate() + 3 * dir);
    else d.setDate(d.getDate() + 7 * dir);
    setCurrentDate(d);
    updateUrl(view, d);
  };

  const goToday = () => {
    const d = new Date();
    setCurrentDate(d);
    updateUrl(view, d);
  };

  const handleCreateEvent = async (data: { title: string; start_time: string; end_time: string; calendar_id: string }) => {
    if (!token) return;
    try {
      await api("/events", {
        method: "POST",
        body: { ...data, is_all_day: false, timezone: "Asia/Seoul" },
        token,
      });
      loadEvents();
    } catch (err) {
      console.error("Create event failed:", err);
    }
  };

  const handleDeleteEvent = async (eventId: string) => {
    if (!token) return;
    try {
      await api(`/events/${eventId}`, { method: "DELETE", token });
      setSelectedEvent(null);
      loadEvents();
    } catch (err) {
      console.error("Delete event failed:", err);
    }
  };

  const headerLabel = getHeaderLabel(currentDate, view);
  const calMap = new Map(calendars.map((c) => [c.id, c]));

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 md:px-6 py-3">
        <div className="flex items-center gap-2 md:gap-3">
          <Button variant="secondary" size="sm" onClick={goToday}>오늘</Button>
          <Button variant="ghost" size="icon-sm" onClick={() => navigate(-1)}>
            <ChevronLeft size={16} />
          </Button>
          <Button variant="ghost" size="icon-sm" onClick={() => navigate(1)}>
            <ChevronRight size={16} />
          </Button>
          <h2 className="text-lg font-semibold text-text-strong">{headerLabel}</h2>
        </div>
        <div className="flex items-center gap-2">
          <ViewDropdown view={view} onChange={handleViewChange} />
          <QuickCreatePopover
            calendarId={calendars[0]?.id || ""}
            onSubmit={handleCreateEvent}
          />
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {loading && events.length === 0 ? (
          <div className="flex items-center justify-center py-20 text-text-muted">불러오는 중...</div>
        ) : view === "year-compact" ? (
          <YearCompactView
            year={currentDate.getFullYear()}
            events={events}
            onMonthClick={(m) => {
              const d = new Date(currentDate.getFullYear(), m, 1);
              setView("month");
              setCurrentDate(d);
              updateUrl("month", d);
            }}
          />
        ) : view === "year-timeline" ? (
          <YearTimelineView
            year={currentDate.getFullYear()}
            events={events}
            getEventColor={getEventColor}
            calendars={calendars}
          />
        ) : view === "month" ? (
          <MonthView
            currentDate={currentDate}
            events={events}
            calendars={calendars}
            onEventClick={setSelectedEvent}
          />
        ) : view === "week" || view === "3day" ? (
          <WeekView
            currentDate={currentDate}
            events={events}
            calendars={calendars}
            days={view === "week" ? 7 : 3}
            onEventClick={setSelectedEvent}
          />
        ) : (
          <AgendaView events={events} calendars={calendars} onEventClick={setSelectedEvent} />
        )}
      </div>

      {/* Event Detail Modal */}
      {selectedEvent && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
          onClick={() => setSelectedEvent(null)}
        >
          <div
            className="w-[calc(100vw-2rem)] max-w-80 rounded-lg border border-border bg-background p-4 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start gap-3">
              <div
                className="mt-1 h-3 w-3 shrink-0 rounded-full"
                style={{ backgroundColor: getEventColor(selectedEvent.color_id, calMap.get(selectedEvent.calendar_id)?.color_id ?? null) }}
              />
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-medium text-text-strong">{selectedEvent.title || "(제목 없음)"}</h3>
                <p className="mt-1 text-xs text-text-secondary">{formatEventTime(selectedEvent)}</p>
                {selectedEvent.location && (
                  <p className="mt-1 text-xs text-text-muted">{selectedEvent.location}</p>
                )}
                {selectedEvent.description && (
                  <p className="mt-2 text-xs text-text-secondary">{selectedEvent.description}</p>
                )}
              </div>
            </div>
            <div className="mt-3 flex justify-between">
              <Button variant="danger" size="sm" onClick={() => handleDeleteEvent(selectedEvent.id)}>삭제</Button>
              <Button variant="ghost" size="sm" onClick={() => setSelectedEvent(null)}>닫기</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ===== Month View ===== */
function MonthView({
  currentDate, events, calendars, onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  calendars: CalendarData[];
  onEventClick: (e: EventData) => void;
}) {
  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();
  const firstDay = new Date(year, month, 1);
  const startOffset = firstDay.getDay();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const today = new Date();

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

  const getEventsForDay = (day: number) => {
    const dateStr = `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
    return events.filter((e) => {
      const start = e.start_time.split("T")[0];
      const end = e.end_time.split("T")[0];
      return dateStr >= start && dateStr <= end;
    });
  };

  const calMap = new Map(calendars.map((c) => [c.id, c]));
  const weekdays = ["일", "월", "화", "수", "목", "금", "토"];

  return (
    <div className="flex h-full flex-col">
      <div className="grid grid-cols-7 border-b border-border">
        {weekdays.map((d, i) => (
          <div
            key={d}
            className={cn(
              "py-2 text-center text-xs font-medium",
              i === 0 ? "text-error" : i === 6 ? "text-info" : "text-text-muted"
            )}
          >
            {d}
          </div>
        ))}
      </div>
      <div className="grid flex-1 grid-cols-7" style={{ gridTemplateRows: `repeat(${weeks.length}, 1fr)` }}>
        {weeks.map((weekDays, wi) =>
          weekDays.map((day, di) => {
            const isToday = day !== null && year === today.getFullYear() && month === today.getMonth() && day === today.getDate();
            const dayEvents = day ? getEventsForDay(day) : [];
            const idx = wi * 7 + di;

            return (
              <div
                key={idx}
                className={cn(
                  "min-h-[60px] md:min-h-[80px] border-b border-r border-border/50 p-1",
                  day === null && "bg-surface-accent/30"
                )}
              >
                {day !== null && (
                  <>
                    <div
                      className={cn(
                        "mb-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full text-xs",
                        isToday ? "bg-primary text-white font-medium" : di === 0 ? "text-error" : di === 6 ? "text-info" : ""
                      )}
                    >
                      {day}
                    </div>
                    <div className="space-y-0.5">
                      {dayEvents.slice(0, 3).map((e, ei) => {
                        const cal = calMap.get(e.calendar_id);
                        return (
                          <div
                            key={e.id}
                            onClick={() => onEventClick(e)}
                            className={cn(
                              "cursor-pointer truncate rounded px-1 py-0.5 text-[10px] leading-tight text-white",
                              ei === 2 && "hidden md:block"
                            )}
                            style={{ backgroundColor: getEventColor(e.color_id, cal?.color_id ?? null) }}
                          >
                            {e.title || "(제목 없음)"}
                          </div>
                        );
                      })}
                      {dayEvents.length > 2 && (
                        <div className="text-[10px] text-text-muted px-1">
                          <span className="md:hidden">+{dayEvents.length - 2}</span>
                          <span className="hidden md:inline">{dayEvents.length > 3 ? `+${dayEvents.length - 3}` : ""}</span>
                        </div>
                      )}
                    </div>
                  </>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

/* ===== Week View ===== */
function WeekView({
  currentDate, events, calendars, days, onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  calendars: CalendarData[];
  days: number;
  onEventClick: (e: EventData) => void;
}) {
  const startDate = new Date(currentDate);
  if (days === 7) startDate.setDate(startDate.getDate() - startDate.getDay());

  const dayDates = Array.from({ length: days }, (_, i) => {
    const d = new Date(startDate);
    d.setDate(d.getDate() + i);
    return d;
  });

  const today = new Date();
  const hours = Array.from({ length: 16 }, (_, i) => i + 7);
  const calMap = new Map(calendars.map((c) => [c.id, c]));

  const getEventsForDay = (date: Date) => {
    const dateStr = date.toISOString().split("T")[0];
    return events.filter((e) => {
      const start = e.start_time.split("T")[0];
      const end = e.end_time.split("T")[0];
      return dateStr >= start && dateStr <= end;
    });
  };

  const weekdays = ["일", "월", "화", "수", "목", "금", "토"];

  return (
    <div className="flex h-full flex-col">
      <div className="flex border-b border-border">
        <div className="w-10 md:w-14 shrink-0" />
        {dayDates.map((d, i) => {
          const isToday = d.toDateString() === today.toDateString();
          return (
            <div
              key={i}
              className={cn(
                "flex-1 border-l border-border/50 py-2 text-center text-xs",
                isToday ? "font-medium text-text-strong" : "text-text-secondary"
              )}
            >
              <span className={isToday ? "inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-primary text-white px-1" : ""}>
                {weekdays[d.getDay()]} {d.getDate()}
              </span>
            </div>
          );
        })}
      </div>
      <div className="flex flex-1 min-h-0 overflow-auto">
        <div className="w-10 md:w-14 shrink-0">
          {hours.map((h) => (
            <div key={h} className="relative h-12">
              <span className="absolute -top-2 right-2 text-[10px] text-text-muted">
                {String(h).padStart(2, "0")}:00
              </span>
            </div>
          ))}
        </div>
        {dayDates.map((d, di) => {
          const dayEvents = getEventsForDay(d).filter((e) => !e.is_all_day);
          // Simple overlap grouping
          const positioned = positionEvents(dayEvents);

          return (
            <div key={di} className="relative flex-1 border-l border-border/50">
              {hours.map((h) => (
                <div key={h} className="h-12 border-b border-border/30" />
              ))}
              {positioned.map(({ event, column, totalColumns }) => {
                const startHour = new Date(event.start_time).getHours() + new Date(event.start_time).getMinutes() / 60;
                const endHour = new Date(event.end_time).getHours() + new Date(event.end_time).getMinutes() / 60;
                const top = (startHour - 7) * 48;
                const height = Math.max((endHour - startHour) * 48, 20);
                const cal = calMap.get(event.calendar_id);
                const width = `calc(${100 / totalColumns}% - 2px)`;
                const left = `calc(${(column / totalColumns) * 100}% + 1px)`;

                return (
                  <div
                    key={event.id}
                    className="absolute cursor-pointer overflow-hidden rounded px-1 py-0.5 text-[10px] text-white"
                    style={{
                      top: `${top}px`,
                      height: `${height}px`,
                      width,
                      left,
                      backgroundColor: getEventColor(event.color_id, cal?.color_id ?? null),
                    }}
                    onClick={() => onEventClick(event)}
                  >
                    <div className="font-medium truncate">{event.title || "(제목 없음)"}</div>
                    {height > 30 && (
                      <div className="truncate opacity-80">
                        {new Date(event.start_time).toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}
                        {" - "}
                        {new Date(event.end_time).toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function positionEvents(events: EventData[]): { event: EventData; column: number; totalColumns: number }[] {
  if (events.length === 0) return [];

  const sorted = [...events].sort((a, b) => a.start_time.localeCompare(b.start_time));
  const groups: EventData[][] = [];
  let currentGroup: EventData[] = [];
  let groupEnd = "";

  for (const e of sorted) {
    if (currentGroup.length === 0 || e.start_time < groupEnd) {
      currentGroup.push(e);
      if (e.end_time > groupEnd) groupEnd = e.end_time;
    } else {
      groups.push(currentGroup);
      currentGroup = [e];
      groupEnd = e.end_time;
    }
  }
  if (currentGroup.length > 0) groups.push(currentGroup);

  const result: { event: EventData; column: number; totalColumns: number }[] = [];
  for (const group of groups) {
    const columns: string[][] = [];
    for (const e of group) {
      let placed = false;
      for (let c = 0; c < columns.length; c++) {
        const lastEnd = columns[c][columns[c].length - 1];
        if (e.start_time >= lastEnd) {
          columns[c].push(e.end_time);
          result.push({ event: e, column: c, totalColumns: 0 });
          placed = true;
          break;
        }
      }
      if (!placed) {
        columns.push([e.end_time]);
        result.push({ event: e, column: columns.length - 1, totalColumns: 0 });
      }
    }
    const totalCols = columns.length;
    for (const r of result) {
      if (group.includes(r.event)) r.totalColumns = totalCols;
    }
  }
  return result;
}

/* ===== Agenda View ===== */
function AgendaView({
  events, calendars, onEventClick,
}: {
  events: EventData[];
  calendars: CalendarData[];
  onEventClick: (e: EventData) => void;
}) {
  const calMap = new Map(calendars.map((c) => [c.id, c]));
  const grouped = new Map<string, EventData[]>();
  for (const e of events) {
    const dateStr = e.start_time.split("T")[0];
    if (!grouped.has(dateStr)) grouped.set(dateStr, []);
    grouped.get(dateStr)!.push(e);
  }
  const sortedDates = Array.from(grouped.keys()).sort();

  if (sortedDates.length === 0) {
    return <div className="flex items-center justify-center py-20 text-text-muted">이 기간에 일정이 없습니다</div>;
  }

  return (
    <div className="divide-y divide-border/50 p-4">
      {sortedDates.map((date) => (
        <div key={date} className="py-3">
          <h3 className="mb-2 text-sm font-medium text-text-secondary">
            {new Date(date + "T00:00:00").toLocaleDateString("ko-KR", {
              year: "numeric", month: "long", day: "numeric", weekday: "short",
            })}
          </h3>
          <div className="space-y-1">
            {grouped.get(date)!.map((e) => {
              const cal = calMap.get(e.calendar_id);
              return (
                <div
                  key={e.id}
                  className="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 hover:bg-surface-accent/50"
                  onClick={() => onEventClick(e)}
                >
                  <div
                    className="h-3 w-3 shrink-0 rounded-full"
                    style={{ backgroundColor: getEventColor(e.color_id, cal?.color_id ?? null) }}
                  />
                  <div className="min-w-0 flex-1">
                    <span className="text-sm text-text-primary">{e.title || "(제목 없음)"}</span>
                  </div>
                  <span className="shrink-0 text-xs text-text-muted">
                    {e.is_all_day
                      ? "종일"
                      : `${new Date(e.start_time).toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })} - ${new Date(e.end_time).toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}`}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

/* ===== Helpers ===== */
function getDateRange(date: Date, view: CalendarViewMode): { start: string; end: string } {
  const d = new Date(date);
  let start: Date, end: Date;

  if (view === "year-compact" || view === "year-timeline") {
    start = new Date(d.getFullYear(), 0, 1);
    end = new Date(d.getFullYear(), 11, 31, 23, 59, 59);
  } else if (view === "month") {
    start = new Date(d.getFullYear(), d.getMonth(), 1);
    end = new Date(d.getFullYear(), d.getMonth() + 1, 0, 23, 59, 59);
  } else if (view === "week") {
    start = new Date(d);
    start.setDate(d.getDate() - d.getDay());
    end = new Date(start);
    end.setDate(start.getDate() + 6);
    end.setHours(23, 59, 59);
  } else if (view === "3day") {
    start = new Date(d);
    end = new Date(d);
    end.setDate(d.getDate() + 2);
    end.setHours(23, 59, 59);
  } else {
    start = new Date(d);
    end = new Date(d);
    end.setDate(d.getDate() + 30);
  }

  return { start: start.toISOString(), end: end.toISOString() };
}

function getHeaderLabel(date: Date, view: CalendarViewMode): string {
  if (view === "year-compact" || view === "year-timeline") return `${date.getFullYear()}년`;
  if (view === "month") return `${date.getFullYear()}년 ${date.getMonth() + 1}월`;
  if (view === "week" || view === "3day")
    return `${date.getFullYear()}년 ${date.getMonth() + 1}월 ${date.getDate()}일`;
  return `${date.getFullYear()}년 ${date.getMonth() + 1}월`;
}

function formatEventTime(e: EventData): string {
  if (e.is_all_day) return "종일";
  const start = new Date(e.start_time);
  const end = new Date(e.end_time);
  const opts: Intl.DateTimeFormatOptions = { month: "long", day: "numeric", hour: "2-digit", minute: "2-digit" };
  return `${start.toLocaleDateString("ko-KR", opts)} - ${end.toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}`;
}
