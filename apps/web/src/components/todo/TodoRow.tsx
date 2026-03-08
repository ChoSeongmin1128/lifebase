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
  const priorityMeta = (() => {
    if (todo.priority === "urgent") {
      return { label: "긴급", className: "border-error/20 bg-error/8 text-error" };
    }
    if (todo.priority === "high") {
      return { label: "높음", className: "border-caution/20 bg-caution/8 text-caution" };
    }
    if (todo.priority === "low") {
      return { label: "낮음", className: "border-border/70 bg-background text-text-muted" };
    }
    return null;
  })();

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
        "group flex items-start gap-3 rounded-2xl border border-transparent py-2.5 pr-4 transition-[background-color,border-color,box-shadow,opacity]",
        !isOverlay && !isExpanded && "hover:border-border/60 hover:bg-surface/70",
        isExpanded && !isOverlay && "border-border/70 bg-surface shadow-sm",
        todo.is_pinned && !todo.is_done && !isOverlay && !isExpanded && "border-primary/15 bg-primary/[0.04]",
        todo.is_done && "opacity-70",
        isDragging && !isOverlay && "opacity-30",
        isOverlay && "border-border bg-surface shadow-lg opacity-90",
      )}
      {...(isOverlay ? {} : sortable.attributes)}
    >
      {/* Drag handle */}
      {showDragHandle ? (
        <button
          {...(isOverlay ? {} : sortable.listeners)}
          className={cn(
            "mt-0.5 shrink-0 cursor-grab touch-none text-text-muted transition-opacity",
            isOverlay ? "opacity-50" : "opacity-0 group-hover:opacity-40",
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
          className="mt-0.5 shrink-0 rounded-md p-0.5 text-text-muted transition-colors hover:bg-surface-accent hover:text-text-primary"
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
                "block line-clamp-3 break-words text-[14px] font-medium leading-5 text-text-primary",
                todo.is_done && "text-text-muted line-through"
              )}
            >
              {todo.title}
            </span>
          )}
          {todo.notes.trim() && !isExpanded ? (
            <p className="mt-1 line-clamp-1 text-[12px] text-text-muted">
              {todo.notes.trim()}
            </p>
          ) : null}
          {!isExpanded && (listLabel || todo.due_date || priorityMeta || (isCollapsed && childCount && childCount.total > 0)) ? (
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              {listLabel ? (
                <span className="inline-flex items-center rounded-full border border-border/70 bg-background px-2 py-1 text-[11px] text-text-muted">
                  {listLabel}
                </span>
              ) : null}
              {todo.due_date && !todo.is_done ? (
                <span
                  className={cn(
                    "inline-flex items-center rounded-full border px-2 py-1 text-[11px]",
                    isOverdue
                      ? "border-error/20 bg-error/8 font-medium text-error"
                      : "border-border/70 bg-background text-text-muted"
                  )}
                >
                  {formatDueLabel(todo.due_date, todo.due_time)}
                </span>
              ) : null}
              {priorityMeta ? (
                <span className={cn("inline-flex items-center gap-1 rounded-full border px-2 py-1 text-[11px]", priorityMeta.className)}>
                  <Flag size={11} />
                  {priorityMeta.label}
                </span>
              ) : null}
              {isCollapsed && childCount && childCount.total > 0 ? (
                <span className="inline-flex items-center rounded-full border border-border/70 bg-background px-2 py-1 text-[11px] text-text-muted">
                  하위 {childCount.done}/{childCount.total}
                </span>
              ) : null}
            </div>
          ) : null}
          <ExpandableDetails open={Boolean(isExpanded)}>
            {expandedContent}
          </ExpandableDetails>
        </div>
      </div>

      {/* Overlay mode: skip interactive buttons */}
      {isOverlay ? null : (
        <>
          {/* Pin */}
          <button
            onClick={onTogglePin}
            className={cn(
              "mt-0.5 shrink-0 rounded-md p-1 transition-colors",
              todo.is_pinned
                ? "text-primary hover:bg-primary/10"
                : isExpanded
                  ? "text-text-muted hover:bg-surface-accent"
                  : "text-text-muted opacity-0 group-hover:opacity-100 hover:bg-surface-accent"
            )}
          >
            <Pin size={14} fill={todo.is_pinned ? "currentColor" : "none"} />
          </button>

          {/* More menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className={cn(
                  "mt-0.5 shrink-0 rounded-md p-1 text-text-muted transition-colors",
                  isExpanded ? "opacity-100 hover:bg-surface-accent" : "opacity-0 group-hover:opacity-100 hover:bg-surface-accent"
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
