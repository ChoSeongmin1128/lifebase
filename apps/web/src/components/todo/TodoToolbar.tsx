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

type SortBy = "manual" | "date" | "due" | "recent_starred" | "title";
type FilterMode = "all" | "has_due" | "has_priority" | "done";

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: "manual", label: "내가 정렬한대로" },
  { value: "date", label: "날짜" },
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
    <PageToolbar className="py-3">
      <h2 className="font-medium text-text-strong">{listName}</h2>
      <PageToolbarGroup className="gap-2">
        <div className="relative">
          <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted" />
          <Input
            placeholder="검색..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="h-8 w-40 pl-8"
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

        <div className="hidden items-center gap-2 text-xs text-text-muted md:flex">
          <span>최근 동기화: {lastSyncedAt ? new Date(lastSyncedAt).toLocaleString("ko-KR") : "-"}</span>
        </div>
        <Button
          variant="secondary"
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
