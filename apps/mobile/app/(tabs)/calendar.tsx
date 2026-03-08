import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
} from "react-native";
import { useCalendarActions } from "../../features/calendar/ui/hooks/useCalendarActions";
import { useAuthFlow } from "../../features/auth/ui/hooks/useAuthFlow";
import { useTodoActions } from "../../features/todo/ui/hooks/useTodoActions";
import {
  buildFixedMonthGridWithWeekStart,
  getFixedMonthFetchRangeWithWeekStart,
} from "../../lib/calendar/month-grid";
import type { CalendarData, CalendarEvent, DaySummaryData } from "../../features/calendar/domain/CalendarEntities";
import { formatDueLabel } from "../../features/todo/lib/formatDueDate";
import { formatDescriptionText } from "../../lib/rich-text-description";

const SHOW_SPECIAL_CALENDARS_SETTING_KEY = "calendar_show_special_calendars";
const SPECIAL_ACCOUNT_SELECTION_SETTING_KEY = "calendar_selected_special_account_ids";
const SPECIAL_CALENDAR_SELECTION_SETTING_KEY = "calendar_selected_special_calendar_ids";

function toDateKey(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function parseStoredIDs(value: string): string[] {
  if (!value) return [];
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeHolidayTitle(title: string): string {
  return title.trim().replace(/\s+/g, " ").toLowerCase();
}

function formatTimeLabel(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" });
}

function formatSummaryEventTime(event: CalendarEvent): string {
  if (event.is_all_day) return "종일";
  const start = formatTimeLabel(event.start_time);
  const end = formatTimeLabel(event.end_time);
  if (start && end) return `${start} - ${end}`;
  return start || end || "";
}

function formatDateLabel(dateKey: string): string {
  const date = new Date(`${dateKey}T00:00:00`);
  if (Number.isNaN(date.getTime())) return dateKey;
  return date.toLocaleDateString("ko-KR", {
    year: "numeric",
    month: "long",
    day: "numeric",
    weekday: "short",
  });
}

function getTodoPriorityRank(priority: string): number {
  if (priority === "urgent") return 0;
  if (priority === "high") return 1;
  if (priority === "normal") return 2;
  if (priority === "low") return 3;
  return 4;
}

function parseWeekStartsOn(settings: Record<string, string>): number {
  const raw = Number.parseInt(settings.calendar_week_start || "0", 10);
  if (Number.isNaN(raw)) return 0;
  const normalized = raw % 7;
  return normalized < 0 ? normalized + 7 : normalized;
}

export default function CalendarScreen() {
  const [calendars, setCalendars] = useState<CalendarData[]>([]);
  const [events, setEvents] = useState<CalendarEvent[]>([]);
  const [currentDate, setCurrentDate] = useState(new Date());
  const [selectedDateKey, setSelectedDateKey] = useState<string | null>(toDateKey(new Date()));
  const [daySummary, setDaySummary] = useState<DaySummaryData | null>(null);
  const [daySummaryLoading, setDaySummaryLoading] = useState(false);
  const [daySummaryError, setDaySummaryError] = useState("");
  const [weekStartsOn, setWeekStartsOn] = useState(0);
  const [selectedSpecialAccountIDs, setSelectedSpecialAccountIDs] = useState<string[]>([]);
  const [selectedSpecialCalendarIDs, setSelectedSpecialCalendarIDs] = useState<string[]>([]);
  const [googleAccountEmails, setGoogleAccountEmails] = useState<Map<string, string>>(new Map());
  const pendingSelectAllSpecialRef = useRef(false);
  const { listCalendars, getSettings, updateSettings, listEvents, getDaySummary, backfillEvents } = useCalendarActions();
  const { listLists: listTodoLists, listTodos: listTodoItems } = useTodoActions();
  const { listGoogleAccounts } = useAuthFlow();
  const timezone = useMemo(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Seoul",
    []
  );

  const loadSettings = useCallback(async () => {
    try {
      const data = await getSettings();
      const settings = data.settings || {};
      setWeekStartsOn(parseWeekStartsOn(settings));
      const selectedAccounts = parseStoredIDs(settings[SPECIAL_ACCOUNT_SELECTION_SETTING_KEY] || "");
      const selectedCalendars = parseStoredIDs(settings[SPECIAL_CALENDAR_SELECTION_SETTING_KEY] || "");
      setSelectedSpecialAccountIDs(selectedAccounts);
      setSelectedSpecialCalendarIDs(selectedCalendars);
      pendingSelectAllSpecialRef.current =
        selectedAccounts.length === 0 &&
        selectedCalendars.length === 0 &&
        settings[SHOW_SPECIAL_CALENDARS_SETTING_KEY] === "true";
    } catch {
      setWeekStartsOn(0);
      setSelectedSpecialAccountIDs([]);
      setSelectedSpecialCalendarIDs([]);
      pendingSelectAllSpecialRef.current = false;
    }
  }, [getSettings]);

  const loadCalendars = useCallback(async () => {
    try {
      const next = await listCalendars();
      setCalendars(next || []);
    } catch {
      setCalendars([]);
    }
  }, [listCalendars]);

  const loadGoogleAccounts = useCallback(async () => {
    try {
      const accounts = await listGoogleAccounts();
      const map = new Map<string, string>();
      (accounts || []).forEach((account) => {
        map.set(account.id, account.google_email);
      });
      setGoogleAccountEmails(map);
    } catch {
      setGoogleAccountEmails(new Map());
    }
  }, [listGoogleAccounts]);

  const specialCalendars = useMemo(
    () => calendars.filter((calendar) => calendar.is_special),
    [calendars]
  );
  const specialAccountIDs = useMemo(() => {
    const ids = new Set<string>();
    specialCalendars.forEach((calendar) => {
      if (calendar.google_account_id) ids.add(calendar.google_account_id);
    });
    return Array.from(ids);
  }, [specialCalendars]);
  const selectedSpecialAccountSet = useMemo(
    () => new Set(selectedSpecialAccountIDs),
    [selectedSpecialAccountIDs]
  );
  const selectedSpecialCalendarSet = useMemo(
    () => new Set(selectedSpecialCalendarIDs),
    [selectedSpecialCalendarIDs]
  );
  const visibleCalendars = useMemo(
    () =>
      calendars.filter((calendar) => {
        if (calendar.is_special) {
          if (calendar.google_account_id && !selectedSpecialAccountSet.has(calendar.google_account_id)) {
            return false;
          }
          if (!selectedSpecialCalendarSet.has(calendar.id)) return false;
        }
        return true;
      }),
    [calendars, selectedSpecialAccountSet, selectedSpecialCalendarSet]
  );
  const visibleCalendarIDs = useMemo(
    () => visibleCalendars.map((calendar) => calendar.id),
    [visibleCalendars]
  );

  const specialAccountLabel = useCallback((accountID: string) => {
    if (googleAccountEmails.has(accountID)) return googleAccountEmails.get(accountID) || accountID;
    return accountID;
  }, [googleAccountEmails]);

  const load = useCallback(async () => {
    const { start, end } = getFixedMonthFetchRangeWithWeekStart(currentDate, weekStartsOn);

    try {
      if (calendars.length > 0 && visibleCalendarIDs.length === 0) {
        setEvents([]);
        return;
      }
      const initial = await listEvents(start, end, visibleCalendarIDs);
      setEvents(initial || []);

      const startAt = new Date(start).getTime();
      const endAt = new Date(end).getTime();
      const needsBackfillCalendarIDs = visibleCalendars
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
        const merged = await listEvents(start, end, visibleCalendarIDs);
        setEvents(merged || []);
      }
    } catch {
      setEvents([]);
    }
  }, [backfillEvents, calendars.length, currentDate, listEvents, visibleCalendarIDs, visibleCalendars, weekStartsOn]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  useEffect(() => {
    loadCalendars();
  }, [loadCalendars]);

  useEffect(() => {
    loadGoogleAccounts();
  }, [loadGoogleAccounts]);

  useEffect(() => {
    if (!pendingSelectAllSpecialRef.current) return;
    if (specialCalendars.length === 0) return;
    pendingSelectAllSpecialRef.current = false;

    const allAccountIDs = Array.from(
      new Set(
        specialCalendars
          .map((calendar) => calendar.google_account_id)
          .filter((id): id is string => !!id)
      )
    );
    const allCalendarIDs = specialCalendars.map((calendar) => calendar.id);
    setSelectedSpecialAccountIDs(allAccountIDs);
    setSelectedSpecialCalendarIDs(allCalendarIDs);
    updateSettings({
      [SPECIAL_ACCOUNT_SELECTION_SETTING_KEY]: allAccountIDs.join(","),
      [SPECIAL_CALENDAR_SELECTION_SETTING_KEY]: allCalendarIDs.join(","),
      [SHOW_SPECIAL_CALENDARS_SETTING_KEY]: "true",
    }).catch(() => undefined);
  }, [specialCalendars, updateSettings]);

  useEffect(() => {
    load();
  }, [load]);

  const loadTodosForDate = useCallback(async (dateKey: string): Promise<DaySummaryData["todos"]> => {
    const lists = await listTodoLists();
    if (!lists || lists.length === 0) return [];

    const grouped = await Promise.all(
      lists.map(async (list) => {
        try {
          const items = await listTodoItems(list.id);
          return items.map((todo) => ({ listID: list.id, todo }));
        } catch {
          return [];
        }
      }),
    );

    return grouped
      .flat()
      .filter(({ todo }) => {
        const dueKey = (todo.due_date || todo.due || "").slice(0, 10);
        return dueKey === dateKey;
      })
      .map(({ listID, todo }) => ({
        id: todo.id,
        list_id: todo.list_id || listID,
        title: todo.title || "(제목 없음)",
        notes: todo.notes || "",
        due: todo.due || todo.due_date || null,
        due_date: todo.due_date || null,
        due_time: todo.due_time || null,
        priority: todo.priority || "normal",
        is_done: todo.done ?? todo.is_done ?? false,
      }))
      .sort((a, b) => {
        const timeA = a.due_time || "99:99";
        const timeB = b.due_time || "99:99";
        if (timeA !== timeB) return timeA.localeCompare(timeB);
        const rankDiff = getTodoPriorityRank(a.priority) - getTodoPriorityRank(b.priority);
        if (rankDiff !== 0) return rankDiff;
        return (a.title || "").localeCompare(b.title || "");
      });
  }, [listTodoItems, listTodoLists]);

  useEffect(() => {
    if (!selectedDateKey) {
      setDaySummary(null);
      setDaySummaryError("");
      setDaySummaryLoading(false);
      return;
    }

    let cancelled = false;
    setDaySummaryLoading(true);
    setDaySummaryError("");
    getDaySummary(selectedDateKey, timezone, visibleCalendarIDs, false)
      .then((summary) => {
        if (cancelled) return;
        const allowedCalendarIDs = new Set(visibleCalendarIDs);
        const nextEvents =
          calendars.length > 0
            ? (summary.events || []).filter((event) => allowedCalendarIDs.has(event.calendar_id))
            : summary.events || [];
        const nextTodosPromise =
          (summary.todos || []).length > 0
            ? Promise.resolve(summary.todos || [])
            : loadTodosForDate(selectedDateKey).catch(() => []);
        return nextTodosPromise.then((nextTodos) => {
          if (cancelled) return;
          setDaySummary({ ...summary, events: nextEvents, todos: nextTodos });
        });
      })
      .catch(async () => {
        if (cancelled) return;
        const allowedCalendarIDs = new Set(visibleCalendarIDs);
        const calendarKindByID = new Map(calendars.map((calendar) => [calendar.id, calendar.kind]));
        const fallbackEvents = (events || []).filter((event) => {
          if (allowedCalendarIDs.size > 0 && !allowedCalendarIDs.has(event.calendar_id)) return false;
          const start = event.start_time.split("T")[0];
          const end = event.end_time.split("T")[0];
          return selectedDateKey >= start && selectedDateKey <= end;
        });
        const holidayByKey = new Map<string, { date: string; name: string }>();
        fallbackEvents.forEach((event) => {
          const kind = calendarKindByID.get(event.calendar_id);
          if (kind !== "holiday" || !event.is_all_day) return;
          const key = normalizeHolidayTitle(event.title);
          if (holidayByKey.has(key)) return;
          holidayByKey.set(key, { date: selectedDateKey, name: event.title });
        });
        const mergedHolidays = Array.from(holidayByKey.values());
        const mergedEvents = fallbackEvents.filter((event) => {
          const kind = calendarKindByID.get(event.calendar_id);
          return !(kind === "holiday" && event.is_all_day);
        });
        const fallbackTodos = await loadTodosForDate(selectedDateKey).catch(() => []);
        setDaySummary({
          date: selectedDateKey,
          timezone,
          holidays: mergedHolidays,
          events: mergedEvents,
          todos: fallbackTodos,
        });
        setDaySummaryError("");
      })
      .finally(() => {
        if (cancelled) return;
        setDaySummaryLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [calendars, events, getDaySummary, loadTodosForDate, selectedDateKey, timezone, visibleCalendarIDs]);

  const prevMonth = () => {
    setCurrentDate(
      new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1)
    );
  };

  const nextMonth = () => {
    setCurrentDate(
      new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1)
    );
  };

  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();
  const todayKey = toDateKey(new Date());

  const cells = useMemo(
    () => buildFixedMonthGridWithWeekStart(currentDate, weekStartsOn),
    [currentDate, weekStartsOn]
  );
  const weeks = useMemo(
    () => Array.from({ length: 6 }, (_, i) => cells.slice(i * 7, i * 7 + 7)),
    [cells]
  );
  const weekLabels = useMemo(() => {
    const labels = ["일", "월", "화", "수", "목", "금", "토"];
    return Array.from({ length: 7 }, (_, i) => labels[(weekStartsOn + i) % 7]);
  }, [weekStartsOn]);

  const calMap = useMemo(
    () => new Map(calendars.map((calendar) => [calendar.id, calendar])),
    [calendars]
  );
  const displayEvents = useMemo(() => {
    const seenHolidayKeys = new Set<string>();
    return events.filter((event) => {
      const calendar = calMap.get(event.calendar_id);
      if (calendar?.kind === "holiday" && event.is_all_day) {
        const key = `${event.start_time.slice(0, 10)}|${normalizeHolidayTitle(event.title)}`;
        if (seenHolidayKeys.has(key)) return false;
        seenHolidayKeys.add(key);
      }
      return true;
    });
  }, [calMap, events]);

  const persistSpecialSelection = useCallback((accounts: string[], calendars: string[]) => {
    setSelectedSpecialAccountIDs(accounts);
    setSelectedSpecialCalendarIDs(calendars);
    updateSettings({
      [SPECIAL_ACCOUNT_SELECTION_SETTING_KEY]: accounts.join(","),
      [SPECIAL_CALENDAR_SELECTION_SETTING_KEY]: calendars.join(","),
      [SHOW_SPECIAL_CALENDARS_SETTING_KEY]: accounts.length > 0 && calendars.length > 0 ? "true" : "false",
    }).catch(() => undefined);
  }, [updateSettings]);

  const handleToggleSpecialAccount = useCallback((accountID: string) => {
    const exists = selectedSpecialAccountIDs.includes(accountID);
    const nextAccounts = exists
      ? selectedSpecialAccountIDs.filter((id) => id !== accountID)
      : [...selectedSpecialAccountIDs, accountID];
    persistSpecialSelection(nextAccounts, selectedSpecialCalendarIDs);
  }, [persistSpecialSelection, selectedSpecialAccountIDs, selectedSpecialCalendarIDs]);

  const handleToggleSpecialCalendar = useCallback((calendarID: string) => {
    const exists = selectedSpecialCalendarIDs.includes(calendarID);
    const nextCalendars = exists
      ? selectedSpecialCalendarIDs.filter((id) => id !== calendarID)
      : [...selectedSpecialCalendarIDs, calendarID];
    persistSpecialSelection(selectedSpecialAccountIDs, nextCalendars);
  }, [persistSpecialSelection, selectedSpecialAccountIDs, selectedSpecialCalendarIDs]);

  const handleSelectAllSpecial = useCallback(() => {
    const allAccountIDs = specialAccountIDs;
    const allCalendarIDs = specialCalendars.map((calendar) => calendar.id);
    persistSpecialSelection(allAccountIDs, allCalendarIDs);
  }, [persistSpecialSelection, specialAccountIDs, specialCalendars]);

  const handleClearSpecial = useCallback(() => {
    persistSpecialSelection([], []);
  }, [persistSpecialSelection]);

  const getEventsForDateKey = (dateKey: string) =>
    displayEvents.filter((event) => {
      const start = event.start_time.split("T")[0];
      const end = event.end_time.split("T")[0];
      return dateKey >= start && dateKey <= end;
    });

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <TouchableOpacity onPress={prevMonth}>
          <Text style={styles.nav}>◀</Text>
        </TouchableOpacity>
        <Text style={styles.title}>
          {year}년 {month + 1}월
        </Text>
        <TouchableOpacity onPress={nextMonth}>
          <Text style={styles.nav}>▶</Text>
        </TouchableOpacity>
      </View>
      <View style={styles.specialToggleRow}>
        <View style={styles.specialActionsRow}>
          <TouchableOpacity onPress={handleSelectAllSpecial} style={styles.specialActionChip}>
            <Text style={styles.specialActionText}>특수 전체 선택</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={handleClearSpecial} style={styles.specialActionChip}>
            <Text style={styles.specialActionText}>모두 숨김</Text>
          </TouchableOpacity>
        </View>
        <Text style={styles.specialSectionTitle}>특수 계정</Text>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.specialScroll}>
          {specialAccountIDs.length === 0 ? (
            <Text style={styles.specialEmpty}>선택 가능한 특수 계정 없음</Text>
          ) : (
            specialAccountIDs.map((accountID) => {
              const selected = selectedSpecialAccountSet.has(accountID);
              return (
                <TouchableOpacity
                  key={accountID}
                  style={[styles.specialToggle, selected && styles.specialToggleOn]}
                  onPress={() => handleToggleSpecialAccount(accountID)}
                >
                  <Text style={[styles.specialToggleText, selected && styles.specialToggleTextOn]} numberOfLines={1}>
                    {specialAccountLabel(accountID)}
                  </Text>
                </TouchableOpacity>
              );
            })
          )}
        </ScrollView>
        <Text style={styles.specialSectionTitle}>특수 캘린더</Text>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.specialScroll}>
          {specialCalendars.length === 0 ? (
            <Text style={styles.specialEmpty}>선택 가능한 특수 캘린더 없음</Text>
          ) : (
            specialCalendars.map((calendar) => {
              const selected = selectedSpecialCalendarSet.has(calendar.id);
              return (
                <TouchableOpacity
                  key={calendar.id}
                  style={[styles.specialToggle, selected && styles.specialToggleOn]}
                  onPress={() => handleToggleSpecialCalendar(calendar.id)}
                >
                  <Text style={[styles.specialToggleText, selected && styles.specialToggleTextOn]} numberOfLines={1}>
                    {calendar.name}
                  </Text>
                </TouchableOpacity>
              );
            })
          )}
        </ScrollView>
      </View>

      <View style={styles.weekHeader}>
        {weekLabels.map((d) => (
          <Text key={d} style={styles.weekDay}>
            {d}
          </Text>
        ))}
      </View>

      <ScrollView>
        {weeks.map((weekCells, i) => (
          <View key={i} style={styles.weekRow}>
            {weekCells.map((cell, j) => {
              const isToday = cell.dateKey === todayKey;
              const isSelected = selectedDateKey === cell.dateKey;
              const dayEvents = getEventsForDateKey(cell.dateKey);

              return (
                <TouchableOpacity
                  key={j}
                  style={[
                    styles.dayCell,
                    !cell.inCurrentMonth && styles.dayCellOutside,
                    isSelected && styles.dayCellSelected,
                  ]}
                  onPress={() => setSelectedDateKey(cell.dateKey)}
                  activeOpacity={0.85}
                >
                  <Text
                    style={[
                      styles.dayText,
                      !cell.inCurrentMonth && styles.dayTextOutside,
                      isToday && styles.todayText,
                    ]}
                  >
                    {cell.day}
                  </Text>

                  {dayEvents.slice(0, 2).map((event) => (
                    <View
                      key={event.id}
                      style={[
                        styles.eventDot,
                        { backgroundColor: event.color_id ? "#34A853" : "#4285F4" },
                        !cell.inCurrentMonth && styles.eventDotOutside,
                      ]}
                    >
                      <Text style={[styles.eventText, !cell.inCurrentMonth && styles.eventTextOutside]} numberOfLines={1}>
                        {event.title}
                      </Text>
                    </View>
                  ))}

                  {dayEvents.length > 2 && (
                    <Text style={styles.moreText}>+{dayEvents.length - 2}</Text>
                  )}
                </TouchableOpacity>
              );
            })}
          </View>
        ))}
        {selectedDateKey ? (
          <View style={styles.summaryPanel}>
            <Text style={styles.summaryTitle}>{formatDateLabel(selectedDateKey)}</Text>
            {daySummaryLoading ? (
              <Text style={styles.summaryEmpty}>일정 정보를 불러오는 중...</Text>
            ) : daySummaryError ? (
              <Text style={styles.summaryError}>{daySummaryError}</Text>
            ) : !daySummary ? (
              <Text style={styles.summaryEmpty}>표시할 데이터가 없습니다.</Text>
            ) : (
              <>
                <View style={styles.summarySection}>
                  <Text style={styles.summarySectionTitle}>공휴일</Text>
                  {daySummary.holidays.length === 0 ? (
                    <Text style={styles.summaryEmpty}>없음</Text>
                  ) : (
                    daySummary.holidays.map((holiday) => (
                      <Text key={`${holiday.date}-${holiday.name}`} style={styles.summaryHolidayText}>
                        {holiday.name}
                      </Text>
                    ))
                  )}
                </View>

                <View style={styles.summarySection}>
                  <Text style={styles.summarySectionTitle}>일정</Text>
                  {daySummary.events.length === 0 ? (
                    <Text style={styles.summaryEmpty}>없음</Text>
                  ) : (
                    daySummary.events.map((event) => (
                      <View key={event.id} style={styles.summaryEventRow}>
                        <View style={styles.summaryEventHead}>
                          <View style={styles.summaryEventDot} />
                          <Text style={styles.summaryEventTitle} numberOfLines={1}>
                            {event.title || "(제목 없음)"}
                          </Text>
                        </View>
                        <Text style={styles.summaryMetaText}>{formatSummaryEventTime(event)}</Text>
                        {formatDescriptionText(event.description) ? (
                          <Text style={styles.summaryDescriptionText} numberOfLines={6}>
                            {formatDescriptionText(event.description)}
                          </Text>
                        ) : null}
                      </View>
                    ))
                  )}
                </View>

                <View style={styles.summarySection}>
                  <Text style={styles.summarySectionTitle}>Todo</Text>
                  {daySummary.todos.length === 0 ? (
                    <Text style={styles.summaryEmpty}>없음</Text>
                  ) : (
                    daySummary.todos.map((todo) => (
                      <View key={todo.id} style={styles.summaryTodoRow}>
                        <Text
                          style={[
                            styles.summaryTodoText,
                            todo.is_done && styles.summaryTodoDoneText,
                          ]}
                          numberOfLines={2}
                        >
                          {todo.title}
                        </Text>
                        <Text style={styles.summaryMetaText}>
                          우선순위 {todo.priority}
                          {todo.due_date ? ` · 기한 ${formatDueLabel(todo.due_date, todo.due_time)}` : ""}
                          {todo.is_done ? " · 완료" : ""}
                        </Text>
                      </View>
                    ))
                  )}
                </View>
              </>
            )}
          </View>
        ) : null}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: "#fff" },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    padding: 16,
  },
  nav: { fontSize: 18, padding: 8, color: "#333" },
  title: { fontSize: 18, fontWeight: "600" },
  specialToggleRow: {
    paddingHorizontal: 16,
    paddingBottom: 8,
  },
  specialActionsRow: {
    flexDirection: "row",
    gap: 8,
    marginBottom: 6,
  },
  specialActionChip: {
    borderWidth: 1,
    borderColor: "#d1d5db",
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 5,
    backgroundColor: "#fff",
  },
  specialActionText: {
    fontSize: 11,
    color: "#374151",
    fontWeight: "500",
  },
  specialSectionTitle: {
    fontSize: 11,
    color: "#6b7280",
    marginBottom: 4,
  },
  specialScroll: {
    maxHeight: 34,
    marginBottom: 6,
  },
  specialEmpty: {
    fontSize: 11,
    color: "#9ca3af",
    paddingVertical: 6,
  },
  specialToggle: {
    alignSelf: "flex-start",
    borderWidth: 1,
    borderColor: "#ddd",
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 6,
    backgroundColor: "#fff",
  },
  specialToggleOn: {
    backgroundColor: "#111827",
    borderColor: "#111827",
  },
  specialToggleText: { fontSize: 12, color: "#4b5563" },
  specialToggleTextOn: { color: "#fff" },
  weekHeader: {
    flexDirection: "row",
    borderBottomWidth: 1,
    borderBottomColor: "#eee",
    paddingBottom: 8,
  },
  weekDay: {
    flex: 1,
    textAlign: "center",
    fontSize: 12,
    color: "#999",
  },
  weekRow: { flexDirection: "row", minHeight: 70 },
  dayCell: {
    flex: 1,
    padding: 4,
    borderBottomWidth: 1,
    borderBottomColor: "#f5f5f5",
  },
  dayCellOutside: {
    backgroundColor: "#fafafa",
  },
  dayCellSelected: {
    backgroundColor: "#eaf4ff",
  },
  dayText: { fontSize: 13, textAlign: "center", marginBottom: 2, color: "#333" },
  dayTextOutside: {
    color: "#9ca3af",
  },
  todayText: {
    color: "#fff",
    backgroundColor: "#4285F4",
    borderRadius: 10,
    overflow: "hidden",
    textAlign: "center",
    width: 20,
    alignSelf: "center",
  },
  eventDot: {
    borderRadius: 3,
    paddingHorizontal: 3,
    paddingVertical: 1,
    marginTop: 1,
  },
  eventDotOutside: {
    opacity: 0.55,
  },
  eventText: { fontSize: 9, color: "#fff" },
  eventTextOutside: {
    color: "#f1f5f9",
  },
  moreText: { fontSize: 9, color: "#999", textAlign: "center", marginTop: 1 },
  summaryPanel: {
    borderTopWidth: 1,
    borderTopColor: "#e5e7eb",
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginTop: 8,
    gap: 10,
  },
  summaryTitle: {
    fontSize: 15,
    fontWeight: "600",
    color: "#111827",
  },
  summarySection: {
    gap: 6,
  },
  summarySectionTitle: {
    fontSize: 11,
    color: "#6b7280",
    fontWeight: "600",
  },
  summaryHolidayText: {
    fontSize: 13,
    color: "#b91c1c",
    fontWeight: "600",
  },
  summaryEventRow: {
    borderWidth: 1,
    borderColor: "#e5e7eb",
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 8,
    gap: 2,
  },
  summaryEventHead: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
  },
  summaryEventDot: {
    width: 7,
    height: 7,
    borderRadius: 999,
    backgroundColor: "#2563eb",
  },
  summaryEventTitle: {
    flex: 1,
    fontSize: 13,
    color: "#111827",
    fontWeight: "500",
  },
  summaryTodoRow: {
    borderWidth: 1,
    borderColor: "#e5e7eb",
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 8,
    gap: 2,
  },
  summaryTodoText: {
    fontSize: 13,
    color: "#111827",
  },
  summaryTodoDoneText: {
    color: "#6b7280",
    textDecorationLine: "line-through",
  },
  summaryMetaText: {
    fontSize: 11,
    color: "#6b7280",
  },
  summaryDescriptionText: {
    fontSize: 12,
    lineHeight: 18,
    color: "#4b5563",
    marginTop: 6,
  },
  summaryEmpty: {
    fontSize: 12,
    color: "#9ca3af",
  },
  summaryError: {
    fontSize: 12,
    color: "#b91c1c",
  },
});
