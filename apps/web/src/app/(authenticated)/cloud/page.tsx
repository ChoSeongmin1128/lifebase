"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState, useEffect, useCallback, useMemo, useRef, Suspense } from "react";
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
import { FileIcon } from "@/components/cloud/FileIcon";
import { ThumbnailImage } from "@/components/cloud/ThumbnailImage";
import { BulkActionBar } from "@/components/cloud/BulkActionBar";
import { PageToolbar, PageToolbarGroup } from "@/components/layout/PageToolbar";
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
interface CloudClipboard {
  mode: ClipboardMode;
  itemType: ClipboardItemType;
  itemID: string;
  itemName: string;
}
const INTERNAL_FILE_DRAG_TYPE = "application/x-lifebase-cloud-file-id";

const isTextInputTarget = (target: EventTarget | null) => {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
};

function CloudPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const section = parseCloudSection(searchParams.get("section")) as CloudSection;
  const folderFromUrl = searchParams.get("folder");
  const isMyFilesSection = section === "";
  const isTrashSection = section === "trash";
  const isRecentSection = section === "recent";
  const isSharedSection = section === "shared";
  const isStarredSection = section === "starred";
  const isSelectableSection = isMyFilesSection || isTrashSection;

  const [items, setItems] = useState<FolderItem[]>([]);
  const [path, setPath] = useState<{ id: string | null; name: string }[]>([
    { id: null, name: "내 클라우드" },
  ]);
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<SortBy>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [renaming, setRenaming] = useState<{ id: string; type: "folder" | "file"; name: string } | null>(null);
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [creatingFolder, setCreatingFolder] = useState(false);
  const [showNewFile, setShowNewFile] = useState(false);
  const [newFileName, setNewFileName] = useState("");
  const [newFileExtension, setNewFileExtension] = useState<"md" | "txt">("md");
  const [creatingFile, setCreatingFile] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingFile, setEditingFile] = useState<CloudFile | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [savingContent, setSavingContent] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<CloudFile[] | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [draggingFileId, setDraggingFileId] = useState<string | null>(null);
  const [dropTargetFolderId, setDropTargetFolderId] = useState<string | null>(null);
  const [movingFileId, setMovingFileId] = useState<string | null>(null);
  const [clipboard, setClipboard] = useState<CloudClipboard | null>(null);
  const [clipboardBusy, setClipboardBusy] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [starredKeys, setStarredKeys] = useState<Set<string>>(new Set());

  const fileInputRef = useRef<HTMLInputElement>(null);

  const currentFolderID = useMemo(() => {
    const fromPath = path[path.length - 1].id;
    if (fromPath) return fromPath;
    if (isMyFilesSection && folderFromUrl) return folderFromUrl;
    return null;
  }, [folderFromUrl, isMyFilesSection, path]);
  const authed = isAuthenticated();
  const cloud = useCloudActions();

  const buildCloudHref = useCallback((targetSection: CloudSection, folderId?: string | null) => {
    const params = new URLSearchParams();
    if (targetSection) {
      params.set("section", targetSection);
    }
    if (!targetSection && folderId) {
      params.set("folder", folderId);
    }
    const query = params.toString();
    return query ? `/cloud?${query}` : "/cloud";
  }, []);

  const syncFolderUrl = useCallback(
    (folderId: string | null, mode: "push" | "replace" = "replace") => {
      const href = buildCloudHref(section, folderId);
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
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const nextItems = await cloud.listItems({
        section,
        folderId: currentFolderID,
        sortBy,
        sortDir,
      });
      setItems(nextItems);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [
    authed,
    currentFolderID,
    cloud,
    section,
    sortBy,
    sortDir,
  ]);

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
    if (!isMyFilesSection) {
      setPath([{ id: null, name: "내 클라우드" }]);
    }
    setRenaming(null);
    setShowNewFolder(false);
    setNewFolderName("");
    setShowNewFile(false);
    setNewFileName("");
    setNewFileExtension("md");
    setSearchResults(null);
    setSearchQuery("");
    setSelectedIds(new Set());
    setDragOver(false);
    setDraggingFileId(null);
    setDropTargetFolderId(null);
    setMovingFileId(null);
    loadStars();
    loadItems();
  }, [isMyFilesSection, loadItems, loadStars]);

  const navigateToFolder = (folder: FolderData) => {
    if (!isMyFilesSection) return;
    setPath((prev) => [...prev, { id: folder.id, name: folder.name }]);
    syncFolderUrl(folder.id, "push");
  };

  const navigateToBreadcrumb = (index: number) => {
    if (!isMyFilesSection) return;
    setPath((prev) => {
      const next = prev.slice(0, index + 1);
      const nextFolderId = next[next.length - 1]?.id ?? null;
      syncFolderUrl(nextFolderId, "push");
      return next;
    });
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
    try {
      if (item.type === "folder" && item.folder) {
        await cloud.deleteFolder(item.folder.id);
      } else if (item.type === "file" && item.file) {
        await cloud.deleteFile(item.file.id);
      }
      loadItems();
    } catch (err) {
      console.error("Delete failed:", err);
    }
  };

  const handleBulkDelete = async () => {
    if (!authed || !isMyFilesSection) return;
    for (const item of displayItems) {
      const id = item.type === "folder" ? item.folder!.id : item.file!.id;
      if (!selectedIds.has(id)) continue;
      try {
        if (item.type === "folder") {
          await cloud.deleteFolder(id);
        } else {
          await cloud.deleteFile(id);
        }
      } catch (err) {
        console.error("Delete failed:", err);
      }
    }
    setSelectedIds(new Set());
    loadItems();
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
    try {
      await cloud.emptyTrash();
      setSelectedIds(new Set());
      await loadItems();
    } catch (err) {
      console.error("Empty trash failed:", err);
    }
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
      setNewFileExtension("md");
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

  const getDraggedFileId = (e: React.DragEvent): string | null => {
    const fromData = e.dataTransfer.getData(INTERNAL_FILE_DRAG_TYPE);
    if (fromData) return fromData;
    return draggingFileId;
  };

  const isUploadDragEvent = (e: React.DragEvent) => Array.from(e.dataTransfer.types).includes("Files");

  const handleMoveFileToFolder = async (fileId: string, folderId: string) => {
    if (!authed || !isMyFilesSection || movingFileId) return;
    const source = displayItems.find((item) => item.type === "file" && item.file?.id === fileId);
    if (!source || source.type !== "file") return;
    if (source.file!.folder_id === folderId) return;

    setMovingFileId(fileId);
    try {
      await cloud.moveFile(fileId, folderId);
      setSelectedIds((prev) => {
        if (!prev.has(fileId)) return prev;
        const next = new Set(prev);
        next.delete(fileId);
        return next;
      });
      await loadItems();
    } catch (err) {
      console.error("Move file failed:", err);
      const msg = err instanceof Error ? err.message : "알 수 없는 오류";
      if (typeof window !== "undefined") {
        window.alert(`파일 이동에 실패했습니다: ${msg}`);
      }
    } finally {
      setMovingFileId(null);
    }
  };

  const handleFolderDragOver = (e: React.DragEvent, folderId: string) => {
    if (!isMyFilesSection || movingFileId) return;
    const fileId = getDraggedFileId(e);
    if (!fileId) return;
    const source = displayItems.find((item) => item.type === "file" && item.file?.id === fileId);
    if (!source || source.type !== "file") return;
    if (source.file!.folder_id === folderId) return;

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
    const fileId = getDraggedFileId(e);
    setDropTargetFolderId(null);
    if (!fileId) return;
    await handleMoveFileToFolder(fileId, folderId);
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

  const showCloudActionError = useCallback((prefix: string, err: unknown) => {
    console.error(prefix, err);
    const msg = err instanceof Error ? err.message : "알 수 없는 오류";
    if (typeof window !== "undefined") {
      window.alert(`${prefix}: ${msg}`);
    }
  }, []);

  const setClipboardFromItem = useCallback((mode: ClipboardMode, item: FolderItem) => {
    if (!isMyFilesSection) return;
    const itemID = item.type === "folder" ? item.folder!.id : item.file!.id;
    const itemName = item.type === "folder" ? item.folder!.name : item.file!.name;
    setClipboard({ mode, itemType: item.type, itemID, itemName });
  }, [isMyFilesSection]);

  const applyClipboardToFolder = useCallback(async (targetFolderID: string | null) => {
    if (!authed || !isMyFilesSection || !clipboard || clipboardBusy) return;

    setClipboardBusy(true);
    try {
      if (clipboard.itemType === "file") {
        if (clipboard.mode === "copy") {
          await cloud.copyFile(clipboard.itemID, targetFolderID);
        } else {
          await cloud.moveFile(clipboard.itemID, targetFolderID);
          setClipboard(null);
        }
      } else {
        if (clipboard.mode === "copy") {
          await cloud.copyFolder(clipboard.itemID, targetFolderID);
        } else {
          await cloud.moveFolder(clipboard.itemID, targetFolderID);
          setClipboard(null);
        }
      }

      await loadItems();
    } catch (err) {
      showCloudActionError("붙여넣기에 실패했습니다", err);
    } finally {
      setClipboardBusy(false);
    }
  }, [authed, clipboard, clipboardBusy, cloud, isMyFilesSection, loadItems, showCloudActionError]);

  const getSingleSelectedItem = useCallback(() => {
    if (!isMyFilesSection || selectedIds.size !== 1) return null;
    const currentItems: FolderItem[] = isMyFilesSection && searchResults
      ? searchResults.map((f) => ({ type: "file" as const, file: f, path: undefined }))
      : items;
    const selectedID = Array.from(selectedIds)[0];
    return currentItems.find((item) =>
      item.type === "folder" ? item.folder?.id === selectedID : item.file?.id === selectedID,
    ) ?? null;
  }, [isMyFilesSection, items, searchResults, selectedIds]);

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
    if (!isMyFilesSection) return;

    const onKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.altKey) return;
      if (isTextInputTarget(e.target)) return;

      const key = e.key.toLowerCase();
      if (key !== "c" && key !== "x" && key !== "v") return;

      const selectedText = window.getSelection()?.toString() || "";
      if (selectedText && key !== "v") return;

      if (key === "v") {
        if (!clipboard || clipboardBusy) return;
        e.preventDefault();
        void applyClipboardToFolder(currentFolderID ?? null);
        return;
      }

      const selectedItem = getSingleSelectedItem();
      if (!selectedItem) return;

      e.preventDefault();
      setClipboardFromItem(key === "x" ? "cut" : "copy", selectedItem);
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [
    applyClipboardToFolder,
    clipboard,
    clipboardBusy,
    currentFolderID,
    getSingleSelectedItem,
    isMyFilesSection,
    setClipboardFromItem,
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
  const showBreadcrumb = isMyFilesSection && path.length > 1;
  const showSearchResultBanner = isMyFilesSection && searchResults !== null;
  const showBulkBar = selectedIds.size > 0 && (isMyFilesSection || isTrashSection);
  const currentViewMode: ViewMode = isMyFilesSection ? viewMode : "list";

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
      {showBreadcrumb && (
        <PageToolbar className="justify-start py-3">
          <nav className="flex items-center gap-1 text-sm">
            {path.map((p, i) => (
              <span key={i} className="flex items-center gap-1">
                {i > 0 && <span className="text-text-muted">/</span>}
                <button
                  onClick={() => navigateToBreadcrumb(i)}
                  className={`hover:underline ${
                    i === path.length - 1
                      ? "font-medium text-text-strong"
                      : "text-text-secondary"
                  }`}
                >
                  {p.name}
                </button>
              </span>
            ))}
          </nav>
        </PageToolbar>
      )}

      <div className="flex gap-1.5 overflow-x-auto border-b border-border px-4 py-2 md:hidden">
        {CLOUD_SECTION_ITEMS.map((item) => {
          const isActive = item.section === section;
          const href = item.section ? `/cloud?section=${item.section}` : "/cloud";
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
              disabled={!isMyFilesSection}
              className="h-8 w-full md:w-48 pl-8"
            />
          </div>

          {/* View toggle */}
          {isMyFilesSection && (
            <div className="flex rounded-lg border border-border">
              <button
                onClick={() => setViewMode("list")}
                className={`flex h-8 w-8 items-center justify-center rounded-l-lg ${
                  currentViewMode === "list" ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
                }`}
              >
                <List size={14} />
              </button>
              <button
                onClick={() => setViewMode("grid")}
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
                onClick={() => {
                  setShowNewFolder(false);
                  setNewFolderName("");
                  setShowNewFile(true);
                }}
                className="gap-1.5"
              >
                <FilePlus size={14} />
                <span className="hidden md:inline">새 파일</span>
              </Button>

              <Button variant="primary" size="sm" onClick={() => fileInputRef.current?.click()} className="gap-1.5">
                <Upload size={14} />
                <span className="hidden md:inline">업로드</span>
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={!clipboard || clipboardBusy}
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
                setNewFileExtension("md");
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
              setNewFileExtension("md");
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
            {clipboard.mode === "copy" ? "복사됨" : "잘라내기됨"}: {clipboard.itemName}
          </span>
          <span className="text-text-muted">붙여넣기: mac ⌘V / windows Ctrl+V</span>
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
      <div className="flex-1 overflow-auto">
        {loading ? (
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
                const isDraggingItem = isMyFilesSection && item.type === "file" && draggingFileId === id;
                const isCutClipboardItem =
                  clipboard?.mode === "cut" && clipboard.itemType === item.type && clipboard.itemID === id;

                return (
                  <tr
                    key={id}
                    className={`border-b border-border/50 hover:bg-surface-accent/50 cursor-default ${
                      isSelected ? "bg-primary/5" : ""
                    } ${isDropTarget ? "bg-surface-accent/80 ring-1 ring-inset ring-primary" : ""} ${
                      isDraggingItem ? "opacity-60" : ""
                    } ${isCutClipboardItem ? "opacity-60" : ""}
                    }`}
                    draggable={isMyFilesSection && item.type === "file"}
                    onDragStart={(e) => {
                      if (!isMyFilesSection || item.type !== "file" || movingFileId) return;
                      setDraggingFileId(item.file!.id);
                      e.dataTransfer.effectAllowed = "move";
                      e.dataTransfer.setData(INTERNAL_FILE_DRAG_TYPE, item.file!.id);
                      e.dataTransfer.setData("text/plain", item.file!.id);
                    }}
                    onDragEnd={() => {
                      setDraggingFileId(null);
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
                      if (isMyFilesSection && item.type === "folder" && item.folder) navigateToFolder(item.folder);
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
                          <Folder size={16} className="text-text-muted shrink-0" />
                        ) : (
                          <FileIcon mimeType={item.file!.mime_type} size={16} className="text-text-muted shrink-0" />
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
                                className={`${item.type === "folder" && isMyFilesSection ? "cursor-pointer hover:underline " : ""}text-text-primary`}
                                onClick={() => {
                                  if (isMyFilesSection && item.type === "folder" && item.folder) navigateToFolder(item.folder);
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
                            <DropdownMenuItem onClick={() => handleRestore(item)}>
                              <Undo2 size={14} /> 복원
                            </DropdownMenuItem>
                          ) : isMyFilesSection ? (
                            <>
                              <DropdownMenuItem onClick={() => setClipboardFromItem("copy", item)}>
                                <Copy size={14} /> 복사
                              </DropdownMenuItem>
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
              const isDraggingItem = isMyFilesSection && item.type === "file" && draggingFileId === id;
              const isCutClipboardItem =
                clipboard?.mode === "cut" && clipboard.itemType === item.type && clipboard.itemID === id;

              return (
                <div
                  key={id}
                  className={`group relative flex flex-col rounded-lg border overflow-hidden cursor-default transition-colors ${
                    isSelected ? "border-primary bg-primary/5" : "border-border hover:bg-surface-accent/50"
                  } ${isDropTarget ? "bg-surface-accent/80 ring-2 ring-primary border-primary" : ""} ${
                    isDraggingItem ? "opacity-60" : ""
                  } ${isCutClipboardItem ? "opacity-60" : ""}
                  }`}
                  draggable={isMyFilesSection && item.type === "file"}
                  onDragStart={(e) => {
                    if (!isMyFilesSection || item.type !== "file" || movingFileId) return;
                    setDraggingFileId(item.file!.id);
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData(INTERNAL_FILE_DRAG_TYPE, item.file!.id);
                    e.dataTransfer.setData("text/plain", item.file!.id);
                  }}
                  onDragEnd={() => {
                    setDraggingFileId(null);
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
                  <div className="relative w-full aspect-[4/3] bg-surface-accent flex items-center justify-center">
                    {item.type === "folder" ? (
                      <Folder size={36} className="text-text-muted" />
                    ) : item.file!.thumb_status === "done" && isMediaFile(item.file!.mime_type) ? (
                      <ThumbnailImage
                        fileId={item.file!.id}
                        size="medium"
                        alt={item.file!.name}
                        fill
                        className="object-cover"
                      />
                    ) : (
                      <FileIcon mimeType={item.file!.mime_type} size={36} className="text-text-muted" />
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
