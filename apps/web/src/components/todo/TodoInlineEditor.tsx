"use client";

import { useEffect, useMemo, useState } from "react";
import {
  CalendarDays,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Clock3,
  Flag,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { buildFixedMonthGrid } from "@/features/calendar/domain/month-grid";
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
    chipClassName: "border-error/30 bg-error/8 text-error",
  },
  {
    value: "high",
    label: "높음",
    chipClassName: "border-caution/30 bg-caution/8 text-caution",
  },
  {
    value: "normal",
    label: "보통",
    chipClassName: "border-border/80 bg-background text-text-secondary",
  },
  {
    value: "low",
    label: "낮음",
    chipClassName: "border-border/80 bg-background text-text-muted",
  },
] as const;

const WEEKDAY_LABELS = ["일", "월", "화", "수", "목", "금", "토"] as const;
const TIME_OPTIONS = Array.from({ length: 24 * 2 }, (_, index) => {
  const hour = Math.floor(index / 2);
  const minute = (index % 2) * 30;
  return `${pad2(hour)}:${pad2(minute)}`;
});

function toDateKey(date: Date) {
  const year = String(date.getFullYear()).padStart(4, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function getToday(offsetDays = 0) {
  const next = new Date();
  next.setHours(0, 0, 0, 0);
  next.setDate(next.getDate() + offsetDays);
  return toDateKey(next);
}

function parseDateKey(value: string) {
  if (!value) return new Date();
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, (month || 1) - 1, day || 1, 0, 0, 0, 0);
}

function formatMonthLabel(value: Date) {
  return new Intl.DateTimeFormat("ko-KR", {
    year: "numeric",
    month: "long",
  }).format(value);
}

function formatDateChip(value: string) {
  if (!value) return "날짜 추가";
  const parsed = parseDateKey(value);
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

function parseFlexibleTimeInput(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return "";

  if (trimmed.includes(":")) {
    const [hourText = "", minuteText = ""] = trimmed.split(":");
    const hour = Number(hourText);
    const minute = Number(minuteText);
    if (Number.isNaN(hour) || Number.isNaN(minute)) return null;
    if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null;
    return `${pad2(hour)}:${pad2(minute)}`;
  }

  const digits = trimmed.replace(/\D/g, "");
  if (!digits) return null;

  if (digits.length <= 2) {
    const hour = Number(digits);
    if (Number.isNaN(hour) || hour < 0 || hour > 23) return null;
    return `${pad2(hour)}:00`;
  }

  const normalizedDigits = digits.slice(0, 4);
  const hourText = normalizedDigits.length === 3 ? normalizedDigits.slice(0, 1) : normalizedDigits.slice(0, 2);
  const minuteText = normalizedDigits.length === 3 ? normalizedDigits.slice(1, 3) : normalizedDigits.slice(2, 4);
  const hour = Number(hourText);
  const minute = Number(minuteText);
  if (Number.isNaN(hour) || Number.isNaN(minute)) return null;
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null;
  return `${pad2(hour)}:${pad2(minute)}`;
}

function pad2(value: number) {
  return String(value).padStart(2, "0");
}

function isNestedRadixLayer(target: EventTarget | null) {
  return target instanceof HTMLElement && Boolean(target.closest("[data-radix-popper-content-wrapper]"));
}

function isTodoEditLayer(target: EventTarget | null) {
  return target instanceof HTMLElement && Boolean(target.closest('[data-todo-edit-layer="true"]'));
}

function OptionRow({
  selected,
  label,
  tone,
  onClick,
}: {
  selected: boolean;
  label: string;
  tone?: "urgent" | "high";
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-surface-accent",
        tone === "urgent" && "text-error",
        tone === "high" && "text-caution",
        !tone && "text-text-primary"
      )}
      onClick={onClick}
    >
      <span>{label}</span>
      <Check size={14} className={cn("text-primary transition-opacity", selected ? "opacity-100" : "opacity-0")} />
    </button>
  );
}

function ChipButton({
  icon,
  label,
  muted,
  className,
}: {
  icon: React.ReactNode;
  label: string;
  muted?: boolean;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "inline-flex min-h-9 items-center gap-2 rounded-xl border px-3 py-2 text-xs transition-colors",
        muted
          ? "border-dashed border-border/70 bg-surface-accent/35 text-text-muted"
          : "border-border/80 bg-background text-text-secondary hover:border-primary/25 hover:bg-surface-accent/55",
        className
      )}
    >
      {icon}
      <span className={cn("font-medium", muted && "text-text-muted")}>{label}</span>
      <ChevronDown size={13} className="text-text-muted" />
    </div>
  );
}

function TodoDatePicker({
  value,
  onChange,
  onClose,
}: {
  value: string;
  onChange: (nextDate: string) => void;
  onClose: () => void;
}) {
  const [viewDate, setViewDate] = useState(() => {
    const base = value ? parseDateKey(value) : new Date();
    return new Date(base.getFullYear(), base.getMonth(), 1);
  });

  const cells = useMemo(() => buildFixedMonthGrid(viewDate), [viewDate]);
  const todayKey = getToday(0);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between px-1">
        <button
          type="button"
          className="inline-flex h-8 w-8 items-center justify-center rounded-full text-text-muted transition-colors hover:bg-surface-accent hover:text-text-primary"
          onClick={() => {
            setViewDate((prev) => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
          }}
        >
          <ChevronLeft size={16} />
        </button>
        <div className="text-sm font-semibold text-text-primary">{formatMonthLabel(viewDate)}</div>
        <button
          type="button"
          className="inline-flex h-8 w-8 items-center justify-center rounded-full text-text-muted transition-colors hover:bg-surface-accent hover:text-text-primary"
          onClick={() => {
            setViewDate((prev) => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
          }}
        >
          <ChevronRight size={16} />
        </button>
      </div>

      <div className="grid grid-cols-7 gap-y-1 px-1">
        {WEEKDAY_LABELS.map((label) => (
          <div key={label} className="pb-1 text-center text-[11px] font-medium text-text-muted">
            {label}
          </div>
        ))}
        {cells.map((cell) => {
          const cellKey = cell.dateKey;
          const isSelected = value === cellKey;
          const isToday = todayKey === cellKey;
          return (
            <button
              key={cellKey}
              type="button"
              className={cn(
                "mx-auto inline-flex h-9 w-9 items-center justify-center rounded-xl text-sm transition-colors",
                isSelected
                  ? "bg-primary text-white shadow-sm"
                  : isToday
                    ? "border border-primary/35 bg-primary/6 text-primary"
                    : cell.inCurrentMonth
                      ? "text-text-primary hover:bg-surface-accent"
                      : "text-text-muted/55 hover:bg-surface-accent/60"
              )}
              onClick={() => {
                onChange(cellKey);
                onClose();
              }}
            >
              {cell.day}
            </button>
          );
        })}
      </div>

      <div className="flex items-center justify-between px-1 pt-1">
        <button
          type="button"
          className="text-sm text-primary transition-opacity hover:opacity-75"
          onClick={() => {
            onChange("");
            onClose();
          }}
        >
          삭제
        </button>
        <button
          type="button"
          className="text-sm text-primary transition-opacity hover:opacity-75"
          onClick={() => {
            onChange(getToday(0));
            onClose();
          }}
        >
          오늘
        </button>
      </div>
    </div>
  );
}

function TodoTimePicker({
  value,
  onChange,
  onClose,
}: {
  value: string;
  onChange: (nextTime: string) => void;
  onClose: () => void;
}) {
  const [timeDraft, setTimeDraft] = useState(() => value || "");
  const normalizedDraft = useMemo(() => parseFlexibleTimeInput(timeDraft), [timeDraft]);

  const applySelectedTime = (nextTime?: string) => {
    const normalizedTime = nextTime ?? normalizedDraft;
    if (!normalizedTime) return;
    onChange(normalizedTime);
    onClose();
  };

  return (
    <div className="space-y-3">
      <div className="rounded-2xl border border-border/70 bg-surface/60 p-3">
        <div className="flex items-center gap-2">
          <Input
            value={timeDraft}
            placeholder="09:47"
            className="h-10 flex-1 rounded-xl bg-background"
            onChange={(event) => {
              setTimeDraft(event.target.value);
            }}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                applySelectedTime();
              }
            }}
          />
          <Button
            type="button"
            size="sm"
            variant="secondary"
            className="h-10 rounded-xl px-3"
            disabled={!normalizedDraft}
            onClick={() => applySelectedTime()}
          >
            적용
          </Button>
        </div>

        <ScrollArea className="mt-3 h-52 rounded-xl border border-border/60 bg-background/70">
          <div className="p-1.5" data-todo-edit-layer="true" onPointerDown={(event) => event.stopPropagation()}>
            {TIME_OPTIONS.map((option) => {
              const selected = (normalizedDraft || value) === option;
              return (
                <button
                  key={option}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-surface-accent",
                    selected ? "bg-primary/8 text-primary" : "text-text-secondary"
                  )}
                  onClick={() => {
                    setTimeDraft(option);
                    applySelectedTime(option);
                  }}
                >
                  <span>{formatTimeChip(option)}</span>
                  <span className="text-[11px] text-text-muted">{option}</span>
                </button>
              );
            })}
          </div>
        </ScrollArea>
      </div>

      <div className="flex justify-start">
        <button
          type="button"
          className="text-sm text-error transition-opacity hover:opacity-75"
          onClick={() => {
            onChange("");
            onClose();
          }}
        >
          삭제
        </button>
      </div>
    </div>
  );
}

export function TodoInlineEditor({
  todo,
  listName,
  className,
  onUpdate,
}: TodoInlineEditorProps) {
  const [dueDate, setDueDate] = useState(() => todo.due_date || "");
  const [dueTime, setDueTime] = useState(() => todo.due_time || "");
  const [priority, setPriority] = useState(() => todo.priority);
  const [notes, setNotes] = useState(() => todo.notes);
  const [dateOpen, setDateOpen] = useState(false);
  const [timeOpen, setTimeOpen] = useState(false);
  const [priorityOpen, setPriorityOpen] = useState(false);

  const overlayOpen = dateOpen || timeOpen || priorityOpen;

  useEffect(() => {
    if (overlayOpen) {
      document.body.dataset.todoEditOverlayOpen = "true";
      return () => {
        if (document.body.dataset.todoEditOverlayOpen === "true") {
          delete document.body.dataset.todoEditOverlayOpen;
        }
      };
    }

    if (document.body.dataset.todoEditOverlayOpen === "true") {
      delete document.body.dataset.todoEditOverlayOpen;
    }
    return undefined;
  }, [overlayOpen]);

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

  const applyPriority = (nextPriority: string) => {
    setPriority(nextPriority);
    void onUpdate({ priority: nextPriority });
    setPriorityOpen(false);
  };

  const selectedPriority =
    PRIORITY_OPTIONS.find((option) => option.value === priority) ?? PRIORITY_OPTIONS[2];

  return (
    <div className={cn("mt-3 space-y-3 rounded-2xl border border-border/70 bg-background/90 p-3 shadow-sm", className)}>
      <Textarea
        value={notes}
        rows={1}
        className="min-h-[76px] rounded-xl border border-border/60 bg-surface/70 px-3 py-2 text-[13px] leading-5 text-text-secondary shadow-none placeholder:text-text-muted/70 focus-visible:ring-1 focus-visible:ring-primary/30"
        placeholder="메모"
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
          <span className="rounded-xl border border-border/70 bg-surface/60 px-2.5 py-1 text-[11px] text-text-muted">
            {listName}
          </span>
        ) : null}

        <Popover open={dateOpen} onOpenChange={setDateOpen}>
          <PopoverTrigger asChild>
            <button type="button" onClick={(event) => event.stopPropagation()}>
              <ChipButton
                icon={<CalendarDays size={14} className="text-text-muted" />}
                label={formatDateChip(dueDate)}
                muted={!dueDate}
              />
            </button>
          </PopoverTrigger>
          <PopoverContent
            align="start"
            className="w-[20rem] rounded-2xl border-border/80 p-3"
            data-todo-edit-layer="true"
            onPointerDown={(event) => event.stopPropagation()}
            onInteractOutside={(event) => {
              if (isTodoEditLayer(event.target) || isNestedRadixLayer(event.target)) {
                event.preventDefault();
              }
            }}
          >
            <TodoDatePicker
              value={dueDate}
              onChange={applyDueDate}
              onClose={() => setDateOpen(false)}
            />
          </PopoverContent>
        </Popover>

        <Popover open={timeOpen} onOpenChange={setTimeOpen}>
          <PopoverTrigger asChild>
            <button
              type="button"
              disabled={!dueDate}
              onClick={(event) => event.stopPropagation()}
            >
              <ChipButton
                icon={<Clock3 size={14} className="text-text-muted" />}
                label={dueDate ? formatTimeChip(dueTime) : "시간 추가"}
                muted={!dueDate}
              />
            </button>
          </PopoverTrigger>
          <PopoverContent
            align="start"
            className="w-[18rem] rounded-2xl border-border/80 p-3"
            data-todo-edit-layer="true"
            onPointerDown={(event) => event.stopPropagation()}
            onInteractOutside={(event) => {
              if (isTodoEditLayer(event.target) || isNestedRadixLayer(event.target)) {
                event.preventDefault();
              }
            }}
          >
            <TodoTimePicker
              value={dueTime}
              onChange={applyDueTime}
              onClose={() => setTimeOpen(false)}
            />
          </PopoverContent>
        </Popover>

        <Popover open={priorityOpen} onOpenChange={setPriorityOpen}>
          <PopoverTrigger asChild>
            <button type="button" onClick={(event) => event.stopPropagation()}>
              <ChipButton
                icon={<Flag size={14} className="text-text-muted" />}
                label={selectedPriority.label}
                className={selectedPriority.chipClassName}
              />
            </button>
          </PopoverTrigger>
          <PopoverContent
            align="start"
            className="w-48 rounded-2xl border-border/80 p-2"
            data-todo-edit-layer="true"
            onPointerDown={(event) => event.stopPropagation()}
            onInteractOutside={(event) => {
              if (isTodoEditLayer(event.target) || isNestedRadixLayer(event.target)) {
                event.preventDefault();
              }
            }}
          >
            <div className="space-y-1">
              <OptionRow selected={priority === "urgent"} label="긴급" tone="urgent" onClick={() => applyPriority("urgent")} />
              <OptionRow selected={priority === "high"} label="높음" tone="high" onClick={() => applyPriority("high")} />
              <OptionRow selected={priority === "normal"} label="보통" onClick={() => applyPriority("normal")} />
              <OptionRow selected={priority === "low"} label="낮음" onClick={() => applyPriority("low")} />
            </div>
          </PopoverContent>
        </Popover>

        {(dueDate || dueTime) ? (
          <button
            type="button"
            className="inline-flex min-h-9 items-center gap-1.5 rounded-xl border border-transparent px-2.5 py-2 text-xs text-text-muted transition-colors hover:border-border/60 hover:bg-surface-accent/55 hover:text-text-secondary"
            onClick={(event) => {
              event.stopPropagation();
              applyDueDate("");
            }}
          >
            <X size={13} />
            일정 제거
          </button>
        ) : null}
      </div>
    </div>
  );
}
