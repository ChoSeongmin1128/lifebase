"use client";

import { useEffect, useState } from "react";
import type { ThumbSize } from "@/features/gallery/domain/MediaFile";
import { useGalleryActions } from "@/features/gallery/ui/hooks/useGalleryActions";

export function useThumbnailSource(fileId: string, size: ThumbSize) {
  const [src, setSrc] = useState<string | null>(null);
  const { loadThumbnail } = useGalleryActions();

  useEffect(() => {
    let objectURL: string | null = null;
    let active = true;

    const load = async () => {
      try {
        const blob = await loadThumbnail(fileId, size);
        objectURL = URL.createObjectURL(blob);
        if (active) {
          setSrc(objectURL);
        }
      } catch {
        if (active) {
          setSrc(null);
        }
      }
    };

    load();

    return () => {
      active = false;
      if (objectURL) {
        URL.revokeObjectURL(objectURL);
      }
    };
  }, [fileId, loadThumbnail, size]);

  return src;
}
