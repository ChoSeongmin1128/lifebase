"use client";

import { useEffect, useRef, useState } from "react";
import { CalendarDays, Clock3, Flag } from "lucide-react";
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
  onUpdate: (updates: Record<string, unknown>) => Promise<void>;
}

export function TodoInlineEditor({
  todo,
  listName,
  className,
  onUpdate,
}: TodoInlineEditorProps) {
  const [dueDate, setDueDate] = useState(todo.due_date || "");
  const [dueTime, setDueTime] = useState(todo.due_time || "");
  const [priority, setPriority] = useState(todo.priority);
  const [notes, setNotes] = useState(todo.notes);
  const notesRef = useRef<HTMLTextAreaElement | null>(null);
  const dateInputRef = useRef<HTMLInputElement | null>(null);
  const timeInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!notesRef.current) return;
    notesRef.current.style.height = "0px";
    notesRef.current.style.height = `${notesRef.current.scrollHeight}px`;
  }, [notes]);

  return (
    <div className={cn("space-y-3 pt-1", className)}>
      <Textarea
        ref={notesRef}
        value={notes}
        rows={1}
        className="min-h-0 rounded-none border-0 bg-transparent px-0 py-0 text-xs leading-5 text-text-muted shadow-none placeholder:text-text-muted/70 focus-visible:ring-0"
        placeholder="세부정보"
        onClick={(event) => event.stopPropagation()}
        onChange={(event) => setNotes(event.target.value)}
        onBlur={() => {
          if (notes !== todo.notes) {
            void onUpdate({ notes });
          }
        }}
      />

      <div className="flex flex-wrap items-center gap-2">
        {listName ? (
          <span className="rounded-full bg-background px-2 py-0.5 text-[11px] text-text-muted">
            {listName}
          </span>
        ) : null}

        <div
          className="flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 transition-colors hover:border-primary/40"
          onClick={() => {
            dateInputRef.current?.focus();
            dateInputRef.current?.showPicker?.();
          }}
        >
          <CalendarDays size={13} className="text-text-muted" />
          <Input
            ref={dateInputRef}
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

        <div
          className={cn(
            "flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 transition-colors",
            dueDate ? "hover:border-primary/40" : "opacity-60"
          )}
          onClick={() => {
            if (!dueDate) return;
            timeInputRef.current?.focus();
            timeInputRef.current?.showPicker?.();
          }}
        >
          <Clock3 size={13} className="text-text-muted" />
          <Input
            ref={timeInputRef}
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
    </div>
  );
}
