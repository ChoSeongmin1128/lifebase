"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import Image from "next/image";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
} from "@/components/ui/tooltip";
import { ExtensionBadge } from "@/components/gallery/ExtensionBadge";
import {
  LayoutGrid,
  List,
  CalendarDays,
  Layers,
  ImageIcon,
  Film,
  Play,
  ArrowUp,
  ArrowDown,
  Loader2,
  AlertCircle,
  Clock,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";

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
type ThumbSize = "small" | "medium";

type ThumbnailImageProps =
  | {
      fileId: string;
      size: ThumbSize;
      token: string | null;
      alt: string;
      className?: string;
      sizes?: string;
      fallback?: React.ReactNode;
      fill: true;
      width?: never;
      height?: never;
    }
  | {
      fileId: string;
      size: ThumbSize;
      token: string | null;
      alt: string;
      className?: string;
      sizes?: string;
      fallback?: React.ReactNode;
      fill?: false;
      width: number;
      height: number;
    };

function ThumbnailImage(props: ThumbnailImageProps) {
  const { fileId, size, token, alt, className, sizes, fallback } = props;
  const [src, setSrc] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      setSrc(null);
      return;
    }

    const controller = new AbortController();
    let objectURL: string | null = null;
    let active = true;

    const load = async () => {
      try {
        const res = await fetch(`${API_URL}/api/v1/gallery/thumbnails/${fileId}/${size}`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });
        if (!res.ok) {
          throw new Error(`thumbnail request failed: ${res.status}`);
        }

        const blob = await res.blob();
        objectURL = URL.createObjectURL(blob);
        if (active) {
          setSrc(objectURL);
        }
      } catch {
        if (active) {
          setSrc(null);
        }
      }
    };

    load();

    return () => {
      active = false;
      controller.abort();
      if (objectURL) {
        URL.revokeObjectURL(objectURL);
      }
    };
  }, [fileId, size, token]);

  if (!src) {
    return <>{fallback ?? null}</>;
  }

  if (props.fill) {
    return (
      <Image
        src={src}
        alt={alt}
        fill
        unoptimized
        sizes={sizes}
        className={className}
      />
    );
  }

  return (
    <Image
      src={src}
      alt={alt}
      width={props.width}
      height={props.height}
      unoptimized
      sizes={sizes}
      className={className}
    />
  );
}

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
        const params = new URLSearchParams({ sort_by: sortBy, sort_dir: sortDir, limit: "50" });
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

  useEffect(() => { loadMedia(); }, [loadMedia]);

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

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  };

  const formatDate = (dateStr: string) =>
    new Date(dateStr).toLocaleDateString("ko-KR", { year: "numeric", month: "long", day: "numeric" });

  const groupByDate = (files: MediaFile[]) => {
    const map = new Map<string, MediaFile[]>();
    for (const f of files) {
      const dateStr = (f.taken_at || f.created_at).split("T")[0];
      if (!map.has(dateStr)) map.set(dateStr, []);
      map.get(dateStr)!.push(f);
    }
    return Array.from(map, ([date, items]) => ({ date, items }));
  };

  const viewModes: { value: ViewMode; icon: React.ComponentType<{ size?: number }>; label: string }[] = [
    { value: "grid", icon: LayoutGrid, label: "그리드" },
    { value: "list", icon: List, label: "목록" },
    { value: "date", icon: CalendarDays, label: "날짜별" },
  ];

  const mediaFilters: { value: MediaType; icon: React.ComponentType<{ size?: number }>; label: string }[] = [
    { value: "all", icon: Layers, label: "전체" },
    { value: "image", icon: ImageIcon, label: "이미지" },
    { value: "video", icon: Film, label: "동영상" },
  ];

  function ThumbStatusIcon({ status }: { status: string }) {
    if (status === "processing") return <Loader2 size={16} className="animate-spin text-text-muted" />;
    if (status === "failed") return <AlertCircle size={16} className="text-error" />;
    return <Clock size={16} className="text-text-muted" />;
  }

  function GridItem({ file }: { file: MediaFile }) {
    return (
      <div
        className="group relative aspect-square cursor-pointer overflow-hidden rounded-lg bg-surface-accent"
        onClick={() => setSelectedFile(file)}
      >
        {file.thumb_status === "done" ? (
          <ThumbnailImage
            fileId={file.id}
            size="small"
            token={token}
            alt={file.name}
            fill
            sizes="(max-width: 768px) 33vw, (max-width: 1280px) 16vw, 150px"
            className="object-cover"
            fallback={
              <div className="flex h-full w-full items-center justify-center">
                <ThumbStatusIcon status="failed" />
              </div>
            }
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <ThumbStatusIcon status={file.thumb_status} />
          </div>
        )}
        {file.mime_type.startsWith("video/") && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-black/50">
              <Play size={14} className="text-white ml-0.5" fill="white" />
            </div>
          </div>
        )}
        {/* Extension badge */}
        <div className="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <ExtensionBadge filename={file.name} />
        </div>
        <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/50 to-transparent p-2 opacity-0 transition-opacity group-hover:opacity-100">
          <p className="truncate text-xs text-white">{file.name}</p>
        </div>
      </div>
    );
  }

  return (
    <TooltipProvider>
      <div className="flex h-full flex-col">
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 md:px-6 py-3">
          <h1 className="text-lg font-semibold text-text-strong">갤러리</h1>
          <div className="flex items-center gap-2 md:gap-3">
            {/* Media type filter */}
            <div className="flex rounded-lg border border-border">
              {mediaFilters.map((f) => (
                <Tooltip key={f.value}>
                  <TooltipTrigger asChild>
                    <button
                      onClick={() => setMediaType(f.value)}
                      className={cn(
                        "flex h-8 w-8 items-center justify-center transition-colors",
                        mediaType === f.value ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
                      )}
                    >
                      <f.icon size={14} />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{f.label}</TooltipContent>
                </Tooltip>
              ))}
            </div>

            {/* Sort */}
            <Select value={sortBy} onValueChange={(v) => setSortBy(v as SortBy)}>
              <SelectTrigger className="h-8 w-24 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="taken_at">촬영일</SelectItem>
                <SelectItem value="created_at">업로드일</SelectItem>
                <SelectItem value="name">이름</SelectItem>
                <SelectItem value="size">크기</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => setSortDir(sortDir === "asc" ? "desc" : "asc")}
            >
              {sortDir === "asc" ? <ArrowUp size={14} /> : <ArrowDown size={14} />}
            </Button>

            {/* View mode */}
            <div className="flex rounded-lg border border-border">
              {viewModes.map((v) => (
                <Tooltip key={v.value}>
                  <TooltipTrigger asChild>
                    <button
                      onClick={() => setViewMode(v.value)}
                      className={cn(
                        "flex h-8 w-8 items-center justify-center transition-colors",
                        viewMode === v.value ? "bg-surface-accent text-text-strong" : "text-text-muted hover:bg-surface-accent"
                      )}
                    >
                      <v.icon size={14} />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{v.label}</TooltipContent>
                </Tooltip>
              ))}
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4">
          {loading ? (
            <div className="flex items-center justify-center py-20 text-text-muted">불러오는 중...</div>
          ) : items.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 text-text-muted">
              <ImageIcon size={48} className="text-border" />
              <p className="mt-3">미디어 파일이 없습니다</p>
              <p className="mt-1 text-sm">Cloud에 이미지나 동영상을 업로드하면 여기에 표시됩니다</p>
            </div>
          ) : viewMode === "grid" ? (
            <div className="grid grid-cols-[repeat(auto-fill,minmax(100px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2">
              {items.map((file) => <GridItem key={file.id} file={file} />)}
            </div>
          ) : viewMode === "list" ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-text-muted">
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
                    className="cursor-pointer border-b border-border/50 hover:bg-surface-accent/50"
                    onClick={() => setSelectedFile(file)}
                  >
                    <td className="px-2 py-1">
                      <div className="relative h-8 w-8 overflow-hidden rounded bg-surface-accent">
                        {file.thumb_status === "done" ? (
                          <ThumbnailImage
                            fileId={file.id}
                            size="small"
                            token={token}
                            alt=""
                            fill
                            sizes="32px"
                            className="object-cover"
                            fallback={
                              <div className="flex h-full w-full items-center justify-center">
                                <ThumbStatusIcon status="failed" />
                              </div>
                            }
                          />
                        ) : (
                          <div className="flex h-full w-full items-center justify-center">
                            <ThumbStatusIcon status={file.thumb_status} />
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-1">
                      <div className="flex items-center gap-2">
                        {file.mime_type.startsWith("video/") && (
                          <Film size={12} className="shrink-0 text-text-muted" />
                        )}
                        <span className="truncate text-text-primary">{file.name}</span>
                        <ExtensionBadge filename={file.name} className="shrink-0" />
                      </div>
                    </td>
                    <td className="hidden md:table-cell px-4 py-1 text-text-muted">
                      {formatSize(file.size_bytes)}
                    </td>
                    <td className="hidden md:table-cell px-4 py-1 text-text-muted">
                      {formatDate(file.taken_at || file.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="space-y-6">
              {groupByDate(items).map((group) => (
                <div key={group.date}>
                  <h3 className="mb-2 text-sm font-medium text-text-secondary">{formatDate(group.date)}</h3>
                  <div className="grid grid-cols-[repeat(auto-fill,minmax(100px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2">
                    {group.items.map((file) => <GridItem key={file.id} file={file} />)}
                  </div>
                </div>
              ))}
            </div>
          )}

          {nextCursor && <div ref={sentinelRef} className="h-4" />}
          {loadingMore && (
            <div className="py-4 text-center text-sm text-text-muted">더 불러오는 중...</div>
          )}
        </div>

        {/* Preview Modal */}
        {selectedFile && (
          <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
            onClick={() => setSelectedFile(null)}
          >
            <div className="relative max-h-[90vh] max-w-[90vw]" onClick={(e) => e.stopPropagation()}>
              <button
                onClick={() => setSelectedFile(null)}
                className="absolute -right-3 -top-3 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
              >
                <X size={16} />
              </button>
              {selectedFile.thumb_status === "done" ? (
                <ThumbnailImage
                  fileId={selectedFile.id}
                  size="medium"
                  token={token}
                  alt={selectedFile.name}
                  width={1200}
                  height={1200}
                  sizes="90vw"
                  className="max-h-[85vh] w-auto rounded-lg object-contain"
                  fallback={
                    <div className="flex h-64 w-64 items-center justify-center rounded-lg bg-surface text-text-muted">
                      썸네일 없음
                    </div>
                  }
                />
              ) : (
                <div className="flex h-64 w-64 items-center justify-center rounded-lg bg-surface text-text-muted">
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
    </TooltipProvider>
  );
}
