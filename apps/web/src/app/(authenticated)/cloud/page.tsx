"use client";

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
import { BulkActionBar } from "@/components/cloud/BulkActionBar";
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

type SortBy = "name" | "size" | "updated_at" | "created_at";
type SortDir = "asc" | "desc";
type ViewMode = "list" | "grid";

export default function CloudPage() {
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
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<CloudFile[] | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<ViewMode>("list");

  const fileInputRef = useRef<HTMLInputElement>(null);

  const currentFolderID = path[path.length - 1].id;
  const token = getAccessToken();

  const loadFolder = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({ sort_by: sortBy, sort_dir: sortDir });
      if (currentFolderID) params.set("folder_id", currentFolderID);
      const data = await api<{ items: FolderItem[] }>(
        `/cloud/folders?${params}`,
        { token }
      );
      setItems(data.items || []);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [token, currentFolderID, sortBy, sortDir]);

  useEffect(() => {
    setSearchResults(null);
    setSearchQuery("");
    setSelectedIds(new Set());
    loadFolder();
  }, [loadFolder]);

  const navigateToFolder = (folder: FolderData) => {
    setPath((prev) => [...prev, { id: folder.id, name: folder.name }]);
  };

  const navigateToBreadcrumb = (index: number) => {
    setPath((prev) => prev.slice(0, index + 1));
  };

  const handleUpload = async (files: FileList | null) => {
    if (!files || !token) return;
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
    loadFolder();
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
    if (!token) return;
    try {
      if (item.type === "folder" && item.folder) {
        await api(`/cloud/folders/${item.folder.id}`, { method: "DELETE", token });
      } else if (item.type === "file" && item.file) {
        await api(`/cloud/files/${item.file.id}`, { method: "DELETE", token });
      }
      loadFolder();
    } catch (err) {
      console.error("Delete failed:", err);
    }
  };

  const handleBulkDelete = async () => {
    if (!token) return;
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
    loadFolder();
  };

  const handleBulkDownload = async () => {
    for (const item of displayItems) {
      if (item.type !== "file" || !item.file) continue;
      const id = item.file.id;
      if (!selectedIds.has(id)) continue;
      await handleDownload(item.file);
    }
  };

  const handleRename = async () => {
    if (!renaming || !token || !renaming.name.trim()) return;
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
      loadFolder();
    } catch (err) {
      console.error("Rename failed:", err);
    }
  };

  const handleCreateFolder = async () => {
    if (!token || !newFolderName.trim()) return;
    try {
      await api("/cloud/folders", {
        method: "POST",
        body: { name: newFolderName, parent_id: currentFolderID },
        token,
      });
      setShowNewFolder(false);
      setNewFolderName("");
      loadFolder();
    } catch (err) {
      console.error("Create folder failed:", err);
    }
  };

  const handleSearch = async () => {
    if (!token || !searchQuery.trim()) {
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
    setDragOver(false);
    handleUpload(e.dataTransfer.files);
  };

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === displayItems.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(displayItems.map((i) => (i.type === "folder" ? i.folder!.id : i.file!.id))));
    }
  };

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

  const displayItems = searchResults
    ? searchResults.map((f) => ({ type: "file" as const, file: f }))
    : items;

  return (
    <div
      className="flex h-full flex-col"
      onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
    >
      {/* Toolbar Row 1: Breadcrumb */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 md:px-6 py-2">
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

      {/* Toolbar Row 2: Actions */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 md:px-6 py-2">
        <div className="flex items-center gap-2">
          {/* Search */}
          <div className="relative">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted" />
            <Input
              placeholder="검색..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                if (!e.target.value) setSearchResults(null);
              }}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              className="h-8 w-full md:w-48 pl-8"
            />
          </div>

          {/* View toggle */}
          <div className="flex rounded-lg border border-border">
            <button
              onClick={() => setViewMode("list")}
              className={`flex h-8 w-8 items-center justify-center rounded-l-lg ${
                viewMode === "list" ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
              }`}
            >
              <List size={14} />
            </button>
            <button
              onClick={() => setViewMode("grid")}
              className={`flex h-8 w-8 items-center justify-center rounded-r-lg ${
                viewMode === "grid" ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
              }`}
            >
              <LayoutGrid size={14} />
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Sort */}
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
        </div>
      </div>

      {/* New Folder Input */}
      {showNewFolder && (
        <div className="flex items-center gap-2 border-b border-border bg-surface-accent/50 px-6 py-2">
          <Folder size={16} className="text-text-muted" />
          <Input
            autoFocus
            placeholder="폴더 이름"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleCreateFolder();
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
      {searchResults !== null && (
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
      {selectedIds.size > 0 && (
        <BulkActionBar
          count={selectedIds.size}
          onDownload={handleBulkDownload}
          onDelete={handleBulkDelete}
          onClear={() => setSelectedIds(new Set())}
        />
      )}

      {/* Drop overlay */}
      {dragOver && (
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
              {searchResults !== null ? "검색 결과가 없습니다" : "폴더가 비어 있습니다"}
            </p>
            {searchResults === null && (
              <p className="mt-1 text-sm">파일을 업로드하거나 폴더를 만들어 보세요</p>
            )}
          </div>
        ) : viewMode === "list" ? (
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
                  <Checkbox
                    checked={selectedIds.size === displayItems.length && displayItems.length > 0}
                    onCheckedChange={toggleSelectAll}
                  />
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
                      if (item.type === "folder" && item.folder) navigateToFolder(item.folder);
                      else if (item.type === "file" && item.file) handleDownload(item.file);
                    }}
                  >
                    <td className="px-4 md:px-6 py-2">
                      <Checkbox
                        checked={isSelected}
                        onCheckedChange={() => toggleSelect(id)}
                      />
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
                          <span
                            className={item.type === "folder" ? "cursor-pointer hover:underline text-text-strong" : "text-text-primary"}
                            onClick={() => {
                              if (item.type === "folder" && item.folder) navigateToFolder(item.folder);
                            }}
                          >
                            {name}
                          </span>
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
                  className={`group relative flex flex-col items-center gap-2 rounded-lg border p-3 cursor-default transition-colors ${
                    isSelected ? "border-primary bg-primary/5" : "border-border hover:bg-surface-accent/50"
                  }`}
                  onClick={() => toggleSelect(id)}
                  onDoubleClick={() => {
                    if (item.type === "folder" && item.folder) navigateToFolder(item.folder);
                    else if (item.type === "file" && item.file) handleDownload(item.file);
                  }}
                >
                  {item.type === "folder" ? (
                    <Folder size={32} className="text-text-muted" />
                  ) : (
                    <FileIcon mimeType={item.file!.mime_type} size={32} className="text-text-muted" />
                  )}
                  <span className="w-full truncate text-center text-xs text-text-primary">{name}</span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
