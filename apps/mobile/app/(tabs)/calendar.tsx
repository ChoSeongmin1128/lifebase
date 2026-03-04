import { useState, useEffect, useCallback, useMemo } from "react";
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
} from "react-native";
import { useCalendarActions } from "../../features/calendar/ui/hooks/useCalendarActions";
import {
  buildFixedMonthGridWithWeekStart,
  getFixedMonthFetchRangeWithWeekStart,
} from "../../lib/calendar/month-grid";
import type { CalendarData, CalendarEvent } from "../../features/calendar/domain/CalendarEntities";

const SHOW_SPECIAL_CALENDARS_SETTING_KEY = "calendar_show_special_calendars";

function toDateKey(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
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
  const [selectedDateKey, setSelectedDateKey] = useState<string | null>(null);
  const [weekStartsOn, setWeekStartsOn] = useState(0);
  const [showSpecialCalendars, setShowSpecialCalendars] = useState(false);
  const { listCalendars, getSettings, listEvents, backfillEvents } = useCalendarActions();

  const loadSettings = useCallback(async () => {
    try {
      const data = await getSettings();
      const settings = data.settings || {};
      setWeekStartsOn(parseWeekStartsOn(settings));
      setShowSpecialCalendars(settings[SHOW_SPECIAL_CALENDARS_SETTING_KEY] === "true");
    } catch {
      setWeekStartsOn(0);
      setShowSpecialCalendars(false);
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

  const load = useCallback(async () => {
    const { start, end } = getFixedMonthFetchRangeWithWeekStart(currentDate, weekStartsOn);
    const visibleCalendars = calendars.filter((calendar) => {
      if (!showSpecialCalendars && calendar.is_special) return false;
      return true;
    });
    const visibleCalendarIDs = visibleCalendars.map((calendar) => calendar.id);

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
  }, [backfillEvents, calendars, currentDate, listEvents, showSpecialCalendars, weekStartsOn]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  useEffect(() => {
    loadCalendars();
  }, [loadCalendars]);

  useEffect(() => {
    load();
  }, [load]);

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

  const getEventsForDateKey = (dateKey: string) =>
    events.filter((event) => {
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
        <TouchableOpacity
          onPress={() => setShowSpecialCalendars((prev) => !prev)}
          style={[styles.specialToggle, showSpecialCalendars && styles.specialToggleOn]}
        >
          <Text style={[styles.specialToggleText, showSpecialCalendars && styles.specialToggleTextOn]}>
            {showSpecialCalendars ? "특수 캘린더 표시" : "특수 캘린더 숨김"}
          </Text>
        </TouchableOpacity>
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
});
