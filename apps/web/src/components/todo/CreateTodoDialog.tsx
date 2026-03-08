"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface CreateTodoDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: {
    title: string;
    dueDate: string | null;
    dueTime: string | null;
    priority: string;
    notes: string;
    parentId?: string;
  }) => void;
  parentId?: string;
  disabled?: boolean;
}

const PRIORITIES = [
  { value: "urgent", label: "긴급", className: "text-error border-error" },
  { value: "high", label: "높음", className: "text-caution border-caution" },
  { value: "normal", label: "보통", className: "text-text-primary border-border" },
  { value: "low", label: "낮음", className: "text-text-muted border-border" },
] as const;

export function CreateTodoDialog({
  open,
  onOpenChange,
  onSubmit,
  parentId,
  disabled,
}: CreateTodoDialogProps) {
  const [title, setTitle] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [dueTime, setDueTime] = useState("");
  const [priority, setPriority] = useState("normal");
  const [notes, setNotes] = useState("");

  const reset = () => {
    setTitle("");
    setDueDate("");
    setDueTime("");
    setPriority("normal");
    setNotes("");
  };

  const handleSubmit = () => {
    if (!title.trim() || disabled) return;
    onSubmit({
      title: title.trim(),
      dueDate: dueDate || null,
      dueTime: dueDate ? (dueTime || null) : null,
      priority,
      notes,
      parentId,
    });
    reset();
  };

  const handleOpenChange = (v: boolean) => {
    if (!v) reset();
    onOpenChange(v);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{parentId ? "하위 Todo 추가" : "새 Todo 추가"}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Input
            autoFocus
            placeholder="제목"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSubmit();
              }
            }}
          />
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="mb-1 block text-xs text-text-muted">마감일</label>
              <Input
                type="date"
                value={dueDate}
                onChange={(e) => {
                  const nextDate = e.target.value;
                  setDueDate(nextDate);
                  if (!nextDate) {
                    setDueTime("");
                  }
                }}
              />
            </div>
            <div className="w-28">
              <label className="mb-1 block text-xs text-text-muted">시간</label>
              <Input
                type="time"
                value={dueTime}
                disabled={!dueDate}
                onChange={(e) => setDueTime(e.target.value)}
              />
            </div>
            <div className="flex-1">
              <label className="mb-1 block text-xs text-text-muted">우선순위</label>
              <div className="flex gap-1">
                {PRIORITIES.map((p) => (
                  <button
                    key={p.value}
                    type="button"
                    onClick={() => setPriority(p.value)}
                    className={cn(
                      "flex-1 rounded-md border px-2 py-1.5 text-xs font-medium transition-colors",
                      priority === p.value
                        ? cn(p.className, "bg-surface-accent")
                        : "border-border text-text-muted hover:bg-surface-accent/50"
                    )}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-xs text-text-muted">메모</label>
            <Textarea
              placeholder="메모 (선택)"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" size="sm" onClick={() => handleOpenChange(false)}>
            취소
          </Button>
          <Button size="sm" onClick={handleSubmit} disabled={!title.trim() || disabled}>
            추가
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
