"use client";

import { ChevronRight, Loader2 } from "lucide-react";

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
  const ancestors = path.slice(0, -1);

  return (
    <div className="border-b border-border px-4 py-3">
      <div className="flex min-h-[76px] flex-col justify-between gap-2">
        <div className="min-h-7 overflow-x-auto">
          {ancestors.length > 0 ? (
            <div
              className={`flex w-max min-w-full items-center gap-1 whitespace-nowrap text-sm transition-opacity duration-150 ${
                loading ? "opacity-90" : "opacity-100"
              }`}
            >
              {ancestors.map((entry, index) => (
                <div key={`${entry.id ?? "root"}-${index}`} className="flex items-center gap-1">
                  {index > 0 ? <ChevronRight size={13} className="shrink-0 text-text-muted" /> : null}
                  <button
                    type="button"
                    onClick={() => onNavigate(entry.id)}
                    className="max-w-32 truncate rounded-full px-2 py-1 text-xs text-text-secondary transition-colors hover:bg-surface-accent hover:text-text-strong md:max-w-40"
                  >
                    {entry.name}
                  </button>
                </div>
              ))}
            </div>
          ) : null}
        </div>
        <div className="flex min-h-9 items-center gap-2">
          <h1
            className={`truncate text-2xl font-semibold tracking-[-0.02em] text-text-strong transition-opacity duration-150 ${
              loading ? "opacity-90" : "opacity-100"
            }`}
          >
            {current?.name ?? "보관함"}
          </h1>
          <span className="flex h-4 w-4 shrink-0 items-center justify-center">
            <Loader2
              size={14}
              className={`text-primary transition-opacity duration-150 ${
                loading ? "animate-spin delay-[180ms] opacity-100" : "opacity-0"
              }`}
            />
          </span>
        </div>
      </div>
    </div>
  );
}
