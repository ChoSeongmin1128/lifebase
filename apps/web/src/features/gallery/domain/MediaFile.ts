export interface MediaFile {
  id: string;
  name: string;
  mime_type: string;
  size_bytes: number;
  thumb_status: string;
  taken_at: string | null;
  created_at: string;
  updated_at: string;
}

export type MediaType = "all" | "image" | "video";
export type GallerySortBy = "taken_at" | "created_at" | "name" | "size";
export type SortDir = "asc" | "desc";
export type ThumbSize = "small" | "medium";

export interface GalleryQuery {
  mediaType: MediaType;
  sortBy: GallerySortBy;
  sortDir: SortDir;
  cursor?: string;
  limit?: number;
}

export interface GalleryPage {
  items: MediaFile[];
  nextCursor: string;
}
