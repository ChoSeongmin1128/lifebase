import {
  Image,
  Film,
  Music,
  FileText,
  Archive,
  File,
  FileSpreadsheet,
  Presentation,
  Code,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

const MIME_MAP: [RegExp, LucideIcon, string][] = [
  [/^image\//, Image, "text-emerald-500"],
  [/^video\//, Film, "text-rose-500"],
  [/^audio\//, Music, "text-violet-500"],
  [/pdf/, FileText, "text-red-500"],
  [/spreadsheet|excel|csv/, FileSpreadsheet, "text-green-600"],
  [/presentation|powerpoint/, Presentation, "text-orange-500"],
  [/zip|archive|compressed|tar|rar|7z/, Archive, "text-amber-600"],
  [/javascript|typescript|json|html|css|xml|python|java|go|rust/, Code, "text-sky-500"],
];

export function getFileIconColorClass(mimeType: string): string {
  for (const [pattern, , colorClass] of MIME_MAP) {
    if (pattern.test(mimeType)) {
      return colorClass;
    }
  }
  return "text-slate-500";
}

export function FileIcon({
  mimeType,
  size = 16,
  className,
}: {
  mimeType: string;
  size?: number;
  className?: string;
}) {
  for (const [pattern, Icon, colorClass] of MIME_MAP) {
    if (pattern.test(mimeType)) {
      return <Icon size={size} className={cn(colorClass, className)} />;
    }
  }
  return <File size={size} className={cn("text-slate-500", className)} />;
}
