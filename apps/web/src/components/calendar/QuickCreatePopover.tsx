"use client";

import { useState } from "react";
import { Popover, PopoverAnchor, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Plus } from "lucide-react";

interface QuickCreateAnchorPoint {
  x: number;
  y: number;
  side?: "left" | "right";
}

interface QuickCreatePopoverProps {
  defaultStart?: string;
  defaultEnd?: string;
  open?: boolean;
  anchorPoint?: QuickCreateAnchorPoint | null;
  calendarId: string;
  onOpenChange?: (open: boolean) => void;
  onSubmit: (data: {
    title: string;
    start_local: string;
    end_local: string;
    calendar_id: string;
  }) => void;
  onDetail?: (data: {
    title: string;
    start_local: string;
    end_local: string;
    calendar_id: string;
  }) => void;
}

export function QuickCreatePopover({
  defaultStart,
  defaultEnd,
  open,
  anchorPoint,
  calendarId,
  onOpenChange,
  onSubmit,
  onDetail,
}: QuickCreatePopoverProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [startTime, setStartTime] = useState(defaultStart || "");
  const [endTime, setEndTime] = useState(defaultEnd || "");

  const isControlled = typeof open === "boolean";
  const isOpen = isControlled ? open : internalOpen;

  const setOpen = (next: boolean) => {
    if (!isControlled) {
      setInternalOpen(next);
    }
    onOpenChange?.(next);
  };

  const buildPayload = () => ({
    title,
    start_local: startTime,
    end_local: endTime,
    calendar_id: calendarId,
  });

  const handleSubmit = () => {
    if (!title || !startTime || !endTime) return;
    onSubmit(buildPayload());
    setTitle("");
    setStartTime(defaultStart || "");
    setEndTime(defaultEnd || "");
    setOpen(false);
  };

  const handleDetail = () => {
    if (!startTime || !endTime || !onDetail) return;
    onDetail(buildPayload());
    setOpen(false);
  };

  return (
    <Popover open={isOpen} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="primary" size="sm" className="gap-1.5">
          <Plus size={14} />
          일정
        </Button>
      </PopoverTrigger>
      {anchorPoint && (
        <PopoverAnchor asChild>
          <span
            aria-hidden
            className="pointer-events-none fixed h-px w-px"
            style={{ left: `${anchorPoint.x}px`, top: `${anchorPoint.y}px` }}
          />
        </PopoverAnchor>
      )}
      <PopoverContent
        side={anchorPoint?.side || (anchorPoint ? "right" : "bottom")}
        align={anchorPoint ? "start" : "end"}
        sideOffset={8}
        className="w-[22rem] max-w-[calc(100vw-1rem)] space-y-3"
      >
        <Input
          autoFocus
          placeholder="제목"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
        />
        <div className="space-y-2">
          <input
            type="datetime-local"
            value={startTime}
            onChange={(e) => setStartTime(e.target.value)}
            className="h-9 w-full min-w-0 rounded-lg border border-border bg-surface px-2 text-sm outline-none"
          />
          <input
            type="datetime-local"
            value={endTime}
            onChange={(e) => setEndTime(e.target.value)}
            className="h-9 w-full min-w-0 rounded-lg border border-border bg-surface px-2 text-sm outline-none"
          />
        </div>
        <div className="flex justify-end gap-2">
          {onDetail && (
            <Button variant="secondary" size="sm" onClick={handleDetail} disabled={!startTime || !endTime}>
              상세
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>취소</Button>
          <Button variant="primary" size="sm" onClick={handleSubmit}>만들기</Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
