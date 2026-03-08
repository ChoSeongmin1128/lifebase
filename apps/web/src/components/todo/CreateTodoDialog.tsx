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

interface CreateTodoDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: {
    title: string;
    dueDate: string | null;
    dueTime: string | null;
    notes: string;
    parentId?: string;
  }) => void;
  parentId?: string;
  disabled?: boolean;
}

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
  const [notes, setNotes] = useState("");

  const reset = () => {
    setTitle("");
    setDueDate("");
    setDueTime("");
    setNotes("");
  };

  const handleSubmit = () => {
    if (!title.trim() || disabled) return;
    onSubmit({
      title: title.trim(),
      dueDate: dueDate || null,
      dueTime: dueDate ? (dueTime || null) : null,
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
