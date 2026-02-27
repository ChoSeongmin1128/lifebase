"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState, useEffect, useCallback, useRef } from "react";
import { api, apiUpload, apiDownload } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
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
import {
  CLOUD_SECTION_ITEMS,
  CLOUD_SECTION_LABELS,
  parseCloudSection,
} from "@/lib/cloud-sections";
import {
  ArrowUpDown,
  FolderPlus,
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
  Star as StarIcon,
  Undo2,
} from "lucide-react";

interface FolderData {
  id: string;
  user_id: string;
  parent_id: string | null;
  name: string;
  created_at: string;
  updated_at: string;
}

interface CloudFile {
  id: string;
  user_id: string;
  folder_id: string | null;
  name: string;
  mime_type: string;
  size_bytes: number;
  thumb_status: string;
  taken_at: string | null;
  created_at: string;
  updated_at: string;
}

interface FolderItem {
  type: "folder" | "file";
  folder?: FolderData;
  file?: CloudFile;
}

interface StarItem {
  id: string;
  type: "folder" | "file";
}

type SortBy = "name" | "size" | "updated_at" | "created_at";
type SortDir = "asc" | "desc";
type ViewMode = "list" | "grid";

export default function CloudPage() {
  const searchParams = useSearchParams();
  const section = parseCloudSection(searchParams.get("section"));
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
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<CloudFile[] | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [starredKeys, setStarredKeys] = useState<Set<string>>(new Set());

  const fileInputRef = useRef<HTMLInputElement>(null);

  const currentFolderID = path[path.length - 1].id;
  const token = getAccessToken();

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
    if (!token) {
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      if (isTrashSection) {
        const data = await api<{ items: FolderItem[] }>("/cloud/trash", { token });
        setItems(data.items || []);
        return;
      }
      if (isRecentSection) {
        const data = await api<{ items: FolderItem[] }>("/cloud/recent", { token });
        setItems(data.items || []);
        return;
      }
      if (isSharedSection) {
        const data = await api<{ items: FolderItem[] }>("/cloud/shared", { token });
        setItems(data.items || []);
        return;
      }
      if (isStarredSection) {
        const data = await api<{ items: FolderItem[] }>("/cloud/starred", { token });
        setItems(data.items || []);
        return;
      }

      const params = new URLSearchParams({ sort_by: sortBy, sort_dir: sortDir });
      if (currentFolderID) {
        params.set("folder_id", currentFolderID);
      }
      const data = await api<{ items: FolderItem[] }>(`/cloud/folders?${params.toString()}`, { token });
      setItems(data.items || []);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [
    currentFolderID,
    isRecentSection,
    isSharedSection,
    isStarredSection,
    isTrashSection,
    sortBy,
    sortDir,
    token,
  ]);

  const loadStars = useCallback(async () => {
    if (!token) {
      setStarredKeys(new Set());
      return;
    }
    try {
      const data = await api<{ stars: StarItem[] }>("/cloud/stars", { token });
      const next = new Set((data.stars || []).map((star) => toStarKey(star.id, star.type)));
      setStarredKeys(next);
    } catch {
      setStarredKeys(new Set());
    }
  }, [token]);

  useEffect(() => {
    if (!isMyFilesSection) {
      setPath([{ id: null, name: "내 클라우드" }]);
    }
    setRenaming(null);
    setShowNewFolder(false);
    setNewFolderName("");
    setSearchResults(null);
    setSearchQuery("");
    setSelectedIds(new Set());
    loadStars();
    loadItems();
  }, [isMyFilesSection, loadItems, loadStars]);

  const navigateToFolder = (folder: FolderData) => {
    if (!isMyFilesSection) return;
    setPath((prev) => [...prev, { id: folder.id, name: folder.name }]);
  };

  const navigateToBreadcrumb = (index: number) => {
    if (!isMyFilesSection) return;
    setPath((prev) => prev.slice(0, index + 1));
  };

  const handleUpload = async (files: FileList | null) => {
    if (!files || !token || !isMyFilesSection) return;
    for (const file of Array.from(files)) {
      const formData = new FormData();
      formData.append("file", file);
      if (currentFolderID) formData.append("folder_id", currentFolderID);
      try {
        await apiUpload("/cloud/files/upload", formData, token);
      } catch (err) {
        console.error("Upload failed:", err);
      }
    }
    loadItems();
  };

  const handleDownload = async (file: CloudFile) => {
    if (!token) return;
    try {
      const { blob, filename } = await apiDownload(
        `/cloud/files/${file.id}/download`,
        token
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download failed:", err);
    }
  };

  const handleDelete = async (item: FolderItem) => {
    if (!token || !isMyFilesSection) return;
    try {
      if (item.type === "folder" && item.folder) {
        await api(`/cloud/folders/${item.folder.id}`, { method: "DELETE", token });
      } else if (item.type === "file" && item.file) {
        await api(`/cloud/files/${item.file.id}`, { method: "DELETE", token });
      }
      loadItems();
    } catch (err) {
      console.error("Delete failed:", err);
    }
  };

  const handleBulkDelete = async () => {
    if (!token || !isMyFilesSection) return;
    for (const item of displayItems) {
      const id = item.type === "folder" ? item.folder!.id : item.file!.id;
      if (!selectedIds.has(id)) continue;
      try {
        if (item.type === "folder") {
          await api(`/cloud/folders/${id}`, { method: "DELETE", token });
        } else {
          await api(`/cloud/files/${id}`, { method: "DELETE", token });
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
    if (!token) return;
    const { id, type } = toItemMeta(item);
    const isStarred = starredKeys.has(toStarKey(id, type));
    try {
      if (isStarred) {
        await api("/cloud/stars", {
          method: "DELETE",
          body: { id, type },
          token,
        });
      } else {
        await api("/cloud/stars", {
          method: "POST",
          body: { id, type },
          token,
        });
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
    if (!token || !isTrashSection) return;
    try {
      const id = item.type === "folder" ? item.folder?.id : item.file?.id;
      if (!id) return;
      await api("/cloud/trash/restore", {
        method: "POST",
        body: { id, type: item.type },
        token,
      });
      await loadItems();
    } catch (err) {
      console.error("Restore failed:", err);
    }
  };

  const handleBulkRestore = async () => {
    if (!token || !isTrashSection) return;
    for (const item of displayItems) {
      const id = item.type === "folder" ? item.folder?.id : item.file?.id;
      if (!id || !selectedIds.has(id)) continue;
      try {
        await api("/cloud/trash/restore", {
          method: "POST",
          body: { id, type: item.type },
          token,
        });
      } catch (err) {
        console.error("Restore failed:", err);
      }
    }
    setSelectedIds(new Set());
    await loadItems();
  };

  const handleEmptyTrash = async () => {
    if (!token || !isTrashSection) return;
    try {
      await api("/cloud/trash", { method: "DELETE", token });
      setSelectedIds(new Set());
      await loadItems();
    } catch (err) {
      console.error("Empty trash failed:", err);
    }
  };

  const handleRename = async () => {
    if (!renaming || !token || !renaming.name.trim() || !isMyFilesSection) return;
    try {
      if (renaming.type === "folder") {
        await api(`/cloud/folders/${renaming.id}/rename`, {
          method: "PATCH",
          body: { name: renaming.name },
          token,
        });
      } else {
        await api(`/cloud/files/${renaming.id}/rename`, {
          method: "PATCH",
          body: { name: renaming.name },
          token,
        });
      }
      setRenaming(null);
      loadItems();
    } catch (err) {
      console.error("Rename failed:", err);
    }
  };

  const handleCreateFolder = async () => {
    if (!token || !newFolderName.trim() || !isMyFilesSection || creatingFolder) return;
    setCreatingFolder(true);
    try {
      await api("/cloud/folders", {
        method: "POST",
        body: { name: newFolderName, parent_id: currentFolderID },
        token,
      });
      setShowNewFolder(false);
      setNewFolderName("");
      loadItems();
    } catch (err) {
      console.error("Create folder failed:", err);
    } finally {
      setCreatingFolder(false);
    }
  };

  const handleSearch = async () => {
    if (!token || !searchQuery.trim() || !isMyFilesSection) {
      setSearchResults(null);
      return;
    }
    try {
      const data = await api<{ files: CloudFile[] }>(
        `/cloud/search?q=${encodeURIComponent(searchQuery)}`,
        { token }
      );
      setSearchResults(data.files || []);
    } catch {
      setSearchResults([]);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    if (!isMyFilesSection) {
      setDragOver(false);
      return;
    }
    setDragOver(false);
    handleUpload(e.dataTransfer.files);
  };

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

  const sortOptions: { value: SortBy; label: string }[] = [
    { value: "name", label: "이름" },
    { value: "updated_at", label: "수정한 날짜" },
    { value: "created_at", label: "생성한 날짜" },
    { value: "size", label: "크기" },
  ];

  const displayItems = isMyFilesSection && searchResults
    ? searchResults.map((f) => ({ type: "file" as const, file: f }))
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
        e.preventDefault();
        if (!isMyFilesSection) return;
        setDragOver(true);
      }}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
    >
      {showBreadcrumb && (
        <div className="flex flex-wrap items-center gap-2 border-b border-border px-4 md:px-6 py-2">
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
        </div>
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
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 md:px-6 py-2">
        <div className="flex items-center gap-2">
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
        </div>

        <div className="flex items-center gap-2">
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

              <Button variant="ghost" size="sm" onClick={() => setShowNewFolder(true)} className="gap-1.5">
                <FolderPlus size={14} />
                <span className="hidden md:inline">새 폴더</span>
              </Button>

              <Button variant="primary" size="sm" onClick={() => fileInputRef.current?.click()} className="gap-1.5">
                <Upload size={14} />
                <span className="hidden md:inline">업로드</span>
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
        </div>
      </div>

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

                return (
                  <tr
                    key={id}
                    className={`border-b border-border/50 hover:bg-surface-accent/50 cursor-default ${
                      isSelected ? "bg-primary/5" : ""
                    }`}
                    onDoubleClick={() => {
                      if (isMyFilesSection && item.type === "folder" && item.folder) navigateToFolder(item.folder);
                      else if (!isTrashSection && item.type === "file" && item.file) handleDownload(item.file);
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
                            <span
                              className={`${item.type === "folder" && isMyFilesSection ? "cursor-pointer hover:underline " : ""}text-text-primary`}
                              onClick={() => {
                                if (isMyFilesSection && item.type === "folder" && item.folder) navigateToFolder(item.folder);
                              }}
                            >
                              {name}
                            </span>
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
                              {item.type === "folder" && item.folder && (
                                <DropdownMenuItem onClick={() => navigateToFolder(item.folder!)}>
                                  <FolderOpen size={14} /> 열기
                                </DropdownMenuItem>
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

              return (
                <div
                  key={id}
                  className={`group relative flex flex-col rounded-lg border overflow-hidden cursor-default transition-colors ${
                    isSelected ? "border-primary bg-primary/5" : "border-border hover:bg-surface-accent/50"
                  }`}
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
                        token={token}
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
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
