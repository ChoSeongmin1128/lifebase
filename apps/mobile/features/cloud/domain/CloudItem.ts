export type FolderItem = {
  type: "folder" | "file";
  id: string;
  name: string;
  mime_type?: string;
  size_bytes?: number;
  updated_at: string;
};
