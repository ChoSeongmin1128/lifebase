"use client";

import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export interface CloudPathEntry {
  id: string | null;
  name: string;
}

interface CloudFolderHeaderProps {
  path: CloudPathEntry[];
  loading: boolean;
  onNavigate: (folderId: string | null) => void;
  currentActions?: React.ReactNode;
}

export function CloudFolderHeader({ path, loading, onNavigate, currentActions }: CloudFolderHeaderProps) {
  const current = path[path.length - 1];
  const ancestors = path.slice(0, -1);

  return (
    <div className="border-b border-border px-4 py-3 md:px-6 lg:px-8">
      <div className="min-w-0 overflow-x-auto">
        <div
          className={`flex min-h-10 min-w-max items-center gap-1 whitespace-nowrap text-sm transition-opacity duration-150 ${
            loading ? "opacity-90" : "opacity-100"
          }`}
        >
          {ancestors.map((entry, index) => (
            <div key={`${entry.id ?? "root"}-${index}`} className="flex items-center gap-1">
              {index > 0 ? <ChevronRight size={13} className="shrink-0 text-text-muted" /> : null}
              <button
                type="button"
                onClick={() => onNavigate(entry.id)}
                className="max-w-32 truncate rounded-full px-2 py-1 text-sm text-text-secondary transition-colors hover:bg-surface-accent hover:text-text-strong md:max-w-40"
              >
                {entry.name}
              </button>
            </div>
          ))}

          {ancestors.length > 0 ? <ChevronRight size={13} className="shrink-0 text-text-muted" /> : null}

          {currentActions ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="inline-flex max-w-[18rem] items-center gap-1 rounded-md px-1.5 py-1 text-base font-semibold text-text-strong transition-colors hover:bg-surface-accent/80 data-[state=open]:bg-surface-accent/80"
                >
                  <span className="truncate">{current?.name ?? "내 드라이브"}</span>
                  <ChevronDown size={14} className="shrink-0 text-text-muted" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                {currentActions}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <span className="inline-flex max-w-[18rem] items-center px-1.5 py-1 text-base font-semibold text-text-strong">
              <span className="truncate">{current?.name ?? "내 드라이브"}</span>
            </span>
          )}

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
