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
import type { CalendarEvent, SettingsResponse } from "../../features/calendar/domain/CalendarEntities";

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
  const [events, setEvents] = useState<CalendarEvent[]>([]);
  const [currentDate, setCurrentDate] = useState(new Date());
  const [selectedDateKey, setSelectedDateKey] = useState<string | null>(null);
  const [weekStartsOn, setWeekStartsOn] = useState(0);
  const { getSettings, listEvents } = useCalendarActions();

  const loadSettings = useCallback(async () => {
    try {
      const data = await getSettings();
      setWeekStartsOn(parseWeekStartsOn(data.settings || {}));
    } catch {
      setWeekStartsOn(0);
    }
  }, [getSettings]);

  const load = useCallback(async () => {
    const { start, end } = getFixedMonthFetchRangeWithWeekStart(currentDate, weekStartsOn);

    try {
      const data = await listEvents(start, end);
      setEvents(data || []);
    } catch {
      setEvents([]);
    }
  }, [currentDate, listEvents, weekStartsOn]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

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
                        { backgroundColor: event.color || "#4285F4" },
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
