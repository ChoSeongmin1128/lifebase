import { cn } from "@/lib/utils";

interface ExtensionBadgeProps {
  filename: string;
  className?: string;
}

export function ExtensionBadge({ filename, className }: ExtensionBadgeProps) {
  const ext = filename.split(".").pop()?.toLowerCase() || "";
  if (!ext) return null;

  return (
    <span
      className={cn(
        "rounded bg-black/50 px-1 py-0.5 text-[9px] font-medium uppercase text-white",
        className
      )}
    >
      {ext}
    </span>
  );
}
