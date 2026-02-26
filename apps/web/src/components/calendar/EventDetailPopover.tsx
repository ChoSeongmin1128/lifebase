"use client";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent } from "@/components/ui/popover";
import { MapPin, Clock, Trash2 } from "lucide-react";

interface EventData {
  id: string;
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  is_all_day: boolean;
}

interface EventDetailPopoverProps {
  event: EventData;
  open: boolean;
  onClose: () => void;
  onDelete: (id: string) => void;
  color: string;
}

function formatEventTime(e: EventData): string {
  if (e.is_all_day) return "종일";
  const start = new Date(e.start_time);
  const end = new Date(e.end_time);
  const opts: Intl.DateTimeFormatOptions = { month: "long", day: "numeric", hour: "2-digit", minute: "2-digit" };
  return `${start.toLocaleDateString("ko-KR", opts)} - ${end.toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}`;
}

export function EventDetailPopover({ event, open, onClose, onDelete, color }: EventDetailPopoverProps) {
  if (!open) return null;

  return (
    <Popover open={open} onOpenChange={(v) => !v && onClose()}>
      <PopoverContent className="w-80" onPointerDownOutside={onClose}>
        <div className="flex items-start gap-3">
          <div className="mt-1 h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: color }} />
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-medium text-text-strong">{event.title || "(제목 없음)"}</h3>
            <div className="mt-1 flex items-center gap-1.5 text-xs text-text-secondary">
              <Clock size={12} />
              {formatEventTime(event)}
            </div>
            {event.location && (
              <div className="mt-1 flex items-center gap-1.5 text-xs text-text-secondary">
                <MapPin size={12} />
                {event.location}
              </div>
            )}
            {event.description && (
              <p className="mt-2 text-xs text-text-secondary">{event.description}</p>
            )}
          </div>
        </div>
        <div className="mt-3 flex justify-between">
          <Button variant="danger" size="sm" onClick={() => onDelete(event.id)} className="gap-1.5">
            <Trash2 size={12} /> 삭제
          </Button>
          <Button variant="ghost" size="sm" onClick={onClose}>닫기</Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
