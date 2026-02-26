import { Flag } from "lucide-react";
import { cn } from "@/lib/utils";

const PRIORITY_STYLES: Record<string, string> = {
  urgent: "text-error",
  high: "text-caution",
  normal: "hidden",
  low: "text-text-muted",
};

interface PriorityFlagProps {
  priority: string;
  size?: number;
}

export function PriorityFlag({ priority, size = 14 }: PriorityFlagProps) {
  const style = PRIORITY_STYLES[priority];
  if (!style || style === "hidden") return null;

  return (
    <Flag
      size={size}
      className={cn("shrink-0", style)}
      fill={priority === "urgent" ? "currentColor" : "none"}
    />
  );
}
