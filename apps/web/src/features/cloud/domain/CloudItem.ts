export interface FolderData {
  id: string;
  user_id: string;
  parent_id: string | null;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface CloudFile {
  id: string;
  user_id: string;
  folder_id: string | null;
  name: string;
  mime_type: string;
  size_bytes: number;
  thumb_status: string;
  taken_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface FolderItem {
  type: "folder" | "file";
  folder?: FolderData;
  file?: CloudFile;
  path?: string;
}

export interface StarItem {
  id: string;
  type: "folder" | "file";
}

export type CloudSection = "" | "trash" | "recent" | "shared" | "starred";
export type CloudSortBy = "name" | "size" | "updated_at" | "created_at";
export type CloudSortDir = "asc" | "desc";

export interface ListCloudItemsInput {
  section: CloudSection;
  folderId?: string | null;
  sortBy?: CloudSortBy;
  sortDir?: CloudSortDir;
}
