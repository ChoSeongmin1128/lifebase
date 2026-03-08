"use client";

import { useEffect, useRef, useState } from "react";
import { CalendarDays, Clock3, Flag, X } from "lucide-react";
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

const PRIORITY_OPTIONS = [
  {
    value: "urgent",
    label: "긴급",
    className: "border-error/30 bg-error/8 text-error hover:border-error/50 hover:bg-error/12",
    activeClassName: "border-error/60 bg-error/14 text-error",
  },
  {
    value: "high",
    label: "높음",
    className: "border-caution/30 bg-caution/8 text-caution hover:border-caution/50 hover:bg-caution/12",
    activeClassName: "border-caution/60 bg-caution/14 text-caution",
  },
  {
    value: "normal",
    label: "보통",
    className: "border-border bg-background text-text-secondary hover:border-primary/30 hover:bg-surface-accent/60",
    activeClassName: "border-primary/40 bg-surface-accent text-text-primary",
  },
  {
    value: "low",
    label: "낮음",
    className: "border-border bg-background text-text-muted hover:border-primary/25 hover:bg-surface-accent/50",
    activeClassName: "border-border bg-surface-accent text-text-secondary",
  },
] as const;

function formatDateChip(value: string) {
  if (!value) return "날짜 추가";
  const parsed = new Date(`${value}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("ko-KR", {
    month: "short",
    day: "numeric",
    weekday: "short",
  }).format(parsed);
}

function formatTimeChip(value: string) {
  if (!value) return "시간 추가";
  const [hourText = "0", minuteText = "0"] = value.split(":");
  const hour = Number(hourText);
  const minute = Number(minuteText);
  if (Number.isNaN(hour) || Number.isNaN(minute)) return value;
  const period = hour >= 12 ? "오후" : "오전";
  const normalizedHour = hour % 12 || 12;
  return `${period} ${normalizedHour}:${String(minute).padStart(2, "0")}`;
}

function getToday(offsetDays = 0) {
  const next = new Date();
  next.setHours(0, 0, 0, 0);
  next.setDate(next.getDate() + offsetDays);
  const year = String(next.getFullYear()).padStart(4, "0");
  const month = String(next.getMonth() + 1).padStart(2, "0");
  const day = String(next.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
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

  const openDatePicker = () => {
    dateInputRef.current?.focus();
    dateInputRef.current?.showPicker?.();
  };

  const openTimePicker = () => {
    if (!dueDate) return;
    timeInputRef.current?.focus();
    timeInputRef.current?.showPicker?.();
  };

  const applyDueDate = (nextDate: string) => {
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
  };

  const applyDueTime = (nextTime: string) => {
    setDueTime(nextTime);
    void onUpdate({
      due_date: dueDate || "",
      due_time: nextTime || "",
    });
  };

  return (
    <div className={cn("space-y-3 pt-1", className)}>
      <Textarea
        ref={notesRef}
        value={notes}
        rows={1}
        className="min-h-[68px] rounded-xl border border-border/70 bg-background px-3 py-2 text-[13px] leading-5 text-text-secondary shadow-none placeholder:text-text-muted/70 focus-visible:ring-1 focus-visible:ring-primary/30"
        placeholder="세부정보"
        onClick={(event) => event.stopPropagation()}
        onChange={(event) => setNotes(event.target.value)}
        onBlur={() => {
          if (notes !== todo.notes) {
            void onUpdate({ notes });
          }
        }}
      />

      <div className="relative flex flex-wrap items-center gap-2">
        <input
          ref={dateInputRef}
          type="date"
          value={dueDate}
          tabIndex={-1}
          aria-hidden="true"
          className="pointer-events-none absolute left-0 top-0 h-0 w-0 opacity-0"
          onChange={(e) => applyDueDate(e.target.value)}
        />
        <input
          ref={timeInputRef}
          type="time"
          value={dueTime}
          disabled={!dueDate}
          tabIndex={-1}
          aria-hidden="true"
          className="pointer-events-none absolute left-0 top-0 h-0 w-0 opacity-0"
          onChange={(e) => applyDueTime(e.target.value)}
        />

        {listName ? (
          <span className="rounded-full border border-border/70 bg-background px-2.5 py-1 text-[11px] text-text-muted">
            {listName}
          </span>
        ) : null}

        <button
          type="button"
          className="inline-flex min-h-9 items-center gap-2 rounded-full border border-border/80 bg-background px-3 py-2 text-xs text-text-secondary transition-colors hover:border-primary/35 hover:bg-surface-accent/65"
          onClick={(event) => {
            event.stopPropagation();
            openDatePicker();
          }}
        >
          <CalendarDays size={14} className="text-text-muted" />
          <span className={cn("font-medium", !dueDate && "text-text-muted")}>
            {formatDateChip(dueDate)}
          </span>
        </button>

        <button
          type="button"
          className={cn(
            "inline-flex min-h-9 items-center gap-2 rounded-full border px-3 py-2 text-xs transition-colors",
            dueDate
              ? "border-border/80 bg-background text-text-secondary hover:border-primary/35 hover:bg-surface-accent/65"
              : "border-dashed border-border/70 bg-surface-accent/40 text-text-muted"
          )}
          onClick={(event) => {
            event.stopPropagation();
            openTimePicker();
          }}
          disabled={!dueDate}
        >
          <Clock3 size={14} className="text-text-muted" />
          <span className={cn("font-medium", !dueDate && "text-text-muted")}>
            {dueDate ? formatTimeChip(dueTime) : "날짜 먼저"}
          </span>
        </button>

        {(dueDate || dueTime) && (
          <button
            type="button"
            className="inline-flex min-h-9 items-center gap-1.5 rounded-full border border-transparent px-2.5 py-2 text-xs text-text-muted transition-colors hover:border-border/60 hover:bg-surface-accent/65 hover:text-text-secondary"
            onClick={(event) => {
              event.stopPropagation();
              applyDueDate("");
            }}
          >
            <X size={13} />
            일정 제거
          </button>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          className="rounded-full border border-border/70 bg-background px-2.5 py-1.5 text-[11px] font-medium text-text-muted transition-colors hover:border-primary/30 hover:bg-surface-accent/65 hover:text-text-secondary"
          onClick={(event) => {
            event.stopPropagation();
            applyDueDate(getToday(0));
          }}
        >
          오늘
        </button>
        <button
          type="button"
          className="rounded-full border border-border/70 bg-background px-2.5 py-1.5 text-[11px] font-medium text-text-muted transition-colors hover:border-primary/30 hover:bg-surface-accent/65 hover:text-text-secondary"
          onClick={(event) => {
            event.stopPropagation();
            applyDueDate(getToday(1));
          }}
        >
          내일
        </button>
        {dueDate ? (
          <>
            <button
              type="button"
              className="rounded-full border border-border/70 bg-background px-2.5 py-1.5 text-[11px] font-medium text-text-muted transition-colors hover:border-primary/30 hover:bg-surface-accent/65 hover:text-text-secondary"
              onClick={(event) => {
                event.stopPropagation();
                applyDueTime("09:00");
              }}
            >
              오전 9:00
            </button>
            <button
              type="button"
              className="rounded-full border border-border/70 bg-background px-2.5 py-1.5 text-[11px] font-medium text-text-muted transition-colors hover:border-primary/30 hover:bg-surface-accent/65 hover:text-text-secondary"
              onClick={(event) => {
                event.stopPropagation();
                applyDueTime("18:00");
              }}
            >
              오후 6:00
            </button>
            {dueTime ? (
              <button
                type="button"
                className="rounded-full border border-transparent px-2.5 py-1.5 text-[11px] font-medium text-text-muted transition-colors hover:border-border/60 hover:bg-surface-accent/65 hover:text-text-secondary"
                onClick={(event) => {
                  event.stopPropagation();
                  applyDueTime("");
                }}
              >
                시간 제거
              </button>
            ) : null}
          </>
        ) : null}
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center gap-2 px-1 text-[11px] font-medium text-text-muted">
          <Flag size={13} />
          우선순위
        </div>
        <div className="flex flex-wrap gap-1.5">
          {PRIORITY_OPTIONS.map((option) => {
            const active = priority === option.value;
            return (
              <button
                key={option.value}
                type="button"
                className={cn(
                  "rounded-full border px-3 py-1.5 text-xs font-medium transition-colors",
                  active ? option.activeClassName : option.className
                )}
                onClick={(event) => {
                  event.stopPropagation();
                  setPriority(option.value);
                  void onUpdate({ priority: option.value });
                }}
              >
                {option.label}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
