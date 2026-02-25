import { useState, useEffect, useCallback } from "react";
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
} from "react-native";
import { api } from "../../lib/api";
import { getAccessToken } from "../../lib/auth";

type CalendarEvent = {
  id: string;
  title: string;
  start_time: string;
  end_time: string;
  all_day: boolean;
  color?: string;
};

export default function CalendarScreen() {
  const [events, setEvents] = useState<CalendarEvent[]>([]);
  const [currentDate, setCurrentDate] = useState(new Date());

  const load = useCallback(async () => {
    const token = await getAccessToken();
    if (!token) return;
    const start = new Date(
      currentDate.getFullYear(),
      currentDate.getMonth(),
      1
    );
    const end = new Date(
      currentDate.getFullYear(),
      currentDate.getMonth() + 1,
      0,
      23,
      59,
      59
    );
    try {
      const data = await api<{ events: CalendarEvent[] }>(
        `/events?start=${start.toISOString()}&end=${end.toISOString()}`,
        { token }
      );
      setEvents(data.events || []);
    } catch {
      setEvents([]);
    }
  }, [currentDate]);

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
  const firstDay = new Date(year, month, 1).getDay();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const today = new Date();

  const weeks: (number | null)[][] = [];
  let week: (number | null)[] = Array(firstDay).fill(null);
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

  const getEventsForDay = (day: number) =>
    events.filter((e) => {
      const d = new Date(e.start_time).getDate();
      return d === day;
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
        {["일", "월", "화", "수", "목", "금", "토"].map((d) => (
          <Text key={d} style={styles.weekDay}>
            {d}
          </Text>
        ))}
      </View>

      <ScrollView>
        {weeks.map((w, i) => (
          <View key={i} style={styles.weekRow}>
            {w.map((day, j) => {
              const isToday =
                day !== null &&
                year === today.getFullYear() &&
                month === today.getMonth() &&
                day === today.getDate();
              const dayEvents = day ? getEventsForDay(day) : [];
              return (
                <View key={j} style={styles.dayCell}>
                  {day !== null && (
                    <>
                      <Text
                        style={[
                          styles.dayText,
                          isToday && styles.todayText,
                        ]}
                      >
                        {day}
                      </Text>
                      {dayEvents.slice(0, 2).map((e) => (
                        <View
                          key={e.id}
                          style={[
                            styles.eventDot,
                            { backgroundColor: e.color || "#4285F4" },
                          ]}
                        >
                          <Text style={styles.eventText} numberOfLines={1}>
                            {e.title}
                          </Text>
                        </View>
                      ))}
                      {dayEvents.length > 2 && (
                        <Text style={styles.moreText}>
                          +{dayEvents.length - 2}
                        </Text>
                      )}
                    </>
                  )}
                </View>
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
  dayText: { fontSize: 13, textAlign: "center", marginBottom: 2 },
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
  eventText: { fontSize: 9, color: "#fff" },
  moreText: { fontSize: 9, color: "#999", textAlign: "center", marginTop: 1 },
});
