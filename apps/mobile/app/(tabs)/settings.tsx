import { View, Text, TouchableOpacity, Switch, StyleSheet, ScrollView } from "react-native";
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
  const [activeSection, setActiveSection] = useState<
    "general" | "calendar" | "todo" | "notifications" | "cloud"
  >("general");

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
      <View style={styles.headerCard}>
        <Text style={styles.screenTitle}>설정</Text>
        <Text style={styles.screenSubtitle}>모바일에서도 같은 섹션 구조를 상단 세그먼트로 유지합니다.</Text>
      </View>

      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.segmentRow}
        style={styles.segmentScroller}
      >
        {[
          { id: "general" as const, label: "일반" },
          { id: "calendar" as const, label: "캘린더" },
          { id: "todo" as const, label: "Todo" },
          { id: "notifications" as const, label: "알림" },
          { id: "cloud" as const, label: "Cloud" },
        ].map((section) => (
          <TouchableOpacity
            key={section.id}
            style={[styles.segmentChip, activeSection === section.id && styles.segmentChipActive]}
            onPress={() => setActiveSection(section.id)}
          >
            <Text style={[styles.segmentChipText, activeSection === section.id && styles.segmentChipTextActive]}>
              {section.label}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      <View style={styles.sectionCard}>
        {activeSection === "general" && (
          <>
            <Text style={styles.sectionTitle}>일반</Text>
            <View style={styles.row}>
              <Text style={styles.label}>다크 모드</Text>
              <Switch value={darkMode} onValueChange={setDarkMode} />
            </View>
            <View style={[styles.row, styles.rowBlock]}>
              <Text style={styles.label}>Google 계정</Text>
              <Text style={styles.value}>연결됨</Text>
            </View>
            <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
              <Text style={styles.logoutText}>로그아웃</Text>
            </TouchableOpacity>
          </>
        )}

        {activeSection === "calendar" && (
          <>
            <Text style={styles.sectionTitle}>캘린더</Text>
            <Text style={styles.supportingText}>주 시작 요일, 기본 보기, 계정 필터 설정은 Web/Desktop과 같은 구조로 후속 확장합니다.</Text>
          </>
        )}

        {activeSection === "notifications" && (
          <>
            <Text style={styles.sectionTitle}>알림</Text>
            <View style={styles.row}>
              <Text style={styles.label}>Push 알림</Text>
              <Switch value={pushEnabled} onValueChange={setPushEnabled} />
            </View>
          </>
        )}

        {activeSection === "todo" && (
          <>
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
          </>
        )}

        {activeSection === "cloud" && (
          <>
            <Text style={styles.sectionTitle}>Cloud</Text>
            <Text style={styles.supportingText}>정렬, 보기 기준, 업로드 관련 설정은 같은 섹션 구조를 유지한 채 모바일 화면에 맞게 이어서 확장합니다.</Text>
          </>
        )}
      </View>

      <Text style={styles.version}>LifeBase v0.1.0</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: "#F7F8F6", padding: 16 },
  headerCard: {
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginBottom: 12,
  },
  screenTitle: { fontSize: 20, fontWeight: "700", color: "#111111" },
  screenSubtitle: { marginTop: 4, fontSize: 13, color: "#5E6B67" },
  segmentScroller: { marginBottom: 12 },
  segmentRow: { gap: 8 },
  segmentChip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
  },
  segmentChipActive: {
    backgroundColor: "#1B998B",
    borderColor: "#1B998B",
  },
  segmentChipText: { fontSize: 13, color: "#5E6B67", fontWeight: "500" },
  segmentChipTextActive: { color: "#FFFFFF" },
  sectionCard: {
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
    borderRadius: 20,
    padding: 16,
  },
  sectionTitle: {
    fontSize: 13,
    fontWeight: "600",
    color: "#7B8784",
    marginBottom: 12,
    textTransform: "uppercase",
  },
  row: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingVertical: 8,
  },
  rowBlock: {
    marginTop: 4,
    marginBottom: 4,
  },
  optionWrap: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
    marginTop: 8,
  },
  optionChip: {
    borderWidth: 1,
    borderColor: "#D8DFDC",
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 5,
    backgroundColor: "#fff",
  },
  optionChipActive: {
    borderColor: "#1B998B",
    backgroundColor: "#1B998B",
  },
  optionChipText: { fontSize: 12, color: "#43514D" },
  optionChipTextActive: { color: "#fff", fontWeight: "600" },
  label: { fontSize: 15 },
  value: { fontSize: 14, color: "#5E6B67" },
  supportingText: {
    fontSize: 14,
    lineHeight: 20,
    color: "#5E6B67",
  },
  logoutButton: {
    marginTop: 12,
    padding: 12,
    backgroundColor: "#FFF0ED",
    borderRadius: 12,
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
