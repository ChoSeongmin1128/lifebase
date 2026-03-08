"use client";

import { type ReactNode, useEffect, useRef, useState } from "react";
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
import { Textarea } from "@/components/ui/textarea";
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
import { formatDueLabel } from "@/features/todo/lib/formatDueDate";

interface TodoItem {
  id: string;
  list_id: string;
  parent_id: string | null;
  title: string;
  notes: string;
  due_date: string | null;
  due_time: string | null;
  priority: string;
  is_done: boolean;
  is_pinned: boolean;
  starred_at?: string | null;
  sort_order: number;
  done_at: string | null;
  created_at: string;
  updated_at?: string;
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
  isExpanded?: boolean;
  lists?: TodoList[];
  onToggleCollapse?: () => void;
  onToggleDone: () => void;
  onTogglePin: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onChangePriority: (priority: string) => void;
  onUpdateTitle?: (title: string) => void;
  expandedContent?: ReactNode;
  onAddSubtask?: () => void;
  onMoveToList?: (listId: string) => void;
}

function ExpandedTitleEditor({
  title,
  isDone,
  onUpdateTitle,
}: {
  title: string;
  isDone: boolean;
  onUpdateTitle?: (title: string) => void;
}) {
  const [draftTitle, setDraftTitle] = useState(title);
  const titleInputRef = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    if (!titleInputRef.current) return;
    titleInputRef.current.style.height = "0px";
    titleInputRef.current.style.height = `${titleInputRef.current.scrollHeight}px`;
  }, [draftTitle]);

  return (
    <Textarea
      ref={titleInputRef}
      value={draftTitle}
      rows={1}
      autoFocus
      className={cn(
        "min-h-0 resize-none border-0 bg-transparent px-0 py-0 text-sm leading-5 text-text-primary shadow-none focus-visible:ring-0",
        isDone && "text-text-muted line-through"
      )}
      onClick={(event) => event.stopPropagation()}
      onChange={(event) => setDraftTitle(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          setDraftTitle(title);
          event.currentTarget.blur();
        }
      }}
      onBlur={() => {
        const nextTitle = draftTitle.trim();
        if (!nextTitle) {
          setDraftTitle(title);
          return;
        }
        if (nextTitle !== title) {
          onUpdateTitle?.(nextTitle);
        }
        if (nextTitle !== draftTitle) {
          setDraftTitle(nextTitle);
        }
      }}
    />
  );
}

function ExpandableDetails({
  open,
  children,
}: {
  open: boolean;
  children?: ReactNode;
}) {
  if (!children) return null;

  return (
    <div
      className={cn(
        "mt-2 grid overflow-hidden transition-[grid-template-rows,opacity,transform] duration-220 ease-[cubic-bezier(0.22,1,0.36,1)]",
        open
          ? "grid-rows-[1fr] opacity-100 translate-y-0"
          : "pointer-events-none grid-rows-[0fr] opacity-0 -translate-y-1"
      )}
    >
      <div className="min-h-0 overflow-hidden">
        <div>
          {children}
        </div>
      </div>
    </div>
  );
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
  isExpanded,
  lists,
  onToggleCollapse,
  onToggleDone,
  onTogglePin,
  onEdit,
  onDelete,
  onChangePriority,
  onUpdateTitle,
  expandedContent,
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
      data-todo-row-id={todo.id}
      className={cn(
        "group flex items-start gap-2 py-2 pr-4 transition-colors",
        !isOverlay && !isExpanded && "hover:bg-surface-accent/50",
        isExpanded && !isOverlay && "bg-transparent",
        todo.is_pinned && !todo.is_done && !isOverlay && !isExpanded && "bg-surface-accent",
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
      <div
        className={cn("min-w-0 flex-1", isExpanded ? "cursor-text" : "cursor-pointer")}
        onClick={isExpanded ? undefined : onEdit}
      >
        <div className="min-w-0">
          {isExpanded ? (
            <ExpandedTitleEditor
              key={`${todo.id}:${todo.title}`}
              title={todo.title}
              isDone={todo.is_done}
              onUpdateTitle={onUpdateTitle}
            />
          ) : (
            <span
              className={cn(
                "block line-clamp-3 break-words text-sm leading-5 text-text-primary",
                todo.is_done && "text-text-muted line-through"
              )}
            >
              {todo.title}
            </span>
          )}
          {todo.notes.trim() && !isExpanded ? (
            <p className="mt-0.5 line-clamp-1 text-xs text-text-muted">
              {todo.notes.trim()}
            </p>
          ) : null}
          <ExpandableDetails open={Boolean(isExpanded)}>
            {expandedContent}
          </ExpandableDetails>
        </div>
        {/* Child count badge when collapsed */}
        {!isExpanded && isCollapsed && childCount && childCount.total > 0 && (
          <span className="ml-2 inline-flex items-center rounded-full bg-surface-accent px-1.5 py-0.5 text-[10px] text-text-muted">
            {childCount.done}/{childCount.total}
          </span>
        )}
      </div>

      {/* Due badge */}
      {!isExpanded && listLabel && (
        <span className="shrink-0 rounded-full bg-surface-accent px-1.5 py-0.5 text-[10px] text-text-muted">
          {listLabel}
        </span>
      )}
      {!isExpanded && todo.due_date && !todo.is_done && (
        <span
          className={cn(
            "shrink-0 text-[11px]",
            isOverdue ? "text-error font-medium" : "text-text-muted"
          )}
        >
          {formatDueLabel(todo.due_date, todo.due_time)}
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
              todo.is_pinned
                ? "text-primary"
                : isExpanded
                  ? "text-text-muted"
                  : "text-text-muted opacity-0 group-hover:opacity-100"
            )}
          >
            <Pin size={14} fill={todo.is_pinned ? "currentColor" : "none"} />
          </button>

          {/* More menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className={cn(
                  "shrink-0 text-text-muted transition-opacity",
                  isExpanded ? "opacity-100" : "opacity-0 group-hover:opacity-100"
                )}
              >
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
