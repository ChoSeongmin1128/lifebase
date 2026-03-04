import { View, Text, TouchableOpacity, Switch, StyleSheet } from "react-native";
import { useCallback, useEffect, useState } from "react";
import { clearTokens, getAccessToken } from "../../features/auth/infrastructure/token-auth";
import { api } from "../../features/shared/infrastructure/http-api";
import { router } from "expo-router";

const RETENTION_OPTIONS = [
  { value: "1m", label: "1달" },
  { value: "3m", label: "3달" },
  { value: "6m", label: "반년" },
  { value: "1y", label: "1년" },
  { value: "3y", label: "3년" },
  { value: "unlimited", label: "무제한" },
];

export default function SettingsScreen() {
  const [pushEnabled, setPushEnabled] = useState(true);
  const [darkMode, setDarkMode] = useState(false);
  const [todoDoneRetention, setTodoDoneRetention] = useState("1y");

  const loadSettings = useCallback(async () => {
    try {
      const token = await getAccessToken();
      if (!token) return;
      const data = await api<{ settings?: Record<string, string> }>("/settings", { token });
      const value = data.settings?.todo_done_retention_period || "1y";
      const valid = RETENTION_OPTIONS.some((item) => item.value === value) ? value : "1y";
      setTodoDoneRetention(valid);
    } catch {
      setTodoDoneRetention("1y");
    }
  }, []);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const updateSetting = async (key: string, value: string) => {
    const token = await getAccessToken();
    if (!token) throw new Error("인증이 필요합니다.");
    await api("/settings", {
      method: "PATCH",
      body: { [key]: value },
      token,
    });
  };

  const handleTodoRetentionChange = async (value: string) => {
    const previous = todoDoneRetention;
    setTodoDoneRetention(value);
    try {
      await updateSetting("todo_done_retention_period", value);
    } catch {
      setTodoDoneRetention(previous);
    }
  };

  const handleLogout = async () => {
    await clearTokens();
    router.replace("/login");
  };

  return (
    <View style={styles.container}>
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>일반</Text>
        <View style={styles.row}>
          <Text style={styles.label}>다크 모드</Text>
          <Switch value={darkMode} onValueChange={setDarkMode} />
        </View>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>알림</Text>
        <View style={styles.row}>
          <Text style={styles.label}>Push 알림</Text>
          <Switch value={pushEnabled} onValueChange={setPushEnabled} />
        </View>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Todo</Text>
        <Text style={styles.label}>완료 항목 보존 기간</Text>
        <View style={styles.optionWrap}>
          {RETENTION_OPTIONS.map((option) => (
            <TouchableOpacity
              key={option.value}
              style={[
                styles.optionChip,
                todoDoneRetention === option.value && styles.optionChipActive,
              ]}
              onPress={() => handleTodoRetentionChange(option.value)}
            >
              <Text
                style={[
                  styles.optionChipText,
                  todoDoneRetention === option.value && styles.optionChipTextActive,
                ]}
              >
                {option.label}
              </Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>계정</Text>
        <View style={styles.row}>
          <Text style={styles.label}>Google 계정</Text>
          <Text style={styles.value}>연결됨</Text>
        </View>
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Text style={styles.logoutText}>로그아웃</Text>
        </TouchableOpacity>
      </View>

      <Text style={styles.version}>LifeBase v0.1.0</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: "#fff", padding: 16 },
  section: {
    marginBottom: 24,
    borderBottomWidth: 1,
    borderBottomColor: "#eee",
    paddingBottom: 16,
  },
  sectionTitle: {
    fontSize: 13,
    fontWeight: "600",
    color: "#999",
    marginBottom: 12,
    textTransform: "uppercase",
  },
  row: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingVertical: 8,
  },
  optionWrap: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
    marginTop: 8,
  },
  optionChip: {
    borderWidth: 1,
    borderColor: "#d1d5db",
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 5,
    backgroundColor: "#fff",
  },
  optionChipActive: {
    borderColor: "#111827",
    backgroundColor: "#111827",
  },
  optionChipText: { fontSize: 12, color: "#374151" },
  optionChipTextActive: { color: "#fff", fontWeight: "600" },
  label: { fontSize: 15 },
  value: { fontSize: 14, color: "#666" },
  logoutButton: {
    marginTop: 12,
    padding: 12,
    backgroundColor: "#fee",
    borderRadius: 8,
    alignItems: "center",
  },
  logoutText: { color: "#DC2626", fontWeight: "600" },
  version: {
    textAlign: "center",
    color: "#ccc",
    fontSize: 12,
    marginTop: 40,
  },
});
