import type { GalleryPage, GalleryQuery, ThumbSize } from "@/features/gallery/domain/MediaFile";

export interface GalleryRepository {
  listMedia(query: GalleryQuery): Promise<GalleryPage>;
  loadThumbnail(fileId: string, size: ThumbSize): Promise<Blob>;
}
