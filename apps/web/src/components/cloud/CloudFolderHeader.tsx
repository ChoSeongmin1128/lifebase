"use client";

import { ChevronLeft, ChevronRight, Loader2, Route } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

export interface CloudPathEntry {
  id: string | null;
  name: string;
}

interface CloudFolderHeaderProps {
  path: CloudPathEntry[];
  loading: boolean;
  onNavigate: (folderId: string | null) => void;
}

export function CloudFolderHeader({ path, loading, onNavigate }: CloudFolderHeaderProps) {
  const current = path[path.length - 1];
  const parent = path[path.length - 2] ?? { id: null, name: "내 클라우드" };
  const isRoot = current?.id == null;

  return (
    <div className="border-b border-border px-4 py-3">
      <div className="flex min-h-[76px] flex-col justify-between gap-2">
        <div className="flex h-7 items-center justify-between gap-3">
          <div className="min-w-0">
            {isRoot ? (
              <div className="flex h-7 items-center text-[11px] font-medium uppercase tracking-[0.16em] text-text-muted">
                Cloud
              </div>
            ) : (
              <button
                type="button"
                onClick={() => onNavigate(parent.id)}
                className="inline-flex h-7 items-center gap-1 text-sm text-text-secondary transition-colors hover:text-text-strong"
              >
                <ChevronLeft size={14} />
                <span className="max-w-48 truncate">{parent.name}</span>
              </button>
            )}
          </div>
          <div className="flex h-8 w-[88px] items-center justify-end">
            {!isRoot ? (
              <Popover>
                <PopoverTrigger asChild>
                  <Button variant="ghost" size="sm" className="h-8 shrink-0 gap-1.5 text-text-secondary">
                    <Route size={14} />
                    경로 보기
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="end" className="w-80 max-w-[calc(100vw-1.5rem)] p-3">
                  <div className="space-y-1">
                    <p className="text-xs font-medium text-text-secondary">전체 경로</p>
                    <p className="text-xs text-text-muted">
                      필요한 폴더를 바로 눌러 이동할 수 있습니다.
                    </p>
                  </div>
                  <div className="mt-3 overflow-hidden rounded-lg border border-border">
                    {path.map((entry, index) => {
                      const isCurrent = index === path.length - 1;
                      return (
                        <div key={`${entry.id ?? "root"}-${index}`}>
                          <button
                            type="button"
                            onClick={() => onNavigate(entry.id)}
                            className={`flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors ${
                              isCurrent
                                ? "bg-surface-accent font-medium text-text-strong"
                                : "text-text-secondary hover:bg-surface-accent hover:text-text-strong"
                            }`}
                          >
                            <span className="truncate">{entry.name}</span>
                            {!isCurrent ? <ChevronRight size={14} className="shrink-0 text-text-muted" /> : null}
                          </button>
                          {index < path.length - 1 ? <div className="border-t border-border" /> : null}
                        </div>
                      );
                    })}
                  </div>
                </PopoverContent>
              </Popover>
            ) : (
              <div className="h-8 w-[88px]" aria-hidden="true" />
            )}
          </div>
        </div>
        <div className="flex min-h-9 items-center gap-2">
          <h1 className="truncate text-2xl font-semibold tracking-[-0.02em] text-text-strong">
            {current?.name ?? "내 클라우드"}
          </h1>
          {loading ? <Loader2 size={14} className="shrink-0 animate-spin text-primary" /> : null}
        </div>
      </div>
    </div>
  );
}
