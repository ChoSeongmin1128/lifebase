"use client";

import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useState, useEffect, useCallback, useMemo, useRef, Suspense } from "react";
import { getCloudItemToken } from "@lifebase/design-tokens";
import { useCloudActions } from "@/features/cloud/ui/hooks/useCloudActions";
import type { CloudFile, CloudSection, FolderData, FolderItem } from "@/features/cloud/domain/CloudItem";
import { isAuthenticated } from "@/features/auth/infrastructure/token-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { CloudFolderHeader, type CloudPathEntry } from "@/components/cloud/CloudFolderHeader";
import { FileIcon } from "@/components/cloud/FileIcon";
import { ThumbnailImage } from "@/components/cloud/ThumbnailImage";
import { BulkActionBar } from "@/components/cloud/BulkActionBar";
import { PageToolbar, PageToolbarGroup } from "@/components/layout/PageToolbar";
import { useToast } from "@/components/providers/ToastProvider";
import {
  CLOUD_SECTION_ITEMS,
  CLOUD_SECTION_LABELS,
  parseCloudSection,
} from "@/lib/cloud-sections";
import {
  ArrowUpDown,
  ClipboardPaste,
  Copy,
  FolderPlus,
  FilePlus,
  Upload,
  Folder,
  Cloud,
  MoreVertical,
  Download,
  Pencil,
  Trash2,
  FolderOpen,
  LayoutGrid,
  List,
  Search,
  Scissors,
  Star as StarIcon,
  Undo2,
} from "lucide-react";

type SortBy = "name" | "size" | "updated_at" | "created_at";
type SortDir = "asc" | "desc";
type ViewMode = "list" | "grid";
type ClipboardMode = "copy" | "cut";
type ClipboardItemType = "file" | "folder";
type FolderRouteState = "ready" | "checking" | "invalid" | "not_found" | "error";
interface CloudClipboardItem {
  itemType: ClipboardItemType;
  itemID: string;
  itemName: string;
}
interface CloudClipboard {
  mode: ClipboardMode;
  items: CloudClipboardItem[];
  summary: string;
}
interface CloudDragItem {
  itemType: ClipboardItemType;
  itemID: string;
  parentFolderID: string | null;
}

interface PendingCloudDeletion {
  cancel: () => void;
  flush: () => Promise<void>;
}

interface PendingTrashEmpty {
  cancel: () => void;
  flush: () => Promise<void>;
}

interface CloudDeleteTarget {
  id: string;
  type: "folder" | "file";
}

const INTERNAL_ITEM_DRAG_TYPE = "application/x-lifebase-cloud-item";
const ROOT_PATH_ENTRY: CloudPathEntry = { id: null, name: "내 클라우드" };
const TRASH_ROOT_PATH_ENTRY: CloudPathEntry = { id: null, name: "휴지통" };
const DELETE_UNDO_WINDOW_MS = 5_000;
const cloudPathCache = new Map<string, CloudPathEntry[]>();
const cloudFolderCache = new Map<string, FolderData>();
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

const getCloudLocationKey = (section: CloudSection, folderId: string | null) => `${section}:${folderId || "root"}`;

const isTextInputTarget = (target: EventTarget | null) => {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
};

const buildClipboardSummary = (items: CloudClipboardItem[]) => {
  if (items.length === 0) return "";
  if (items.length === 1) return items[0].itemName;

  const fileCount = items.filter((item) => item.itemType === "file").length;
  const folderCount = items.length - fileCount;
  if (fileCount === items.length) return `파일 ${items.length}개`;
  if (folderCount === items.length) return `폴더 ${items.length}개`;
  return `항목 ${items.length}개`;
};

const arePathEntriesEqual = (a: CloudPathEntry[], b: CloudPathEntry[]) => (
  a.length === b.length
  && a.every((entry, index) => entry.id === b[index]?.id && entry.name === b[index]?.name)
);

const isValidUUID = (value: string) => UUID_PATTERN.test(value);

const isNotFoundError = (error: unknown) => (
  error instanceof Error
  && error.message.toLowerCase().includes("not found")
);

function CloudPageInner() {
  const params = useParams<{ folderId?: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const routeFolderId = typeof params.folderId === "string" ? params.folderId : null;
  const folderQuery = searchParams.get("folder");
  const sectionQuery = parseCloudSection(searchParams.get("section")) as CloudSection;
  const section = routeFolderId ? "" : sectionQuery;
  const quickAction = searchParams.get("quick");
  const isMyFilesSection = section === "";
  const isTrashSection = section === "trash";
  const isRecentSection = section === "recent";
  const isSharedSection = section === "shared";
  const isStarredSection = section === "starred";
  const isSelectableSection = isMyFilesSection || isTrashSection;

  const resolvedFolderID = useMemo(() => {
    if (section === "") {
      return routeFolderId || folderQuery || null;
    }
    if (section === "trash") {
      return folderQuery || null;
    }
    return null;
  }, [folderQuery, routeFolderId, section]);
  const locationKey = useMemo(() => getCloudLocationKey(section, resolvedFolderID), [resolvedFolderID, section]);
  const initialRootPathEntry = section === "trash" ? TRASH_ROOT_PATH_ENTRY : ROOT_PATH_ENTRY;
  const initialPath = useMemo(() => {
    const cached = cloudPathCache.get(locationKey);
    if (cached && cached.length > 0) {
      return cached;
    }
    return [initialRootPathEntry];
  }, [initialRootPathEntry, locationKey]);
  const [activeFolderID, setActiveFolderID] = useState<string | null>(resolvedFolderID);
  const [items, setItems] = useState<FolderItem[]>([]);
  const [path, setPath] = useState<CloudPathEntry[]>(initialPath);
  const [pathLoading, setPathLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [folderRouteState, setFolderRouteState] = useState<FolderRouteState>(resolvedFolderID ? "checking" : "ready");
  const [folderRouteReloadKey, setFolderRouteReloadKey] = useState(0);
  const [hasLoadedItems, setHasLoadedItems] = useState(false);
  const [sortBy, setSortBy] = useState<SortBy>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [renaming, setRenaming] = useState<{ id: string; type: "folder" | "file"; name: string } | null>(null);
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [creatingFolder, setCreatingFolder] = useState(false);
  const [showNewFile, setShowNewFile] = useState(false);
  const [newFileName, setNewFileName] = useState("");
  const [newFileExtension, setNewFileExtension] = useState<"md" | "txt">("txt");
  const [creatingFile, setCreatingFile] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingFile, setEditingFile] = useState<CloudFile | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [savingContent, setSavingContent] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<CloudFile[] | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [draggingItem, setDraggingItem] = useState<CloudDragItem | null>(null);
  const [dropTargetFolderId, setDropTargetFolderId] = useState<string | null>(null);
  const [movingItemKey, setMovingItemKey] = useState<string | null>(null);
  const [clipboard, setClipboard] = useState<CloudClipboard | null>(null);
  const [clipboardBusy, setClipboardBusy] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [starredKeys, setStarredKeys] = useState<Set<string>>(new Set());

  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderCacheRef = useRef<Map<string, FolderData>>(new Map(cloudFolderCache));
  const itemsRequestRef = useRef(0);
  const pathRequestRef = useRef(0);
  const folderRouteRequestRef = useRef(0);
  const quickActionHandledRef = useRef(false);
  const pendingDeletionRef = useRef<PendingCloudDeletion | null>(null);
  const pendingTrashEmptyRef = useRef<PendingTrashEmpty | null>(null);
  const locationKeyRef = useRef(locationKey);

  const currentFolderID = activeFolderID;
  const hasLegacyFolderQuery = isMyFilesSection && !routeFolderId && !!folderQuery;
  const rootPathEntry = isTrashSection ? TRASH_ROOT_PATH_ENTRY : ROOT_PATH_ENTRY;
  const authed = isAuthenticated();
  const cloud = useCloudActions();
  const toast = useToast();

  const updatePathState = useCallback((nextPath: CloudPathEntry[], folderId: string | null = currentFolderID) => {
    setPath((prev) => (arePathEntriesEqual(prev, nextPath) ? prev : nextPath));
    cloudPathCache.set(getCloudLocationKey(section, folderId), nextPath);
  }, [currentFolderID, section]);

  useEffect(() => {
    setActiveFolderID(resolvedFolderID);
  }, [resolvedFolderID]);

  useEffect(() => {
    if ((!isMyFilesSection && !isTrashSection) || !authed) {
      folderRouteRequestRef.current += 1;
      setFolderRouteState("ready");
      return;
    }
    if (!currentFolderID) {
      folderRouteRequestRef.current += 1;
      setFolderRouteState("ready");
      return;
    }
    if (!isValidUUID(currentFolderID)) {
      folderRouteRequestRef.current += 1;
      itemsRequestRef.current += 1;
      pathRequestRef.current += 1;
      setFolderRouteState("invalid");
      setItems([]);
      setLoading(false);
      setPathLoading(false);
      updatePathState([rootPathEntry], null);
      return;
    }

    const requestId = folderRouteRequestRef.current + 1;
    folderRouteRequestRef.current = requestId;
    setFolderRouteState("checking");
    setLoading(true);
    setPathLoading(true);

    void (async () => {
      try {
        const folder = isTrashSection
          ? await cloud.getTrashFolder(currentFolderID)
          : await cloud.getFolder(currentFolderID);
        if (requestId !== folderRouteRequestRef.current) return;
        folderCacheRef.current.set(folder.id, folder);
        cloudFolderCache.set(folder.id, folder);
        setFolderRouteState("ready");
      } catch (error) {
        if (requestId !== folderRouteRequestRef.current) return;
        itemsRequestRef.current += 1;
        pathRequestRef.current += 1;
        setFolderRouteState(isNotFoundError(error) ? "not_found" : "error");
        setItems([]);
        setLoading(false);
        setPathLoading(false);
        updatePathState([rootPathEntry], null);
      }
    })();
  }, [
    authed,
    cloud,
    currentFolderID,
    folderRouteReloadKey,
    isMyFilesSection,
    isTrashSection,
    rootPathEntry,
    updatePathState,
  ]);

  const buildCloudHref = useCallback((targetSection: CloudSection, folderId?: string | null, quick?: string | null) => {
    const params = new URLSearchParams();
    if (targetSection) {
      params.set("section", targetSection);
    }
    if (targetSection === "trash" && folderId) {
      params.set("folder", folderId);
    }
    if (quick) {
      params.set("quick", quick);
    }
    const query = params.toString();
    if (!targetSection && folderId) {
      const basePath = `/cloud/folders/${encodeURIComponent(folderId)}`;
      return query ? `${basePath}?${query}` : basePath;
    }
    return query ? `/cloud?${query}` : "/cloud";
  }, []);

  const syncFolderUrl = useCallback(
    (folderId: string | null, mode: "push" | "replace" = "replace", nextQuickAction?: string | null) => {
      const href = buildCloudHref(section, folderId, nextQuickAction);
      if (mode === "push") {
        router.push(href, { scroll: false });
        return;
      }
      router.replace(href, { scroll: false });
    },
    [buildCloudHref, router, section],
  );

  const toItemMeta = (item: FolderItem): { id: string; type: "folder" | "file" } => {
    if (item.type === "folder") {
      return { id: item.folder!.id, type: "folder" };
    }
    return { id: item.file!.id, type: "file" };
  };

  const toStarKey = (itemID: string, itemType: "folder" | "file") => `${itemType}:${itemID}`;

  const isItemStarred = (item: FolderItem) => {
    const { id, type } = toItemMeta(item);
    return starredKeys.has(toStarKey(id, type));
  };

  const loadItems = useCallback(async () => {
    if (!authed) {
      itemsRequestRef.current += 1;
      setLoading(false);
      return;
    }

    if ((isMyFilesSection || isTrashSection) && currentFolderID && folderRouteState !== "ready") {
      itemsRequestRef.current += 1;
      setItems([]);
      setLoading(folderRouteState === "checking");
      setHasLoadedItems(folderRouteState !== "checking");
      return;
    }

    const requestId = itemsRequestRef.current + 1;
    itemsRequestRef.current = requestId;
    const showInitialLoading = !hasLoadedItems;
    setLoading(showInitialLoading);
    try {
      const nextItems = await cloud.listItems({
        section,
        folderId: currentFolderID,
        sortBy,
        sortDir,
      });
      if (requestId !== itemsRequestRef.current) return;
      setItems(nextItems);
      if (isMyFilesSection || isTrashSection) {
        nextItems.forEach((item) => {
          if (item.type === "folder" && item.folder) {
            folderCacheRef.current.set(item.folder.id, item.folder);
            cloudFolderCache.set(item.folder.id, item.folder);
          }
        });
      }
    } catch {
      if (requestId !== itemsRequestRef.current) return;
      setItems([]);
    } finally {
      if (requestId !== itemsRequestRef.current) return;
      setLoading(false);
      setHasLoadedItems(true);
    }
  }, [
    authed,
    hasLoadedItems,
    currentFolderID,
    cloud,
    section,
    sortBy,
    sortDir,
    folderRouteState,
    isMyFilesSection,
    isTrashSection,
  ]);

  const loadFolderPath = useCallback(async () => {
    if ((!isMyFilesSection && !isTrashSection) || !authed) {
      pathRequestRef.current += 1;
      updatePathState([rootPathEntry], null);
      setPathLoading(false);
      return;
    }
    if (!currentFolderID) {
      pathRequestRef.current += 1;
      updatePathState([rootPathEntry], null);
      setPathLoading(false);
      return;
    }
    if (folderRouteState !== "ready") {
      pathRequestRef.current += 1;
      if (folderRouteState !== "checking") {
        updatePathState([rootPathEntry], null);
      }
      setPathLoading(folderRouteState === "checking");
      return;
    }

    const lastEntry = path[path.length - 1];
    const hasStablePath =
      path.length > 1
      && path[0]?.id === rootPathEntry.id
      && lastEntry?.id === currentFolderID;
    if (hasStablePath) {
      pathRequestRef.current += 1;
      setPathLoading(false);
      return;
    }

    const requestId = pathRequestRef.current + 1;
    pathRequestRef.current = requestId;
    setPathLoading(true);
    try {
      const visited = new Set<string>();
      const nextEntries: CloudPathEntry[] = [];
      let cursor: string | null = currentFolderID;

      while (cursor && !visited.has(cursor)) {
        visited.add(cursor);
        let folder = folderCacheRef.current.get(cursor);
        if (!folder) {
          folder = isTrashSection ? await cloud.getTrashFolder(cursor) : await cloud.getFolder(cursor);
          folderCacheRef.current.set(folder.id, folder);
          cloudFolderCache.set(folder.id, folder);
        }
        nextEntries.unshift({ id: folder.id, name: folder.name });
        cursor = folder.parent_id;
      }

      if (requestId !== pathRequestRef.current) return;
      updatePathState([rootPathEntry, ...nextEntries], currentFolderID);
    } catch {
      if (requestId !== pathRequestRef.current) return;
      const fallbackFolder = folderCacheRef.current.get(currentFolderID);
      updatePathState(
        fallbackFolder
          ? [rootPathEntry, { id: currentFolderID, name: fallbackFolder.name }]
          : [rootPathEntry],
        fallbackFolder ? currentFolderID : null,
      );
    } finally {
      if (requestId !== pathRequestRef.current) return;
      setPathLoading(false);
    }
  }, [authed, cloud, currentFolderID, folderRouteState, isMyFilesSection, isTrashSection, path, rootPathEntry, updatePathState]);

  const loadStars = useCallback(async () => {
    if (!authed) {
      setStarredKeys(new Set());
      return;
    }
    try {
      const stars = await cloud.listStars();
      const next = new Set((stars || []).map((star) => toStarKey(star.id, star.type)));
      setStarredKeys(next);
    } catch {
      setStarredKeys(new Set());
    }
  }, [authed, cloud]);

  useEffect(() => {
    if (!isMyFilesSection && !isTrashSection) {
      setPathLoading(false);
      updatePathState([rootPathEntry], null);
    }
    setRenaming(null);
    setShowNewFolder(false);
    setNewFolderName("");
    setShowNewFile(false);
    setNewFileName("");
    setNewFileExtension("txt");
    setSearchResults(null);
    setSearchQuery("");
    setSelectedIds(new Set());
    setDragOver(false);
    setDraggingItem(null);
    setDropTargetFolderId(null);
    setMovingItemKey(null);
    loadStars();
    loadItems();
  }, [isMyFilesSection, isTrashSection, loadItems, loadStars, rootPathEntry, updatePathState]);

  useEffect(() => {
    const nextKey = getCloudLocationKey(section, currentFolderID);
    if (locationKeyRef.current !== nextKey && pendingDeletionRef.current) {
      void pendingDeletionRef.current.flush();
    }
    if (locationKeyRef.current !== nextKey && pendingTrashEmptyRef.current) {
      void pendingTrashEmptyRef.current.flush();
    }
    locationKeyRef.current = nextKey;
  }, [currentFolderID, section]);

  useEffect(() => {
    return () => {
      if (pendingDeletionRef.current) {
        void pendingDeletionRef.current.flush();
      }
      if (pendingTrashEmptyRef.current) {
        void pendingTrashEmptyRef.current.flush();
      }
    };
  }, []);

  useEffect(() => {
    if (!hasLegacyFolderQuery || !folderQuery) return;
    router.replace(buildCloudHref("", folderQuery, quickAction), { scroll: false });
  }, [buildCloudHref, hasLegacyFolderQuery, folderQuery, quickAction, router]);

  useEffect(() => {
    if (!isMyFilesSection && !isTrashSection) {
      return;
    }
    loadFolderPath();
  }, [isMyFilesSection, isTrashSection, loadFolderPath]);

  useEffect(() => {
    if (quickAction !== "upload") return;
    if (quickActionHandledRef.current) return;
    if (!isMyFilesSection || !authed) return;

    quickActionHandledRef.current = true;
    setTimeout(() => {
      fileInputRef.current?.click();
    }, 0);

    router.replace(buildCloudHref(section, currentFolderID), { scroll: false });
  }, [quickAction, isMyFilesSection, authed, router, buildCloudHref, section, currentFolderID]);

  const navigateToFolder = (folder: FolderData) => {
    if (!isMyFilesSection && !isTrashSection) return;
    setActiveFolderID(folder.id);
    folderCacheRef.current.set(folder.id, folder);
    cloudFolderCache.set(folder.id, folder);
    setPath((prev) => {
      const lastEntry = prev[prev.length - 1];
      if (lastEntry?.id === folder.id) {
        return prev;
      }
      const nextPath = [...prev, { id: folder.id, name: folder.name }];
      cloudPathCache.set(getCloudLocationKey(section, folder.id), nextPath);
      return nextPath;
    });
    syncFolderUrl(folder.id, "push");
  };

  const navigateToHeaderPath = (folderId: string | null) => {
    if (!isMyFilesSection && !isTrashSection) return;
    setActiveFolderID(folderId);
    setPath((prev) => {
      if (folderId === null) {
        const nextPath = [rootPathEntry];
        cloudPathCache.set(getCloudLocationKey(section, null), nextPath);
        return nextPath;
      }
      const existingIndex = prev.findIndex((entry) => entry.id === folderId);
      if (existingIndex >= 0) {
        const nextPath = prev.slice(0, existingIndex + 1);
        cloudPathCache.set(getCloudLocationKey(section, folderId), nextPath);
        return nextPath;
      }
      const cached = folderCacheRef.current.get(folderId);
      const nextPath = [rootPathEntry, { id: folderId, name: cached?.name || "폴더" }];
      cloudPathCache.set(getCloudLocationKey(section, folderId), nextPath);
      return nextPath;
    });
    syncFolderUrl(folderId, "push");
  };

  const handleUpload = async (files: FileList | null) => {
    if (!files || !authed || !isMyFilesSection) return;
    for (const file of Array.from(files)) {
      try {
        await cloud.uploadFile(file, currentFolderID);
      } catch (err) {
        console.error("Upload failed:", err);
      }
    }
    loadItems();
  };

  const handleDownload = async (file: CloudFile) => {
    if (!authed) return;
    try {
      const { blob, filename } = await cloud.downloadFile(file.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename && filename !== "download" ? filename : file.name;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download failed:", err);
    }
  };

  const isEditableTextFile = (file: CloudFile) => {
    const lowerName = file.name.toLowerCase();
    if (lowerName.endsWith(".md") || lowerName.endsWith(".txt")) return true;
    return file.mime_type.startsWith("text/");
  };

  const openTextEditor = async (file: CloudFile) => {
    if (!authed || !isMyFilesSection || !isEditableTextFile(file)) return;
    try {
      const content = await cloud.getTextFileContent(file.id);
      setEditingFile(file);
      setEditorContent(content);
      setEditorOpen(true);
    } catch (err) {
      console.error("Open editor failed:", err);
    }
  };

  const saveTextEditor = async () => {
    if (!editingFile || savingContent) return;
    setSavingContent(true);
    try {
      await cloud.updateTextFileContent(editingFile.id, editorContent);
      setEditorOpen(false);
      setEditingFile(null);
      await loadItems();
    } catch (err) {
      console.error("Save editor failed:", err);
    } finally {
      setSavingContent(false);
    }
  };

  const handleDelete = async (item: FolderItem) => {
    if (!authed || !isMyFilesSection) return;
    const target =
      item.type === "folder" && item.folder
        ? { id: item.folder.id, type: "folder" as const }
        : item.type === "file" && item.file
          ? { id: item.file.id, type: "file" as const }
          : null;
    if (!target) return;
    await queueCloudDelete([target]);
  };

  const handleBulkDelete = async () => {
    if (!authed || !isMyFilesSection) return;
    const targets = displayItems
      .map((item) =>
        item.type === "folder"
          ? { id: item.folder!.id, type: "folder" as const }
          : { id: item.file!.id, type: "file" as const },
      )
      .filter((target) => selectedIds.has(target.id));
    if (targets.length === 0) return;
    await queueCloudDelete(targets);
  };

  const handleBulkDownload = async () => {
    if (!isMyFilesSection) return;
    for (const item of displayItems) {
      if (item.type !== "file" || !item.file) continue;
      const id = item.file.id;
      if (!selectedIds.has(id)) continue;
      await handleDownload(item.file);
    }
  };

  const handleToggleStar = async (item: FolderItem) => {
    if (!authed) return;
    const { id, type } = toItemMeta(item);
    const isStarred = starredKeys.has(toStarKey(id, type));
    try {
      if (isStarred) {
        await cloud.removeStar(id, type);
      } else {
        await cloud.addStar(id, type);
      }
      await loadStars();
      if (isStarredSection) {
        await loadItems();
      }
    } catch (err) {
      console.error("Toggle star failed:", err);
    }
  };

  const handleRestore = async (item: FolderItem) => {
    if (!authed || !isTrashSection) return;
    try {
      const id = item.type === "folder" ? item.folder?.id : item.file?.id;
      if (!id) return;
      await cloud.restoreTrashItem(id, item.type);
      await loadItems();
    } catch (err) {
      console.error("Restore failed:", err);
    }
  };

  const handleBulkRestore = async () => {
    if (!authed || !isTrashSection) return;
    for (const item of displayItems) {
      const id = item.type === "folder" ? item.folder?.id : item.file?.id;
      if (!id || !selectedIds.has(id)) continue;
      try {
        await cloud.restoreTrashItem(id, item.type);
      } catch (err) {
        console.error("Restore failed:", err);
      }
    }
    setSelectedIds(new Set());
    await loadItems();
  };

  const handleEmptyTrash = async () => {
    if (!authed || !isTrashSection) return;
    if (pendingTrashEmptyRef.current) {
      await pendingTrashEmptyRef.current.flush();
    }

    const itemsSnapshot = items;
    const pathSnapshot = path;
    const selectedIdsSnapshot = selectedIds;
    const folderSnapshot = currentFolderID;
    let settled = false;
    let timerID = 0;

    const restore = () => {
      setItems(itemsSnapshot);
      setPath(pathSnapshot);
      setSelectedIds(selectedIdsSnapshot);
      setActiveFolderID(folderSnapshot);
      syncFolderUrl(folderSnapshot, "replace");
    };

    setItems([]);
    setPath([TRASH_ROOT_PATH_ENTRY]);
    setSelectedIds(new Set());
    setActiveFolderID(null);
    syncFolderUrl(null, "replace");

    const finalize = async () => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timerID);
      pendingTrashEmptyRef.current = null;

      try {
        await cloud.emptyTrash();
        await loadItems();
      } catch (err) {
        restore();
        console.error("Empty trash failed:", err);
        const msg = err instanceof Error ? err.message : "알 수 없는 오류";
        toast.error("휴지통 비우기에 실패했습니다", msg);
      }
    };

    const cancel = () => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timerID);
      pendingTrashEmptyRef.current = null;
      restore();
      toast.success("복원됨");
    };

    pendingTrashEmptyRef.current = { cancel, flush: finalize };
    timerID = window.setTimeout(() => {
      void finalize();
    }, DELETE_UNDO_WINDOW_MS);

    toast.show({
      variant: "warning",
      title: "휴지통 비워짐",
      duration: DELETE_UNDO_WINDOW_MS,
      actionLabel: "실행 취소",
      onAction: cancel,
    });
  };

  const handleRename = async () => {
    if (!renaming || !authed || !renaming.name.trim() || !isMyFilesSection) return;
    try {
      if (renaming.type === "folder") {
        await cloud.renameFolder(renaming.id, renaming.name);
      } else {
        await cloud.renameFile(renaming.id, renaming.name);
      }
      setRenaming(null);
      loadItems();
    } catch (err) {
      console.error("Rename failed:", err);
    }
  };

  const handleCreateFolder = async () => {
    if (!authed || !newFolderName.trim() || !isMyFilesSection || creatingFolder) return;
    setCreatingFolder(true);
    try {
      await cloud.createFolder(newFolderName, currentFolderID);
      setShowNewFolder(false);
      setNewFolderName("");
      loadItems();
    } catch (err) {
      console.error("Create folder failed:", err);
    } finally {
      setCreatingFolder(false);
    }
  };

  const handleCreateFile = async () => {
    if (!authed || !newFileName.trim() || !isMyFilesSection || creatingFile) return;
    setCreatingFile(true);
    try {
      await cloud.createTextFile(newFileName, newFileExtension, currentFolderID);
      setShowNewFile(false);
      setNewFileName("");
      setNewFileExtension("txt");
      loadItems();
    } catch (err) {
      console.error("Create file failed:", err);
    } finally {
      setCreatingFile(false);
    }
  };

  const handleSearch = async () => {
    if (!authed || !searchQuery.trim() || !isMyFilesSection) {
      setSearchResults(null);
      return;
    }
    try {
      const files = await cloud.searchFiles(searchQuery);
      setSearchResults(files || []);
    } catch {
      setSearchResults([]);
    }
  };

  const refreshVisibleItems = useCallback(async () => {
    await loadItems();
    if (!authed || !isMyFilesSection || searchResults === null || !searchQuery.trim()) {
      return;
    }
    try {
      const files = await cloud.searchFiles(searchQuery);
      setSearchResults(files || []);
    } catch {
      setSearchResults([]);
    }
  }, [authed, cloud, isMyFilesSection, loadItems, searchQuery, searchResults]);

  const queueCloudDelete = useCallback(async (targets: CloudDeleteTarget[]) => {
    if (!authed || !isMyFilesSection || targets.length === 0) return;

    if (pendingDeletionRef.current) {
      await pendingDeletionRef.current.flush();
    }

    const targetIDs = new Set(targets.map((target) => target.id));
    const itemsSnapshot = items;
    const searchResultsSnapshot = searchResults;
    const selectedIdsSnapshot = selectedIds;
    let settled = false;

    setItems((prev) =>
      prev.filter((item) => {
        const id = item.type === "folder" ? item.folder!.id : item.file!.id;
        return !targetIDs.has(id);
      }),
    );
    setSearchResults((prev) => (prev ? prev.filter((file) => !targetIDs.has(file.id)) : prev));
    setSelectedIds((prev) => {
      const next = new Set(prev);
      targets.forEach((target) => next.delete(target.id));
      return next;
    });

    let timerID = 0;
    const restore = () => {
      setItems(itemsSnapshot);
      setSearchResults(searchResultsSnapshot);
      setSelectedIds(selectedIdsSnapshot);
    };

    const finalize = async () => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timerID);
      pendingDeletionRef.current = null;

      try {
        const results = await Promise.allSettled(
          targets.map((target) =>
            target.type === "folder" ? cloud.deleteFolder(target.id) : cloud.deleteFile(target.id),
          ),
        );
        const failed = results.filter((result) => result.status === "rejected").length;

        await refreshVisibleItems();

        if (failed !== 0) {
          toast.warning("일부 항목 삭제 실패", `${failed}개 항목을 삭제하지 못했습니다.`);
        }
      } catch (err) {
        restore();
        console.error("Delete failed:", err);
        const msg = err instanceof Error ? err.message : "알 수 없는 오류";
        toast.error("삭제에 실패했습니다", msg);
      }
    };

    const cancel = () => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timerID);
      pendingDeletionRef.current = null;
      restore();
      toast.success(targets.length === 1 ? "복원됨" : `항목 ${targets.length}개 복원됨`);
    };

    pendingDeletionRef.current = { cancel, flush: finalize };
    timerID = window.setTimeout(() => {
      void finalize();
    }, DELETE_UNDO_WINDOW_MS);

    const singleTarget = targets[0];
    const title = targets.length === 1
      ? singleTarget.type === "folder" ? "폴더 삭제됨" : "파일 삭제됨"
      : `항목 ${targets.length}개 삭제됨`;

    toast.show({
      variant: "warning",
      title,
      duration: DELETE_UNDO_WINDOW_MS,
      actionLabel: "실행 취소",
      onAction: cancel,
    });
  }, [
    authed,
    cloud,
    isMyFilesSection,
    items,
    refreshVisibleItems,
    searchResults,
    selectedIds,
    toast,
  ]);

  const getDragItemFromRow = (item: FolderItem): CloudDragItem => {
    if (item.type === "folder") {
      return {
        itemType: "folder",
        itemID: item.folder!.id,
        parentFolderID: item.folder!.parent_id,
      };
    }
    return {
      itemType: "file",
      itemID: item.file!.id,
      parentFolderID: item.file!.folder_id,
    };
  };

  const parseDraggedItem = (raw: string): CloudDragItem | null => {
    if (!raw) return null;
    try {
      const parsed = JSON.parse(raw) as Partial<CloudDragItem>;
      if (
        (parsed.itemType === "file" || parsed.itemType === "folder")
        && typeof parsed.itemID === "string"
        && (parsed.parentFolderID === null || typeof parsed.parentFolderID === "string")
      ) {
        return {
          itemType: parsed.itemType,
          itemID: parsed.itemID,
          parentFolderID: parsed.parentFolderID,
        };
      }
      return null;
    } catch {
      return null;
    }
  };

  const getDraggedItem = (e: React.DragEvent): CloudDragItem | null => {
    const fromData = parseDraggedItem(e.dataTransfer.getData(INTERNAL_ITEM_DRAG_TYPE));
    if (fromData) return fromData;
    return draggingItem;
  };

  const canDropToFolder = (dragItem: CloudDragItem, folderId: string) => {
    if (dragItem.itemType === "folder" && dragItem.itemID === folderId) return false;
    if (dragItem.parentFolderID === folderId) return false;
    return true;
  };

  const isUploadDragEvent = (e: React.DragEvent) => Array.from(e.dataTransfer.types).includes("Files");

  const handleMoveItemToFolder = async (dragItem: CloudDragItem, folderId: string) => {
    if (!authed || !isMyFilesSection || movingItemKey) return;
    if (!canDropToFolder(dragItem, folderId)) return;

    setMovingItemKey(`${dragItem.itemType}:${dragItem.itemID}`);
    try {
      if (dragItem.itemType === "file") {
        await cloud.moveFile(dragItem.itemID, folderId);
      } else {
        await cloud.moveFolder(dragItem.itemID, folderId);
      }
      setSelectedIds((prev) => {
        if (!prev.has(dragItem.itemID)) return prev;
        const next = new Set(prev);
        next.delete(dragItem.itemID);
        return next;
      });
      await loadItems();
    } catch (err) {
      const prefix = dragItem.itemType === "folder" ? "폴더 이동에 실패했습니다" : "파일 이동에 실패했습니다";
      console.error(prefix, err);
      const msg = err instanceof Error ? err.message : "알 수 없는 오류";
      toast.error(prefix, msg);
    } finally {
      setMovingItemKey(null);
    }
  };

  const handleFolderDragOver = (e: React.DragEvent, folderId: string) => {
    if (!isMyFilesSection || movingItemKey) return;
    const dragItem = getDraggedItem(e);
    if (!dragItem) return;
    if (!canDropToFolder(dragItem, folderId)) return;

    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = "move";
    if (dropTargetFolderId !== folderId) {
      setDropTargetFolderId(folderId);
    }
  };

  const handleFolderDrop = async (e: React.DragEvent, folderId: string) => {
    e.preventDefault();
    e.stopPropagation();
    const dragItem = getDraggedItem(e);
    setDropTargetFolderId(null);
    if (!dragItem) return;
    await handleMoveItemToFolder(dragItem, folderId);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    if (!isMyFilesSection) {
      setDragOver(false);
      return;
    }

    if (!isUploadDragEvent(e)) {
      setDragOver(false);
      return;
    }

    setDragOver(false);
    handleUpload(e.dataTransfer.files);
  };

  const toClipboardItem = useCallback((item: FolderItem): CloudClipboardItem => {
    if (item.type === "folder") {
      return {
        itemType: "folder",
        itemID: item.folder!.id,
        itemName: item.folder!.name,
      };
    }
    return {
      itemType: "file",
      itemID: item.file!.id,
      itemName: item.file!.name,
    };
  }, []);

  const setClipboardFromItems = useCallback((mode: ClipboardMode, items: FolderItem[]) => {
    if (!isMyFilesSection) return;
    if (items.length === 0) return;
    if (mode === "copy" && items.some((item) => item.type === "folder")) {
      toast.warning("폴더 복사는 지원되지 않습니다");
      return;
    }
    const clipboardItems = items.map(toClipboardItem);
    setClipboard({
      mode,
      items: clipboardItems,
      summary: buildClipboardSummary(clipboardItems),
    });
  }, [isMyFilesSection, toast, toClipboardItem]);

  const setClipboardFromItem = useCallback((mode: ClipboardMode, item: FolderItem) => {
    setClipboardFromItems(mode, [item]);
  }, [setClipboardFromItems]);

  const applyClipboardToFolder = useCallback(async (targetFolderID: string | null) => {
    if (!authed || !isMyFilesSection || !clipboard || clipboardBusy) return;

    setClipboardBusy(true);
    try {
      const results = await Promise.allSettled(
        clipboard.items.map((item) => {
          if (item.itemType === "file") {
            return clipboard.mode === "copy"
              ? cloud.copyFile(item.itemID, targetFolderID)
              : cloud.moveFile(item.itemID, targetFolderID);
          }
          return clipboard.mode === "copy"
            ? cloud.copyFolder(item.itemID, targetFolderID)
            : cloud.moveFolder(item.itemID, targetFolderID);
        }),
      );

      const failedItems = clipboard.items.filter((_, index) => results[index].status === "rejected");
      if (clipboard.mode === "cut") {
        if (failedItems.length === 0) {
          setClipboard(null);
        } else {
          setClipboard({
            mode: "cut",
            items: failedItems,
            summary: buildClipboardSummary(failedItems),
          });
        }
      }

      if (failedItems.length > 0) {
        toast.warning("일부 항목 붙여넣기 실패", `${failedItems.length}개 항목을 처리하지 못했습니다.`);
      }

      await loadItems();
    } finally {
      setClipboardBusy(false);
    }
  }, [authed, clipboard, clipboardBusy, cloud, isMyFilesSection, loadItems, toast]);

  const getSelectedItems = useCallback(() => {
    if (!isMyFilesSection || selectedIds.size === 0) return [];
    const currentItems: FolderItem[] = isMyFilesSection && searchResults
      ? searchResults.map((f) => ({ type: "file" as const, file: f, path: undefined }))
      : items;
    return currentItems.filter((item) => {
      const id = item.type === "folder" ? item.folder?.id : item.file?.id;
      return !!id && selectedIds.has(id);
    });
  }, [isMyFilesSection, items, searchResults, selectedIds]);

  const isClipboardCutItem = useCallback((itemType: ClipboardItemType, itemID: string) => {
    if (clipboard?.mode !== "cut") return false;
    return clipboard.items.some((item) => item.itemType === itemType && item.itemID === itemID);
  }, [clipboard]);

  const toggleSelect = (id: string) => {
    if (!isSelectableSection) return;
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (!isSelectableSection) return;
    if (selectedIds.size === displayItems.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(displayItems.map((i) => (i.type === "folder" ? i.folder!.id : i.file!.id))));
    }
  };

  useEffect(() => {
    if (!isSelectableSection) return;

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isTextInputTarget(e.target)) {
        if (selectedIds.size === 0 && !clipboard) return;
        e.preventDefault();
        setSelectedIds(new Set());
        setClipboard(null);
        return;
      }

      if (!(e.metaKey || e.ctrlKey) || e.altKey) return;
      if (isTextInputTarget(e.target)) return;

      const key = e.key.toLowerCase();
      const visibleItems: FolderItem[] = isMyFilesSection && searchResults
        ? searchResults.map((f) => ({ type: "file" as const, file: f, path: undefined }))
        : items;
      if (key === "a") {
        if (visibleItems.length === 0) return;
        e.preventDefault();
        setSelectedIds(new Set(visibleItems.map((item) => (item.type === "folder" ? item.folder!.id : item.file!.id))));
        return;
      }

      if (!isMyFilesSection) return;
      if (key !== "c" && key !== "x" && key !== "v") return;

      const selectedText = window.getSelection()?.toString() || "";
      if (selectedText && key !== "v") return;

      if (key === "v") {
        if (!clipboard || clipboardBusy) return;
        e.preventDefault();
        void applyClipboardToFolder(currentFolderID ?? null);
        return;
      }

      const selectedItems = getSelectedItems();
      if (selectedItems.length === 0) return;

      e.preventDefault();
      setClipboardFromItems(key === "x" ? "cut" : "copy", selectedItems);
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [
    applyClipboardToFolder,
    clipboard,
    clipboardBusy,
    currentFolderID,
    getSelectedItems,
    isMyFilesSection,
    isSelectableSection,
    items,
    searchResults,
    selectedIds,
    setClipboardFromItems,
  ]);

  const isMediaFile = (mimeType: string) =>
    mimeType.startsWith("image/") || mimeType.startsWith("video/");

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("ko-KR", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const formatRecentFilePath = (item: FolderItem) => {
    if (item.type !== "file") return "";
    if (item.path && item.path.startsWith("/")) return item.path;

    const folderSegments = (item.path || "")
      .split("/")
      .map((segment) => segment.trim())
      .filter((segment) => segment.length > 0 && segment !== "내 클라우드");
    return `/${[...folderSegments, item.file!.name].join("/")}`;
  };

  const sortOptions: { value: SortBy; label: string }[] = [
    { value: "name", label: "이름" },
    { value: "updated_at", label: "수정한 날짜" },
    { value: "created_at", label: "생성한 날짜" },
    { value: "size", label: "크기" },
  ];

  const displayItems: FolderItem[] = isMyFilesSection && searchResults
    ? searchResults.map((f) => ({ type: "file" as const, file: f, path: undefined }))
    : items;

  const sectionLabel = CLOUD_SECTION_LABELS[section];
  const hasFolderRouteError =
    currentFolderID !== null
    && (isMyFilesSection || isTrashSection)
    && folderRouteState !== "ready"
    && folderRouteState !== "checking";
  const folderActionsEnabled =
    !currentFolderID
    || !(isMyFilesSection || isTrashSection)
    || folderRouteState === "ready";
  const displayPath = (() => {
    if (!isMyFilesSection && !isTrashSection) return path;
    if (currentFolderID === null) return [rootPathEntry];
    if (folderRouteState !== "ready" && folderRouteState !== "checking") return [rootPathEntry];
    const lastEntry = path[path.length - 1];
    if (lastEntry?.id === currentFolderID) return path;
    const existingIndex = path.findIndex((entry) => entry.id === currentFolderID);
    if (existingIndex >= 0) return path.slice(0, existingIndex + 1);
    const cached = folderCacheRef.current.get(currentFolderID);
    return cached
      ? [rootPathEntry, { id: currentFolderID, name: cached.name }]
      : [rootPathEntry];
  })();
  const showFolderHeader = isMyFilesSection || isTrashSection;
  const headerLoading = folderRouteState === "checking" || (
    folderRouteState === "ready"
    && (pathLoading || (currentFolderID !== null && displayPath[displayPath.length - 1]?.id !== currentFolderID))
  );
  const showSearchResultBanner = isMyFilesSection && searchResults !== null;
  const showBulkBar = selectedIds.size > 0 && (isMyFilesSection || isTrashSection);
  const currentViewMode: ViewMode = isMyFilesSection ? viewMode : "list";
  const routeActionLabel = isTrashSection ? "휴지통으로 이동" : "내 클라우드로 이동";
  const routeStateCopy = (() => {
    if (!hasFolderRouteError) return null;
    if (folderRouteState === "invalid") {
      return {
        title: "잘못된 경로입니다",
        description: "폴더 주소를 다시 확인해 주세요.",
      };
    }
    if (folderRouteState === "not_found") {
      return {
        title: "폴더를 찾을 수 없습니다",
        description: "삭제되었거나 이동되었을 수 있습니다.",
      };
    }
    return {
      title: "폴더를 불러오지 못했습니다",
      description: "잠시 후 다시 시도해 주세요.",
    };
  })();

  const navigateToSectionRoot = useCallback(() => {
    syncFolderUrl(null, "replace");
  }, [syncFolderUrl]);

  const retryFolderRoute = useCallback(() => {
    setFolderRouteReloadKey((prev) => prev + 1);
  }, []);

  return (
    <div
      className="flex h-full flex-col"
      onDragOver={(e) => {
        if (!isMyFilesSection) return;
        if (!isUploadDragEvent(e)) return;
        e.preventDefault();
        setDragOver(true);
      }}
      onDragLeave={() => {
        setDragOver(false);
      }}
      onDrop={handleDrop}
    >
      {showFolderHeader && (
        <CloudFolderHeader
          path={displayPath}
          loading={headerLoading}
          onNavigate={navigateToHeaderPath}
        />
      )}

      <div className="flex gap-1.5 overflow-x-auto border-b border-border px-4 py-2 md:hidden">
        {CLOUD_SECTION_ITEMS.map((item) => {
          const isActive = item.section === section;
          const href = buildCloudHref(item.section);
          return (
            <Link
              key={item.section || "root"}
              href={href}
              className={`whitespace-nowrap rounded-full border px-3 py-1 text-xs transition-colors ${
                isActive
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border text-text-secondary"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </div>

      {/* Toolbar Row 2: Actions */}
      <PageToolbar>
        <PageToolbarGroup className="gap-2">
          {/* Search */}
          <div className="relative">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted" />
            <Input
              placeholder={isMyFilesSection ? "검색..." : `${sectionLabel}에서는 검색 미지원`}
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                if (!e.target.value) setSearchResults(null);
              }}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              disabled={!isMyFilesSection || !folderActionsEnabled}
              className="h-8 w-full md:w-48 pl-8"
            />
          </div>

          {/* View toggle */}
          {isMyFilesSection && (
            <div className="flex rounded-lg border border-border">
              <button
                onClick={() => setViewMode("list")}
                disabled={!folderActionsEnabled}
                className={`flex h-8 w-8 items-center justify-center rounded-l-lg ${
                  currentViewMode === "list" ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
                }`}
              >
                <List size={14} />
              </button>
              <button
                onClick={() => setViewMode("grid")}
                disabled={!folderActionsEnabled}
                className={`flex h-8 w-8 items-center justify-center rounded-r-lg ${
                  currentViewMode === "grid" ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
                }`}
              >
                <LayoutGrid size={14} />
              </button>
            </div>
          )}
        </PageToolbarGroup>

        <PageToolbarGroup className="gap-2">
          {isMyFilesSection && (
            <>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="gap-1.5">
                    <ArrowUpDown size={14} />
                    <span className="hidden md:inline">정렬</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {sortOptions.map((opt) => (
                    <DropdownMenuItem
                      key={opt.value}
                      disabled={!folderActionsEnabled}
                      onClick={() => {
                        if (sortBy === opt.value) {
                          setSortDir(sortDir === "asc" ? "desc" : "asc");
                        } else {
                          setSortBy(opt.value);
                          setSortDir(opt.value === "name" ? "asc" : "desc");
                        }
                      }}
                      className="justify-between"
                    >
                      {opt.label}
                      {sortBy === opt.value && (
                        <span className="text-xs text-text-muted">
                          {sortDir === "asc" ? "↑" : "↓"}
                        </span>
                      )}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>

              <Button
                variant="ghost"
                size="sm"
                disabled={!folderActionsEnabled}
                onClick={() => {
                  setShowNewFile(false);
                  setNewFileName("");
                  setShowNewFolder(true);
                }}
                className="gap-1.5"
              >
                <FolderPlus size={14} />
                <span className="hidden md:inline">새 폴더</span>
              </Button>

              <Button
                variant="ghost"
                size="sm"
                disabled={!folderActionsEnabled}
                onClick={() => {
                  setShowNewFolder(false);
                  setNewFolderName("");
                  setNewFileExtension("txt");
                  setShowNewFile(true);
                }}
                className="gap-1.5"
              >
                <FilePlus size={14} />
                <span className="hidden md:inline">새 파일</span>
              </Button>

              <Button
                variant="primary"
                size="sm"
                disabled={!folderActionsEnabled}
                onClick={() => fileInputRef.current?.click()}
                className="gap-1.5"
              >
                <Upload size={14} />
                <span className="hidden md:inline">업로드</span>
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={!clipboard || clipboardBusy || !folderActionsEnabled}
                onClick={() => void applyClipboardToFolder(currentFolderID ?? null)}
                className="gap-1.5"
              >
                <ClipboardPaste size={14} />
                <span className="hidden md:inline">붙여넣기</span>
              </Button>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={(e) => handleUpload(e.target.files)}
              />
            </>
          )}

          {isTrashSection && (
            <Button
              variant="ghost"
              size="sm"
              disabled={!folderActionsEnabled}
              onClick={handleEmptyTrash}
              className="gap-1.5 text-error hover:text-error"
            >
              <Trash2 size={14} />
              <span className="hidden md:inline">휴지통 비우기</span>
            </Button>
          )}
        </PageToolbarGroup>
      </PageToolbar>

      {/* New Folder Input */}
      {isMyFilesSection && showNewFolder && (
        <div className="flex items-center gap-2 border-b border-border bg-surface-accent/50 px-6 py-2">
          <Folder size={16} className="text-text-muted" />
          <Input
            autoFocus
            placeholder="폴더 이름"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") { e.preventDefault(); handleCreateFolder(); }
              if (e.key === "Escape") { setShowNewFolder(false); setNewFolderName(""); }
            }}
            className="h-7 flex-1"
          />
          <Button size="sm" onClick={handleCreateFolder}>만들기</Button>
          <Button variant="ghost" size="sm" onClick={() => { setShowNewFolder(false); setNewFolderName(""); }}>
            취소
          </Button>
        </div>
      )}

      {/* New File Input */}
      {isMyFilesSection && showNewFile && (
        <div className="flex items-center gap-2 border-b border-border bg-surface-accent/50 px-6 py-2">
          <FilePlus size={16} className="text-text-muted" />
          <Input
            autoFocus
            placeholder="파일 이름"
            value={newFileName}
            onChange={(e) => setNewFileName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleCreateFile();
              }
              if (e.key === "Escape") {
                setShowNewFile(false);
                setNewFileName("");
                setNewFileExtension("txt");
              }
            }}
            className="h-7 flex-1"
          />
          <div className="flex items-center gap-1">
            <Button
              variant={newFileExtension === "md" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setNewFileExtension("md")}
            >
              .md
            </Button>
            <Button
              variant={newFileExtension === "txt" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setNewFileExtension("txt")}
            >
              .txt
            </Button>
          </div>
          <Button size="sm" disabled={creatingFile} onClick={handleCreateFile}>
            만들기
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setShowNewFile(false);
              setNewFileName("");
              setNewFileExtension("txt");
            }}
          >
            취소
          </Button>
        </div>
      )}

      {/* Search indicator */}
      {showSearchResultBanner && (
        <div className="flex items-center gap-2 border-b border-border bg-surface-accent/50 px-6 py-2 text-sm">
          <span className="text-text-secondary">
            &quot;{searchQuery}&quot; 검색 결과: {searchResults.length}건
          </span>
          <button
            onClick={() => { setSearchQuery(""); setSearchResults(null); }}
            className="text-text-muted hover:text-text-primary"
          >
            ✕
          </button>
        </div>
      )}

      {isMyFilesSection && clipboard && (
        <div className="flex items-center gap-2 border-b border-border bg-surface-accent/40 px-6 py-2 text-xs text-text-secondary">
          <span>
            {clipboard.mode === "copy" ? "복사됨" : "잘라내기됨"}: {clipboard.summary}
          </span>
          <span className="text-text-muted">붙여넣기: mac ⌘V / windows Ctrl+V, 해제: Esc</span>
        </div>
      )}

      {/* Bulk action bar */}
      {showBulkBar && (
        <>
          {isMyFilesSection && (
            <BulkActionBar
              count={selectedIds.size}
              onDownload={handleBulkDownload}
              onDelete={handleBulkDelete}
              onClear={() => setSelectedIds(new Set())}
            />
          )}
          {isTrashSection && (
            <div className="flex items-center justify-between gap-2 border-b border-border bg-surface-accent/50 px-4 md:px-6 py-2">
              <span className="text-sm text-text-secondary">{selectedIds.size}개 선택됨</span>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" onClick={handleBulkRestore} className="gap-1.5">
                  <Undo2 size={14} />
                  복원
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleEmptyTrash}
                  className="gap-1.5 text-error hover:text-error"
                >
                  <Trash2 size={14} />
                  비우기
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setSelectedIds(new Set())}>
                  선택 해제
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Drop overlay */}
      {dragOver && isMyFilesSection && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/80">
          <div className="rounded-xl border-2 border-dashed border-primary/40 p-12 text-center">
            <Upload size={48} className="mx-auto text-primary" />
            <p className="mt-2 text-text-secondary">여기에 파일을 놓으세요</p>
          </div>
        </div>
      )}

      {/* File list */}
      <div className="relative flex-1 overflow-auto">
        {routeStateCopy ? (
          <div className="flex flex-col items-center justify-center py-20 text-center text-text-muted">
            <Cloud size={48} className="text-border" />
            <p className="mt-3 text-base font-medium text-text-strong">{routeStateCopy.title}</p>
            <p className="mt-1 text-sm">{routeStateCopy.description}</p>
            <div className="mt-4 flex items-center gap-2">
              <Button size="sm" onClick={navigateToSectionRoot}>
                {routeActionLabel}
              </Button>
              {folderRouteState === "error" && (
                <Button variant="secondary" size="sm" onClick={retryFolderRoute}>
                  다시 시도
                </Button>
              )}
            </div>
          </div>
        ) : loading ? (
          <div className="flex items-center justify-center py-20 text-text-muted">
            불러오는 중...
          </div>
        ) : displayItems.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-text-muted">
            <Cloud size={48} className="text-border" />
            <p className="mt-3">
              {searchResults !== null
                ? "검색 결과가 없습니다"
                : isTrashSection
                  ? "휴지통이 비어 있습니다"
                  : isRecentSection
                    ? "최근 파일이 없습니다"
                    : isSharedSection
                      ? "공유 받은 폴더가 없습니다"
                      : isStarredSection
                        ? "중요 표시한 항목이 없습니다"
                        : "폴더가 비어 있습니다"}
            </p>
            {searchResults === null && isMyFilesSection && (
              <p className="mt-1 text-sm">파일을 업로드하거나 폴더를 만들어 보세요</p>
            )}
          </div>
        ) : currentViewMode === "list" ? (
          <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
            <colgroup>
              <col className="w-10" />
              <col />
              <col className="hidden md:table-column" style={{ width: "112px" }} />
              <col className="hidden md:table-column" style={{ width: "144px" }} />
              <col className="w-10" />
            </colgroup>
            <thead>
              <tr className="border-b border-border text-left text-text-muted">
                <th className="w-10 px-4 md:px-6 py-2 font-normal">
                  {isSelectableSection ? (
                    <Checkbox
                      checked={selectedIds.size === displayItems.length && displayItems.length > 0}
                      onCheckedChange={toggleSelectAll}
                    />
                  ) : null}
                </th>
                <th className="px-2 py-2 font-normal border-r border-border/30">이름</th>
                <th className="hidden md:table-cell px-4 py-2 font-normal border-r border-border/30">크기</th>
                <th className="hidden md:table-cell px-4 py-2 font-normal border-r border-border/30">수정한 날짜</th>
                <th className="w-10 px-2 py-2 font-normal"></th>
              </tr>
            </thead>
            <tbody>
              {displayItems.map((item) => {
                const id = item.type === "folder" ? item.folder!.id : item.file!.id;
                const name = item.type === "folder" ? item.folder!.name : item.file!.name;
                const isRenaming = renaming?.id === id;
                const isSelected = selectedIds.has(id);
                const recentPath = isRecentSection && item.type === "file" ? formatRecentFilePath(item) : "";
                const isDropTarget = isMyFilesSection && item.type === "folder" && dropTargetFolderId === id;
                const isDraggingItem =
                  isMyFilesSection && draggingItem?.itemType === item.type && draggingItem.itemID === id;
                const isCutClipboardItem = isClipboardCutItem(item.type, id);
                const itemToken = item.type === "folder"
                  ? getCloudItemToken({ type: "folder" })
                  : getCloudItemToken({ type: "file", mimeType: item.file!.mime_type });

                return (
                  <tr
                    key={id}
                    className={`border-b border-border/50 hover:bg-surface-accent/50 cursor-default ${
                      isSelected ? "bg-primary/5" : ""
                    } ${isDropTarget ? "bg-surface-accent/80 ring-1 ring-inset ring-primary" : ""} ${
                      isDraggingItem ? "opacity-60" : ""
                    } ${isCutClipboardItem ? "opacity-60" : ""}
                    }`}
                    draggable={isMyFilesSection}
                    onDragStart={(e) => {
                      if (!isMyFilesSection || movingItemKey) return;
                      const dragItem = getDragItemFromRow(item);
                      setDraggingItem(dragItem);
                      e.dataTransfer.effectAllowed = "move";
                      e.dataTransfer.setData(INTERNAL_ITEM_DRAG_TYPE, JSON.stringify(dragItem));
                      e.dataTransfer.setData("text/plain", dragItem.itemID);
                    }}
                    onDragEnd={() => {
                      setDraggingItem(null);
                      setDropTargetFolderId(null);
                    }}
                    onDragOver={(e) => {
                      if (item.type !== "folder") return;
                      handleFolderDragOver(e, item.folder!.id);
                    }}
                    onDragLeave={() => {
                      if (item.type !== "folder") return;
                      if (dropTargetFolderId === item.folder!.id) {
                        setDropTargetFolderId(null);
                      }
                    }}
                    onDrop={(e) => {
                      if (item.type !== "folder") return;
                      void handleFolderDrop(e, item.folder!.id);
                    }}
                    onDoubleClick={() => {
                      if ((isMyFilesSection || isTrashSection) && item.type === "folder" && item.folder) {
                        navigateToFolder(item.folder);
                      }
                      else if (!isTrashSection && item.type === "file" && item.file) {
                        if (isMyFilesSection && isEditableTextFile(item.file)) {
                          openTextEditor(item.file);
                        } else {
                          handleDownload(item.file);
                        }
                      }
                    }}
                  >
                    <td className="px-4 md:px-6 py-2">
                      {isSelectableSection ? (
                        <Checkbox
                          checked={isSelected}
                          onCheckedChange={() => toggleSelect(id)}
                        />
                      ) : null}
                    </td>
                    <td className="px-2 py-2">
                      <div className="flex items-center gap-2">
                        {item.type === "folder" ? (
                          <Folder size={16} className="shrink-0" style={{ color: itemToken.foreground }} />
                        ) : (
                          <FileIcon mimeType={item.file!.mime_type} size={16} className="shrink-0" />
                        )}
                        {isRenaming ? (
                          <Input
                            autoFocus
                            value={renaming.name}
                            onChange={(e) => setRenaming({ ...renaming, name: e.target.value })}
                            onKeyDown={(e) => {
                              if (e.key === "Enter") handleRename();
                              if (e.key === "Escape") setRenaming(null);
                            }}
                            onBlur={handleRename}
                            className="h-6 text-sm"
                          />
                        ) : (
                          <>
                            <div className="min-w-0">
                              <span
                                className={`${item.type === "folder" && (isMyFilesSection || isTrashSection) ? "cursor-pointer hover:underline " : ""}text-text-primary`}
                                onClick={() => {
                                  if ((isMyFilesSection || isTrashSection) && item.type === "folder" && item.folder) {
                                    navigateToFolder(item.folder);
                                  }
                                }}
                              >
                                {name}
                              </span>
                              {isRecentSection && item.type === "file" && (
                                <p className="truncate text-[11px] text-text-muted">{recentPath}</p>
                              )}
                            </div>
                            {isItemStarred(item) && (
                              <StarIcon size={12} className="shrink-0 text-amber-500 fill-amber-500" />
                            )}
                          </>
                        )}
                      </div>
                    </td>
                    <td className="hidden md:table-cell px-4 py-2 text-text-muted">
                      {item.type === "file" ? formatSize(item.file!.size_bytes) : "—"}
                    </td>
                    <td className="hidden md:table-cell px-4 py-2 text-text-muted">
                      {formatDate(item.type === "folder" ? item.folder!.updated_at : item.file!.updated_at)}
                    </td>
                    <td className="px-2 py-2">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button className="flex h-7 w-7 items-center justify-center rounded-lg text-text-muted hover:bg-surface-accent">
                            <MoreVertical size={14} />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {isTrashSection ? (
                            <>
                              {item.type === "folder" && item.folder ? (
                                <DropdownMenuItem onClick={() => navigateToFolder(item.folder!)}>
                                  <FolderOpen size={14} /> 열기
                                </DropdownMenuItem>
                              ) : null}
                              <DropdownMenuItem onClick={() => handleRestore(item)}>
                                <Undo2 size={14} /> 복원
                              </DropdownMenuItem>
                            </>
                          ) : isMyFilesSection ? (
                            <>
                              {item.type === "file" ? (
                                <DropdownMenuItem onClick={() => setClipboardFromItem("copy", item)}>
                                  <Copy size={14} /> 복사
                                </DropdownMenuItem>
                              ) : (
                                <DropdownMenuItem disabled>
                                  <Copy size={14} /> 복사 (폴더 미지원)
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem onClick={() => setClipboardFromItem("cut", item)}>
                                <Scissors size={14} /> 잘라내기
                              </DropdownMenuItem>
                              {item.type === "folder" && item.folder && (
                                <DropdownMenuItem
                                  onClick={() => void applyClipboardToFolder(item.folder!.id)}
                                  disabled={!clipboard || clipboardBusy}
                                >
                                  <ClipboardPaste size={14} /> 여기에 붙여넣기
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuSeparator />
                              {item.type === "folder" && item.folder && (
                                <DropdownMenuItem onClick={() => navigateToFolder(item.folder!)}>
                                  <FolderOpen size={14} /> 열기
                                </DropdownMenuItem>
                              )}
                              {item.type === "file" && item.file && (
                                isEditableTextFile(item.file) ? (
                                  <DropdownMenuItem onClick={() => openTextEditor(item.file!)}>
                                    <Pencil size={14} /> 편집
                                  </DropdownMenuItem>
                                ) : null
                              )}
                              {item.type === "file" && item.file && (
                                <DropdownMenuItem onClick={() => handleDownload(item.file!)}>
                                  <Download size={14} /> 다운로드
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem onClick={() => handleToggleStar(item)}>
                                <StarIcon size={14} />
                                {isItemStarred(item) ? "중요 해제" : "중요 표시"}
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => {
                                  const itemId = item.type === "folder" ? item.folder!.id : item.file!.id;
                                  const itemName = item.type === "folder" ? item.folder!.name : item.file!.name;
                                  setRenaming({ id: itemId, type: item.type, name: itemName });
                                }}
                              >
                                <Pencil size={14} /> 이름 변경
                              </DropdownMenuItem>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onClick={() => handleDelete(item)}
                                className="text-error focus:text-error"
                              >
                                <Trash2 size={14} /> 삭제
                              </DropdownMenuItem>
                            </>
                          ) : (
                            <>
                              {isSharedSection && item.type === "folder" && (
                                <DropdownMenuItem disabled>읽기 전용</DropdownMenuItem>
                              )}
                              {item.type === "file" && item.file && (
                                <DropdownMenuItem onClick={() => handleDownload(item.file!)}>
                                  <Download size={14} /> 다운로드
                                </DropdownMenuItem>
                              )}
                              {!isSharedSection && (
                                <DropdownMenuItem onClick={() => handleToggleStar(item)}>
                                  <StarIcon size={14} />
                                  {isItemStarred(item) ? "중요 해제" : "중요 표시"}
                                </DropdownMenuItem>
                              )}
                            </>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          /* Grid view */
          <div className="grid grid-cols-[repeat(auto-fill,minmax(120px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(140px,1fr))] gap-3 p-4">
            {displayItems.map((item) => {
              const id = item.type === "folder" ? item.folder!.id : item.file!.id;
              const name = item.type === "folder" ? item.folder!.name : item.file!.name;
              const isSelected = selectedIds.has(id);
              const recentPath = isRecentSection && item.type === "file" ? formatRecentFilePath(item) : "";
              const isDropTarget = isMyFilesSection && item.type === "folder" && dropTargetFolderId === id;
              const isDraggingItem =
                isMyFilesSection && draggingItem?.itemType === item.type && draggingItem.itemID === id;
              const isCutClipboardItem = isClipboardCutItem(item.type, id);
              const itemToken = item.type === "folder"
                ? getCloudItemToken({ type: "folder" })
                : getCloudItemToken({ type: "file", mimeType: item.file!.mime_type });

              return (
                <div
                  key={id}
                  className={`group relative flex flex-col rounded-lg border overflow-hidden cursor-default transition-colors ${
                    isSelected ? "border-primary bg-primary/5" : "border-border hover:bg-surface-accent/50"
                  } ${isDropTarget ? "bg-surface-accent/80 ring-2 ring-primary border-primary" : ""} ${
                    isDraggingItem ? "opacity-60" : ""
                  } ${isCutClipboardItem ? "opacity-60" : ""}
                  }`}
                  draggable={isMyFilesSection}
                  onDragStart={(e) => {
                    if (!isMyFilesSection || movingItemKey) return;
                    const dragItem = getDragItemFromRow(item);
                    setDraggingItem(dragItem);
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData(INTERNAL_ITEM_DRAG_TYPE, JSON.stringify(dragItem));
                    e.dataTransfer.setData("text/plain", dragItem.itemID);
                  }}
                  onDragEnd={() => {
                    setDraggingItem(null);
                    setDropTargetFolderId(null);
                  }}
                  onDragOver={(e) => {
                    if (item.type !== "folder") return;
                    handleFolderDragOver(e, item.folder!.id);
                  }}
                  onDragLeave={() => {
                    if (item.type !== "folder") return;
                    if (dropTargetFolderId === item.folder!.id) {
                      setDropTargetFolderId(null);
                    }
                  }}
                  onDrop={(e) => {
                    if (item.type !== "folder") return;
                    void handleFolderDrop(e, item.folder!.id);
                  }}
                  onClick={() => toggleSelect(id)}
                  onDoubleClick={() => {
                    if (isMyFilesSection && item.type === "folder" && item.folder) navigateToFolder(item.folder);
                    else if (isMyFilesSection && item.type === "file" && item.file) handleDownload(item.file);
                  }}
                >
                  {/* 4:3 미리보기 영역 */}
                  <div
                    className="relative flex aspect-[4/3] w-full items-center justify-center"
                    style={{
                      backgroundColor:
                        item.type === "folder" || !(item.file!.thumb_status === "done" && isMediaFile(item.file!.mime_type))
                          ? itemToken.background
                          : undefined,
                    }}
                  >
                    {item.type === "folder" ? (
                      <Folder size={36} style={{ color: itemToken.foreground }} />
                    ) : item.file!.thumb_status === "done" && isMediaFile(item.file!.mime_type) ? (
                      <ThumbnailImage
                        fileId={item.file!.id}
                        size="medium"
                        alt={item.file!.name}
                        fill
                        className="object-cover"
                      />
                    ) : (
                      <FileIcon mimeType={item.file!.mime_type} size={36} />
                    )}
                  </div>
                  {/* 파일명 */}
                  <div className="px-2 py-1.5">
                    <span className="block w-full truncate text-center text-xs text-text-primary">{name}</span>
                    {isRecentSection && item.type === "file" && (
                      <p className="mt-0.5 truncate text-center text-[10px] text-text-muted">
                        {recentPath}
                      </p>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <Dialog open={editorOpen} onOpenChange={(open) => {
        setEditorOpen(open);
        if (!open) {
          setEditingFile(null);
          setEditorContent("");
        }
      }}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{editingFile ? `${editingFile.name} 편집` : "파일 편집"}</DialogTitle>
          </DialogHeader>
          <Textarea
            value={editorContent}
            onChange={(e) => setEditorContent(e.target.value)}
            rows={18}
            className="font-mono text-sm"
          />
          <DialogFooter>
            <Button variant="ghost" onClick={() => setEditorOpen(false)}>
              취소
            </Button>
            <Button onClick={saveTextEditor} disabled={savingContent}>
              저장
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default function CloudPage() {
  return (
    <Suspense fallback={<div className="flex h-full items-center justify-center text-text-muted">불러오는 중...</div>}>
      <CloudPageInner />
    </Suspense>
  );
}
