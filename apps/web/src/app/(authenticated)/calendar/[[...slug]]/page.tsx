"use client";

import { useState, useEffect, useCallback, useMemo, useRef, type MouseEvent as ReactMouseEvent } from "react";
import { useRouter, useParams } from "next/navigation";
import { useCalendarActions } from "@/features/calendar/ui/hooks/useCalendarActions";
import type {
  CalendarData,
  CreateEventInput,
  EventData,
  EventPayload,
} from "@/features/calendar/domain/CalendarEntities";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { ViewDropdown, type CalendarViewMode } from "@/components/calendar/ViewDropdown";
import { YearCompactView } from "@/components/calendar/YearCompactView";
import { YearTimelineView } from "@/components/calendar/YearTimelineView";
import { QuickCreatePopover } from "@/components/calendar/QuickCreatePopover";
import { PageToolbar, PageToolbarGroup } from "@/components/layout/PageToolbar";
import {
  getFixedMonthFetchRangeWithWeekStart,
  buildFixedMonthGridWithWeekStart,
  type MonthCell,
} from "@/lib/calendar/month-grid";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface EventEditorForm {
  title: string;
  description: string;
  location: string;
  startLocal: string;
  endLocal: string;
  isAllDay: boolean;
  calendarId: string;
}

interface QuickCreateAnchorPoint {
  x: number;
  y: number;
  side?: "left" | "right";
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
      date = new Date(parseInt(dateStr, 10), 0, 1);
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

function toDateStr(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

function toLocalDateTimeInput(date: Date): string {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

function fromLocalDateTimeInput(local: string): Date {
  return new Date(local);
}

function formatTimeLabel(iso: string): string {
  const date = new Date(iso);
  let hour = date.getHours();
  const minute = date.getMinutes();
  const ampm = hour >= 12 ? "PM" : "AM";
  hour = hour % 12;
  if (hour === 0) hour = 12;
  if (minute === 0) return `${ampm} ${hour}시`;
  return `${ampm} ${hour}:${String(minute).padStart(2, "0")}`;
}

function formatTimeRangeLabel(startISO: string, endISO: string): string {
  return `${formatTimeLabel(startISO)} - ${formatTimeLabel(endISO)}`;
}

function parseWeekHourRange(settings: Record<string, string>): { start: number; end: number } {
  const rawStart = Number.parseInt(settings.week_start_hour || "8", 10);
  const rawEnd = Number.parseInt(settings.week_end_hour || "22", 10);

  const start = Number.isNaN(rawStart) ? 8 : Math.min(Math.max(rawStart, 0), 23);
  const end = Number.isNaN(rawEnd) ? 22 : Math.min(Math.max(rawEnd, 1), 24);

  if (start >= end) return { start: 8, end: 22 };
  return { start, end };
}

function parseWeekStartsOn(settings: Record<string, string>): number {
  const raw = Number.parseInt(settings.calendar_week_start || "0", 10);
  if (Number.isNaN(raw)) return 0;
  const normalized = raw % 7;
  return normalized < 0 ? normalized + 7 : normalized;
}

function makeDefaultEditorForm(start: Date, end: Date, calendarId: string): EventEditorForm {
  return {
    title: "",
    description: "",
    location: "",
    startLocal: toLocalDateTimeInput(start),
    endLocal: toLocalDateTimeInput(end),
    isAllDay: false,
    calendarId,
  };
}

function buildEventPayload(form: EventEditorForm, timezone: string): EventPayload {
  const startDate = fromLocalDateTimeInput(form.startLocal);
  let endDate = fromLocalDateTimeInput(form.endLocal);

  if (endDate <= startDate) {
    endDate = new Date(startDate.getTime() + 30 * 60 * 1000);
  }

  if (form.isAllDay) {
    const start = new Date(startDate.getFullYear(), startDate.getMonth(), startDate.getDate(), 0, 0, 0, 0);
    const end = new Date(endDate.getFullYear(), endDate.getMonth(), endDate.getDate(), 23, 59, 59, 999);
    return {
      title: form.title,
      description: form.description,
      location: form.location,
      start_time: start.toISOString(),
      end_time: end.toISOString(),
      timezone,
      is_all_day: true,
    };
  }

  return {
    title: form.title,
    description: form.description,
    location: form.location,
    start_time: startDate.toISOString(),
    end_time: endDate.toISOString(),
    timezone,
    is_all_day: false,
  };
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
  const [selectedDateKey, setSelectedDateKey] = useState<string | null>(null);

  const [weekHours, setWeekHours] = useState({ start: 8, end: 22 });
  const [weekStartsOn, setWeekStartsOn] = useState(0);

  const [quickCreateOpen, setQuickCreateOpen] = useState(false);
  const [quickDefaultStart, setQuickDefaultStart] = useState<string>("");
  const [quickDefaultEnd, setQuickDefaultEnd] = useState<string>("");
  const [quickCreateAnchor, setQuickCreateAnchor] = useState<QuickCreateAnchorPoint | null>(null);

  const [editorOpen, setEditorOpen] = useState(false);
  const [editorMode, setEditorMode] = useState<"create" | "edit">("create");
  const [editingEventID, setEditingEventID] = useState<string | null>(null);
  const [editorForm, setEditorForm] = useState<EventEditorForm>(
    makeDefaultEditorForm(new Date(), new Date(Date.now() + 60 * 60 * 1000), "")
  );

  const { listCalendars, getSettings, listEvents, createEvent, updateEvent, deleteEvent } = useCalendarActions();

  const timezone = useMemo(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Seoul",
    []
  );

  const defaultCalendarID = useMemo(
    () => calendars.find((cal) => cal.is_visible)?.id || calendars[0]?.id || "",
    [calendars]
  );

  useEffect(() => {
    if (!editorForm.calendarId && defaultCalendarID) {
      setEditorForm((prev) => ({ ...prev, calendarId: defaultCalendarID }));
    }
  }, [defaultCalendarID, editorForm.calendarId]);

  const loadCalendars = useCallback(async () => {
    try {
      const next = await listCalendars();
      setCalendars(next || []);
    } catch {
      setCalendars([]);
    }
  }, [listCalendars]);

  const loadSettings = useCallback(async () => {
    try {
      const data = await getSettings();
      const settings = data.settings || {};
      setWeekHours(parseWeekHourRange(settings));
      setWeekStartsOn(parseWeekStartsOn(settings));
    } catch {
      setWeekHours({ start: 8, end: 22 });
      setWeekStartsOn(0);
    }
  }, [getSettings]);

  const loadEvents = useCallback(async () => {
    setLoading(true);
    const { start, end } = getDateRange(currentDate, view, weekStartsOn);
    try {
      const next = await listEvents(start, end);
      setEvents(next || []);
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [currentDate, listEvents, view, weekStartsOn]);

  useEffect(() => { loadCalendars(); }, [loadCalendars]);
  useEffect(() => { loadSettings(); }, [loadSettings]);
  useEffect(() => { loadEvents(); }, [loadEvents]);

  const updateUrl = useCallback((newView: CalendarViewMode, date: Date) => {
    router.replace(buildCalendarUrl(newView, date), { scroll: false });
  }, [router]);

  const handleViewChange = (newView: CalendarViewMode) => {
    setView(newView);
    updateUrl(newView, currentDate);
  };

  const navigate = (dir: number) => {
    const next = new Date(currentDate);
    if (view === "year-compact" || view === "year-timeline") next.setFullYear(next.getFullYear() + dir);
    else if (view === "month") next.setMonth(next.getMonth() + dir);
    else if (view === "week") next.setDate(next.getDate() + 7 * dir);
    else if (view === "3day") next.setDate(next.getDate() + 3 * dir);
    else next.setDate(next.getDate() + 7 * dir);
    setCurrentDate(next);
    updateUrl(view, next);
  };

  const goToday = () => {
    const today = new Date();
    setCurrentDate(today);
    setSelectedDateKey(toDateStr(today));
    updateUrl(view, today);
  };

  const openQuickCreate = (
    start: Date,
    end: Date,
    selectedKey?: string,
    anchorPoint?: QuickCreateAnchorPoint | null
  ) => {
    setQuickDefaultStart(toLocalDateTimeInput(start));
    setQuickDefaultEnd(toLocalDateTimeInput(end));
    setQuickCreateAnchor(anchorPoint ?? null);
    if (selectedKey) setSelectedDateKey(selectedKey);
    setQuickCreateOpen(true);
  };

  const getQuickCreateAnchorFromEvent = (
    event: ReactMouseEvent<HTMLDivElement | HTMLButtonElement>
  ): QuickCreateAnchorPoint => {
    const rect = event.currentTarget.getBoundingClientRect();
    const estimatedPopoverWidth = 352;
    const edgePadding = 12;
    const isLeftRegion = rect.left < window.innerWidth * 0.4;
    let side: "left" | "right" = isLeftRegion ? "right" : "left";

    if (side === "left" && rect.left < estimatedPopoverWidth + edgePadding) {
      side = "right";
    } else if (
      side === "right" &&
      window.innerWidth - rect.right < estimatedPopoverWidth + edgePadding
    ) {
      side = "left";
    }

    return {
      x: side === "left" ? rect.left + 4 : rect.right - 4,
      y: rect.top + 8,
      side,
    };
  };

  const handleDayClick = (date: Date, event: ReactMouseEvent<HTMLDivElement>) => {
    const start = new Date(date.getFullYear(), date.getMonth(), date.getDate(), 9, 0, 0, 0);
    const end = new Date(start.getTime() + 60 * 60 * 1000);
    openQuickCreate(start, end, toDateStr(date), getQuickCreateAnchorFromEvent(event));
  };

  const handleWeekSlotClick = (date: Date, hour: number, event: ReactMouseEvent<HTMLButtonElement>) => {
    const start = new Date(date.getFullYear(), date.getMonth(), date.getDate(), hour, 0, 0, 0);
    const end = new Date(start.getTime() + 60 * 60 * 1000);
    openQuickCreate(start, end, toDateStr(date), getQuickCreateAnchorFromEvent(event));
  };

  const handleCreateEvent = async (data: {
    title: string;
    start_local: string;
    end_local: string;
    calendar_id: string;
  }) => {
    try {
      const payload: CreateEventInput = {
        calendar_id: data.calendar_id || defaultCalendarID,
        title: data.title,
        start_time: fromLocalDateTimeInput(data.start_local).toISOString(),
        end_time: fromLocalDateTimeInput(data.end_local).toISOString(),
        is_all_day: false,
        timezone,
        description: "",
        location: "",
      };
      await createEvent(payload);
      await loadEvents();
      setQuickCreateOpen(false);
    } catch (err) {
      console.error("Create event failed:", err);
    }
  };

  const openCreateEditor = (draft?: {
    title: string;
    start_local: string;
    end_local: string;
    calendar_id: string;
  }) => {
    const fallbackStart = new Date();
    const fallbackEnd = new Date(Date.now() + 60 * 60 * 1000);

    const start = draft?.start_local ? fromLocalDateTimeInput(draft.start_local) : fallbackStart;
    const end = draft?.end_local ? fromLocalDateTimeInput(draft.end_local) : fallbackEnd;

    setEditorMode("create");
    setEditingEventID(null);
    setEditorForm({
      title: draft?.title || "",
      description: "",
      location: "",
      startLocal: toLocalDateTimeInput(start),
      endLocal: toLocalDateTimeInput(end),
      isAllDay: false,
      calendarId: draft?.calendar_id || defaultCalendarID,
    });
    setEditorOpen(true);
  };

  const openEditEditor = (event: EventData) => {
    setEditorMode("edit");
    setEditingEventID(event.id);
    setEditorForm({
      title: event.title || "",
      description: event.description || "",
      location: event.location || "",
      startLocal: toLocalDateTimeInput(new Date(event.start_time)),
      endLocal: toLocalDateTimeInput(new Date(event.end_time)),
      isAllDay: event.is_all_day,
      calendarId: event.calendar_id,
    });
    setEditorOpen(true);
  };

  const handleEditorSubmit = async () => {
    if (!editorForm.title.trim()) return;
    if (!editorForm.startLocal || !editorForm.endLocal) return;

    try {
      const payload = buildEventPayload(editorForm, timezone);
      if (editorMode === "create") {
        await createEvent({
          calendar_id: editorForm.calendarId || defaultCalendarID,
          ...payload,
        });
      } else if (editingEventID) {
        await updateEvent(editingEventID, payload);
      }

      setEditorOpen(false);
      setSelectedEvent(null);
      await loadEvents();
    } catch (err) {
      console.error("Save event failed:", err);
    }
  };

  const handleDeleteEvent = async (eventID: string) => {
    try {
      await deleteEvent(eventID);
      setSelectedEvent(null);
      await loadEvents();
    } catch (err) {
      console.error("Delete event failed:", err);
    }
  };

  const headerLabel = getHeaderLabel(currentDate, view);
  const calMap = useMemo(() => new Map(calendars.map((cal) => [cal.id, cal])), [calendars]);

  return (
    <div className="flex h-full flex-col">
      <PageToolbar>
        <PageToolbarGroup>
          <Button variant="secondary" size="sm" className="rounded-full" onClick={goToday}>오늘</Button>
          <Button variant="ghost" size="icon-sm" onClick={() => navigate(-1)}>
            <ChevronLeft size={16} />
          </Button>
          <Button variant="ghost" size="icon-sm" onClick={() => navigate(1)}>
            <ChevronRight size={16} />
          </Button>
          <h2 className="text-lg font-semibold text-text-strong">{headerLabel}</h2>
        </PageToolbarGroup>
        <PageToolbarGroup className="gap-2">
          <ViewDropdown view={view} onChange={handleViewChange} />
          <QuickCreatePopover
            key={`${quickDefaultStart}|${quickDefaultEnd}|${defaultCalendarID}`}
            open={quickCreateOpen}
            anchorPoint={quickCreateAnchor}
            onOpenChange={(open) => {
              setQuickCreateOpen(open);
              if (!open) setQuickCreateAnchor(null);
            }}
            defaultStart={quickDefaultStart}
            defaultEnd={quickDefaultEnd}
            calendarId={defaultCalendarID}
            onSubmit={handleCreateEvent}
            onDetail={openCreateEditor}
          />
        </PageToolbarGroup>
      </PageToolbar>

      <div className="flex-1 min-h-0 overflow-auto">
        {loading && events.length === 0 ? (
          <div className="flex items-center justify-center py-20 text-text-muted">불러오는 중...</div>
        ) : view === "year-compact" ? (
          <YearCompactView
            year={currentDate.getFullYear()}
            events={events}
            onMonthClick={(month) => {
              const date = new Date(currentDate.getFullYear(), month, 1);
              setView("month");
              setCurrentDate(date);
              updateUrl("month", date);
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
            weekStartsOn={weekStartsOn}
            selectedDateKey={selectedDateKey}
            onDayClick={handleDayClick}
            onEventClick={setSelectedEvent}
          />
        ) : view === "week" || view === "3day" ? (
          <WeekView
            currentDate={currentDate}
            events={events}
            calendars={calendars}
            days={view === "week" ? 7 : 3}
            startHour={weekHours.start}
            endHour={weekHours.end}
            weekStartsOn={weekStartsOn}
            onSlotClick={handleWeekSlotClick}
            onEventClick={setSelectedEvent}
          />
        ) : (
          <AgendaView events={events} calendars={calendars} onEventClick={setSelectedEvent} />
        )}
      </div>

      {selectedEvent && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
          onClick={() => setSelectedEvent(null)}
        >
          <div
            className="w-[calc(100vw-2rem)] max-w-80 rounded-lg border border-border bg-background p-4 shadow-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-start gap-3">
              <div
                className="mt-1 h-3 w-3 shrink-0 rounded-full"
                style={{
                  backgroundColor: getEventColor(
                    selectedEvent.color_id,
                    calMap.get(selectedEvent.calendar_id)?.color_id ?? null
                  ),
                }}
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
            <div className="mt-3 flex justify-between gap-2">
              <Button variant="danger" size="sm" onClick={() => handleDeleteEvent(selectedEvent.id)}>삭제</Button>
              <div className="flex gap-2">
                <Button variant="secondary" size="sm" onClick={() => openEditEditor(selectedEvent)}>수정</Button>
                <Button variant="ghost" size="sm" onClick={() => setSelectedEvent(null)}>닫기</Button>
              </div>
            </div>
          </div>
        </div>
      )}

      <Dialog open={editorOpen} onOpenChange={setEditorOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editorMode === "create" ? "일정 추가" : "일정 수정"}</DialogTitle>
          </DialogHeader>

          <div className="space-y-3">
            <Input
              placeholder="제목"
              value={editorForm.title}
              onChange={(event) => setEditorForm((prev) => ({ ...prev, title: event.target.value }))}
            />

            {editorMode === "create" && (
              <Select
                value={editorForm.calendarId}
                onValueChange={(value) => setEditorForm((prev) => ({ ...prev, calendarId: value }))}
              >
                <SelectTrigger className="h-9">
                  <SelectValue placeholder="캘린더" />
                </SelectTrigger>
                <SelectContent>
                  {calendars.map((calendar) => (
                    <SelectItem key={calendar.id} value={calendar.id}>
                      {calendar.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}

            <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
              <div className="space-y-1">
                <label className="text-xs text-text-muted">시작</label>
                <Input
                  type="datetime-local"
                  value={editorForm.startLocal}
                  onChange={(event) => setEditorForm((prev) => ({ ...prev, startLocal: event.target.value }))}
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-text-muted">종료</label>
                <Input
                  type="datetime-local"
                  value={editorForm.endLocal}
                  onChange={(event) => setEditorForm((prev) => ({ ...prev, endLocal: event.target.value }))}
                />
              </div>
            </div>

            <label className="flex items-center gap-2 text-sm text-text-secondary">
              <input
                type="checkbox"
                checked={editorForm.isAllDay}
                onChange={(event) => setEditorForm((prev) => ({ ...prev, isAllDay: event.target.checked }))}
              />
              종일
            </label>

            <Input
              placeholder="장소"
              value={editorForm.location}
              onChange={(event) => setEditorForm((prev) => ({ ...prev, location: event.target.value }))}
            />

            <Textarea
              placeholder="메모"
              rows={3}
              value={editorForm.description}
              onChange={(event) => setEditorForm((prev) => ({ ...prev, description: event.target.value }))}
            />
          </div>

          <DialogFooter className="justify-between">
            <Button variant="ghost" size="sm" onClick={() => setEditorOpen(false)}>취소</Button>
            <Button size="sm" onClick={handleEditorSubmit} disabled={!editorForm.title.trim()}>
              저장
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ===== Month View with 42 Cells ===== */
function MonthView({
  currentDate,
  events,
  calendars,
  weekStartsOn,
  selectedDateKey,
  onDayClick,
  onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  calendars: CalendarData[];
  weekStartsOn: number;
  selectedDateKey: string | null;
  onDayClick: (date: Date, event: ReactMouseEvent<HTMLDivElement>) => void;
  onEventClick: (event: EventData) => void;
}) {
  const cells = useMemo(
    () => buildFixedMonthGridWithWeekStart(currentDate, weekStartsOn),
    [currentDate, weekStartsOn]
  );
  const weeks = useMemo(
    () => Array.from({ length: 6 }, (_, index) => cells.slice(index * 7, index * 7 + 7)),
    [cells]
  );
  const todayKey = toDateStr(new Date());

  const calMap = new Map(calendars.map((calendar) => [calendar.id, calendar]));
  const weekdayLabels = ["일", "월", "화", "수", "목", "금", "토"];
  const weekdays = Array.from({ length: 7 }, (_, index) => {
    const day = (weekStartsOn + index) % 7;
    return { label: weekdayLabels[day], day };
  });

  const multiDayEvents = events.filter((event) => {
    const start = event.start_time.split("T")[0];
    const end = event.end_time.split("T")[0];
    return event.is_all_day || start !== end;
  });

  const singleDayEvents = events.filter((event) => {
    const start = event.start_time.split("T")[0];
    const end = event.end_time.split("T")[0];
    return !event.is_all_day && start === end;
  });

  function getWeekSpanBars(weekCells: MonthCell[]) {
    const bars: { event: EventData; startCol: number; endCol: number; lane: number }[] = [];
    const lanes: string[][] = [];

    for (const event of multiDayEvents) {
      const eventStart = event.start_time.split("T")[0];
      const eventEnd = event.end_time.split("T")[0];

      let startCol = -1;
      let endCol = -1;

      for (let col = 0; col < 7; col += 1) {
        const key = weekCells[col].dateKey;
        if (key >= eventStart && key <= eventEnd) {
          if (startCol === -1) startCol = col;
          endCol = col;
        }
      }

      if (startCol === -1) continue;

      let assignedLane = -1;
      for (let lane = 0; lane < lanes.length; lane += 1) {
        const occupied = lanes[lane].some((id) => {
          const bar = bars.find((item) => item.event.id === id && item.lane === lane);
          if (!bar) return false;
          return !(endCol < bar.startCol || startCol > bar.endCol);
        });

        if (!occupied) {
          assignedLane = lane;
          lanes[lane].push(event.id);
          break;
        }
      }

      if (assignedLane === -1 && lanes.length < 3) {
        assignedLane = lanes.length;
        lanes.push([event.id]);
      }
      if (assignedLane === -1) continue;

      bars.push({ event, startCol, endCol, lane: assignedLane });
    }

    return bars;
  }

  function getSingleDayEvents(dateKey: string) {
    return singleDayEvents.filter((event) => event.start_time.split("T")[0] === dateKey);
  }

  function countAllEventsForDate(dateKey: string) {
    return events.filter((event) => {
      const start = event.start_time.split("T")[0];
      const end = event.end_time.split("T")[0];
      return dateKey >= start && dateKey <= end;
    }).length;
  }

  return (
    <div className="flex h-full flex-col">
      <div className="grid grid-cols-7 border-b border-border">
        {weekdays.map(({ label, day }) => (
          <div
            key={`${label}-${day}`}
            className={cn(
              "py-2 text-center text-xs font-medium",
              day === 0 ? "text-error" : day === 6 ? "text-info" : "text-text-muted"
            )}
          >
            {label}
          </div>
        ))}
      </div>

      <div className="grid flex-1 grid-cols-7" style={{ gridTemplateRows: "repeat(6, 1fr)" }}>
        {weeks.map((weekCells, weekIndex) => {
          const spanBars = getWeekSpanBars(weekCells);
          const spanBarHeight = Math.max(
            spanBars.length > 0 ? (Math.max(...spanBars.map((bar) => bar.lane)) + 1) * 18 : 0,
            0
          );

          return weekCells.map((cell, dayIndex) => {
            const isToday = cell.dateKey === todayKey;
            const isSelected = selectedDateKey === cell.dateKey;
            const singleEvents = getSingleDayEvents(cell.dateKey);
            const totalCount = countAllEventsForDate(cell.dateKey);
            const index = weekIndex * 7 + dayIndex;
            const barsStartingHere = spanBars.filter((bar) => bar.startCol === dayIndex);

            return (
              <div
                key={index}
                className={cn(
                  "relative min-h-[60px] md:min-h-[80px] border-b border-r border-border/50 p-1 overflow-hidden cursor-pointer",
                  !cell.inCurrentMonth && "bg-surface-accent/20",
                  isSelected && "ring-1 ring-inset ring-primary/50"
                )}
                onClick={(event) => onDayClick(cell.date, event)}
              >
                <div
                  className={cn(
                    "mb-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full text-xs",
                    isToday
                      ? "bg-primary text-white font-medium"
                      : !cell.inCurrentMonth
                          ? "text-text-muted"
                        : cell.date.getDay() === 0
                          ? "text-error"
                          : cell.date.getDay() === 6
                            ? "text-info"
                            : "text-text-primary"
                  )}
                >
                  {cell.day}
                </div>

                {barsStartingHere.map((bar) => {
                  const spanCols = bar.endCol - bar.startCol + 1;
                  const calendar = calMap.get(bar.event.calendar_id);
                  return (
                    <div
                      key={bar.event.id}
                      className={cn(
                        "absolute left-0.5 cursor-pointer truncate rounded px-1 text-[10px] leading-[16px] text-white z-10",
                        !cell.inCurrentMonth && "opacity-60"
                      )}
                      style={{
                        top: `${26 + bar.lane * 18}px`,
                        width: `calc(${spanCols * 100}% - 4px)`,
                        backgroundColor: getEventColor(bar.event.color_id, calendar?.color_id ?? null),
                      }}
                      onClick={(event) => {
                        event.stopPropagation();
                        onEventClick(bar.event);
                      }}
                    >
                      {bar.event.title || "(제목 없음)"}
                    </div>
                  );
                })}

                <div className="space-y-0.5" style={{ marginTop: `${spanBarHeight}px` }}>
                  {singleEvents.slice(0, 2).map((event) => {
                    const calendar = calMap.get(event.calendar_id);
                    return (
                      <div
                        key={event.id}
                        onClick={(clickEvent) => {
                          clickEvent.stopPropagation();
                          onEventClick(event);
                        }}
                        className={cn(
                          "flex items-center gap-1 cursor-pointer truncate px-0.5 text-[10px] leading-tight text-text-primary",
                          !cell.inCurrentMonth && "opacity-60"
                        )}
                      >
                        <div
                          className="h-1.5 w-1.5 shrink-0 rounded-full"
                          style={{ backgroundColor: getEventColor(event.color_id, calendar?.color_id ?? null) }}
                        />
                        <span className="inline-block w-[64px] shrink-0 tabular-nums text-text-muted">
                          {formatTimeLabel(event.start_time)}
                        </span>
                        <span className="truncate">{event.title || "(제목 없음)"}</span>
                      </div>
                    );
                  })}
                  {totalCount > 3 && (
                    <div className="text-[10px] text-text-muted px-0.5">+{totalCount - 3}</div>
                  )}
                </div>
              </div>
            );
          });
        })}
      </div>
    </div>
  );
}

/* ===== Week View ===== */
function WeekView({
  currentDate,
  events,
  calendars,
  days,
  startHour,
  endHour,
  weekStartsOn,
  onSlotClick,
  onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  calendars: CalendarData[];
  days: number;
  startHour: number;
  endHour: number;
  weekStartsOn: number;
  onSlotClick: (date: Date, hour: number, event: ReactMouseEvent<HTMLButtonElement>) => void;
  onEventClick: (event: EventData) => void;
}) {
  const startDate = new Date(currentDate);
  if (days === 7) {
    const offset = (startDate.getDay() - weekStartsOn + 7) % 7;
    startDate.setDate(startDate.getDate() - offset);
  }

  const dayDates = Array.from({ length: days }, (_, index) => {
    const date = new Date(startDate);
    date.setDate(date.getDate() + index);
    return date;
  });

  const TOP_SPACER_HEIGHT = 24;
  const DEFAULT_HOUR_HEIGHT = 48;

  const today = new Date();
  const hours = Array.from({ length: endHour - startHour }, (_, index) => index + startHour);
  const calMap = new Map(calendars.map((calendar) => [calendar.id, calendar]));
  const gridBodyRef = useRef<HTMLDivElement | null>(null);
  const [hourHeight, setHourHeight] = useState(DEFAULT_HOUR_HEIGHT);

  useEffect(() => {
    const element = gridBodyRef.current;
    if (!element) return;

    const recalculateHourHeight = () => {
      const totalSlots = Math.max(hours.length, 1);
      const availableHeight = element.clientHeight - TOP_SPACER_HEIGHT;
      if (availableHeight <= 0) {
        setHourHeight(DEFAULT_HOUR_HEIGHT);
        return;
      }
      setHourHeight(availableHeight / totalSlots);
    };

    recalculateHourHeight();
    const observer = new ResizeObserver(recalculateHourHeight);
    observer.observe(element);
    return () => observer.disconnect();
  }, [hours.length]);

  const getEventsForDay = (date: Date) => {
    const dateKey = toDateStr(date);
    return events.filter((event) => {
      const start = event.start_time.split("T")[0];
      const end = event.end_time.split("T")[0];
      return dateKey >= start && dateKey <= end;
    });
  };

  const getAllDayEvents = (date: Date) => getEventsForDay(date).filter((event) => event.is_all_day);
  const getTimedEvents = (date: Date) => getEventsForDay(date).filter((event) => !event.is_all_day);

  const hasAnyAllDay = dayDates.some((date) => getAllDayEvents(date).length > 0);
  const weekdays = ["일", "월", "화", "수", "목", "금", "토"];

  return (
    <div className="flex h-full flex-col">
      <div className="flex border-b border-border">
        <div className="w-10 md:w-14 shrink-0" />
        {dayDates.map((date, index) => {
          const isToday = date.toDateString() === today.toDateString();
          return (
            <div
              key={index}
              className={cn(
                "flex-1 border-l border-border/50 py-2 text-center text-xs",
                isToday ? "font-medium text-text-strong" : "text-text-secondary"
              )}
            >
              <span className={isToday ? "inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-primary px-1 text-white" : ""}>
                {weekdays[date.getDay()]} {date.getDate()}
              </span>
            </div>
          );
        })}
      </div>

      {hasAnyAllDay && (
        <div className="flex border-b border-border">
          <div className="w-10 md:w-14 shrink-0 flex items-center justify-end pr-2">
            <span className="text-[10px] text-text-muted">종일</span>
          </div>
          {dayDates.map((date, index) => {
            const allDayEvents = getAllDayEvents(date);
            return (
              <div key={index} className="flex-1 border-l border-border/50 px-0.5 py-1 space-y-0.5">
                {allDayEvents.map((event) => {
                  const calendar = calMap.get(event.calendar_id);
                  return (
                    <div
                      key={event.id}
                      className="cursor-pointer truncate rounded px-1 py-0.5 text-[10px] leading-tight text-white"
                      style={{ backgroundColor: getEventColor(event.color_id, calendar?.color_id ?? null) }}
                      onClick={() => onEventClick(event)}
                    >
                      {event.title || "(제목 없음)"}
                    </div>
                  );
                })}
              </div>
            );
          })}
        </div>
      )}

      <div ref={gridBodyRef} className="flex flex-1 min-h-0 overflow-auto">
        <div className="w-10 md:w-14 shrink-0">
          <div className="relative h-6" />
          {hours.map((hour) => (
            <div key={hour} className="relative" style={{ height: `${hourHeight}px` }}>
              <span className="absolute -top-2 right-2 text-[10px] text-text-muted tabular-nums">
                {String(hour).padStart(2, "0")}:00
              </span>
            </div>
          ))}
        </div>

        {dayDates.map((date, dayIndex) => {
          const dayEvents = getTimedEvents(date);
          const positioned = positionEvents(dayEvents);

          return (
            <div key={dayIndex} className="relative flex-1 border-l border-border/50">
              <button
                type="button"
                className="block h-6 w-full border-b border-dashed border-border/40"
                onClick={(event) => onSlotClick(date, startHour, event)}
              />
              {hours.map((hour) => (
                <button
                  key={hour}
                  type="button"
                  className="block w-full border-b border-border/30 hover:bg-surface-accent/30"
                  style={{ height: `${hourHeight}px` }}
                  onClick={(event) => onSlotClick(date, hour, event)}
                />
              ))}

              {positioned.map(({ event, column, totalColumns }) => {
                const start = new Date(event.start_time);
                const end = new Date(event.end_time);
                const startPosition = start.getHours() + start.getMinutes() / 60;
                const endPosition = end.getHours() + end.getMinutes() / 60;
                const top = (startPosition - startHour) * hourHeight + TOP_SPACER_HEIGHT;
                const height = Math.max((endPosition - startPosition) * hourHeight, 20);
                const calendar = calMap.get(event.calendar_id);
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
                      backgroundColor: getEventColor(event.color_id, calendar?.color_id ?? null),
                    }}
                    onClick={(clickEvent) => {
                      clickEvent.stopPropagation();
                      onEventClick(event);
                    }}
                  >
                    <div className="truncate font-medium">{event.title || "(제목 없음)"}</div>
                    {height > 30 && (
                      <div className="truncate opacity-80 tabular-nums">
                        {formatTimeRangeLabel(event.start_time, event.end_time)}
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

  for (const event of sorted) {
    if (currentGroup.length === 0 || event.start_time < groupEnd) {
      currentGroup.push(event);
      if (event.end_time > groupEnd) groupEnd = event.end_time;
    } else {
      groups.push(currentGroup);
      currentGroup = [event];
      groupEnd = event.end_time;
    }
  }
  if (currentGroup.length > 0) groups.push(currentGroup);

  const result: { event: EventData; column: number; totalColumns: number }[] = [];
  for (const group of groups) {
    const columns: string[][] = [];
    for (const event of group) {
      let placed = false;
      for (let column = 0; column < columns.length; column += 1) {
        const lastEnd = columns[column][columns[column].length - 1];
        if (event.start_time >= lastEnd) {
          columns[column].push(event.end_time);
          result.push({ event, column, totalColumns: 0 });
          placed = true;
          break;
        }
      }
      if (!placed) {
        columns.push([event.end_time]);
        result.push({ event, column: columns.length - 1, totalColumns: 0 });
      }
    }

    const totalColumns = columns.length;
    for (const row of result) {
      if (group.includes(row.event)) row.totalColumns = totalColumns;
    }
  }

  return result;
}

/* ===== Agenda View ===== */
function AgendaView({
  events,
  calendars,
  onEventClick,
}: {
  events: EventData[];
  calendars: CalendarData[];
  onEventClick: (event: EventData) => void;
}) {
  const calMap = new Map(calendars.map((calendar) => [calendar.id, calendar]));
  const grouped = new Map<string, EventData[]>();

  for (const event of events) {
    const dateKey = event.start_time.split("T")[0];
    if (!grouped.has(dateKey)) grouped.set(dateKey, []);
    grouped.get(dateKey)!.push(event);
  }

  const sortedDates = Array.from(grouped.keys()).sort();

  if (sortedDates.length === 0) {
    return <div className="flex items-center justify-center py-20 text-text-muted">이 기간에 일정이 없습니다</div>;
  }

  return (
    <div className="divide-y divide-border/50 p-4">
      {sortedDates.map((dateKey) => (
        <div key={dateKey} className="py-3">
          <h3 className="mb-2 text-sm font-medium text-text-secondary">
            {new Date(dateKey + "T00:00:00").toLocaleDateString("ko-KR", {
              year: "numeric", month: "long", day: "numeric", weekday: "short",
            })}
          </h3>
          <div className="space-y-1">
            {grouped.get(dateKey)!.map((event) => {
              const calendar = calMap.get(event.calendar_id);
              return (
                <div
                  key={event.id}
                  className="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 hover:bg-surface-accent/50"
                  onClick={() => onEventClick(event)}
                >
                  <div
                    className="h-3 w-3 shrink-0 rounded-full"
                    style={{ backgroundColor: getEventColor(event.color_id, calendar?.color_id ?? null) }}
                  />
                  <div className="min-w-0 flex-1">
                    <span className="text-sm text-text-primary">{event.title || "(제목 없음)"}</span>
                  </div>
                  <span className="shrink-0 text-xs text-text-muted tabular-nums inline-block min-w-[140px] text-right">
                    {event.is_all_day ? "종일" : formatTimeRangeLabel(event.start_time, event.end_time)}
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
function getDateRange(date: Date, view: CalendarViewMode, weekStartsOn: number): { start: string; end: string } {
  const target = new Date(date);
  let start: Date;
  let end: Date;

  if (view === "year-compact" || view === "year-timeline") {
    start = new Date(target.getFullYear(), 0, 1);
    end = new Date(target.getFullYear(), 11, 31, 23, 59, 59);
  } else if (view === "month") {
    return getFixedMonthFetchRangeWithWeekStart(target, weekStartsOn);
  } else if (view === "week") {
    start = new Date(target);
    start.setDate(target.getDate() - ((target.getDay() - weekStartsOn + 7) % 7));
    end = new Date(start);
    end.setDate(start.getDate() + 6);
    end.setHours(23, 59, 59);
  } else if (view === "3day") {
    start = new Date(target);
    end = new Date(target);
    end.setDate(target.getDate() + 2);
    end.setHours(23, 59, 59);
  } else {
    start = new Date(target);
    end = new Date(target);
    end.setDate(target.getDate() + 30);
  }

  return { start: start.toISOString(), end: end.toISOString() };
}

function getHeaderLabel(date: Date, view: CalendarViewMode): string {
  if (view === "year-compact" || view === "year-timeline") return `${date.getFullYear()}년`;
  if (view === "month") return `${date.getFullYear()}년 ${date.getMonth() + 1}월`;
  if (view === "week" || view === "3day") {
    return `${date.getFullYear()}년 ${date.getMonth() + 1}월 ${date.getDate()}일`;
  }
  return `${date.getFullYear()}년 ${date.getMonth() + 1}월`;
}

function formatEventTime(event: EventData): string {
  if (event.is_all_day) return "종일";
  const start = new Date(event.start_time);
  const dateLabel = start.toLocaleDateString("ko-KR", { month: "long", day: "numeric" });
  return `${dateLabel} ${formatTimeRangeLabel(event.start_time, event.end_time)}`;
}
