import type {
  CloudFile,
  FolderItem,
  ListCloudItemsInput,
  StarItem,
} from "@/features/cloud/domain/CloudItem";

export interface CloudRepository {
  listItems(input: ListCloudItemsInput): Promise<FolderItem[]>;
  listStars(): Promise<StarItem[]>;
  uploadFile(file: File, folderId?: string | null): Promise<void>;
  downloadFile(fileId: string): Promise<{ blob: Blob; filename: string }>;
  deleteFolder(folderId: string): Promise<void>;
  deleteFile(fileId: string): Promise<void>;
  addStar(id: string, type: "folder" | "file"): Promise<void>;
  removeStar(id: string, type: "folder" | "file"): Promise<void>;
  restoreTrashItem(id: string, type: "folder" | "file"): Promise<void>;
  emptyTrash(): Promise<void>;
  renameFolder(folderId: string, name: string): Promise<void>;
  renameFile(fileId: string, name: string): Promise<void>;
  createFolder(name: string, parentId?: string | null): Promise<void>;
  searchFiles(query: string): Promise<CloudFile[]>;
}
