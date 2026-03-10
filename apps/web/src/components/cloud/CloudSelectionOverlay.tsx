"use client";

export interface CloudSelectionRect {
  left: number;
  top: number;
  width: number;
  height: number;
}

interface CloudSelectionOverlayProps {
  rect: CloudSelectionRect | null;
}

export function CloudSelectionOverlay({ rect }: CloudSelectionOverlayProps) {
  if (!rect) return null;

  return (
    <div
      className="pointer-events-none absolute z-10 rounded-md border border-primary/50 bg-primary/10 shadow-[0_0_0_1px_rgba(27,153,139,0.08)]"
      style={{
        left: rect.left,
        top: rect.top,
        width: rect.width,
        height: rect.height,
      }}
    />
  );
}
