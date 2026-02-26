"use client";

import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { ChevronDown } from "lucide-react";

export type CalendarViewMode = "year-compact" | "year-timeline" | "month" | "week" | "3day" | "agenda";

const VIEW_OPTIONS: { value: CalendarViewMode; label: string; group: number }[] = [
  { value: "year-compact", label: "연간 컴팩트", group: 0 },
  { value: "year-timeline", label: "연간 타임라인", group: 0 },
  { value: "month", label: "월간", group: 1 },
  { value: "week", label: "주간", group: 1 },
  { value: "3day", label: "3일", group: 1 },
  { value: "agenda", label: "일정", group: 2 },
];

const VIEW_LABELS: Record<CalendarViewMode, string> = {
  "year-compact": "연간 컴팩트",
  "year-timeline": "연간 타임라인",
  month: "월간",
  week: "주간",
  "3day": "3일",
  agenda: "일정",
};

interface ViewDropdownProps {
  view: CalendarViewMode;
  onChange: (view: CalendarViewMode) => void;
}

export function ViewDropdown({ view, onChange }: ViewDropdownProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="secondary" size="sm" className="gap-1.5">
          {VIEW_LABELS[view]}
          <ChevronDown size={14} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {VIEW_OPTIONS.map((opt, i) => (
          <div key={opt.value}>
            {i > 0 && VIEW_OPTIONS[i - 1].group !== opt.group && <DropdownMenuSeparator />}
            <DropdownMenuItem
              onClick={() => onChange(opt.value)}
              className={view === opt.value ? "font-medium text-primary" : ""}
            >
              {opt.label}
            </DropdownMenuItem>
          </div>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
