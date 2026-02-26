import { Button } from "@/components/ui/button";
import { Download, Trash2, X } from "lucide-react";

interface BulkActionBarProps {
  count: number;
  onDownload: () => void;
  onDelete: () => void;
  onClear: () => void;
}

export function BulkActionBar({ count, onDownload, onDelete, onClear }: BulkActionBarProps) {
  return (
    <div className="flex items-center gap-2 border-b border-border bg-surface-accent px-4 md:px-6 py-2">
      <span className="text-sm text-text-primary font-medium">{count}개 선택됨</span>
      <div className="flex items-center gap-1 ml-auto">
        <Button variant="ghost" size="sm" onClick={onDownload} className="gap-1.5">
          <Download size={14} />
          다운로드
        </Button>
        <Button variant="danger" size="sm" onClick={onDelete} className="gap-1.5">
          <Trash2 size={14} />
          삭제
        </Button>
        <Button variant="ghost" size="icon-sm" onClick={onClear}>
          <X size={14} />
        </Button>
      </div>
    </div>
  );
}
