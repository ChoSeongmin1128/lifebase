import type { MediaItem } from "../domain/MediaItem";

export interface GalleryRepository {
  listMedia(): Promise<MediaItem[]>;
}
