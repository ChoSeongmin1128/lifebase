"use client";

import { useState } from "react";
import { CalendarDays, Clock3, Flag, Trash2, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

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

interface TodoInlineEditorProps {
  todo: TodoItem;
  listName?: string;
  className?: string;
  onClose: () => void;
  onDelete: () => void;
  onUpdate: (updates: Record<string, unknown>) => Promise<void>;
}

export function TodoInlineEditor({
  todo,
  listName,
  className,
  onClose,
  onDelete,
  onUpdate,
}: TodoInlineEditorProps) {
  const [dueDate, setDueDate] = useState(todo.due_date || "");
  const [dueTime, setDueTime] = useState(todo.due_time || "");
  const [priority, setPriority] = useState(todo.priority);

  return (
    <div className={cn("space-y-3 border-t border-border/60 pt-3", className)}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          {listName ? (
            <span className="rounded-full bg-background px-2 py-0.5 text-[11px] text-text-muted">
              {listName}
            </span>
          ) : null}
          <span className="text-[11px] text-text-muted">세부 정보</span>
        </div>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={onDelete}
            className="rounded-md p-1 text-text-muted transition-colors hover:bg-background hover:text-error"
            aria-label="Todo 삭제"
          >
            <Trash2 size={14} />
          </button>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-text-muted transition-colors hover:bg-background hover:text-text-primary"
            aria-label="Todo 상세 닫기"
          >
            <X size={14} />
          </button>
        </div>
      </div>

      <Textarea
        defaultValue={todo.title}
        rows={2}
        className="min-h-[72px] resize-none border-0 bg-transparent px-0 text-sm font-medium leading-5 text-text-primary shadow-none focus-visible:ring-0"
        onBlur={(e) => {
          const nextTitle = e.target.value.trim();
          if (nextTitle && nextTitle !== todo.title) {
            void onUpdate({ title: nextTitle });
          } else if (!nextTitle) {
            e.target.value = todo.title;
          }
        }}
      />

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1">
          <CalendarDays size={13} className="text-text-muted" />
          <Input
            type="date"
            value={dueDate}
            className="h-auto min-w-[132px] border-0 bg-transparent px-0 py-0 text-xs shadow-none focus-visible:ring-0"
            onChange={(e) => {
              const nextDate = e.target.value;
              setDueDate(nextDate);
              if (!nextDate) {
                setDueTime("");
                void onUpdate({ due_date: "", due_time: "" });
                return;
              }
              void onUpdate({
                due_date: nextDate,
                due_time: dueTime || "",
              });
            }}
          />
        </div>

        <div className="flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1">
          <Clock3 size={13} className="text-text-muted" />
          <Input
            type="time"
            value={dueTime}
            disabled={!dueDate}
            className="h-auto w-[84px] border-0 bg-transparent px-0 py-0 text-xs shadow-none focus-visible:ring-0"
            onChange={(e) => {
              const nextTime = e.target.value;
              setDueTime(nextTime);
              void onUpdate({
                due_date: dueDate || "",
                due_time: nextTime || "",
              });
            }}
          />
        </div>

        <div className="flex items-center gap-1.5 rounded-full border border-border bg-background px-2 py-0.5">
          <Flag size={13} className="text-text-muted" />
          <Select
            value={priority}
            onValueChange={(value) => {
              setPriority(value);
              void onUpdate({ priority: value });
            }}
          >
            <SelectTrigger className="h-7 w-[104px] border-0 bg-transparent px-0 text-xs shadow-none focus:ring-0">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="urgent">긴급</SelectItem>
              <SelectItem value="high">높음</SelectItem>
              <SelectItem value="normal">보통</SelectItem>
              <SelectItem value="low">낮음</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <Textarea
        defaultValue={todo.notes}
        rows={3}
        className="min-h-[96px] resize-none border-0 bg-background/70 text-sm shadow-none focus-visible:ring-1"
        placeholder="세부 설명 추가"
        onBlur={(e) => {
          if (e.target.value !== todo.notes) {
            void onUpdate({ notes: e.target.value });
          }
        }}
      />
    </div>
  );
}
