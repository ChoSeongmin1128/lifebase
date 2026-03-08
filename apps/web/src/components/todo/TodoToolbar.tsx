"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { PageToolbar } from "@/components/layout/PageToolbar";
import { Search, ArrowUpDown, ChevronDown, MoreVertical, Plus, RefreshCw, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";

type SortBy = "manual" | "due" | "recent_starred" | "title";

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: "manual", label: "내가 정렬한대로" },
  { value: "due", label: "기한" },
  { value: "recent_starred", label: "최근 별표한 항목" },
  { value: "title", label: "제목" },
];

interface TodoListOption {
  id: string;
  name: string;
  activeCount: number;
}

interface TodoToolbarProps {
  currentListName: string;
  lists: TodoListOption[];
  activeListId: string;
  onActiveListChange: (id: string) => void;
  onCreateList: () => void;
  onDeleteCurrentList?: () => void;
  searchQuery: string;
  onSearchChange: (q: string) => void;
  sortBy: SortBy;
  onSortChange: (s: SortBy) => void;
  lastSyncedAt?: string;
  syncingNow?: boolean;
  onManualSync?: () => void;
}

export function TodoToolbar({
  currentListName,
  lists,
  activeListId,
  onActiveListChange,
  onCreateList,
  onDeleteCurrentList,
  searchQuery,
  onSearchChange,
  sortBy,
  onSortChange,
  lastSyncedAt,
  syncingNow = false,
  onManualSync,
}: TodoToolbarProps) {
  return (
    <PageToolbar className="flex-col items-stretch gap-2 py-3">
      <div className="flex min-w-0 items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="w-[12rem] justify-between gap-2 px-2.5">
              <span className="min-w-0 flex-1 truncate text-left text-base font-semibold text-text-strong">
                {currentListName}
              </span>
              <ChevronDown size={14} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-56">
            {lists.map((list) => (
              <DropdownMenuItem
                key={list.id}
                onClick={() => onActiveListChange(list.id)}
                className={cn(
                  "justify-between gap-3",
                  activeListId === list.id && "font-medium text-primary"
                )}
              >
                <span className="truncate">{list.name}</span>
                <span className="shrink-0 text-[10px] tabular-nums text-text-muted">
                  {list.activeCount}
                </span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <p className="hidden min-w-0 truncate text-xs text-text-muted md:block">
          {syncingNow
            ? "Google 동기화 중"
            : lastSyncedAt
              ? `최근 동기화 ${new Date(lastSyncedAt).toLocaleString("ko-KR")}`
              : "빠른 편집과 계층 정리를 바로 이어서 할 수 있습니다"}
        </p>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon-sm" className="ml-auto shrink-0" title="목록 메뉴">
              <MoreVertical size={14} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onCreateList}>
              <Plus size={14} />
              새 목록
            </DropdownMenuItem>
            {onDeleteCurrentList ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={onDeleteCurrentList} className="text-error focus:text-error">
                  <Trash2 size={14} />
                  현재 목록 삭제
                </DropdownMenuItem>
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full min-w-0 sm:w-56 md:w-64">
          <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted" />
          <Input
            placeholder="검색..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="h-8 w-full pl-8"
          />
        </div>

        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={onManualSync}
            disabled={!onManualSync || syncingNow}
            title="지금 동기화"
          >
            <RefreshCw className={cn("h-4 w-4", syncingNow && "animate-spin")} />
          </Button>

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
        </div>
      </div>
    </PageToolbar>
  );
}

export type { SortBy as TodoSortBy };
