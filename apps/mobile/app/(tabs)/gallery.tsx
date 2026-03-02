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
import { useGalleryActions } from "../../features/gallery/ui/hooks/useGalleryActions";
import type { MediaItem } from "../../features/gallery/domain/MediaItem";

const COLUMN_COUNT = 3;
const SCREEN_WIDTH = Dimensions.get("window").width;
const ITEM_SIZE = SCREEN_WIDTH / COLUMN_COUNT - 2;

import Constants from "expo-constants";

const API_BASE =
  (Constants.expoConfig?.extra?.apiUrl as string) ||
  "https://lifebase.cc/api/v1";

export default function GalleryScreen() {
  const [media, setMedia] = useState<MediaItem[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const { listMedia } = useGalleryActions();

  const load = useCallback(async () => {
    try {
      const data = await listMedia();
      setMedia(data || []);
    } catch {
      setMedia([]);
    }
  }, [listMedia]);

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
