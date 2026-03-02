import type { GalleryQuery, ThumbSize } from "@/features/gallery/domain/MediaFile";
import type { GalleryRepository } from "@/features/gallery/repository/GalleryRepository";

export class BrowseGalleryUseCase {
  constructor(private readonly repo: GalleryRepository) {}

  listMedia(query: GalleryQuery) {
    return this.repo.listMedia(query);
  }

  loadThumbnail(fileId: string, size: ThumbSize) {
    return this.repo.loadThumbnail(fileId, size);
  }
}
