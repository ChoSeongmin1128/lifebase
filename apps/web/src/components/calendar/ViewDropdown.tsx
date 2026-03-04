"use client";

import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { CalendarDays, Check, ChevronDown, Columns2, Columns3, LayoutGrid, List, Rows3, type LucideIcon } from "lucide-react";

export type CalendarViewMode = "year-compact" | "year-timeline" | "month" | "week" | "3day" | "agenda";

const VIEW_OPTIONS: { value: CalendarViewMode; label: string; group: number; icon: LucideIcon }[] = [
  { value: "year-compact", label: "연간", group: 0, icon: LayoutGrid },
  { value: "year-timeline", label: "타임라인", group: 0, icon: Rows3 },
  { value: "month", label: "월간", group: 1, icon: CalendarDays },
  { value: "week", label: "주간", group: 1, icon: Columns2 },
  { value: "3day", label: "3일", group: 1, icon: Columns3 },
  { value: "agenda", label: "일정", group: 2, icon: List },
];

const VIEW_LABELS: Record<CalendarViewMode, string> = {
  "year-compact": "연간",
  "year-timeline": "타임라인",
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
  const selectedOption = VIEW_OPTIONS.find((option) => option.value === view) || VIEW_OPTIONS[0];
  const SelectedIcon = selectedOption.icon;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="secondary" size="sm" className="gap-1.5">
          <SelectedIcon size={14} />
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
              <opt.icon size={14} />
              {opt.label}
              {view === opt.value ? <Check size={14} className="ml-auto" /> : null}
            </DropdownMenuItem>
          </div>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
