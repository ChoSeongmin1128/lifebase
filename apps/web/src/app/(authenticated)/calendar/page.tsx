"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";

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

type ViewMode = "month" | "week" | "3day" | "agenda";

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

export default function CalendarPage() {
  const [view, setView] = useState<ViewMode>("month");
  const [currentDate, setCurrentDate] = useState(new Date());
  const [calendars, setCalendars] = useState<CalendarData[]>([]);
  const [events, setEvents] = useState<EventData[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedEvent, setSelectedEvent] = useState<EventData | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState({
    title: "",
    start_time: "",
    end_time: "",
    calendar_id: "",
    is_all_day: false,
  });

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

  useEffect(() => {
    loadCalendars();
  }, [loadCalendars]);

  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  useEffect(() => {
    if (calendars.length > 0 && !createForm.calendar_id) {
      setCreateForm((f) => ({ ...f, calendar_id: calendars[0].id }));
    }
  }, [calendars, createForm.calendar_id]);

  const navigate = (dir: number) => {
    const d = new Date(currentDate);
    if (view === "month") d.setMonth(d.getMonth() + dir);
    else if (view === "week") d.setDate(d.getDate() + 7 * dir);
    else if (view === "3day") d.setDate(d.getDate() + 3 * dir);
    else d.setDate(d.getDate() + 7 * dir);
    setCurrentDate(d);
  };

  const goToday = () => setCurrentDate(new Date());

  const handleCreateEvent = async () => {
    if (!token || !createForm.title || !createForm.start_time || !createForm.end_time) return;
    try {
      await api("/events", {
        method: "POST",
        body: {
          calendar_id: createForm.calendar_id,
          title: createForm.title,
          start_time: createForm.start_time,
          end_time: createForm.end_time,
          is_all_day: createForm.is_all_day,
          timezone: "Asia/Seoul",
        },
        token,
      });
      setShowCreateModal(false);
      setCreateForm({ title: "", start_time: "", end_time: "", calendar_id: calendars[0]?.id || "", is_all_day: false });
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

  const viewOptions: { value: ViewMode; label: string }[] = [
    { value: "month", label: "월" },
    { value: "week", label: "주" },
    { value: "3day", label: "3일" },
    { value: "agenda", label: "일정" },
  ];

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-foreground/10 px-4 md:px-6 py-3">
        <div className="flex items-center gap-2 md:gap-3">
          <button
            onClick={goToday}
            className="rounded-full border border-foreground/10 px-3 py-1 text-sm hover:bg-foreground/5"
          >
            오늘
          </button>
          <button onClick={() => navigate(-1)} className="rounded p-1 hover:bg-foreground/5">
            <ChevronLeft />
          </button>
          <button onClick={() => navigate(1)} className="rounded p-1 hover:bg-foreground/5">
            <ChevronRight />
          </button>
          <h2 className="text-lg font-semibold">{headerLabel}</h2>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex rounded-md border border-foreground/10">
            {viewOptions.map((v) => (
              <button
                key={v.value}
                onClick={() => setView(v.value)}
                className={`px-3 py-1 text-xs ${
                  view === v.value
                    ? "bg-foreground/10 font-medium"
                    : "hover:bg-foreground/5 text-foreground/60"
                }`}
              >
                {v.label}
              </button>
            ))}
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex h-8 items-center gap-1 rounded-md bg-foreground px-3 text-sm text-background hover:opacity-90"
          >
            + 일정
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {loading && events.length === 0 ? (
          <div className="flex items-center justify-center py-20 text-foreground/40">
            불러오는 중...
          </div>
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
          <AgendaView
            events={events}
            calendars={calendars}
            onEventClick={setSelectedEvent}
          />
        )}
      </div>

      {/* Event Detail Popover */}
      {selectedEvent && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
          onClick={() => setSelectedEvent(null)}
        >
          <div
            className="w-[calc(100vw-2rem)] max-w-80 md:w-80 rounded-lg border border-foreground/10 bg-background p-4 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-base font-medium">{selectedEvent.title || "(제목 없음)"}</h3>
            <p className="mt-1 text-sm text-foreground/60">
              {formatEventTime(selectedEvent)}
            </p>
            {selectedEvent.location && (
              <p className="mt-1 text-sm text-foreground/50">{selectedEvent.location}</p>
            )}
            {selectedEvent.description && (
              <p className="mt-2 text-sm text-foreground/70">{selectedEvent.description}</p>
            )}
            <div className="mt-4 flex gap-2">
              <button
                onClick={() => handleDeleteEvent(selectedEvent.id)}
                className="rounded px-3 py-1.5 text-sm text-red-500 hover:bg-red-50"
              >
                삭제
              </button>
              <button
                onClick={() => setSelectedEvent(null)}
                className="ml-auto rounded px-3 py-1.5 text-sm hover:bg-foreground/5"
              >
                닫기
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Event Modal */}
      {showCreateModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
          onClick={() => setShowCreateModal(false)}
        >
          <div
            className="w-[calc(100vw-2rem)] max-w-96 md:w-96 rounded-lg border border-foreground/10 bg-background p-4 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-base font-medium">새 일정</h3>
            <div className="mt-3 space-y-3">
              <input
                autoFocus
                placeholder="제목"
                value={createForm.title}
                onChange={(e) => setCreateForm({ ...createForm, title: e.target.value })}
                className="w-full rounded border border-foreground/10 bg-background px-3 py-2 text-sm outline-none focus:border-foreground/30"
              />
              <div className="flex gap-2">
                <input
                  type="datetime-local"
                  value={createForm.start_time.slice(0, 16)}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, start_time: e.target.value + ":00+09:00" })
                  }
                  className="flex-1 rounded border border-foreground/10 bg-background px-2 py-2 text-sm outline-none"
                />
                <input
                  type="datetime-local"
                  value={createForm.end_time.slice(0, 16)}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, end_time: e.target.value + ":00+09:00" })
                  }
                  className="flex-1 rounded border border-foreground/10 bg-background px-2 py-2 text-sm outline-none"
                />
              </div>
              {calendars.length > 1 && (
                <select
                  value={createForm.calendar_id}
                  onChange={(e) => setCreateForm({ ...createForm, calendar_id: e.target.value })}
                  className="w-full rounded border border-foreground/10 bg-background px-3 py-2 text-sm outline-none"
                >
                  {calendars.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              )}
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => setShowCreateModal(false)}
                className="rounded px-3 py-1.5 text-sm hover:bg-foreground/5"
              >
                취소
              </button>
              <button
                onClick={handleCreateEvent}
                className="rounded bg-foreground px-3 py-1.5 text-sm text-background hover:opacity-90"
              >
                만들기
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Month View Component
function MonthView({
  currentDate,
  events,
  calendars,
  onEventClick,
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
    if (week.length === 7) {
      weeks.push(week);
      week = [];
    }
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
      <div className="grid grid-cols-7 border-b border-foreground/10">
        {weekdays.map((d, i) => (
          <div
            key={d}
            className={`py-2 text-center text-xs font-medium ${
              i === 0 ? "text-red-400" : i === 6 ? "text-blue-400" : "text-foreground/50"
            }`}
          >
            {d}
          </div>
        ))}
      </div>
      <div className="grid flex-1 grid-cols-7 grid-rows-[repeat(auto-fill,minmax(0,1fr))]">
        {weeks.map((weekDays, wi) =>
          weekDays.map((day, di) => {
            const isToday =
              day !== null &&
              year === today.getFullYear() &&
              month === today.getMonth() &&
              day === today.getDate();
            const dayEvents = day ? getEventsForDay(day) : [];
            const idx = wi * 7 + di;

            return (
              <div
                key={idx}
                className={`min-h-[60px] md:min-h-[80px] border-b border-r border-foreground/5 p-1 ${
                  day === null ? "bg-foreground/[0.02]" : ""
                }`}
              >
                {day !== null && (
                  <>
                    <div
                      className={`mb-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full text-xs ${
                        isToday
                          ? "bg-foreground text-background font-medium"
                          : di === 0
                          ? "text-red-400"
                          : di === 6
                          ? "text-blue-400"
                          : ""
                      }`}
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
                            className={`cursor-pointer truncate rounded px-1 py-0.5 text-[10px] leading-tight text-white ${ei === 2 ? "hidden md:block" : ""}`}
                            style={{
                              backgroundColor: getEventColor(e.color_id, cal?.color_id ?? null),
                            }}
                          >
                            {e.title || "(제목 없음)"}
                          </div>
                        );
                      })}
                      {dayEvents.length > 2 && (
                        <div className="text-[10px] text-foreground/40 px-1">
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

// Week View Component
function WeekView({
  currentDate,
  events,
  calendars,
  days,
  onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  calendars: CalendarData[];
  days: number;
  onEventClick: (e: EventData) => void;
}) {
  const startDate = new Date(currentDate);
  if (days === 7) {
    startDate.setDate(startDate.getDate() - startDate.getDay());
  }

  const dayDates = Array.from({ length: days }, (_, i) => {
    const d = new Date(startDate);
    d.setDate(d.getDate() + i);
    return d;
  });

  const today = new Date();
  const hours = Array.from({ length: 16 }, (_, i) => i + 7); // 7:00 - 22:00
  const calMap = new Map(calendars.map((c) => [c.id, c]));

  const getEventsForDay = (date: Date) => {
    const dateStr = date.toISOString().split("T")[0];
    return events.filter((e) => {
      const start = e.start_time.split("T")[0];
      const end = e.end_time.split("T")[0];
      return dateStr >= start && dateStr <= end;
    });
  };

  const formatDayHeader = (d: Date) => {
    const weekdays = ["일", "월", "화", "수", "목", "금", "토"];
    return `${weekdays[d.getDay()]} ${d.getDate()}`;
  };

  return (
    <div className="flex h-full flex-col">
      {/* Day headers */}
      <div className="flex border-b border-foreground/10">
        <div className="w-10 md:w-14 shrink-0" />
        {dayDates.map((d, i) => {
          const isToday = d.toDateString() === today.toDateString();
          return (
            <div
              key={i}
              className={`flex-1 border-l border-foreground/5 py-2 text-center text-xs ${
                isToday ? "font-medium" : "text-foreground/60"
              }`}
            >
              <span className={isToday ? "inline-flex h-6 w-6 items-center justify-center rounded-full bg-foreground text-background" : ""}>
                {formatDayHeader(d)}
              </span>
            </div>
          );
        })}
      </div>

      {/* Time grid */}
      <div className="flex flex-1 overflow-auto">
        <div className="w-10 md:w-14 shrink-0">
          {hours.map((h) => (
            <div key={h} className="relative h-12">
              <span className="absolute -top-2 right-2 text-[10px] text-foreground/40">
                {String(h).padStart(2, "0")}:00
              </span>
            </div>
          ))}
        </div>
        {dayDates.map((d, di) => {
          const dayEvents = getEventsForDay(d);
          return (
            <div key={di} className="relative flex-1 border-l border-foreground/5">
              {hours.map((h) => (
                <div
                  key={h}
                  className="h-12 border-b border-foreground/5"
                />
              ))}
              {/* Event blocks */}
              {dayEvents
                .filter((e) => !e.is_all_day)
                .map((e) => {
                  const startHour = new Date(e.start_time).getHours() + new Date(e.start_time).getMinutes() / 60;
                  const endHour = new Date(e.end_time).getHours() + new Date(e.end_time).getMinutes() / 60;
                  const top = (startHour - 7) * 48;
                  const height = Math.max((endHour - startHour) * 48, 20);
                  const cal = calMap.get(e.calendar_id);

                  return (
                    <div
                      key={e.id}
                      className="absolute left-0.5 right-1 cursor-pointer overflow-hidden rounded px-1 py-0.5 text-[10px] text-white"
                      style={{
                        top: `${top}px`,
                        height: `${height}px`,
                        backgroundColor: getEventColor(e.color_id, cal?.color_id ?? null),
                      }}
                      onClick={() => onEventClick(e)}
                    >
                      <div className="font-medium truncate">{e.title || "(제목 없음)"}</div>
                      {height > 30 && (
                        <div className="truncate opacity-80">
                          {new Date(e.start_time).toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}
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

// Agenda View Component
function AgendaView({
  events,
  calendars,
  onEventClick,
}: {
  events: EventData[];
  calendars: CalendarData[];
  onEventClick: (e: EventData) => void;
}) {
  const calMap = new Map(calendars.map((c) => [c.id, c]));

  // Group events by date
  const grouped = new Map<string, EventData[]>();
  for (const e of events) {
    const dateStr = e.start_time.split("T")[0];
    if (!grouped.has(dateStr)) grouped.set(dateStr, []);
    grouped.get(dateStr)!.push(e);
  }

  const sortedDates = Array.from(grouped.keys()).sort();

  if (sortedDates.length === 0) {
    return (
      <div className="flex items-center justify-center py-20 text-foreground/40">
        이 기간에 일정이 없습니다
      </div>
    );
  }

  return (
    <div className="divide-y divide-foreground/5 p-4">
      {sortedDates.map((date) => (
        <div key={date} className="py-3">
          <h3 className="mb-2 text-sm font-medium text-foreground/70">
            {new Date(date + "T00:00:00").toLocaleDateString("ko-KR", {
              year: "numeric",
              month: "long",
              day: "numeric",
              weekday: "short",
            })}
          </h3>
          <div className="space-y-1">
            {grouped.get(date)!.map((e) => {
              const cal = calMap.get(e.calendar_id);
              return (
                <div
                  key={e.id}
                  className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 hover:bg-foreground/[0.03]"
                  onClick={() => onEventClick(e)}
                >
                  <div
                    className="h-3 w-3 shrink-0 rounded-full"
                    style={{ backgroundColor: getEventColor(e.color_id, cal?.color_id ?? null) }}
                  />
                  <div className="min-w-0 flex-1">
                    <span className="text-sm">{e.title || "(제목 없음)"}</span>
                  </div>
                  <span className="shrink-0 text-xs text-foreground/50">
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

// Helpers

function getDateRange(date: Date, view: ViewMode): { start: string; end: string } {
  const d = new Date(date);
  let start: Date;
  let end: Date;

  if (view === "month") {
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

  return {
    start: start.toISOString(),
    end: end.toISOString(),
  };
}

function getHeaderLabel(date: Date, view: ViewMode): string {
  if (view === "month") {
    return `${date.getFullYear()}년 ${date.getMonth() + 1}월`;
  }
  if (view === "week" || view === "3day") {
    return `${date.getFullYear()}년 ${date.getMonth() + 1}월 ${date.getDate()}일`;
  }
  return `${date.getFullYear()}년 ${date.getMonth() + 1}월`;
}

function formatEventTime(e: EventData): string {
  if (e.is_all_day) return "종일";
  const start = new Date(e.start_time);
  const end = new Date(e.end_time);
  const opts: Intl.DateTimeFormatOptions = { month: "long", day: "numeric", hour: "2-digit", minute: "2-digit" };
  return `${start.toLocaleDateString("ko-KR", opts)} - ${end.toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}`;
}

function ChevronLeft() {
  return (
    <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="m15 18-6-6 6-6" />
    </svg>
  );
}

function ChevronRight() {
  return (
    <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="m9 18 6-6-6-6" />
    </svg>
  );
}
