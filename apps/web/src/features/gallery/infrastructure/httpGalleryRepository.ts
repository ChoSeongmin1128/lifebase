import { api } from "@/features/shared/infrastructure/http-api";
import { getApiUrl } from "@/features/shared/infrastructure/api-url";
import { getAccessToken, isSessionMarkerToken } from "@/features/auth/infrastructure/token-auth";
import type { GalleryPage, GalleryQuery, MediaFile, ThumbSize } from "@/features/gallery/domain/MediaFile";
import type { GalleryRepository } from "@/features/gallery/repository/GalleryRepository";

interface GalleryApiResponse {
  items?: MediaFile[];
  next_cursor?: string;
}

export class HttpGalleryRepository implements GalleryRepository {
  async listMedia(query: GalleryQuery): Promise<GalleryPage> {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }

    const params = new URLSearchParams({
      sort_by: query.sortBy,
      sort_dir: query.sortDir,
      limit: String(query.limit ?? 50),
    });

    if (query.mediaType !== "all") {
      params.set("type", query.mediaType);
    }

    if (query.cursor) {
      params.set("cursor", query.cursor);
    }

    const data = await api<GalleryApiResponse>(`/gallery?${params.toString()}`, { token });
    return {
      items: data.items || [],
      nextCursor: data.next_cursor || "",
    };
  }

  async loadThumbnail(fileId: string, size: ThumbSize): Promise<Blob> {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }

    const headers: HeadersInit = {};
    if (!isSessionMarkerToken(token)) {
      headers.Authorization = `Bearer ${token}`;
    }

    const res = await fetch(getApiUrl(`/gallery/thumbnails/${fileId}/${size}`), {
      headers,
      credentials: "include",
    });

    if (!res.ok) {
      throw new Error(`thumbnail ${res.status}`);
    }

    return res.blob();
  }
}
