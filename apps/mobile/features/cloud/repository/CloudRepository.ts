import type { FolderItem } from "../domain/CloudItem";

export interface CloudRepository {
  listItems(folderId?: string | null): Promise<FolderItem[]>;
}
