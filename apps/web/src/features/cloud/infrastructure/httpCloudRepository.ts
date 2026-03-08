import { api, apiDownload, apiUpload } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";
import type {
  CloudFile,
  FolderData,
  FolderItem,
  ListCloudItemsInput,
  StarItem,
} from "@/features/cloud/domain/CloudItem";
import type { CloudRepository } from "@/features/cloud/repository/CloudRepository";

interface ItemsResponse {
  items?: FolderItem[];
}

interface StarsResponse {
  stars?: StarItem[];
}

interface SearchResponse {
  files?: CloudFile[];
}

interface TextFileContentResponse {
  content: string;
}

export class HttpCloudRepository implements CloudRepository {
  private getToken(): string {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async listItems(input: ListCloudItemsInput): Promise<FolderItem[]> {
    const token = this.getToken();

    if (input.section === "trash") {
      const data = await api<ItemsResponse>("/cloud/trash", { token });
      return data.items || [];
    }
    if (input.section === "recent") {
      const data = await api<ItemsResponse>("/cloud/recent", { token });
      return data.items || [];
    }
    if (input.section === "shared") {
      const data = await api<ItemsResponse>("/cloud/shared", { token });
      return data.items || [];
    }
    if (input.section === "starred") {
      const data = await api<ItemsResponse>("/cloud/starred", { token });
      return data.items || [];
    }

    const params = new URLSearchParams({
      sort_by: input.sortBy || "name",
      sort_dir: input.sortDir || "asc",
    });

    if (input.folderId) {
      params.set("folder_id", input.folderId);
    }

    const data = await api<ItemsResponse>(`/cloud/folders?${params.toString()}`, { token });
    return data.items || [];
  }

  getFolder(folderId: string): Promise<FolderData> {
    const token = this.getToken();
    return api<FolderData>(`/cloud/folders/${folderId}`, { token });
  }

  async listStars(): Promise<StarItem[]> {
    const token = this.getToken();
    const data = await api<StarsResponse>("/cloud/stars", { token });
    return data.stars || [];
  }

  async uploadFile(file: File, folderId?: string | null): Promise<void> {
    const token = this.getToken();
    const formData = new FormData();
    formData.append("file", file);
    if (folderId) {
      formData.append("folder_id", folderId);
    }
    await apiUpload("/cloud/files/upload", formData, token);
  }

  async createTextFile(
    name: string,
    content: string,
    mimeType: string,
    folderId?: string | null,
  ): Promise<void> {
    const file = new File([content], name, { type: mimeType });
    await this.uploadFile(file, folderId);
  }

  downloadFile(fileId: string): Promise<{ blob: Blob; filename: string }> {
    const token = this.getToken();
    return apiDownload(`/cloud/files/${fileId}/download`, token);
  }

  async getTextFileContent(fileId: string): Promise<string> {
    const token = this.getToken();
    const data = await api<TextFileContentResponse>(`/cloud/files/${fileId}/content`, { token });
    return data.content || "";
  }

  async updateTextFileContent(fileId: string, content: string): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/files/${fileId}/content`, {
      method: "PATCH",
      body: { content },
      token,
    });
  }

  async deleteFolder(folderId: string): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/folders/${folderId}`, { method: "DELETE", token });
  }

  async deleteFile(fileId: string): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/files/${fileId}`, { method: "DELETE", token });
  }

  async addStar(id: string, type: "folder" | "file"): Promise<void> {
    const token = this.getToken();
    await api("/cloud/stars", {
      method: "POST",
      body: { id, type },
      token,
    });
  }

  async removeStar(id: string, type: "folder" | "file"): Promise<void> {
    const token = this.getToken();
    await api("/cloud/stars", {
      method: "DELETE",
      body: { id, type },
      token,
    });
  }

  async restoreTrashItem(id: string, type: "folder" | "file"): Promise<void> {
    const token = this.getToken();
    await api("/cloud/trash/restore", {
      method: "POST",
      body: { id, type },
      token,
    });
  }

  async emptyTrash(): Promise<void> {
    const token = this.getToken();
    await api("/cloud/trash", {
      method: "DELETE",
      token,
    });
  }

  async renameFolder(folderId: string, name: string): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/folders/${folderId}/rename`, {
      method: "PATCH",
      body: { name },
      token,
    });
  }

  async moveFolder(folderId: string, parentId?: string | null): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/folders/${folderId}/move`, {
      method: "PATCH",
      body: { parent_id: parentId ?? null },
      token,
    });
  }

  async copyFolder(folderId: string, parentId?: string | null): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/folders/${folderId}/copy`, {
      method: "PATCH",
      body: { parent_id: parentId ?? null },
      token,
    });
  }

  async renameFile(fileId: string, name: string): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/files/${fileId}/rename`, {
      method: "PATCH",
      body: { name },
      token,
    });
  }

  async moveFile(fileId: string, folderId?: string | null): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/files/${fileId}/move`, {
      method: "PATCH",
      body: { folder_id: folderId ?? null },
      token,
    });
  }

  async copyFile(fileId: string, folderId?: string | null): Promise<void> {
    const token = this.getToken();
    await api(`/cloud/files/${fileId}/copy`, {
      method: "PATCH",
      body: { folder_id: folderId ?? null },
      token,
    });
  }

  async createFolder(name: string, parentId?: string | null): Promise<void> {
    const token = this.getToken();
    await api("/cloud/folders", {
      method: "POST",
      body: { name, parent_id: parentId || null },
      token,
    });
  }

  async searchFiles(query: string): Promise<CloudFile[]> {
    const token = this.getToken();
    const data = await api<SearchResponse>(`/cloud/search?q=${encodeURIComponent(query)}`, { token });
    return data.files || [];
  }
}
