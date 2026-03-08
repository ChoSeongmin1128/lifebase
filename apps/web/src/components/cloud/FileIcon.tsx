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
import { getCloudFileTypeKey, getCloudItemToken } from "@lifebase/design-tokens";
import { cn } from "@/lib/utils";

const ICON_MAP: Partial<Record<ReturnType<typeof getCloudFileTypeKey>, LucideIcon>> = {
  image: Image,
  video: Film,
  audio: Music,
  pdf: FileText,
  spreadsheet: FileSpreadsheet,
  presentation: Presentation,
  code: Code,
  document: FileText,
  archive: Archive,
  unknown: File,
};

export function getFileIconColorClass(mimeType: string): string {
  return getCloudItemToken({ type: "file", mimeType }).foreground;
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
  const token = getCloudItemToken({ type: "file", mimeType });
  const Icon = ICON_MAP[getCloudFileTypeKey(mimeType)] ?? File;

  return <Icon size={size} className={cn(className)} style={{ color: token.foreground }} />;
}
