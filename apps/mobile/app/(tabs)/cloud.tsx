import { useState, useEffect, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  RefreshControl,
} from "react-native";
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

  const getIcon = (item: FolderItem): { label: string; color: string; background: string } => {
    if (item.type === "folder") {
      return { label: "FD", color: "#92400e", background: "#fef3c7" };
    }

    const mime = (item.mime_type || "").toLowerCase();
    if (mime.startsWith("image/")) return { label: "IMG", color: "#065f46", background: "#d1fae5" };
    if (mime.startsWith("video/")) return { label: "VID", color: "#991b1b", background: "#fee2e2" };
    if (mime.startsWith("audio/")) return { label: "AUD", color: "#5b21b6", background: "#ede9fe" };
    if (mime.includes("pdf")) return { label: "PDF", color: "#991b1b", background: "#fee2e2" };
    if (mime.includes("javascript") || mime.includes("typescript") || mime.includes("json") || mime.includes("html") || mime.includes("css")) {
      return { label: "</>", color: "#0c4a6e", background: "#e0f2fe" };
    }
    return { label: "DOC", color: "#334155", background: "#e2e8f0" };
  };

  return (
    <View style={styles.container}>
      {path.length > 1 && (
        <TouchableOpacity style={styles.breadcrumb} onPress={goBack}>
          <Text style={styles.breadcrumbText}>
            ← {path[path.length - 2].name}
          </Text>
        </TouchableOpacity>
      )}
      <FlatList
        data={items}
        keyExtractor={(item) => item.id}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={styles.empty}>파일이 없습니다</Text>
        }
        renderItem={({ item }) => (
          (() => {
            const icon = getIcon(item);
            return (
              <TouchableOpacity
                style={styles.row}
                onPress={() => {
                  if (item.type === "folder") openFolder(item.id, item.name);
                }}
              >
                <View style={[styles.iconWrap, { backgroundColor: icon.background }]}>
                  <Text style={[styles.iconLabel, { color: icon.color }]}>{icon.label}</Text>
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
  container: { flex: 1, backgroundColor: "#fff" },
  breadcrumb: { padding: 12, borderBottomWidth: 1, borderBottomColor: "#eee" },
  breadcrumbText: { fontSize: 14, color: "#666" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    padding: 14,
    borderBottomWidth: 1,
    borderBottomColor: "#f0f0f0",
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
  meta: { fontSize: 12, color: "#999", marginTop: 2 },
  empty: { textAlign: "center", marginTop: 60, color: "#999", fontSize: 14 },
});
