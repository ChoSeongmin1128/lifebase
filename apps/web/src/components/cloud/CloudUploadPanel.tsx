"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CheckCircle2, ChevronDown, Loader2, RotateCcw, Upload, X, XCircle } from "lucide-react";

export type CloudUploadStatus = "queued" | "uploading" | "processing" | "completed" | "failed" | "canceled";

export interface CloudUploadQueueItem {
  id: string;
  file: File;
  fileName: string;
  size: number;
  folderId: string | null;
  folderName: string;
  status: CloudUploadStatus;
  loadedBytes: number;
  totalBytes: number;
  progressPercent: number;
  errorMessage?: string;
}

interface CloudUploadPanelProps {
  items: CloudUploadQueueItem[];
  expanded: boolean;
  completedCount: number;
  totalCount: number;
  uploadingCount: number;
  overallPercent: number;
  onToggleExpanded: () => void;
  onClose: () => void;
  onCancelItem: (id: string) => void;
  onRetryItem: (id: string) => void;
  onCancelAll: () => void;
  onClearCompleted: () => void;
}

const formatSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
};

const getStatusLabel = (status: CloudUploadStatus) => {
  switch (status) {
    case "queued":
      return "대기 중";
    case "uploading":
      return "업로드 중";
    case "processing":
      return "처리 중";
    case "completed":
      return "완료";
    case "failed":
      return "실패";
    case "canceled":
      return "취소됨";
    default:
      return status;
  }
};

function UploadStatusIcon({ status }: { status: CloudUploadStatus }) {
  if (status === "completed") {
    return <CheckCircle2 size={15} className="mt-0.5 shrink-0 text-success" />;
  }
  if (status === "failed" || status === "canceled") {
    return <XCircle size={15} className="mt-0.5 shrink-0 text-error" />;
  }
  return <Upload size={15} className="mt-0.5 shrink-0 text-info" />;
}

export function CloudUploadPanel({
  items,
  expanded,
  completedCount,
  totalCount,
  uploadingCount,
  overallPercent,
  onToggleExpanded,
  onClose,
  onCancelItem,
  onRetryItem,
  onCancelAll,
  onClearCompleted,
}: CloudUploadPanelProps) {
  if (items.length === 0) return null;

  const hasCancelable = items.some((item) => item.status === "queued" || item.status === "uploading");
  const hasCompleted = items.some((item) => item.status === "completed");

  return (
    <div className="pointer-events-none fixed bottom-24 right-4 z-[110] flex w-[min(92vw,360px)] flex-col gap-2 max-sm:left-1/2 max-sm:right-auto max-sm:-translate-x-1/2">
      <div className="pointer-events-auto rounded-lg border border-info/30 bg-surface shadow-lg">
        <button
          type="button"
          onClick={onToggleExpanded}
          className="flex w-full items-start gap-3 p-3 text-left"
        >
          <Upload size={16} className="mt-0.5 shrink-0 text-info" />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-text-strong">Cloud 업로드</p>
            <p className="mt-0.5 text-xs text-text-secondary">
              {completedCount} / {totalCount} 업로드됨
            </p>
            <div className="mt-2 h-1.5 rounded-full bg-surface-accent">
              <div
                className="h-full rounded-full bg-info transition-[width] duration-200"
                style={{ width: `${Math.max(0, Math.min(100, overallPercent))}%` }}
              />
            </div>
            <p className="mt-1 text-[11px] text-text-muted">
              현재 {uploadingCount}개 업로드 중 · {Math.round(overallPercent)}%
            </p>
          </div>
          <ChevronDown
            size={16}
            className={cn("mt-0.5 shrink-0 text-text-muted transition-transform", expanded && "rotate-180")}
          />
        </button>

        {expanded ? (
          <div className="border-t border-border/70 px-3 pb-3 pt-2">
            <div className="mb-2 flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                {hasCancelable ? (
                  <Button variant="ghost" size="sm" onClick={onCancelAll}>
                    전체 취소
                  </Button>
                ) : null}
                {hasCompleted ? (
                  <Button variant="ghost" size="sm" onClick={onClearCompleted}>
                    완료 항목 정리
                  </Button>
                ) : null}
              </div>
              <Button variant="ghost" size="icon-sm" onClick={onClose} aria-label="업로드 패널 닫기">
                <X size={14} />
              </Button>
            </div>

            <div className="max-h-[320px] space-y-2 overflow-y-auto pr-1">
              {items.map((item) => {
                const isUploading = item.status === "uploading" || item.status === "processing" || item.status === "queued";
                const canCancel = item.status === "queued" || item.status === "uploading";
                const canRetry = item.status === "failed";
                return (
                  <div key={item.id} className="rounded-lg border border-border/70 bg-background px-3 py-2">
                    <div className="flex items-start gap-2">
                      <UploadStatusIcon status={item.status} />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-start justify-between gap-2">
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium text-text-strong">{item.fileName}</p>
                            <p className="mt-0.5 text-[11px] text-text-muted">
                              {item.folderName} · {formatSize(item.size)}
                            </p>
                          </div>
                          <span className="shrink-0 text-[11px] font-medium text-text-secondary">
                            {getStatusLabel(item.status)}
                          </span>
                        </div>

                        <div className="mt-2 h-1.5 rounded-full bg-surface-accent">
                          <div
                            className={cn(
                              "h-full rounded-full transition-[width] duration-150",
                              item.status === "failed" || item.status === "canceled"
                                ? "bg-error/70"
                                : item.status === "completed"
                                  ? "bg-success"
                                  : "bg-info"
                            )}
                            style={{ width: `${Math.max(0, Math.min(100, item.progressPercent))}%` }}
                          />
                        </div>

                        <div className="mt-1 flex items-center justify-between gap-2">
                          <p className="text-[11px] text-text-muted">
                            {isUploading
                              ? `${formatSize(item.loadedBytes)} / ${formatSize(item.totalBytes || item.size)} · ${Math.round(item.progressPercent)}%`
                              : `${Math.round(item.progressPercent)}%`}
                          </p>
                          <div className="flex items-center gap-1">
                            {canRetry ? (
                              <Button variant="ghost" size="sm" onClick={() => onRetryItem(item.id)} className="h-7 px-2 text-[11px]">
                                <RotateCcw size={12} />
                                재시도
                              </Button>
                            ) : null}
                            {canCancel ? (
                              <Button variant="ghost" size="sm" onClick={() => onCancelItem(item.id)} className="h-7 px-2 text-[11px]">
                                취소
                              </Button>
                            ) : null}
                            {item.status === "processing" ? (
                              <Loader2 size={12} className="animate-spin text-text-muted" />
                            ) : null}
                          </div>
                        </div>

                        {item.errorMessage ? (
                          <p className="mt-1 text-[11px] text-error">{item.errorMessage}</p>
                        ) : null}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
