"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import Image from "next/image";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:38117";

interface MediaFile {
  id: string;
  name: string;
  mime_type: string;
  size_bytes: number;
  thumb_status: string;
  taken_at: string | null;
  created_at: string;
  updated_at: string;
}

type ViewMode = "grid" | "list" | "date";
type MediaType = "all" | "image" | "video";
type SortBy = "taken_at" | "created_at" | "name" | "size";

export default function GalleryPage() {
  const [items, setItems] = useState<MediaFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>("grid");
  const [mediaType, setMediaType] = useState<MediaType>("all");
  const [sortBy, setSortBy] = useState<SortBy>("taken_at");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");
  const [nextCursor, setNextCursor] = useState<string>("");
  const [loadingMore, setLoadingMore] = useState(false);
  const [selectedFile, setSelectedFile] = useState<MediaFile | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  const token = getAccessToken();

  const loadMedia = useCallback(
    async (cursor?: string) => {
      if (!token) return;
      const isLoadMore = !!cursor;
      if (isLoadMore) setLoadingMore(true);
      else setLoading(true);

      try {
        const params = new URLSearchParams({
          sort_by: sortBy,
          sort_dir: sortDir,
          limit: "50",
        });
        if (mediaType !== "all") params.set("type", mediaType);
        if (cursor) params.set("cursor", cursor);

        const data = await api<{ items: MediaFile[]; next_cursor?: string }>(
          `/gallery?${params}`,
          { token }
        );

        if (isLoadMore) {
          setItems((prev) => [...prev, ...(data.items || [])]);
        } else {
          setItems(data.items || []);
        }
        setNextCursor(data.next_cursor || "");
      } catch {
        if (!isLoadMore) setItems([]);
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [token, mediaType, sortBy, sortDir]
  );

  useEffect(() => {
    loadMedia();
  }, [loadMedia]);

  // Infinite scroll
  useEffect(() => {
    if (!sentinelRef.current || !nextCursor) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && nextCursor && !loadingMore) {
          loadMedia(nextCursor);
        }
      },
      { threshold: 0.1 }
    );
    observer.observe(sentinelRef.current);
    return () => observer.disconnect();
  }, [nextCursor, loadingMore, loadMedia]);

  const thumbUrl = (fileId: string, size: "small" | "medium") =>
    `${API_URL}/api/v1/gallery/thumbnails/${fileId}/${size}`;

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024)
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("ko-KR", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  const groupByDate = (files: MediaFile[]) => {
    const groups: { date: string; items: MediaFile[] }[] = [];
    const map = new Map<string, MediaFile[]>();
    for (const f of files) {
      const dateStr = (f.taken_at || f.created_at).split("T")[0];
      if (!map.has(dateStr)) {
        map.set(dateStr, []);
      }
      map.get(dateStr)!.push(f);
    }
    for (const [date, dateItems] of map) {
      groups.push({ date, items: dateItems });
    }
    return groups;
  };

  const viewModes: { value: ViewMode; icon: React.ReactNode }[] = [
    {
      value: "grid",
      icon: (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" />
          <rect x="3" y="14" width="7" height="7" /><rect x="14" y="14" width="7" height="7" />
        </svg>
      ),
    },
    {
      value: "list",
      icon: (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <line x1="8" x2="21" y1="6" y2="6" /><line x1="8" x2="21" y1="12" y2="12" />
          <line x1="8" x2="21" y1="18" y2="18" /><line x1="3" x2="3.01" y1="6" y2="6" />
          <line x1="3" x2="3.01" y1="12" y2="12" /><line x1="3" x2="3.01" y1="18" y2="18" />
        </svg>
      ),
    },
    {
      value: "date",
      icon: (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <rect width="18" height="18" x="3" y="4" rx="2" />
          <line x1="16" x2="16" y1="2" y2="6" /><line x1="8" x2="8" y1="2" y2="6" />
          <line x1="3" x2="21" y1="10" y2="10" />
        </svg>
      ),
    },
  ];

  const mediaFilters: { value: MediaType; label: string }[] = [
    { value: "all", label: "전체" },
    { value: "image", label: "이미지" },
    { value: "video", label: "동영상" },
  ];

  const sortOptions: { value: SortBy; label: string }[] = [
    { value: "taken_at", label: "촬영일" },
    { value: "created_at", label: "업로드일" },
    { value: "name", label: "이름" },
    { value: "size", label: "크기" },
  ];

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-foreground/10 px-4 md:px-6 py-3">
        <h1 className="text-lg font-semibold">갤러리</h1>
        <div className="flex items-center gap-2 md:gap-3">
          {/* Media type filter */}
          <div className="flex rounded-md border border-foreground/10">
            {mediaFilters.map((f) => (
              <button
                key={f.value}
                onClick={() => setMediaType(f.value)}
                className={`px-3 py-1 text-xs ${
                  mediaType === f.value
                    ? "bg-foreground/10 font-medium"
                    : "hover:bg-foreground/5 text-foreground/60"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {/* Sort */}
          <select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as SortBy)}
            className="h-7 rounded-md border border-foreground/10 bg-background px-2 text-xs outline-none"
          >
            {sortOptions.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
          <button
            onClick={() => setSortDir(sortDir === "asc" ? "desc" : "asc")}
            className="flex h-7 w-7 items-center justify-center rounded-md border border-foreground/10 text-xs hover:bg-foreground/5"
          >
            {sortDir === "asc" ? "↑" : "↓"}
          </button>

          {/* View mode */}
          <div className="flex rounded-md border border-foreground/10">
            {viewModes.map((v) => (
              <button
                key={v.value}
                onClick={() => setViewMode(v.value)}
                className={`flex h-7 w-7 items-center justify-center ${
                  viewMode === v.value
                    ? "bg-foreground/10"
                    : "hover:bg-foreground/5 text-foreground/60"
                }`}
              >
                {v.icon}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4">
        {loading ? (
          <div className="flex items-center justify-center py-20 text-foreground/40">
            불러오는 중...
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-foreground/40">
            <svg width={48} height={48} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-foreground/20">
              <rect width="18" height="18" x="3" y="3" rx="2" />
              <circle cx="9" cy="9" r="2" />
              <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
            </svg>
            <p className="mt-3">미디어 파일이 없습니다</p>
            <p className="mt-1 text-sm">Cloud에 이미지나 동영상을 업로드하면 여기에 표시됩니다</p>
          </div>
        ) : viewMode === "grid" ? (
          <div className="grid grid-cols-[repeat(auto-fill,minmax(100px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2">
            {items.map((file) => (
              <div
                key={file.id}
                className="group relative aspect-square cursor-pointer overflow-hidden rounded-lg bg-foreground/5"
                onClick={() => setSelectedFile(file)}
              >
                {file.thumb_status === "done" ? (
                  <Image
                    src={thumbUrl(file.id, "small")}
                    alt={file.name}
                    fill
                    unoptimized
                    sizes="(max-width: 768px) 33vw, (max-width: 1280px) 16vw, 150px"
                    className="object-cover"
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-foreground/30">
                    {file.thumb_status === "processing" ? (
                      <span className="text-xs">처리 중...</span>
                    ) : file.thumb_status === "failed" ? (
                      <span className="text-xs">실패</span>
                    ) : (
                      <span className="text-xs">대기 중</span>
                    )}
                  </div>
                )}
                {file.mime_type.startsWith("video/") && (
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-black/50">
                      <svg width={14} height={14} viewBox="0 0 24 24" fill="white">
                        <polygon points="5,3 19,12 5,21" />
                      </svg>
                    </div>
                  </div>
                )}
                <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/50 to-transparent p-2 opacity-0 transition-opacity group-hover:opacity-100">
                  <p className="truncate text-xs text-white">{file.name}</p>
                </div>
              </div>
            ))}
          </div>
        ) : viewMode === "list" ? (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-foreground/10 text-left text-foreground/50">
                <th className="w-12 px-2 py-2 font-normal"></th>
                <th className="px-4 py-2 font-normal">이름</th>
                <th className="hidden md:table-cell w-24 px-4 py-2 font-normal">크기</th>
                <th className="hidden md:table-cell w-36 px-4 py-2 font-normal">날짜</th>
              </tr>
            </thead>
            <tbody>
              {items.map((file) => (
                <tr
                  key={file.id}
                  className="cursor-pointer border-b border-foreground/5 hover:bg-foreground/[0.03]"
                  onClick={() => setSelectedFile(file)}
                >
                  <td className="px-2 py-1">
                    <div className="relative h-8 w-8 overflow-hidden rounded bg-foreground/5">
                      {file.thumb_status === "done" ? (
                        <Image
                          src={thumbUrl(file.id, "small")}
                          alt=""
                          fill
                          unoptimized
                          sizes="32px"
                          className="object-cover"
                        />
                      ) : (
                        <div className="flex h-full w-full items-center justify-center text-[8px] text-foreground/30">
                          {file.name.split(".").pop()?.toUpperCase()}
                        </div>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-1">
                    <div className="flex items-center gap-2">
                      {file.mime_type.startsWith("video/") && (
                        <svg width={12} height={12} viewBox="0 0 24 24" fill="currentColor" className="shrink-0 text-foreground/40">
                          <polygon points="5,3 19,12 5,21" />
                        </svg>
                      )}
                      <span className="truncate">{file.name}</span>
                    </div>
                  </td>
                  <td className="hidden md:table-cell px-4 py-1 text-foreground/50">
                    {formatSize(file.size_bytes)}
                  </td>
                  <td className="hidden md:table-cell px-4 py-1 text-foreground/50">
                    {formatDate(file.taken_at || file.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          /* date grouped view */
          <div className="space-y-6">
            {groupByDate(items).map((group) => (
              <div key={group.date}>
                <h3 className="mb-2 text-sm font-medium text-foreground/70">
                  {formatDate(group.date)}
                </h3>
                <div className="grid grid-cols-[repeat(auto-fill,minmax(100px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2">
                  {group.items.map((file) => (
                    <div
                      key={file.id}
                      className="group relative aspect-square cursor-pointer overflow-hidden rounded-lg bg-foreground/5"
                      onClick={() => setSelectedFile(file)}
                    >
                      {file.thumb_status === "done" ? (
                        <Image
                          src={thumbUrl(file.id, "small")}
                          alt={file.name}
                          fill
                          unoptimized
                          sizes="(max-width: 768px) 33vw, (max-width: 1280px) 16vw, 150px"
                          className="object-cover"
                        />
                      ) : (
                        <div className="flex h-full w-full items-center justify-center text-xs text-foreground/30">
                          {file.name.split(".").pop()?.toUpperCase()}
                        </div>
                      )}
                      {file.mime_type.startsWith("video/") && (
                        <div className="absolute inset-0 flex items-center justify-center">
                          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-black/50">
                            <svg width={14} height={14} viewBox="0 0 24 24" fill="white">
                              <polygon points="5,3 19,12 5,21" />
                            </svg>
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Infinite scroll sentinel */}
        {nextCursor && <div ref={sentinelRef} className="h-4" />}
        {loadingMore && (
          <div className="py-4 text-center text-sm text-foreground/40">
            더 불러오는 중...
          </div>
        )}
      </div>

      {/* Preview Modal */}
      {selectedFile && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
          onClick={() => setSelectedFile(null)}
        >
          <div
            className="relative max-h-[90vh] max-w-[90vw]"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={() => setSelectedFile(null)}
              className="absolute -right-3 -top-3 flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
            >
              ✕
            </button>
            {selectedFile.thumb_status === "done" ? (
              <Image
                src={thumbUrl(selectedFile.id, "medium")}
                alt={selectedFile.name}
                width={1200}
                height={1200}
                unoptimized
                sizes="90vw"
                className="max-h-[85vh] w-auto rounded-lg object-contain"
              />
            ) : (
              <div className="flex h-64 w-64 items-center justify-center rounded-lg bg-foreground/10 text-foreground/40">
                썸네일 없음
              </div>
            )}
            <div className="mt-2 text-center">
              <p className="text-sm text-white/80">{selectedFile.name}</p>
              <p className="text-xs text-white/50">
                {formatSize(selectedFile.size_bytes)} · {formatDate(selectedFile.taken_at || selectedFile.created_at)}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
