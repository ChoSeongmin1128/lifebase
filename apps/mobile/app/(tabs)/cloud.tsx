import { useState, useEffect, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  RefreshControl,
} from "react-native";
import { getCloudItemToken } from "@lifebase/design-tokens";
import { useCloudActions } from "../../features/cloud/ui/hooks/useCloudActions";
import type { FolderItem } from "../../features/cloud/domain/CloudItem";

export default function CloudScreen() {
  const [items, setItems] = useState<FolderItem[]>([]);
  const [folderId, setFolderId] = useState<string | null>(null);
  const [path, setPath] = useState<{ id: string | null; name: string }[]>([
    { id: null, name: "내 파일" },
  ]);
  const [refreshing, setRefreshing] = useState(false);
  const { listItems } = useCloudActions();

  const load = useCallback(async () => {
    try {
      const data = await listItems(folderId);
      setItems(data || []);
    } catch {
      setItems([]);
    }
  }, [folderId, listItems]);

  useEffect(() => {
    load();
  }, [load]);

  const onRefresh = async () => {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  };

  const openFolder = (id: string, name: string) => {
    setFolderId(id);
    setPath((prev) => [...prev, { id, name }]);
  };

  const goBack = () => {
    if (path.length <= 1) return;
    const newPath = path.slice(0, -1);
    setPath(newPath);
    setFolderId(newPath[newPath.length - 1].id);
  };

  const formatSize = (bytes?: number) => {
    if (!bytes) return "";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <View style={styles.container}>
      <View style={styles.headerCard}>
        <Text style={styles.screenTitle}>Cloud</Text>
        <Text style={styles.screenSubtitle}>파일과 폴더를 같은 작업 톤으로 탐색합니다.</Text>
      </View>
      {path.length > 1 ? (
        <TouchableOpacity style={styles.breadcrumbCard} onPress={goBack}>
          <Text style={styles.breadcrumbText}>← {path[path.length - 2].name}</Text>
        </TouchableOpacity>
      ) : (
        <View style={styles.breadcrumbCard}>
          <Text style={styles.breadcrumbText}>내 파일</Text>
        </View>
      )}
      <FlatList
        data={items}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={styles.empty}>파일이 없습니다</Text>
        }
        renderItem={({ item }) => (
          (() => {
            const icon = item.type === "folder"
              ? getCloudItemToken({ type: "folder" })
              : getCloudItemToken({ type: "file", mimeType: item.mime_type });
            return (
              <TouchableOpacity
                style={styles.row}
                onPress={() => {
                  if (item.type === "folder") openFolder(item.id, item.name);
                }}
              >
                <View style={[styles.iconWrap, { backgroundColor: icon.background }]}>
                  <Text style={[styles.iconLabel, { color: icon.foreground }]}>{icon.label}</Text>
                </View>
                <View style={styles.info}>
                  <Text style={styles.name} numberOfLines={1}>
                    {item.name}
                  </Text>
                  <Text style={styles.meta}>
                    {item.type === "file" ? formatSize(item.size_bytes) : "폴더"}
                  </Text>
                </View>
              </TouchableOpacity>
            );
          })()
        )}
      />
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
  breadcrumbCard: {
    paddingHorizontal: 14,
    paddingVertical: 12,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
    borderRadius: 16,
    marginBottom: 12,
  },
  breadcrumbText: { fontSize: 14, color: "#5E6B67", fontWeight: "500" },
  listContent: {
    paddingBottom: 16,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    borderRadius: 20,
    overflow: "hidden",
    backgroundColor: "#FFFFFF",
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    padding: 14,
    borderBottomWidth: 1,
    borderBottomColor: "#E5ECE9",
  },
  iconWrap: {
    width: 28,
    height: 28,
    borderRadius: 8,
    alignItems: "center",
    justifyContent: "center",
    marginRight: 12,
  },
  iconLabel: { fontSize: 10, fontWeight: "700" },
  info: { flex: 1 },
  name: { fontSize: 15, fontWeight: "500" },
  meta: { fontSize: 12, color: "#7B8784", marginTop: 2 },
  empty: { textAlign: "center", marginTop: 60, color: "#7B8784", fontSize: 14 },
});
