"use client";

import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
} from "@/components/ui/dropdown-menu";
import { PriorityFlag } from "./PriorityFlag";
import {
  MoreVertical,
  Pencil,
  Flag,
  Trash2,
  GripVertical,
  Pin,
  ChevronRight,
  ChevronDown,
  Plus,
  ArrowRightLeft,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { formatDueYYMMDD } from "@/features/todo/lib/formatDueDate";

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

interface TodoList {
  id: string;
  name: string;
}

interface TodoRowProps {
  todo: TodoItem;
  listLabel?: string;
  depth?: number;
  isOverdue: boolean;
  hasChildren?: boolean;
  isCollapsed?: boolean;
  childCount?: { total: number; done: number };
  showDragHandle?: boolean;
  isDragging?: boolean;
  isOverlay?: boolean;
  lists?: TodoList[];
  onToggleCollapse?: () => void;
  onToggleDone: () => void;
  onTogglePin: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onChangePriority: (priority: string) => void;
  onAddSubtask?: () => void;
  onMoveToList?: (listId: string) => void;
}

export function TodoRow({
  todo,
  listLabel,
  depth = 0,
  isOverdue,
  hasChildren,
  isCollapsed,
  childCount,
  showDragHandle,
  isDragging,
  isOverlay,
  lists,
  onToggleCollapse,
  onToggleDone,
  onTogglePin,
  onEdit,
  onDelete,
  onChangePriority,
  onAddSubtask,
  onMoveToList,
}: TodoRowProps) {
  const sortable = useSortable({ id: todo.id, disabled: !showDragHandle || isOverlay });

  const style = isOverlay
    ? { paddingLeft: `${depth * 24 + 16}px` }
    : {
        transform: CSS.Transform.toString(sortable.transform),
        transition: sortable.transition,
        paddingLeft: `${depth * 24 + 16}px`,
      };

  return (
    <div
      ref={isOverlay ? undefined : sortable.setNodeRef}
      style={style}
      className={cn(
        "group flex items-center gap-2 py-2 pr-4 transition-colors",
        !isOverlay && "hover:bg-surface-accent/50",
        todo.is_pinned && !todo.is_done && !isOverlay && "bg-surface-accent",
        todo.is_done && "opacity-60",
        isDragging && !isOverlay && "opacity-30",
        isOverlay && "rounded-lg bg-surface shadow-lg border border-border opacity-90",
      )}
      {...(isOverlay ? {} : sortable.attributes)}
    >
      {/* Drag handle */}
      {showDragHandle ? (
        <button
          {...(isOverlay ? {} : sortable.listeners)}
          className={cn(
            "shrink-0 text-text-muted cursor-grab touch-none",
            isOverlay ? "opacity-50" : "opacity-0 group-hover:opacity-50",
          )}
        >
          <GripVertical size={14} />
        </button>
      ) : (
        <div className="w-[14px] shrink-0" />
      )}

      {/* Collapse chevron (for parents) */}
      {hasChildren ? (
        <button
          onClick={onToggleCollapse}
          className="shrink-0 text-text-muted hover:text-text-primary transition-colors"
        >
          {isCollapsed ? <ChevronRight size={14} /> : <ChevronDown size={14} />}
        </button>
      ) : (
        <div className="w-[14px] shrink-0" />
      )}

      {/* Checkbox */}
      <Checkbox
        checked={todo.is_done}
        onCheckedChange={onToggleDone}
        className="mt-0.5 self-start"
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
        {/* Child count badge when collapsed */}
        {isCollapsed && childCount && childCount.total > 0 && (
          <span className="ml-2 inline-flex items-center rounded-full bg-surface-accent px-1.5 py-0.5 text-[10px] text-text-muted">
            {childCount.done}/{childCount.total}
          </span>
        )}
      </div>

      {/* Due badge */}
      {listLabel && (
        <span className="shrink-0 rounded-full bg-surface-accent px-1.5 py-0.5 text-[10px] text-text-muted">
          {listLabel}
        </span>
      )}
      {todo.due && !todo.is_done && (
        <span
          className={cn(
            "shrink-0 text-[11px]",
            isOverdue ? "text-error font-medium" : "text-text-muted"
          )}
        >
          {formatDueYYMMDD(todo.due)}
        </span>
      )}

      {/* Overlay mode: skip interactive buttons */}
      {isOverlay ? null : (
        <>
          {/* Add subtask button */}
          {onAddSubtask && !todo.is_done && (
            <button
              onClick={onAddSubtask}
              className="shrink-0 text-text-muted opacity-0 group-hover:opacity-100 transition-opacity hover:text-primary"
              title="하위 Todo 추가"
            >
              <Plus size={14} />
            </button>
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
              {onAddSubtask && (
                <DropdownMenuItem onClick={onAddSubtask}>
                  <Plus size={14} /> 하위 Todo 추가
                </DropdownMenuItem>
              )}
              {onMoveToList && lists && lists.length > 1 && (
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <ArrowRightLeft size={14} /> 다른 목록으로 이동
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    {lists
                      .filter((l) => l.id !== todo.list_id)
                      .map((l) => (
                        <DropdownMenuItem key={l.id} onClick={() => onMoveToList(l.id)}>
                          {l.name}
                        </DropdownMenuItem>
                      ))}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              )}
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
              <DropdownMenuItem onClick={onDelete} className="text-error focus:text-error">
                <Trash2 size={14} /> 삭제
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </>
      )}
    </div>
  );
}
