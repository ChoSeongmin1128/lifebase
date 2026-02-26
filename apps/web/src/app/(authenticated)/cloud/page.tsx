"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { api, apiUpload, apiDownload } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";

interface Folder {
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
  folder?: Folder;
  file?: CloudFile;
}

type SortBy = "name" | "size" | "updated_at" | "created_at";
type SortDir = "asc" | "desc";

export default function CloudPage() {
  const [items, setItems] = useState<FolderItem[]>([]);
  const [path, setPath] = useState<{ id: string | null; name: string }[]>([
    { id: null, name: "내 클라우드" },
  ]);
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<SortBy>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [showSortMenu, setShowSortMenu] = useState(false);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    item: FolderItem;
  } | null>(null);
  const [renaming, setRenaming] = useState<{ id: string; type: "folder" | "file"; name: string } | null>(null);
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<CloudFile[] | null>(null);
  const [dragOver, setDragOver] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const sortMenuRef = useRef<HTMLDivElement>(null);

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
    loadFolder();
  }, [loadFolder]);

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      setContextMenu(null);
      if (sortMenuRef.current && !sortMenuRef.current.contains(e.target as Node)) {
        setShowSortMenu(false);
      }
    };
    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, []);

  const navigateToFolder = (folder: Folder) => {
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
        body: {
          name: newFolderName,
          parent_id: currentFolderID,
        },
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

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  };

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    return d.toLocaleDateString("ko-KR", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const getFileIcon = (mimeType: string) => {
    if (mimeType.startsWith("image/")) return "🖼️";
    if (mimeType.startsWith("video/")) return "🎬";
    if (mimeType.startsWith("audio/")) return "🎵";
    if (mimeType.includes("pdf")) return "📄";
    if (mimeType.includes("zip") || mimeType.includes("archive")) return "📦";
    return "📄";
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
      onDragOver={(e) => {
        e.preventDefault();
        setDragOver(true);
      }}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
    >
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-foreground/10 px-4 md:px-6 py-3">
        <div className="flex items-center gap-2">
          {/* Breadcrumb */}
          <nav className="flex items-center gap-1 text-sm">
            {path.map((p, i) => (
              <span key={i} className="flex items-center gap-1">
                {i > 0 && <span className="text-foreground/30">/</span>}
                <button
                  onClick={() => navigateToBreadcrumb(i)}
                  className={`hover:underline ${
                    i === path.length - 1
                      ? "font-medium"
                      : "text-foreground/60"
                  }`}
                >
                  {p.name}
                </button>
              </span>
            ))}
          </nav>
        </div>

        <div className="flex items-center gap-2">
          {/* Search */}
          <div className="relative">
            <input
              type="text"
              placeholder="검색..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                if (!e.target.value) setSearchResults(null);
              }}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              className="h-8 w-full md:w-48 rounded-md border border-foreground/10 bg-background px-3 text-sm outline-none focus:border-foreground/30"
            />
          </div>

          {/* Sort */}
          <div className="relative" ref={sortMenuRef}>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowSortMenu(!showSortMenu);
              }}
              className="flex h-8 items-center gap-1 rounded-md border border-foreground/10 px-2 text-sm hover:bg-foreground/5"
            >
              <SortIcon size={14} />
              <span className="hidden md:inline">정렬</span>
            </button>
            {showSortMenu && (
              <div className="absolute right-0 top-full z-10 mt-1 w-40 rounded-md border border-foreground/10 bg-background py-1 shadow-lg">
                {sortOptions.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => {
                      if (sortBy === opt.value) {
                        setSortDir(sortDir === "asc" ? "desc" : "asc");
                      } else {
                        setSortBy(opt.value);
                        setSortDir(opt.value === "name" ? "asc" : "desc");
                      }
                      setShowSortMenu(false);
                    }}
                    className={`flex w-full items-center justify-between px-3 py-1.5 text-sm hover:bg-foreground/5 ${
                      sortBy === opt.value ? "font-medium" : ""
                    }`}
                  >
                    {opt.label}
                    {sortBy === opt.value && (
                      <span className="text-xs text-foreground/50">
                        {sortDir === "asc" ? "↑" : "↓"}
                      </span>
                    )}
                  </button>
                ))}
                <div className="my-1 border-t border-foreground/10" />
                <button
                  onClick={() => {
                    setSortBy("name");
                    setSortDir("asc");
                    setShowSortMenu(false);
                  }}
                  className="w-full px-3 py-1.5 text-left text-sm text-foreground/50 hover:bg-foreground/5"
                >
                  기본 정렬로 돌아가기
                </button>
              </div>
            )}
          </div>

          {/* New Folder */}
          <button
            onClick={() => setShowNewFolder(true)}
            className="flex h-8 items-center gap-1 rounded-md border border-foreground/10 px-2 text-sm hover:bg-foreground/5"
          >
            <FolderPlusIcon size={14} />
            <span className="hidden md:inline">새 폴더</span>
          </button>

          {/* Upload */}
          <button
            onClick={() => fileInputRef.current?.click()}
            className="flex h-8 items-center gap-1 rounded-md bg-foreground px-2 md:px-3 text-sm text-background hover:opacity-90"
          >
            <UploadIcon size={14} />
            <span className="hidden md:inline">업로드</span>
          </button>
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
        <div className="flex items-center gap-2 border-b border-foreground/10 bg-foreground/[0.02] px-6 py-2">
          <FolderIcon size={16} />
          <input
            autoFocus
            type="text"
            placeholder="폴더 이름"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleCreateFolder();
              if (e.key === "Escape") {
                setShowNewFolder(false);
                setNewFolderName("");
              }
            }}
            className="h-7 flex-1 rounded border border-foreground/10 bg-background px-2 text-sm outline-none focus:border-foreground/30"
          />
          <button
            onClick={handleCreateFolder}
            className="rounded px-2 py-1 text-xs font-medium hover:bg-foreground/5"
          >
            만들기
          </button>
          <button
            onClick={() => {
              setShowNewFolder(false);
              setNewFolderName("");
            }}
            className="rounded px-2 py-1 text-xs text-foreground/50 hover:bg-foreground/5"
          >
            취소
          </button>
        </div>
      )}

      {/* Search indicator */}
      {searchResults !== null && (
        <div className="flex items-center gap-2 border-b border-foreground/10 bg-foreground/[0.02] px-6 py-2 text-sm">
          <span className="text-foreground/60">
            &quot;{searchQuery}&quot; 검색 결과: {searchResults.length}건
          </span>
          <button
            onClick={() => {
              setSearchQuery("");
              setSearchResults(null);
            }}
            className="text-foreground/40 hover:text-foreground"
          >
            ✕
          </button>
        </div>
      )}

      {/* Drop overlay */}
      {dragOver && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/80">
          <div className="rounded-xl border-2 border-dashed border-foreground/20 p-12 text-center">
            <UploadIcon size={48} />
            <p className="mt-2 text-foreground/60">여기에 파일을 놓으세요</p>
          </div>
        </div>
      )}

      {/* File list */}
      <div className="flex-1 overflow-auto">
        {loading ? (
          <div className="flex items-center justify-center py-20 text-foreground/40">
            불러오는 중...
          </div>
        ) : displayItems.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-foreground/40">
            <CloudEmptyIcon size={48} />
            <p className="mt-3">
              {searchResults !== null ? "검색 결과가 없습니다" : "폴더가 비어 있습니다"}
            </p>
            {searchResults === null && (
              <p className="mt-1 text-sm">파일을 업로드하거나 폴더를 만들어 보세요</p>
            )}
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-foreground/10 text-left text-foreground/50">
                <th className="px-4 md:px-6 py-2 font-normal">이름</th>
                <th className="hidden md:table-cell px-4 py-2 font-normal w-28">크기</th>
                <th className="hidden md:table-cell px-4 py-2 font-normal w-36">수정한 날짜</th>
              </tr>
            </thead>
            <tbody>
              {displayItems.map((item) => {
                const id = item.type === "folder" ? item.folder!.id : item.file!.id;
                const name = item.type === "folder" ? item.folder!.name : item.file!.name;
                const isRenaming = renaming?.id === id;

                return (
                  <tr
                    key={id}
                    className="border-b border-foreground/5 hover:bg-foreground/[0.03] cursor-default"
                    onContextMenu={(e) => {
                      e.preventDefault();
                      setContextMenu({ x: e.clientX, y: e.clientY, item });
                    }}
                    onDoubleClick={() => {
                      if (item.type === "folder" && item.folder) {
                        navigateToFolder(item.folder);
                      } else if (item.type === "file" && item.file) {
                        handleDownload(item.file);
                      }
                    }}
                  >
                    <td className="px-4 md:px-6 py-2">
                      <div className="flex items-center gap-2">
                        {item.type === "folder" ? (
                          <FolderIcon size={16} />
                        ) : (
                          <span className="text-base leading-none">
                            {getFileIcon(item.file!.mime_type)}
                          </span>
                        )}
                        {isRenaming ? (
                          <input
                            autoFocus
                            type="text"
                            value={renaming.name}
                            onChange={(e) =>
                              setRenaming({ ...renaming, name: e.target.value })
                            }
                            onKeyDown={(e) => {
                              if (e.key === "Enter") handleRename();
                              if (e.key === "Escape") setRenaming(null);
                            }}
                            onBlur={handleRename}
                            className="h-6 rounded border border-foreground/10 bg-background px-1 text-sm outline-none"
                          />
                        ) : (
                          <span
                            className={
                              item.type === "folder"
                                ? "cursor-pointer hover:underline"
                                : ""
                            }
                            onClick={() => {
                              if (item.type === "folder" && item.folder) {
                                navigateToFolder(item.folder);
                              }
                            }}
                          >
                            {name}
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="hidden md:table-cell px-4 py-2 text-foreground/50">
                      {item.type === "file" ? formatSize(item.file!.size_bytes) : "—"}
                    </td>
                    <td className="hidden md:table-cell px-4 py-2 text-foreground/50">
                      {formatDate(
                        item.type === "folder"
                          ? item.folder!.updated_at
                          : item.file!.updated_at
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Context Menu */}
      {contextMenu && (
        <div
          className="fixed z-50 min-w-[160px] rounded-md border border-foreground/10 bg-background py-1 shadow-lg"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          {contextMenu.item.type === "folder" && contextMenu.item.folder && (
            <button
              onClick={() => {
                navigateToFolder(contextMenu.item.folder!);
                setContextMenu(null);
              }}
              className="flex w-full px-3 py-1.5 text-sm hover:bg-foreground/5"
            >
              열기
            </button>
          )}
          {contextMenu.item.type === "file" && contextMenu.item.file && (
            <button
              onClick={() => {
                handleDownload(contextMenu.item.file!);
                setContextMenu(null);
              }}
              className="flex w-full px-3 py-1.5 text-sm hover:bg-foreground/5"
            >
              다운로드
            </button>
          )}
          <button
            onClick={() => {
              const item = contextMenu.item;
              const id = item.type === "folder" ? item.folder!.id : item.file!.id;
              const name = item.type === "folder" ? item.folder!.name : item.file!.name;
              setRenaming({ id, type: item.type, name });
              setContextMenu(null);
            }}
            className="flex w-full px-3 py-1.5 text-sm hover:bg-foreground/5"
          >
            이름 변경
          </button>
          <div className="my-1 border-t border-foreground/10" />
          <button
            onClick={() => {
              handleDelete(contextMenu.item);
              setContextMenu(null);
            }}
            className="flex w-full px-3 py-1.5 text-sm text-red-500 hover:bg-foreground/5"
          >
            삭제
          </button>
        </div>
      )}
    </div>
  );
}

// Icons

function SortIcon({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="m3 16 4 4 4-4" /><line x1="7" x2="7" y1="20" y2="4" />
      <path d="m21 8-4-4-4 4" /><line x1="17" x2="17" y1="4" y2="20" />
    </svg>
  );
}

function FolderIcon({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>
    </svg>
  );
}

function FolderPlusIcon({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 10v6"/><path d="M9 13h6"/>
      <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>
    </svg>
  );
}

function UploadIcon({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
      <polyline points="17 8 12 3 7 8"/><line x1="12" x2="12" y1="3" y2="15"/>
    </svg>
  );
}

function CloudEmptyIcon({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-foreground/20">
      <path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/>
    </svg>
  );
}
