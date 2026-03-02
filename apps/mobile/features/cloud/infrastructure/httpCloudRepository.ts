import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";
import type { FolderItem } from "../domain/CloudItem";
import type { CloudRepository } from "../repository/CloudRepository";

interface FolderItemsResponse {
  items?: FolderItem[];
}

export class HttpCloudRepository implements CloudRepository {
  async listItems(folderId?: string | null): Promise<FolderItem[]> {
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }

    const query = folderId ? `?folder_id=${folderId}` : "";
    const data = await api<FolderItemsResponse>(`/cloud/folders${query}`, { token });
    return data.items || [];
  }
}
