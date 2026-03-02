import type { CloudRepository } from "@/features/cloud/repository/CloudRepository";
import type { ListCloudItemsInput } from "@/features/cloud/domain/CloudItem";

export class ManageCloudUseCase {
  constructor(private readonly repo: CloudRepository) {}

  listItems(input: ListCloudItemsInput) {
    return this.repo.listItems(input);
  }

  listStars() {
    return this.repo.listStars();
  }

  uploadFile(file: File, folderId?: string | null) {
    return this.repo.uploadFile(file, folderId);
  }

  createTextFile(name: string, extension: "md" | "txt", folderId?: string | null) {
    const normalized = name.trim();
    if (!normalized) {
      throw new Error("파일 이름이 비어 있습니다.");
    }

    const hasExt = normalized.includes(".");
    const fileName = hasExt ? normalized : `${normalized}.${extension}`;
    const mimeType = extension === "md" ? "text/markdown" : "text/plain";
    return this.repo.createTextFile(fileName, "", mimeType, folderId);
  }

  getTextFileContent(fileId: string) {
    return this.repo.getTextFileContent(fileId);
  }

  updateTextFileContent(fileId: string, content: string) {
    return this.repo.updateTextFileContent(fileId, content);
  }

  downloadFile(fileId: string) {
    return this.repo.downloadFile(fileId);
  }

  deleteFolder(folderId: string) {
    return this.repo.deleteFolder(folderId);
  }

  deleteFile(fileId: string) {
    return this.repo.deleteFile(fileId);
  }

  addStar(id: string, type: "folder" | "file") {
    return this.repo.addStar(id, type);
  }

  removeStar(id: string, type: "folder" | "file") {
    return this.repo.removeStar(id, type);
  }

  restoreTrashItem(id: string, type: "folder" | "file") {
    return this.repo.restoreTrashItem(id, type);
  }

  emptyTrash() {
    return this.repo.emptyTrash();
  }

  renameFolder(folderId: string, name: string) {
    return this.repo.renameFolder(folderId, name.trim());
  }

  renameFile(fileId: string, name: string) {
    return this.repo.renameFile(fileId, name.trim());
  }

  createFolder(name: string, parentId?: string | null) {
    const normalized = name.trim();
    if (!normalized) {
      throw new Error("폴더 이름이 비어 있습니다.");
    }
    return this.repo.createFolder(normalized, parentId);
  }

  searchFiles(query: string) {
    const normalized = query.trim();
    if (!normalized) {
      return Promise.resolve([]);
    }
    return this.repo.searchFiles(normalized);
  }
}
