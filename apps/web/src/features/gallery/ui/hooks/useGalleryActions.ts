"use client";

import { useMemo } from "react";
import { BrowseGalleryUseCase } from "@/features/gallery/usecase/BrowseGallery";
import { HttpGalleryRepository } from "@/features/gallery/infrastructure/httpGalleryRepository";
import type { GalleryQuery, ThumbSize } from "@/features/gallery/domain/MediaFile";

export function useGalleryActions() {
  const useCase = useMemo(() => {
    return new BrowseGalleryUseCase(new HttpGalleryRepository());
  }, []);

  return useMemo(
    () => ({
      listMedia: (query: GalleryQuery) => useCase.listMedia(query),
      loadThumbnail: (fileId: string, size: ThumbSize) => useCase.loadThumbnail(fileId, size),
    }),
    [useCase],
  );
}
