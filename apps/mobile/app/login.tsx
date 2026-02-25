import { View, Text, TouchableOpacity, StyleSheet } from "react-native";
import * as Linking from "expo-linking";
import { api } from "../lib/api";

export default function LoginScreen() {
  const handleLogin = async () => {
    try {
      const data = await api<{ url: string }>("/auth/url");
      if (data.url) {
        Linking.openURL(data.url);
      }
    } catch (err) {
      console.error("Failed to get auth URL:", err);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>LifeBase</Text>
      <Text style={styles.subtitle}>클라우드 + 캘린더 + Todo</Text>
      <TouchableOpacity style={styles.button} onPress={handleLogin}>
        <Text style={styles.buttonText}>Google로 로그인</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    backgroundColor: "#fff",
    padding: 24,
  },
  title: {
    fontSize: 32,
    fontWeight: "700",
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: "#666",
    marginBottom: 48,
  },
  button: {
    backgroundColor: "#4285F4",
    paddingHorizontal: 32,
    paddingVertical: 14,
    borderRadius: 8,
  },
  buttonText: {
    color: "#fff",
    fontSize: 16,
    fontWeight: "600",
  },
});
