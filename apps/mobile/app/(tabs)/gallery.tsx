import { useState, useEffect, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  Image,
  StyleSheet,
  Dimensions,
  RefreshControl,
} from "react-native";
import { api } from "../../lib/api";
import { getAccessToken } from "../../lib/auth";

const COLUMN_COUNT = 3;
const SCREEN_WIDTH = Dimensions.get("window").width;
const ITEM_SIZE = SCREEN_WIDTH / COLUMN_COUNT - 2;

type MediaItem = {
  id: string;
  name: string;
  mime_type: string;
  taken_at?: string;
  created_at: string;
};

import Constants from "expo-constants";

const API_BASE =
  (Constants.expoConfig?.extra?.apiUrl as string) ||
  "https://lifebase.cc/api/v1";

export default function GalleryScreen() {
  const [media, setMedia] = useState<MediaItem[]>([]);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    const token = await getAccessToken();
    if (!token) return;
    try {
      const data = await api<{ items: MediaItem[] }>("/gallery", { token });
      setMedia(data.items || []);
    } catch {
      setMedia([]);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const onRefresh = async () => {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  };

  return (
    <View style={styles.container}>
      <FlatList
        data={media}
        keyExtractor={(item) => item.id}
        numColumns={COLUMN_COUNT}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={styles.empty}>사진/동영상이 없습니다</Text>
        }
        renderItem={({ item }) => (
          <Image
            source={{
              uri: `${API_BASE}/gallery/thumbnails/${item.id}/small`,
            }}
            style={styles.thumb}
          />
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: "#fff" },
  thumb: {
    width: ITEM_SIZE,
    height: ITEM_SIZE,
    margin: 1,
    backgroundColor: "#f0f0f0",
  },
  empty: { textAlign: "center", marginTop: 60, color: "#999", fontSize: 14 },
});
