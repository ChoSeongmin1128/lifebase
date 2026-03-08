"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { PageToolbar, PageToolbarGroup } from "@/components/layout/PageToolbar";
import { Search, ArrowUpDown, Filter, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";

type SortBy = "manual" | "due" | "recent_starred" | "title";
type FilterMode = "all" | "has_due" | "has_priority" | "done";

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: "manual", label: "내가 정렬한대로" },
  { value: "due", label: "기한" },
  { value: "recent_starred", label: "최근 별표한 항목" },
  { value: "title", label: "제목" },
];

const FILTER_OPTIONS: { value: FilterMode; label: string }[] = [
  { value: "all", label: "전체" },
  { value: "has_due", label: "기한 있음" },
  { value: "has_priority", label: "우선순위 있음" },
  { value: "done", label: "완료됨" },
];

interface TodoToolbarProps {
  listName: string;
  searchQuery: string;
  onSearchChange: (q: string) => void;
  sortBy: SortBy;
  onSortChange: (s: SortBy) => void;
  filter: FilterMode;
  onFilterChange: (f: FilterMode) => void;
  lastSyncedAt?: string;
  syncingNow?: boolean;
  onManualSync?: () => void;
}

export function TodoToolbar({
  listName,
  searchQuery,
  onSearchChange,
  sortBy,
  onSortChange,
  filter,
  onFilterChange,
  lastSyncedAt,
  syncingNow = false,
  onManualSync,
}: TodoToolbarProps) {
  const activeFilterLabel =
    FILTER_OPTIONS.find((item) => item.value === filter)?.label || "전체";

  return (
    <PageToolbar className="gap-3 py-3">
      <div className="min-w-0">
        <h2 className="truncate text-base font-semibold text-text-strong">{listName}</h2>
        <p className="mt-0.5 hidden text-xs text-text-muted md:block">
          {syncingNow
            ? "Google 동기화 중"
            : lastSyncedAt
              ? `최근 동기화 ${new Date(lastSyncedAt).toLocaleString("ko-KR")}`
              : "빠른 편집과 계층 정리를 바로 이어서 할 수 있습니다"}
        </p>
      </div>
      <PageToolbarGroup className="gap-2">
        <div className="relative">
          <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted" />
          <Input
            placeholder="검색..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="h-8 w-full pl-8 md:w-48"
          />
        </div>

        {/* Filter */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-1.5">
              <Filter size={14} />
              <span>{activeFilterLabel}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {FILTER_OPTIONS.map((f) => (
              <DropdownMenuItem
                key={f.value}
                onClick={() => onFilterChange(f.value)}
                className={filter === f.value ? "font-medium text-primary" : ""}
              >
                {f.label}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <Button
          variant="ghost"
          size="icon-sm"
          onClick={onManualSync}
          disabled={!onManualSync || syncingNow}
          title="지금 동기화"
        >
          <RefreshCw className={cn("h-4 w-4", syncingNow && "animate-spin")} />
        </Button>

        {/* Sort */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-1.5">
              <ArrowUpDown size={14} />
              <span className="hidden md:inline">정렬</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {SORT_OPTIONS.map((s) => (
              <DropdownMenuItem
                key={s.value}
                onClick={() => onSortChange(s.value)}
                className={sortBy === s.value ? "font-medium text-primary" : ""}
              >
                {s.label}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

      </PageToolbarGroup>
    </PageToolbar>
  );
}

export type { SortBy as TodoSortBy, FilterMode as TodoFilterMode };
