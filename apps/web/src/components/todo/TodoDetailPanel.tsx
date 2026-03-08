"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
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

interface TodoDetailPanelProps {
  todo: TodoItem;
  listName?: string;
  className?: string;
  onClose: () => void;
  onDelete: () => void;
  onUpdate: (updates: Record<string, unknown>) => Promise<void>;
}

export function TodoDetailPanel({
  todo,
  listName,
  className,
  onClose,
  onDelete,
  onUpdate,
}: TodoDetailPanelProps) {
  const [title, setTitle] = useState(todo.title);
  const [notes, setNotes] = useState(todo.notes);
  const [dueDate, setDueDate] = useState(todo.due_date || "");
  const [dueTime, setDueTime] = useState(todo.due_time || "");
  const [priority, setPriority] = useState(todo.priority);

  return (
    <aside className={cn("flex h-full flex-col border-l border-border bg-surface/40", className)}>
      <div className="flex items-start justify-between gap-3 border-b border-border px-4 py-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-text-strong">Todo 상세</p>
          {listName ? <p className="mt-1 text-xs text-text-muted">{listName}</p> : null}
        </div>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg p-1 text-text-muted transition-colors hover:bg-surface-accent hover:text-text-primary"
          aria-label="Todo 상세 닫기"
        >
          <X size={16} />
        </button>
      </div>

      <div className="flex-1 space-y-4 overflow-auto px-4 py-4">
        <div>
          <label className="mb-1.5 block text-xs text-text-muted">제목</label>
          <Textarea
            value={title}
            rows={3}
            className="min-h-[88px] resize-none"
            onChange={(e) => setTitle(e.target.value)}
            onBlur={() => {
              const nextTitle = title.trim();
              if (nextTitle && nextTitle !== todo.title) {
                void onUpdate({ title: nextTitle });
              } else if (!nextTitle) {
                setTitle(todo.title);
              }
            }}
          />
        </div>

        <div>
          <label className="mb-1.5 block text-xs text-text-muted">메모</label>
          <Textarea
            value={notes}
            rows={8}
            className="min-h-[180px] resize-y"
            placeholder="메모"
            onChange={(e) => setNotes(e.target.value)}
            onBlur={() => {
              if (notes !== todo.notes) {
                void onUpdate({ notes });
              }
            }}
          />
        </div>

        <div className="grid gap-3">
          <div>
            <label className="mb-1.5 block text-xs text-text-muted">기한 날짜</label>
            <Input
              type="date"
              value={dueDate}
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

          <div>
            <label className="mb-1.5 block text-xs text-text-muted">시간</label>
            <Input
              type="time"
              value={dueTime}
              disabled={!dueDate}
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

          <div>
            <label className="mb-1.5 block text-xs text-text-muted">우선순위</label>
            <Select
              value={priority}
              onValueChange={(value) => {
                setPriority(value);
                void onUpdate({ priority: value });
              }}
            >
              <SelectTrigger className="h-9">
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
      </div>

      <div className="flex items-center justify-between border-t border-border px-4 py-3">
        <Button variant="danger" size="sm" onClick={onDelete}>
          삭제
        </Button>
        <Button variant="ghost" size="sm" onClick={onClose}>
          닫기
        </Button>
      </div>
    </aside>
  );
}
