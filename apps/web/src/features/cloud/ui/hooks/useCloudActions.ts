"use client";

import { useMemo } from "react";
import { ManageCloudUseCase } from "@/features/cloud/usecase/ManageCloud";
import { HttpCloudRepository } from "@/features/cloud/infrastructure/httpCloudRepository";
import type { ListCloudItemsInput } from "@/features/cloud/domain/CloudItem";

export function useCloudActions() {
  const useCase = useMemo(() => {
    return new ManageCloudUseCase(new HttpCloudRepository());
  }, []);

  return useMemo(
    () => ({
      listItems: (input: ListCloudItemsInput) => useCase.listItems(input),
      listStars: () => useCase.listStars(),
      uploadFile: (file: File, folderId?: string | null) => useCase.uploadFile(file, folderId),
      downloadFile: (fileId: string) => useCase.downloadFile(fileId),
      deleteFolder: (folderId: string) => useCase.deleteFolder(folderId),
      deleteFile: (fileId: string) => useCase.deleteFile(fileId),
      addStar: (id: string, type: "folder" | "file") => useCase.addStar(id, type),
      removeStar: (id: string, type: "folder" | "file") => useCase.removeStar(id, type),
      restoreTrashItem: (id: string, type: "folder" | "file") => useCase.restoreTrashItem(id, type),
      emptyTrash: () => useCase.emptyTrash(),
      renameFolder: (folderId: string, name: string) => useCase.renameFolder(folderId, name),
      renameFile: (fileId: string, name: string) => useCase.renameFile(fileId, name),
      createFolder: (name: string, parentId?: string | null) => useCase.createFolder(name, parentId),
      searchFiles: (query: string) => useCase.searchFiles(query),
    }),
    [useCase],
  );
}
