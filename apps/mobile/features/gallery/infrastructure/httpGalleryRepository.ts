import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";
import type { MediaItem } from "../domain/MediaItem";
import type { GalleryRepository } from "../repository/GalleryRepository";

interface MediaResponse {
  items?: MediaItem[];
}

export class HttpGalleryRepository implements GalleryRepository {
  async listMedia(): Promise<MediaItem[]> {
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }

    const data = await api<MediaResponse>("/gallery", { token });
    return data.items || [];
  }
}
