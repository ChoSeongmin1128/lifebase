"use client";

import { useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Plus } from "lucide-react";

interface QuickCreatePopoverProps {
  defaultStart?: string;
  defaultEnd?: string;
  calendarId: string;
  onSubmit: (data: { title: string; start_time: string; end_time: string; calendar_id: string }) => void;
}

export function QuickCreatePopover({
  defaultStart,
  defaultEnd,
  calendarId,
  onSubmit,
}: QuickCreatePopoverProps) {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [startTime, setStartTime] = useState(defaultStart || "");
  const [endTime, setEndTime] = useState(defaultEnd || "");

  const handleSubmit = () => {
    if (!title || !startTime || !endTime) return;
    onSubmit({
      title,
      start_time: startTime + ":00+09:00",
      end_time: endTime + ":00+09:00",
      calendar_id: calendarId,
    });
    setTitle("");
    setStartTime("");
    setEndTime("");
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="primary" size="sm" className="gap-1.5">
          <Plus size={14} />
          일정
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 space-y-3">
        <Input
          autoFocus
          placeholder="제목"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
        />
        <div className="flex gap-2">
          <input
            type="datetime-local"
            value={startTime}
            onChange={(e) => setStartTime(e.target.value)}
            className="flex-1 h-9 rounded-lg border border-border bg-surface px-2 text-sm outline-none"
          />
          <input
            type="datetime-local"
            value={endTime}
            onChange={(e) => setEndTime(e.target.value)}
            className="flex-1 h-9 rounded-lg border border-border bg-surface px-2 text-sm outline-none"
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>취소</Button>
          <Button variant="primary" size="sm" onClick={handleSubmit}>만들기</Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
