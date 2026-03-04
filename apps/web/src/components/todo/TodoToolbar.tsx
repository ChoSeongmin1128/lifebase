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
import { Search, ArrowUpDown, Filter } from "lucide-react";
import { cn } from "@/lib/utils";

type SortBy = "due" | "priority" | "created_at" | "manual";
type FilterMode = "all" | "has_due" | "has_priority" | "done";

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: "due", label: "마감일" },
  { value: "priority", label: "우선순위" },
  { value: "created_at", label: "생성일" },
  { value: "manual", label: "수동" },
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
}

export function TodoToolbar({
  listName,
  searchQuery,
  onSearchChange,
  sortBy,
  onSortChange,
  filter,
  onFilterChange,
}: TodoToolbarProps) {
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

        {/* Filter chips */}
        <div className="hidden md:flex gap-1">
          {FILTER_OPTIONS.map((f) => (
            <button
              key={f.value}
              onClick={() => onFilterChange(f.value)}
              className={cn(
                "rounded-full px-2.5 py-1 text-xs transition-colors",
                filter === f.value
                  ? "bg-primary/10 text-primary font-medium"
                  : "text-text-muted hover:bg-surface-accent"
              )}
            >
              {f.label}
            </button>
          ))}
        </div>

        {/* Mobile filter */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon-sm" className="md:hidden">
              <Filter size={14} />
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
