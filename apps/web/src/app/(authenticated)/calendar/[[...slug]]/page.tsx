"use client";

import { useState, useEffect, useCallback, useMemo, useRef, type MouseEvent as ReactMouseEvent } from "react";
import { useRouter, useParams, useSearchParams } from "next/navigation";
import { useCalendarActions } from "@/features/calendar/ui/hooks/useCalendarActions";
import { useTodoActions } from "@/features/todo/ui/hooks/useTodoActions";
import { useSettingsActions } from "@/features/settings/ui/hooks/useSettingsActions";
import { useAuthFlow } from "@/features/auth/ui/hooks/useAuthFlow";
import type { GoogleAccountSummary } from "@/features/auth/domain/AuthSession";
import type {
  CalendarData,
  CreateEventInput,
  DaySummaryData,
  EventData,
  EventPayload,
  HolidayData,
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Checkbox } from "@/components/ui/checkbox";
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
import { Badge } from "@/components/ui/badge";
import { ChevronLeft, ChevronRight, Filter, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  MULTI_ACCOUNT_FALLBACK_COLORS,
  getGoogleAccountCustomColor,
  getGoogleAccountDisplayName,
} from "@/lib/google-account-preferences";
import { getEventEndDateKey, getEventStartDateKey } from "@/lib/calendar/event-date";
import { formatDueLabel } from "@/features/todo/lib/formatDueDate";
import { RichTextDescription } from "@/lib/rich-text-description";
import { useToast } from "@/components/providers/ToastProvider";

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

interface PendingEventDeletion {
  cancel: () => void;
  flush: () => Promise<void>;
}

const COLORS = [
  "#4285f4", "#7986cb", "#33b679", "#8e24aa", "#e67c73",
  "#f6bf26", "#f4511e", "#039be5", "#616161", "#3f51b5",
  "#0b8043", "#d50000",
];

const UNASSIGNED_ACCOUNT_COLOR = "#9ca3af";
const ACCOUNT_FILTER_SETTING_KEY = "calendar_selected_google_account_ids";
const CALENDAR_LAST_SYNC_AT_SETTING_KEY = "calendar_last_sync_at";
const CALENDAR_SHOW_PUBLIC_HOLIDAYS_SETTING_KEY = "calendar_show_public_holidays";
const PAGE_ACTION_SYNC_COOLDOWN_MS = 15_000;
const DELETE_UNDO_WINDOW_MS = 5_000;

function toErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message.trim()) {
    return err.message;
  }
  return fallback;
}

function getGoogleEventColor(colorId: string | null, calColorId: string | null): string {
  const id = colorId || calColorId;
  if (!id) return COLORS[0];
  const num = parseInt(id, 10);
  return COLORS[(num - 1) % COLORS.length] || COLORS[0];
}

function parseStoredAccountIDs(value: string): string[] {
  if (!value) return [];
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function getGoogleAccountStatusLabel(status: string): string {
  if (status === "active") return "정상";
  if (status === "reauth_required") return "재인증 필요";
  if (status === "revoked") return "해지";
  return status;
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

function isValidDateKey(value: string | null | undefined): value is string {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const date = new Date(`${value}T00:00:00`);
  return !Number.isNaN(date.getTime()) && toDateStr(date) === value;
}

function toDateOnlyFromISO(value: string): string {
  if (!value) return "";
  return toDateStr(new Date(value));
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
  const searchParams = useSearchParams();
  const slug = params.slug as string[] | undefined;
  const quickAction = searchParams.get("quick");
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
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [lastSyncedAt, setLastSyncedAt] = useState<string>("");
  const [holidaysByDate, setHolidaysByDate] = useState<Map<string, string[]>>(new Map());
  const [syncingNow, setSyncingNow] = useState(false);
  const [googleAccounts, setGoogleAccounts] = useState<GoogleAccountSummary[]>([]);
  const [selectedGoogleAccountIDs, setSelectedGoogleAccountIDs] = useState<string[]>([]);
  const [settingsLoaded, setSettingsLoaded] = useState(false);

  const [quickCreateOpen, setQuickCreateOpen] = useState(false);
  const [quickDefaultStart, setQuickDefaultStart] = useState<string>("");
  const [quickDefaultEnd, setQuickDefaultEnd] = useState<string>("");
  const [quickCreateAnchor, setQuickCreateAnchor] = useState<QuickCreateAnchorPoint | null>(null);
  const quickActionHandledRef = useRef(false);
  const accountSelectionInitializedRef = useRef(false);
  const accountSelectionPersistedRef = useRef("");
  const syncThrottleRef = useRef<Record<string, number>>({});
  const pendingDeletionRef = useRef<PendingEventDeletion | null>(null);
  const locationKeyRef = useRef(buildCalendarUrl(initialView, initialDate));

  const [editorOpen, setEditorOpen] = useState(false);
  const [editorMode, setEditorMode] = useState<"create" | "edit">("create");
  const [editingEventID, setEditingEventID] = useState<string | null>(null);
  const [daySummary, setDaySummary] = useState<DaySummaryData | null>(null);
  const [daySummaryLoading, setDaySummaryLoading] = useState(false);
  const [daySummaryError, setDaySummaryError] = useState("");
  const [yearTimelineTodosByDate, setYearTimelineTodosByDate] = useState<Map<string, DaySummaryData["todos"]>>(new Map());
  const [yearTimelineTodosLoadedKey, setYearTimelineTodosLoadedKey] = useState("");
  const [yearTimelineTodosLoadingKey, setYearTimelineTodosLoadingKey] = useState("");
  const [editorForm, setEditorForm] = useState<EventEditorForm>(
    makeDefaultEditorForm(new Date(), new Date(Date.now() + 60 * 60 * 1000), "")
  );

  const {
    listCalendars,
    getSettings,
    listEvents,
    listHolidays,
    getDaySummary,
    backfillEvents,
    createEvent,
    updateEvent,
    deleteEvent,
  } = useCalendarActions();
  const { listLists: listTodoLists, listTodos } = useTodoActions();
  const { updateSetting } = useSettingsActions();
  const { listGoogleAccounts, triggerGoogleSync } = useAuthFlow();
  const toast = useToast();

  const timezone = useMemo(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Seoul",
    []
  );

  const activeGoogleAccounts = useMemo(
    () => googleAccounts.filter((account) => account.status === "active"),
    [googleAccounts]
  );

  const syncEnabledGoogleAccounts = useMemo(
    () =>
      activeGoogleAccounts.filter(
        (account) => settings[`google_account_sync_calendar_${account.id}`] !== "false",
      ),
    [activeGoogleAccounts, settings],
  );
  const syncEnabledGoogleAccountIDs = useMemo(
    () => syncEnabledGoogleAccounts.map((account) => account.id),
    [syncEnabledGoogleAccounts],
  );
  const syncEnabledGoogleAccountSet = useMemo(
    () => new Set(syncEnabledGoogleAccountIDs),
    [syncEnabledGoogleAccountIDs],
  );

  const selectableGoogleAccountIDs = useMemo(
    () => selectedGoogleAccountIDs.filter((accountID) => syncEnabledGoogleAccountSet.has(accountID)),
    [selectedGoogleAccountIDs, syncEnabledGoogleAccountSet],
  );
  const selectedGoogleAccountSet = useMemo(
    () => new Set(selectableGoogleAccountIDs),
    [selectableGoogleAccountIDs]
  );
  const effectiveSelectedGoogleAccountSet = useMemo(
    () => (selectedGoogleAccountSet.size > 0 ? selectedGoogleAccountSet : syncEnabledGoogleAccountSet),
    [selectedGoogleAccountSet, syncEnabledGoogleAccountSet],
  );
  const selectedSyncEnabledGoogleAccountIDs = useMemo(
    () => syncEnabledGoogleAccountIDs.filter((id) => effectiveSelectedGoogleAccountSet.has(id)),
    [syncEnabledGoogleAccountIDs, effectiveSelectedGoogleAccountSet],
  );
  const useAccountUnifiedColors = selectedSyncEnabledGoogleAccountIDs.length > 1;
  const accountColorMap = useMemo(() => {
    const map = new Map<string, string>();
    selectedSyncEnabledGoogleAccountIDs.forEach((accountID, index) => {
      const fallbackColor = MULTI_ACCOUNT_FALLBACK_COLORS[index % MULTI_ACCOUNT_FALLBACK_COLORS.length];
      map.set(accountID, getGoogleAccountCustomColor(settings, accountID) || fallbackColor);
    });
    return map;
  }, [selectedSyncEnabledGoogleAccountIDs, settings]);

  const allSyncEnabledAccountsSelected =
    syncEnabledGoogleAccountIDs.length > 0 &&
    syncEnabledGoogleAccountIDs.every((accountID) => effectiveSelectedGoogleAccountSet.has(accountID));

  const accountFilterLabel = allSyncEnabledAccountsSelected
    ? "계정: 전체"
    : `계정: ${selectedSyncEnabledGoogleAccountIDs.length}/${syncEnabledGoogleAccountIDs.length}`;
  const showPublicHolidays = settings[CALENDAR_SHOW_PUBLIC_HOLIDAYS_SETTING_KEY] !== "false";
  const yearViewFocusDate = useMemo(() => {
    if (view !== "year-compact" && view !== "year-timeline") return null;
    const candidate = searchParams.get("focusDate");
    if (isValidDateKey(candidate)) {
      const focusYear = Number.parseInt(candidate.slice(0, 4), 10);
      if (focusYear === currentDate.getFullYear()) {
        return candidate;
      }
    }
    const today = new Date();
    if (today.getFullYear() === currentDate.getFullYear()) {
      return toDateStr(today);
    }
    return `${currentDate.getFullYear()}-01-01`;
  }, [currentDate, searchParams, view]);
  const yearTimelineTodoCacheKey = useMemo(
    () => `${currentDate.getFullYear()}|${Array.from(effectiveSelectedGoogleAccountSet).sort().join(",")}`,
    [currentDate, effectiveSelectedGoogleAccountSet],
  );

  const accountDisplayNameByID = useMemo(
    () =>
      new Map(
        googleAccounts.map((account) => [
          account.id,
          getGoogleAccountDisplayName(settings, account.id, account.google_email),
        ]),
      ),
    [googleAccounts, settings]
  );

  const isCalendarIncludedByAccountFilter = useCallback(
    (calendar: CalendarData) => {
      if (!calendar.google_account_id) return true;
      if (!syncEnabledGoogleAccountSet.has(calendar.google_account_id)) return false;
      return effectiveSelectedGoogleAccountSet.has(calendar.google_account_id);
    },
    [effectiveSelectedGoogleAccountSet, syncEnabledGoogleAccountSet]
  );

  const filteredCalendars = useMemo(
    () =>
      calendars.filter((calendar) => {
        if (!isCalendarIncludedByAccountFilter(calendar)) return false;
        if (calendar.is_special || calendar.kind === "holiday" || calendar.kind === "birthday") return false;
        return true;
      }),
    [calendars, isCalendarIncludedByAccountFilter]
  );
  const writableFilteredCalendars = useMemo(
    () => filteredCalendars.filter((calendar) => !calendar.is_readonly),
    [filteredCalendars]
  );
  const filteredCalendarIDs = useMemo(
    () => filteredCalendars.map((calendar) => calendar.id),
    [filteredCalendars]
  );

  const defaultCalendarID = useMemo(
    () => writableFilteredCalendars.find((cal) => cal.is_visible)?.id || writableFilteredCalendars[0]?.id || "",
    [writableFilteredCalendars]
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
      setSettings(settings);
      setWeekHours(parseWeekHourRange(settings));
      setWeekStartsOn(parseWeekStartsOn(settings));
      setLastSyncedAt(settings[CALENDAR_LAST_SYNC_AT_SETTING_KEY] || "");
      const storedAccountIDs = parseStoredAccountIDs(settings[ACCOUNT_FILTER_SETTING_KEY] || "");
      setSelectedGoogleAccountIDs(storedAccountIDs);
      accountSelectionPersistedRef.current = storedAccountIDs.join(",");
    } catch {
      setSettings({});
      setWeekHours({ start: 8, end: 22 });
      setWeekStartsOn(0);
      setLastSyncedAt("");
      setSelectedGoogleAccountIDs([]);
      accountSelectionPersistedRef.current = "";
    } finally {
      setSettingsLoaded(true);
    }
  }, [getSettings]);

  const loadGoogleAccounts = useCallback(async () => {
    try {
      const accounts = await listGoogleAccounts();
      setGoogleAccounts(accounts || []);
    } catch {
      setGoogleAccounts([]);
    }
  }, [listGoogleAccounts]);

  const loadEvents = useCallback(async () => {
    setLoading(true);
    const { start, end } = getDateRange(currentDate, view, weekStartsOn);
    try {
      if (calendars.length > 0 && filteredCalendarIDs.length === 0) {
        setEvents([]);
        setLoading(false);
        return;
      }

      const shouldFilterByCalendarIDs =
        filteredCalendarIDs.length > 0 && filteredCalendarIDs.length !== calendars.length;
      const initial = await listEvents(start, end, shouldFilterByCalendarIDs ? filteredCalendarIDs : undefined);
      setEvents(initial || []);
      setLoading(false);

      const startAt = new Date(start).getTime();
      const endAt = new Date(end).getTime();
      const needsBackfillCalendarIDs = filteredCalendars
        .filter((calendar) => !!calendar.google_account_id)
        .filter((calendar) => {
          if (!calendar.synced_start || !calendar.synced_end) return true;
          const syncedStart = new Date(calendar.synced_start).getTime();
          const syncedEnd = new Date(calendar.synced_end).getTime();
          return startAt < syncedStart || endAt > syncedEnd;
        })
        .map((calendar) => calendar.id);
      if (needsBackfillCalendarIDs.length > 0) {
        await backfillEvents(start, end, needsBackfillCalendarIDs);
        setCalendars((prev) =>
          prev.map((calendar) => {
            if (!needsBackfillCalendarIDs.includes(calendar.id)) return calendar;
            const nextStart = calendar.synced_start
              ? new Date(Math.min(new Date(calendar.synced_start).getTime(), startAt)).toISOString()
              : new Date(startAt).toISOString();
            const nextEnd = calendar.synced_end
              ? new Date(Math.max(new Date(calendar.synced_end).getTime(), endAt)).toISOString()
              : new Date(endAt).toISOString();
            return { ...calendar, synced_start: nextStart, synced_end: nextEnd };
          }),
        );
        const merged = await listEvents(start, end, shouldFilterByCalendarIDs ? filteredCalendarIDs : undefined);
        setEvents(merged || []);
      }
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [backfillEvents, calendars.length, currentDate, filteredCalendarIDs, filteredCalendars, listEvents, view, weekStartsOn]);

  const loadHolidays = useCallback(async () => {
    if (!showPublicHolidays) {
      setHolidaysByDate(new Map());
      return;
    }
    const { start, end } = getDateRange(currentDate, view, weekStartsOn);
    const startDate = toDateOnlyFromISO(start);
    const endDate = toDateOnlyFromISO(end);
    if (!startDate || !endDate) {
      setHolidaysByDate(new Map());
      return;
    }
    try {
      const holidays = await listHolidays(startDate, endDate);
      const grouped = new Map<string, string[]>();
      (holidays || []).forEach((holiday: HolidayData) => {
        if (!holiday.date || !holiday.name) return;
        const prev = grouped.get(holiday.date) || [];
        if (!prev.includes(holiday.name)) {
          grouped.set(holiday.date, [...prev, holiday.name]);
        }
      });
      setHolidaysByDate(grouped);
    } catch {
      setHolidaysByDate(new Map());
    }
  }, [currentDate, listHolidays, showPublicHolidays, view, weekStartsOn]);

  const loadTodosForDate = useCallback(async (dateKey: string): Promise<DaySummaryData["todos"]> => {
    const lists = await listTodoLists();
    if (!lists || lists.length === 0) {
      return [];
    }
    const filteredLists = lists.filter((list) => {
      if (!list.google_account_id) return true;
      return effectiveSelectedGoogleAccountSet.has(list.google_account_id);
    });
    if (filteredLists.length === 0) {
      return [];
    }

    const grouped = await Promise.all(
      filteredLists.map(async (list) => {
        try {
          const items = await listTodos(list.id, false);
          return items.map((todo) => ({ listID: list.id, todo }));
        } catch {
          return [];
        }
      }),
    );

    const flattened = grouped.flat();
    const filtered = flattened
      .filter(({ todo }) => (todo.due_date || "").slice(0, 10) === dateKey)
      .map(({ listID, todo }) => ({
        id: todo.id,
        list_id: todo.list_id || listID,
        title: todo.title || "(제목 없음)",
        due_date: todo.due_date || null,
        due_time: todo.due_time || null,
        is_done: todo.is_done,
      }))
      .sort((a, b) => {
        const timeA = a.due_time || "99:99";
        const timeB = b.due_time || "99:99";
        if (timeA !== timeB) return timeA.localeCompare(timeB);
        return (a.title || "").localeCompare(b.title || "");
      });

    return filtered;
  }, [effectiveSelectedGoogleAccountSet, listTodoLists, listTodos]);

  useEffect(() => {
    if (view !== "year-timeline") return;
    if (yearTimelineTodosLoadedKey === yearTimelineTodoCacheKey) return;
    if (yearTimelineTodosLoadingKey === yearTimelineTodoCacheKey) return;

    let cancelled = false;
    const year = currentDate.getFullYear();
    const startKey = `${year}-01-01`;
    const endKey = `${year}-12-31`;

    const loadYearTimelineTodos = async () => {
      setYearTimelineTodosLoadingKey(yearTimelineTodoCacheKey);
      setYearTimelineTodosByDate(new Map());
      try {
        const lists = await listTodoLists();
        if (!lists || lists.length === 0) {
          if (!cancelled) {
            setYearTimelineTodosByDate(new Map());
            setYearTimelineTodosLoadedKey(yearTimelineTodoCacheKey);
          }
          return;
        }

        const filteredLists = lists.filter((list) => {
          if (!list.google_account_id) return true;
          return effectiveSelectedGoogleAccountSet.has(list.google_account_id);
        });
        if (filteredLists.length === 0) {
          if (!cancelled) {
            setYearTimelineTodosByDate(new Map());
            setYearTimelineTodosLoadedKey(yearTimelineTodoCacheKey);
          }
          return;
        }

        const grouped = await Promise.all(
          filteredLists.map(async (list) => {
            try {
              const items = await listTodos(list.id, false);
              return items.map((todo) => ({ listID: list.id, todo }));
            } catch {
              return [];
            }
          }),
        );

        if (cancelled) return;

        const byDate = new Map<string, DaySummaryData["todos"]>();
        grouped.flat().forEach(({ listID, todo }) => {
          const dueDate = (todo.due_date || "").slice(0, 10);
          if (!isValidDateKey(dueDate)) return;
          if (dueDate < startKey || dueDate > endKey) return;

          const prev = byDate.get(dueDate) || [];
          prev.push({
            id: todo.id,
            list_id: todo.list_id || listID,
            title: todo.title || "(제목 없음)",
            due_date: todo.due_date || null,
            due_time: todo.due_time || null,
            is_done: todo.is_done,
          });
          byDate.set(dueDate, prev);
        });

        for (const [dateKey, items] of byDate.entries()) {
          items.sort((a, b) => {
            const timeA = a.due_time || "99:99";
            const timeB = b.due_time || "99:99";
            if (timeA !== timeB) return timeA.localeCompare(timeB);
            return (a.title || "").localeCompare(b.title || "");
          });
          byDate.set(dateKey, items);
        }

        setYearTimelineTodosByDate(byDate);
        setYearTimelineTodosLoadedKey(yearTimelineTodoCacheKey);
      } catch {
        if (!cancelled) {
          setYearTimelineTodosByDate(new Map());
          setYearTimelineTodosLoadedKey(yearTimelineTodoCacheKey);
        }
      } finally {
        setYearTimelineTodosLoadingKey((prev) => (prev === yearTimelineTodoCacheKey ? "" : prev));
      }
    };

    void loadYearTimelineTodos();

    return () => {
      cancelled = true;
    };
  }, [
    currentDate,
    effectiveSelectedGoogleAccountSet,
    listTodoLists,
    listTodos,
    view,
    yearTimelineTodoCacheKey,
    yearTimelineTodosLoadedKey,
    yearTimelineTodosLoadingKey,
  ]);

  const loadDaySummary = useCallback(async () => {
    if ((view !== "year-compact" && view !== "year-timeline") || !yearViewFocusDate) {
      setDaySummary(null);
      setDaySummaryError("");
      setDaySummaryLoading(false);
      return;
    }

    setDaySummaryLoading(true);
    setDaySummaryError("");
    try {
      const summary = await getDaySummary(
        yearViewFocusDate,
        timezone,
        filteredCalendarIDs.length > 0 ? filteredCalendarIDs : undefined,
        false
      );
      const allowedCalendarIDs = new Set(filteredCalendarIDs);
      const visibleEvents =
        calendars.length > 0
          ? (summary.events || []).filter((event) => allowedCalendarIDs.has(event.calendar_id))
          : summary.events || [];
      const visibleTodos =
        (summary.todos || []).length > 0
          ? summary.todos || []
          : (view === "year-timeline" ? yearTimelineTodosByDate.get(yearViewFocusDate) : undefined) ||
            await loadTodosForDate(yearViewFocusDate).catch(() => []);
      setDaySummary({ ...summary, events: visibleEvents, todos: visibleTodos });
    } catch {
      const allowedCalendarIDs = new Set(filteredCalendarIDs);
      const fallbackEvents = (events || []).filter((event) => {
        if (allowedCalendarIDs.size > 0 && !allowedCalendarIDs.has(event.calendar_id)) return false;
        const start = getEventStartDateKey(event);
        const end = getEventEndDateKey(event);
        return yearViewFocusDate >= start && yearViewFocusDate <= end;
      });
      const fallbackHolidays = (holidaysByDate.get(yearViewFocusDate) || []).map((name) => ({
        date: yearViewFocusDate,
        name,
      }));
      const fallbackTodos =
        (view === "year-timeline" ? yearTimelineTodosByDate.get(yearViewFocusDate) : undefined) ||
        await loadTodosForDate(yearViewFocusDate).catch(() => []);
      setDaySummary({
        date: yearViewFocusDate,
        timezone,
        holidays: fallbackHolidays,
        events: fallbackEvents,
        todos: fallbackTodos,
      });
      setDaySummaryError("");
    } finally {
      setDaySummaryLoading(false);
    }
  }, [
    calendars.length,
    events,
    filteredCalendarIDs,
    getDaySummary,
    holidaysByDate,
    loadTodosForDate,
    timezone,
    view,
    yearTimelineTodosByDate,
    yearViewFocusDate,
  ]);

  const triggerCalendarSync = useCallback(
    async (reason: "page_enter" | "page_action" | "tab_heartbeat" | "manual", throttleMs = 0) => {
      const key = `calendar:${reason}`;
      const now = Date.now();
      const last = syncThrottleRef.current[key] || 0;
      if (throttleMs > 0 && now - last < throttleMs) {
        return 0;
      }
      syncThrottleRef.current[key] = now;
      const scheduled = await triggerGoogleSync({ area: "calendar", reason });
      if (scheduled > 0) {
        const stamp = new Date().toISOString();
        setLastSyncedAt(stamp);
        updateSetting(CALENDAR_LAST_SYNC_AT_SETTING_KEY, stamp).catch(() => undefined);
      }
      return scheduled;
    },
    [triggerGoogleSync, updateSetting]
  );

  useEffect(() => { loadCalendars(); }, [loadCalendars]);
  useEffect(() => { loadSettings(); }, [loadSettings]);
  useEffect(() => { loadGoogleAccounts(); }, [loadGoogleAccounts]);
  useEffect(() => { loadEvents(); }, [loadEvents]);
  useEffect(() => { loadHolidays(); }, [loadHolidays]);
  useEffect(() => { loadDaySummary(); }, [loadDaySummary]);
  useEffect(() => {
    void triggerCalendarSync("page_enter", 60_000);
  }, [triggerCalendarSync]);
  useEffect(() => {
    const timer = window.setInterval(() => {
      void triggerCalendarSync("tab_heartbeat", 9 * 60_000);
    }, 10 * 60_000);
    return () => window.clearInterval(timer);
  }, [triggerCalendarSync]);
  useEffect(() => {
    const nextKey = buildCalendarUrl(view, currentDate);
    if (locationKeyRef.current !== nextKey && pendingDeletionRef.current) {
      void pendingDeletionRef.current.flush();
    }
    locationKeyRef.current = nextKey;
  }, [currentDate, view]);
  useEffect(() => {
    return () => {
      if (pendingDeletionRef.current) {
        void pendingDeletionRef.current.flush();
      }
    };
  }, []);
  useEffect(() => {
    if (!settingsLoaded) return;
    if (accountSelectionInitializedRef.current) return;

    const activeSet = new Set(syncEnabledGoogleAccountIDs);
    const validStored = selectedGoogleAccountIDs.filter((accountID) => activeSet.has(accountID));
    const normalized = validStored.length > 0 ? validStored : syncEnabledGoogleAccountIDs;

    setSelectedGoogleAccountIDs(normalized);
    accountSelectionInitializedRef.current = true;

    const serialized = normalized.join(",");
    if (serialized !== accountSelectionPersistedRef.current) {
      accountSelectionPersistedRef.current = serialized;
      updateSetting(ACCOUNT_FILTER_SETTING_KEY, serialized).catch(() => undefined);
    }
  }, [selectedGoogleAccountIDs, settingsLoaded, syncEnabledGoogleAccountIDs, updateSetting]);

  useEffect(() => {
    if (!accountSelectionInitializedRef.current) return;
    const serialized = selectedGoogleAccountIDs.join(",");
    if (serialized === accountSelectionPersistedRef.current) return;

    accountSelectionPersistedRef.current = serialized;
    updateSetting(ACCOUNT_FILTER_SETTING_KEY, serialized).catch(() => undefined);
  }, [selectedGoogleAccountIDs, updateSetting]);
  useEffect(() => {
    if (quickAction !== "create") return;
    if (quickActionHandledRef.current) return;
    if (!defaultCalendarID) return;

    quickActionHandledRef.current = true;

    const now = new Date();
    const start = new Date(now);
    start.setMinutes(0, 0, 0);
    start.setHours(start.getHours() + 1);
    const end = new Date(start.getTime() + 60 * 60 * 1000);

    setQuickDefaultStart(toLocalDateTimeInput(start));
    setQuickDefaultEnd(toLocalDateTimeInput(end));
    setQuickCreateAnchor(null);
    setSelectedDateKey(toDateStr(start));
    setQuickCreateOpen(true);

    router.replace(buildCalendarUrl(view, currentDate), { scroll: false });
  }, [quickAction, defaultCalendarID, router, view, currentDate]);

  const updateUrl = useCallback((newView: CalendarViewMode, date: Date) => {
    router.replace(buildCalendarUrl(newView, date), { scroll: false });
  }, [router]);

  const setYearViewPanelDate = useCallback(
    (dateKey: string, baseDate?: Date, yearView: "year-compact" | "year-timeline" = "year-compact") => {
      const base = buildCalendarUrl(yearView, baseDate || currentDate);
      const params = new URLSearchParams(searchParams.toString());
      params.set("panel", "day");
      params.set("focusDate", dateKey);
      router.replace(`${base}?${params.toString()}`, { scroll: false });
    },
    [currentDate, router, searchParams]
  );

  useEffect(() => {
    if ((view !== "year-compact" && view !== "year-timeline") || !yearViewFocusDate) return;
    const currentFocusDate = searchParams.get("focusDate");
    const currentPanel = searchParams.get("panel");
    if (currentFocusDate === yearViewFocusDate && currentPanel === "day") return;
    setYearViewPanelDate(yearViewFocusDate, undefined, view);
  }, [searchParams, setYearViewPanelDate, view, yearViewFocusDate]);

  const handleViewChange = (newView: CalendarViewMode) => {
    setView(newView);
    updateUrl(newView, currentDate);
  };

  const handleSelectAllAccounts = () => {
    setSelectedGoogleAccountIDs(syncEnabledGoogleAccountIDs);
  };

  const handleToggleAccount = (accountID: string) => {
    setSelectedGoogleAccountIDs((prev) => {
      const exists = prev.includes(accountID);
      if (exists) {
        if (prev.length <= 1 && syncEnabledGoogleAccountIDs.length > 0) {
          return prev;
        }
        return prev.filter((id) => id !== accountID);
      }
      return [...prev, accountID];
    });
  };

  const handleManualSync = async () => {
    if (syncingNow) return;
    setSyncingNow(true);
    try {
      await triggerCalendarSync("manual", 0);
      await loadCalendars();
      await loadEvents();
    } catch {
      // noop
    } finally {
      setSyncingNow(false);
    }
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
    const todayKey = toDateStr(today);
    setSelectedDateKey(todayKey);
    if (view === "year-compact" || view === "year-timeline") {
      setYearViewPanelDate(todayKey, today, view);
      return;
    }
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
      void triggerCalendarSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
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
      void triggerCalendarSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      console.error("Save event failed:", err);
    }
  };

  const handleDeleteEvent = async (eventID: string) => {
    if (pendingDeletionRef.current) {
      await pendingDeletionRef.current.flush();
    }

    const eventsSnapshot = events;
    if (!eventsSnapshot.some((event) => event.id === eventID)) return;

    let settled = false;
    setEvents((prev) => prev.filter((event) => event.id !== eventID));
    setSelectedEvent(null);

    let timerID = 0;
    const restore = () => {
      setEvents(eventsSnapshot);
    };

    const finalize = async () => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timerID);
      pendingDeletionRef.current = null;

      try {
        await deleteEvent(eventID);
        await loadEvents();
        void triggerCalendarSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
      } catch (err) {
        restore();
        console.error("Delete event failed:", err);
        toast.error("일정 삭제 실패", toErrorMessage(err, "일정을 삭제하지 못했습니다."));
      }
    };

    const cancel = () => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timerID);
      pendingDeletionRef.current = null;
      restore();
      toast.success("복원됨");
    };

    pendingDeletionRef.current = { cancel, flush: finalize };
    timerID = window.setTimeout(() => {
      void finalize();
    }, DELETE_UNDO_WINDOW_MS);

    toast.show({
      variant: "warning",
      title: "일정 삭제됨",
      duration: DELETE_UNDO_WINDOW_MS,
      actionLabel: "실행 취소",
      onAction: cancel,
    });
  };

  const headerLabel = getHeaderLabel(currentDate, view);
  const calMap = useMemo(() => new Map(calendars.map((cal) => [cal.id, cal])), [calendars]);
  const displayEvents = useMemo(() => {
    return events.filter((event) => {
      const calendar = calMap.get(event.calendar_id);
      if (calendar?.is_special) return false;
      if (calendar?.kind === "holiday" || calendar?.kind === "birthday") return false;
      return true;
    });
  }, [calMap, events]);
  const selectedEventCalendar = selectedEvent ? calMap.get(selectedEvent.calendar_id) : undefined;
  const isSelectedEventReadOnly = !!selectedEventCalendar?.is_readonly;
  const formatCalendarLabel = useCallback((calendar: CalendarData) => {
    const badges: string[] = [];
    if (calendar.is_readonly) badges.push("읽기 전용");
    if (badges.length === 0) return calendar.name;
    return `${calendar.name} · ${badges.join(" · ")}`;
  }, []);
  const getDisplayEventColor = useCallback(
    (eventColorID: string | null, calendarID: string): string => {
      const calendar = calMap.get(calendarID);
      if (useAccountUnifiedColors) {
        const accountID = calendar?.google_account_id || "";
        if (accountID && accountColorMap.has(accountID)) {
          return accountColorMap.get(accountID) || UNASSIGNED_ACCOUNT_COLOR;
        }
        return UNASSIGNED_ACCOUNT_COLOR;
      }
      return getGoogleEventColor(eventColorID, calendar?.color_id ?? null);
    },
    [accountColorMap, calMap, useAccountUnifiedColors]
  );

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
          {syncEnabledGoogleAccountIDs.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="secondary" size="sm" className="gap-1.5">
                  <Filter size={14} />
                  {accountFilterLabel}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-[280px]">
                <DropdownMenuLabel>Google 계정 필터</DropdownMenuLabel>
                <DropdownMenuItem
                  onSelect={(event) => {
                    event.preventDefault();
                    handleSelectAllAccounts();
                  }}
                >
                  <Checkbox checked={allSyncEnabledAccountsSelected} />
                  <span>전체 선택</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {googleAccounts
                  .filter((account) => syncEnabledGoogleAccountSet.has(account.id))
                  .map((account) => {
                  const checked = selectedGoogleAccountSet.has(account.id);
                  const isActive = account.status === "active";
                  const displayName = getGoogleAccountDisplayName(settings, account.id, account.google_email);
                  const statusLabel = getGoogleAccountStatusLabel(account.status);
                  const accountColor = accountColorMap.get(account.id) || UNASSIGNED_ACCOUNT_COLOR;

                  return (
                    <DropdownMenuItem
                      key={account.id}
                      className={!isActive ? "opacity-60" : ""}
                      onSelect={(event) => {
                        event.preventDefault();
                        if (!isActive) return;
                        handleToggleAccount(account.id);
                      }}
                    >
                      <Checkbox checked={checked} disabled={!isActive} />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm">
                          <span
                            className="mr-1 inline-block h-2 w-2 rounded-full align-middle"
                            style={{ backgroundColor: accountColor }}
                          />
                          {displayName}
                        </p>
                        <p className="text-xs text-text-muted">{statusLabel}</p>
                      </div>
                      {account.is_primary ? <Badge variant="primary">기본</Badge> : null}
                    </DropdownMenuItem>
                  );
                })}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
          <div className="hidden items-center gap-2 text-xs text-text-muted md:flex">
            <span>최근 동기화: {lastSyncedAt ? new Date(lastSyncedAt).toLocaleString("ko-KR") : "-"}</span>
          </div>
          <Button
            variant="secondary"
            size="icon-sm"
            onClick={handleManualSync}
            disabled={syncingNow}
            title="지금 동기화"
          >
            <RefreshCw className={cn("h-4 w-4", syncingNow && "animate-spin")} />
          </Button>
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
      {useAccountUnifiedColors && selectedSyncEnabledGoogleAccountIDs.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 border-b border-border/70 px-4 py-2 text-xs text-text-muted">
          <span className="font-medium text-text-secondary">계정 색상</span>
          {selectedSyncEnabledGoogleAccountIDs.map((accountID) => (
            <span key={accountID} className="inline-flex items-center gap-1.5 rounded-full border border-border/70 px-2 py-1">
              <span
                className="h-2.5 w-2.5 rounded-full"
                style={{ backgroundColor: accountColorMap.get(accountID) || UNASSIGNED_ACCOUNT_COLOR }}
              />
              <span className="max-w-[12rem] truncate">
                {accountDisplayNameByID.get(accountID) || "계정 미확인"}
              </span>
            </span>
          ))}
        </div>
      )}

      <div className="flex-1 min-h-0 overflow-auto">
        {loading && events.length === 0 ? (
          <div className="flex items-center justify-center py-20 text-text-muted">불러오는 중...</div>
        ) : view === "year-compact" ? (
          <div className="flex h-full min-h-0 flex-col lg:flex-row">
            <div className="min-h-0 flex-1">
              <YearCompactView
                year={currentDate.getFullYear()}
                events={displayEvents}
                weekStartsOn={weekStartsOn}
                holidaysByDate={holidaysByDate}
                selectedDateKey={yearViewFocusDate}
                onMonthClick={(month) => {
                  const date = new Date(currentDate.getFullYear(), month, 1);
                  setView("month");
                  setCurrentDate(date);
                  updateUrl("month", date);
                }}
                onDateClick={(_, dateKey) => {
                  setSelectedDateKey(dateKey);
                  setYearViewPanelDate(dateKey, undefined, "year-compact");
                }}
              />
            </div>
            {yearViewFocusDate ? (
              <YearCompactDayPanel
                dateKey={yearViewFocusDate}
                summary={daySummary}
                loading={daySummaryLoading}
                error={daySummaryError}
                onEventClick={setSelectedEvent}
                getEventColorByCalendar={getDisplayEventColor}
              />
            ) : null}
          </div>
        ) : view === "year-timeline" ? (
          <div className="flex h-full min-h-0 flex-col lg:flex-row">
            <div className="min-h-0 flex-1">
              <YearTimelineView
                year={currentDate.getFullYear()}
                events={displayEvents}
                holidaysByDate={holidaysByDate}
                todosByDate={yearTimelineTodosByDate}
                selectedDateKey={yearViewFocusDate}
                onDateClick={(_, dateKey) => {
                  setSelectedDateKey(dateKey);
                  setYearViewPanelDate(dateKey, undefined, "year-timeline");
                }}
                getEventColor={(colorID, calendar) => {
                  if (useAccountUnifiedColors) {
                    const accountID = calendar?.google_account_id || "";
                    if (accountID && accountColorMap.has(accountID)) {
                      return accountColorMap.get(accountID) || UNASSIGNED_ACCOUNT_COLOR;
                    }
                    return UNASSIGNED_ACCOUNT_COLOR;
                  }
                  return getGoogleEventColor(colorID, calendar?.color_id ?? null);
                }}
                calendars={filteredCalendars}
              />
            </div>
            {yearViewFocusDate ? (
              <YearCompactDayPanel
                dateKey={yearViewFocusDate}
                summary={daySummary}
                loading={daySummaryLoading}
                error={daySummaryError}
                onEventClick={setSelectedEvent}
                getEventColorByCalendar={getDisplayEventColor}
              />
            ) : null}
          </div>
        ) : view === "month" ? (
          <MonthView
            currentDate={currentDate}
            events={displayEvents}
            holidaysByDate={holidaysByDate}
            weekStartsOn={weekStartsOn}
            selectedDateKey={selectedDateKey}
            getEventColorByCalendar={getDisplayEventColor}
            onDayClick={handleDayClick}
            onEventClick={setSelectedEvent}
          />
        ) : view === "week" || view === "3day" ? (
          <WeekView
            currentDate={currentDate}
            events={displayEvents}
            holidaysByDate={holidaysByDate}
            days={view === "week" ? 7 : 3}
            startHour={weekHours.start}
            endHour={weekHours.end}
            weekStartsOn={weekStartsOn}
            getEventColorByCalendar={getDisplayEventColor}
            onSlotClick={handleWeekSlotClick}
            onEventClick={setSelectedEvent}
          />
        ) : (
          <AgendaView
            events={displayEvents}
            holidaysByDate={holidaysByDate}
            calendars={filteredCalendars}
            getEventColorByCalendar={getDisplayEventColor}
            onEventClick={setSelectedEvent}
          />
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
                  backgroundColor: getDisplayEventColor(selectedEvent.color_id, selectedEvent.calendar_id),
                }}
              />
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-medium text-text-strong">{selectedEvent.title || "(제목 없음)"}</h3>
                <p className="mt-1 text-xs text-text-secondary">{formatEventTime(selectedEvent)}</p>
                {selectedEvent.location && (
                  <p className="mt-1 text-xs text-text-muted">{selectedEvent.location}</p>
                )}
                <RichTextDescription value={selectedEvent.description} />
              </div>
            </div>
            <div className="mt-3 flex justify-between gap-2">
              <Button
                variant="danger"
                size="sm"
                disabled={isSelectedEventReadOnly}
                onClick={() => handleDeleteEvent(selectedEvent.id)}
              >
                {isSelectedEventReadOnly ? "읽기 전용" : "삭제"}
              </Button>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={isSelectedEventReadOnly}
                  onClick={() => openEditEditor(selectedEvent)}
                >
                  {isSelectedEventReadOnly ? "수정 불가" : "수정"}
                </Button>
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
                  {writableFilteredCalendars.map((calendar) => (
                    <SelectItem key={calendar.id} value={calendar.id}>
                      {formatCalendarLabel(calendar)}
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

function YearCompactDayPanel({
  dateKey,
  summary,
  loading,
  error,
  onEventClick,
  getEventColorByCalendar,
}: {
  dateKey: string;
  summary: DaySummaryData | null;
  loading: boolean;
  error: string;
  onEventClick: (event: EventData) => void;
  getEventColorByCalendar: (eventColorID: string | null, calendarID: string) => string;
}) {
  const date = new Date(`${dateKey}T00:00:00`);
  const title = Number.isNaN(date.getTime())
    ? dateKey
    : date.toLocaleDateString("ko-KR", {
        year: "numeric",
        month: "long",
        day: "numeric",
        weekday: "short",
      });
  const hasHolidays = !!summary && summary.holidays.length > 0;
  const hasEvents = !!summary && summary.events.length > 0;
  const hasTodos = !!summary && summary.todos.length > 0;
  const hasAnyData = hasHolidays || hasEvents || hasTodos;

  return (
    <aside className="w-full shrink-0 border-t border-border bg-background lg:h-full lg:w-[340px] lg:border-l lg:border-t-0">
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div>
            <p className="text-xs text-text-muted">선택 날짜</p>
            <h3 className="text-sm font-semibold text-text-strong">{title}</h3>
          </div>
        </div>
        <div className="flex-1 overflow-auto p-4">
          {loading ? (
            <p className="text-sm text-text-muted">일정을 불러오는 중...</p>
          ) : error ? (
            <p className="text-sm text-error">{error}</p>
          ) : !summary ? (
            <p className="text-sm text-text-muted">표시할 데이터가 없습니다.</p>
          ) : (
            <>
              {!hasAnyData ? (
                <p className="text-sm text-text-muted">표시할 데이터가 없습니다.</p>
              ) : (
                <div className="space-y-5">
                  {hasHolidays ? (
                    <section>
                      <p className="mb-2 text-xs font-medium text-text-muted">공휴일</p>
                      <div className="space-y-1">
                        {summary.holidays.map((holiday) => (
                          <p key={`${holiday.date}-${holiday.name}`} className="text-sm font-medium text-error">
                            {holiday.name}
                          </p>
                        ))}
                      </div>
                    </section>
                  ) : null}

                  {hasEvents ? (
                    <section>
                      <p className="mb-2 text-xs font-medium text-text-muted">일정</p>
                      <div className="space-y-1.5">
                        {summary.events.map((event) => (
                          <button
                            key={event.id}
                            type="button"
                            className="flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-surface-accent/60"
                            onClick={() => onEventClick(event)}
                          >
                            <span
                              className="mt-1.5 h-2 w-2 shrink-0 rounded-full"
                              style={{ backgroundColor: getEventColorByCalendar(event.color_id, event.calendar_id) }}
                            />
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-sm text-text-primary">{event.title || "(제목 없음)"}</span>
                              <span className="block text-xs text-text-muted">{formatEventTime(event)}</span>
                            </span>
                          </button>
                        ))}
                      </div>
                    </section>
                  ) : null}

                  {hasTodos ? (
                    <section>
                      <p className="mb-2 text-xs font-medium text-text-muted">Todo</p>
                      <div className="space-y-1.5">
                        {summary.todos.map((todo) => {
                          const dueLabel = formatDueLabel(todo.due_date, todo.due_time);
                          return (
                            <div key={todo.id} className="rounded-md border border-border/70 px-2 py-1.5">
                              <p className={cn("text-sm", todo.is_done ? "text-text-muted line-through" : "text-text-primary")}>
                                {todo.title}
                              </p>
                              <p className="text-[11px] text-text-muted">
                                {dueLabel ? `기한 ${dueLabel}` : "기한 없음"}
                                {todo.is_done ? " · 완료" : ""}
                              </p>
                            </div>
                          );
                        })}
                      </div>
                    </section>
                  ) : null}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </aside>
  );
}

/* ===== Month View with 42 Cells ===== */
function MonthView({
  currentDate,
  events,
  holidaysByDate,
  weekStartsOn,
  selectedDateKey,
  getEventColorByCalendar,
  onDayClick,
  onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  holidaysByDate: Map<string, string[]>;
  weekStartsOn: number;
  selectedDateKey: string | null;
  getEventColorByCalendar: (eventColorID: string | null, calendarID: string) => string;
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
  const spanBarRowHeight = 24;
  const spanBarTopOffset = 28;

  const weekdayLabels = ["일", "월", "화", "수", "목", "금", "토"];
  const weekdays = Array.from({ length: 7 }, (_, index) => {
    const day = (weekStartsOn + index) % 7;
    return { label: weekdayLabels[day], day };
  });

  const multiDayEvents = events.filter((event) => {
    const start = getEventStartDateKey(event);
    const end = getEventEndDateKey(event);
    return event.is_all_day || start !== end;
  });

  const singleDayEvents = events.filter((event) => {
    const start = getEventStartDateKey(event);
    const end = getEventEndDateKey(event);
    return !event.is_all_day && start === end;
  });

  function getWeekSpanBars(weekCells: MonthCell[]) {
    const bars: { event: EventData; startCol: number; endCol: number; lane: number }[] = [];
    const lanes: string[][] = [];

    for (const event of multiDayEvents) {
      const eventStart = getEventStartDateKey(event);
      const eventEnd = getEventEndDateKey(event);

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
    return singleDayEvents.filter((event) => getEventStartDateKey(event) === dateKey);
  }

  function countAllEventsForDate(dateKey: string) {
    return events.filter((event) => {
      const start = getEventStartDateKey(event);
      const end = getEventEndDateKey(event);
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

      <div className="grid flex-1 grid-rows-6">
        {weeks.map((weekCells, weekIndex) => {
          const spanBars = getWeekSpanBars(weekCells);
          const spanBarHeight = Math.max(
            spanBars.length > 0 ? (Math.max(...spanBars.map((bar) => bar.lane)) + 1) * spanBarRowHeight : 0,
            0
          );

          return (
            <div key={weekIndex} className="relative grid min-h-0 grid-cols-7 border-b border-border/50">
              {spanBars.map((bar) => {
                const spanCols = bar.endCol - bar.startCol + 1;
                const startCell = weekCells[bar.startCol];
                return (
                  <div
                    key={bar.event.id}
                    className={cn(
                      "absolute z-20 cursor-pointer truncate px-2 py-[2px] text-[11px] font-medium leading-[16px] text-white shadow-sm",
                      !startCell.inCurrentMonth && "opacity-60"
                    )}
                    style={{
                      top: `${spanBarTopOffset + bar.lane * spanBarRowHeight}px`,
                      left: `calc((100% / 7) * ${bar.startCol} + 3px)`,
                      width: `calc((100% / 7) * ${spanCols} - 6px)`,
                      backgroundColor: getEventColorByCalendar(bar.event.color_id, bar.event.calendar_id),
                      borderTopLeftRadius: bar.startCol === 0 ? "4px" : "10px",
                      borderBottomLeftRadius: bar.startCol === 0 ? "4px" : "10px",
                      borderTopRightRadius: bar.endCol === 6 ? "4px" : "10px",
                      borderBottomRightRadius: bar.endCol === 6 ? "4px" : "10px",
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
              {weekCells.map((cell) => {
                const isToday = cell.dateKey === todayKey;
                const isSelected = selectedDateKey === cell.dateKey;
                const singleEvents = getSingleDayEvents(cell.dateKey);
                const totalCount = countAllEventsForDate(cell.dateKey);
                const holidayLabels = holidaysByDate.get(cell.dateKey) || [];
                const hasHoliday = holidayLabels.length > 0;

                return (
                  <div
                    key={cell.dateKey}
                    className={cn(
                      "relative min-h-[66px] border-r border-border/50 p-1.5 cursor-pointer md:min-h-[96px]",
                      !cell.inCurrentMonth && "bg-surface-accent/20",
                      isSelected && "ring-1 ring-inset ring-primary/50"
                    )}
                    onClick={(event) => onDayClick(cell.date, event)}
                  >
                    <div className="mb-0.5 flex items-center gap-1">
                      <div
                        className={cn(
                          "inline-flex h-6 w-6 items-center justify-center rounded-full text-xs",
                          isToday
                            ? "bg-primary text-white font-medium"
                            : hasHoliday
                              ? "text-error font-semibold"
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
                      {hasHoliday ? (
                        <div
                          className={cn(
                            "min-w-0 flex-1 truncate text-[10px] font-semibold leading-tight text-error",
                            !cell.inCurrentMonth && "opacity-70"
                          )}
                          title={holidayLabels.join(", ")}
                        >
                          {holidayLabels[0]}
                        </div>
                      ) : null}
                    </div>

                    <div className="space-y-0.5" style={{ marginTop: `${spanBarHeight}px` }}>
                      {singleEvents.slice(0, 2).map((event) => {
                        return (
                          <div
                            key={event.id}
                            onClick={(clickEvent) => {
                              clickEvent.stopPropagation();
                              onEventClick(event);
                            }}
                            className={cn(
                              "flex cursor-pointer items-center gap-0.5 truncate px-0.5 text-[12px] leading-[1.25] text-text-primary",
                              !cell.inCurrentMonth && "opacity-60"
                            )}
                          >
                            <div
                              className="h-1.5 w-1.5 shrink-0 rounded-full"
                              style={{ backgroundColor: getEventColorByCalendar(event.color_id, event.calendar_id) }}
                            />
                            <span className="inline-block w-[52px] shrink-0 tabular-nums text-text-muted">
                              {formatTimeLabel(event.start_time)}
                            </span>
                            <span className="truncate">{event.title || "(제목 없음)"}</span>
                          </div>
                        );
                      })}
                      {totalCount > 3 && (
                        <div className="px-0.5 text-[12px] text-text-muted">+{totalCount - 3}</div>
                      )}
                    </div>
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

/* ===== Week View ===== */
function WeekView({
  currentDate,
  events,
  holidaysByDate,
  days,
  startHour,
  endHour,
  weekStartsOn,
  getEventColorByCalendar,
  onSlotClick,
  onEventClick,
}: {
  currentDate: Date;
  events: EventData[];
  holidaysByDate: Map<string, string[]>;
  days: number;
  startHour: number;
  endHour: number;
  weekStartsOn: number;
  getEventColorByCalendar: (eventColorID: string | null, calendarID: string) => string;
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
      const start = getEventStartDateKey(event);
      const end = getEventEndDateKey(event);
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
          const holidayLabels = holidaysByDate.get(toDateStr(date)) || [];
          const hasHoliday = holidayLabels.length > 0;
          return (
            <div
              key={index}
              className={cn(
                "flex-1 border-l border-border/50 px-1 py-1 text-center text-xs",
                isToday ? "font-medium text-text-strong" : hasHoliday ? "font-medium text-error" : "text-text-secondary"
              )}
            >
              <div className="flex min-w-0 items-center justify-center gap-1">
                <span className={isToday ? "inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-primary px-1 text-white" : ""}>
                  {weekdays[date.getDay()]} {date.getDate()}
                </span>
                {hasHoliday ? (
                  <span className="min-w-0 max-w-[84px] truncate text-[10px] font-semibold text-error" title={holidayLabels.join(", ")}>
                    {holidayLabels[0]}
                  </span>
                ) : null}
              </div>
            </div>
          );
        })}
      </div>

      {hasAnyAllDay && (
        <div className="flex border-b border-border">
          <div className="w-10 md:w-14 shrink-0 flex items-center justify-end pr-2">
            <span className="text-[11px] text-text-muted">종일</span>
          </div>
          {dayDates.map((date, index) => {
            const allDayEvents = getAllDayEvents(date);
            return (
              <div key={index} className="flex-1 border-l border-border/50 px-1 py-1.5 space-y-1.5">
                {allDayEvents.map((event) => {
                  return (
                    <div
                      key={event.id}
                      className="cursor-pointer truncate rounded-lg px-2.5 py-1.5 text-xs font-medium leading-[1.35] text-white shadow-sm"
                      style={{ backgroundColor: getEventColorByCalendar(event.color_id, event.calendar_id) }}
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
                const width = `calc(${100 / totalColumns}% - 2px)`;
                const left = `calc(${(column / totalColumns) * 100}% + 1px)`;

                return (
                  <div
                    key={event.id}
                    className="absolute cursor-pointer overflow-hidden rounded-md px-1.5 py-1 text-[11px] leading-[1.25] text-white shadow-sm"
                    style={{
                      top: `${top}px`,
                      height: `${height}px`,
                      width,
                      left,
                      backgroundColor: getEventColorByCalendar(event.color_id, event.calendar_id),
                    }}
                    onClick={(clickEvent) => {
                      clickEvent.stopPropagation();
                      onEventClick(event);
                    }}
                  >
                    <div className="truncate font-medium">{event.title || "(제목 없음)"}</div>
                    {height > 30 && (
                      <div className="truncate tabular-nums opacity-85">
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
  holidaysByDate,
  calendars,
  getEventColorByCalendar,
  onEventClick,
}: {
  events: EventData[];
  holidaysByDate: Map<string, string[]>;
  calendars: CalendarData[];
  getEventColorByCalendar: (eventColorID: string | null, calendarID: string) => string;
  onEventClick: (event: EventData) => void;
}) {
  const visibleCalendarIDs = useMemo(
    () => new Set(calendars.map((calendar) => calendar.id)),
    [calendars]
  );
  const grouped = new Map<string, EventData[]>();

  for (const event of events) {
    if (!visibleCalendarIDs.has(event.calendar_id)) continue;
    const dateKey = getEventStartDateKey(event);
    if (!grouped.has(dateKey)) grouped.set(dateKey, []);
    grouped.get(dateKey)!.push(event);
  }
  for (const [dateKey] of holidaysByDate) {
    if (!grouped.has(dateKey)) grouped.set(dateKey, []);
  }

  const sortedDates = Array.from(grouped.keys()).sort();

  if (sortedDates.length === 0) {
    return <div className="flex items-center justify-center py-20 text-text-muted">이 기간에 일정이 없습니다</div>;
  }

  return (
    <div className="divide-y divide-border/50 p-4">
      {sortedDates.map((dateKey) => (
        <div key={dateKey} className="py-3">
          <h3 className={cn("mb-2 text-sm font-medium text-text-secondary", (holidaysByDate.get(dateKey) || []).length > 0 && "text-error")}>
            {new Date(dateKey + "T00:00:00").toLocaleDateString("ko-KR", {
              year: "numeric", month: "long", day: "numeric", weekday: "short",
            })}
          </h3>
          {(holidaysByDate.get(dateKey) || []).length > 0 ? (
            <p className="mb-2 truncate text-xs font-semibold text-error" title={(holidaysByDate.get(dateKey) || []).join(", ")}>
              {(holidaysByDate.get(dateKey) || [])[0]}
            </p>
          ) : null}
          <div className="space-y-1">
            {grouped.get(dateKey)!.map((event) => {
              return (
                <div
                  key={event.id}
                  className="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 hover:bg-surface-accent/50"
                  onClick={() => onEventClick(event)}
                >
                  <div
                    className="h-3 w-3 shrink-0 rounded-full"
                    style={{ backgroundColor: getEventColorByCalendar(event.color_id, event.calendar_id) }}
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
