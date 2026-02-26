"use client";

import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { PriorityFlag } from "./PriorityFlag";
import { MoreVertical, Pencil, Flag, CalendarDays, Trash2, GripVertical, Pin } from "lucide-react";
import { cn } from "@/lib/utils";

interface TodoItem {
  id: string;
  list_id: string;
  parent_id: string | null;
  title: string;
  notes: string;
  due: string | null;
  priority: string;
  is_done: boolean;
  is_pinned: boolean;
  sort_order: number;
  done_at: string | null;
  created_at: string;
}

interface TodoRowProps {
  todo: TodoItem;
  isOverdue: boolean;
  onToggleDone: () => void;
  onTogglePin: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onChangePriority: (priority: string) => void;
}

export function TodoRow({
  todo,
  isOverdue,
  onToggleDone,
  onTogglePin,
  onEdit,
  onDelete,
  onChangePriority,
}: TodoRowProps) {
  return (
    <div
      className={cn(
        "group flex items-center gap-2 px-4 py-2 hover:bg-surface-accent/50 transition-colors",
        todo.parent_id && "pl-10",
        todo.is_pinned && !todo.is_done && "bg-surface-accent",
        todo.is_done && "opacity-60"
      )}
    >
      {/* Drag handle */}
      <GripVertical size={14} className="shrink-0 text-text-muted opacity-0 group-hover:opacity-50 cursor-grab" />

      {/* Checkbox */}
      <Checkbox
        checked={todo.is_done}
        onCheckedChange={onToggleDone}
      />

      {/* Priority flag */}
      <PriorityFlag priority={todo.priority} />

      {/* Content */}
      <div className="min-w-0 flex-1 cursor-pointer" onClick={onEdit}>
        <span
          className={cn(
            "text-sm text-text-primary",
            todo.is_done && "text-text-muted line-through"
          )}
        >
          {todo.title}
        </span>
      </div>

      {/* Due badge */}
      {todo.due && !todo.is_done && (
        <span
          className={cn(
            "shrink-0 text-[11px]",
            isOverdue ? "text-error font-medium" : "text-text-muted"
          )}
        >
          {new Date(todo.due).toLocaleDateString("ko-KR", { month: "numeric", day: "numeric" })}
        </span>
      )}

      {/* Pin */}
      <button
        onClick={onTogglePin}
        className={cn(
          "shrink-0 transition-opacity",
          todo.is_pinned ? "text-primary" : "text-text-muted opacity-0 group-hover:opacity-100"
        )}
      >
        <Pin size={14} fill={todo.is_pinned ? "currentColor" : "none"} />
      </button>

      {/* More menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button className="shrink-0 text-text-muted opacity-0 group-hover:opacity-100 transition-opacity">
            <MoreVertical size={14} />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={onEdit}>
            <Pencil size={14} /> 수정
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => onChangePriority("urgent")} className="text-error">
            <Flag size={14} /> 긴급
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onChangePriority("high")} className="text-caution">
            <Flag size={14} /> 높음
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onChangePriority("normal")}>
            <Flag size={14} /> 보통
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onChangePriority("low")} className="text-text-muted">
            <Flag size={14} /> 낮음
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem>
            <CalendarDays size={14} /> 마감일 설정
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={onDelete} className="text-error focus:text-error">
            <Trash2 size={14} /> 삭제
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
