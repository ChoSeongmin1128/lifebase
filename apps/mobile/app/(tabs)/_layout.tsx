import { Tabs } from "expo-router";
import { Text } from "react-native";

function TabIcon({ name, focused }: { name: string; focused: boolean }) {
  return (
    <Text style={{ fontSize: 12, color: focused ? "#000" : "#999" }}>
      {name}
    </Text>
  );
}

export default function TabLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: true,
        tabBarActiveTintColor: "#000",
        tabBarInactiveTintColor: "#999",
      }}
    >
      <Tabs.Screen
        name="cloud"
        options={{
          title: "Cloud",
          tabBarIcon: ({ focused }) => (
            <TabIcon name="☁️" focused={focused} />
          ),
        }}
      />
      <Tabs.Screen
        name="gallery"
        options={{
          title: "갤러리",
          tabBarIcon: ({ focused }) => (
            <TabIcon name="🖼" focused={focused} />
          ),
        }}
      />
      <Tabs.Screen
        name="calendar"
        options={{
          title: "캘린더",
          tabBarIcon: ({ focused }) => (
            <TabIcon name="📅" focused={focused} />
          ),
        }}
      />
      <Tabs.Screen
        name="todo"
        options={{
          title: "Todo",
          tabBarIcon: ({ focused }) => (
            <TabIcon name="✓" focused={focused} />
          ),
        }}
      />
      <Tabs.Screen
        name="settings"
        options={{
          title: "설정",
          tabBarIcon: ({ focused }) => (
            <TabIcon name="⚙" focused={focused} />
          ),
        }}
      />
    </Tabs>
  );
}
