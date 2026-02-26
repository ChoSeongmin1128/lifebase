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
} from "lucide-react";

const MIME_MAP: [RegExp, React.ComponentType<{ size?: number; className?: string }>][] = [
  [/^image\//, Image],
  [/^video\//, Film],
  [/^audio\//, Music],
  [/pdf/, FileText],
  [/spreadsheet|excel|csv/, FileSpreadsheet],
  [/presentation|powerpoint/, Presentation],
  [/zip|archive|compressed|tar|rar|7z/, Archive],
  [/javascript|typescript|json|html|css|xml|python|java|go|rust/, Code],
];

export function FileIcon({
  mimeType,
  size = 16,
  className,
}: {
  mimeType: string;
  size?: number;
  className?: string;
}) {
  for (const [pattern, Icon] of MIME_MAP) {
    if (pattern.test(mimeType)) {
      return <Icon size={size} className={className} />;
    }
  }
  return <File size={size} className={className} />;
}
