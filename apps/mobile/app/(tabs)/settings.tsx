import { View, Text, TouchableOpacity, Switch, StyleSheet } from "react-native";
import { useState } from "react";
import { clearTokens } from "../../lib/auth";
import { router } from "expo-router";

export default function SettingsScreen() {
  const [pushEnabled, setPushEnabled] = useState(true);
  const [darkMode, setDarkMode] = useState(false);

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
