"use client";

import { useMemo } from "react";
import { ManageCloudUseCase } from "@/features/cloud/usecase/ManageCloud";
import { HttpCloudRepository } from "@/features/cloud/infrastructure/httpCloudRepository";
import type { ListCloudItemsInput } from "@/features/cloud/domain/CloudItem";
import type { CloudUploadOptions } from "@/features/cloud/repository/CloudRepository";

export function useCloudActions() {
  const useCase = useMemo(() => {
    return new ManageCloudUseCase(new HttpCloudRepository());
  }, []);

  return useMemo(
    () => ({
      listItems: (input: ListCloudItemsInput) => useCase.listItems(input),
      getFolder: (folderId: string) => useCase.getFolder(folderId),
      getTrashFolder: (folderId: string) => useCase.getTrashFolder(folderId),
      listStars: () => useCase.listStars(),
      uploadFile: (file: File, folderId?: string | null, options?: CloudUploadOptions) =>
        useCase.uploadFile(file, folderId, options),
      createTextFile: (name: string, extension: "md" | "txt", folderId?: string | null) =>
        useCase.createTextFile(name, extension, folderId),
      getTextFileContent: (fileId: string) => useCase.getTextFileContent(fileId),
      updateTextFileContent: (fileId: string, content: string) => useCase.updateTextFileContent(fileId, content),
      downloadFile: (fileId: string) => useCase.downloadFile(fileId),
      deleteFolder: (folderId: string) => useCase.deleteFolder(folderId),
      deleteFile: (fileId: string) => useCase.deleteFile(fileId),
      addStar: (id: string, type: "folder" | "file") => useCase.addStar(id, type),
      removeStar: (id: string, type: "folder" | "file") => useCase.removeStar(id, type),
      restoreTrashItem: (id: string, type: "folder" | "file") => useCase.restoreTrashItem(id, type),
      emptyTrash: () => useCase.emptyTrash(),
      renameFolder: (folderId: string, name: string) => useCase.renameFolder(folderId, name),
      moveFolder: (folderId: string, parentId?: string | null) => useCase.moveFolder(folderId, parentId),
      copyFolder: (folderId: string, parentId?: string | null) => useCase.copyFolder(folderId, parentId),
      renameFile: (fileId: string, name: string) => useCase.renameFile(fileId, name),
      moveFile: (fileId: string, folderId?: string | null) => useCase.moveFile(fileId, folderId),
      copyFile: (fileId: string, folderId?: string | null) => useCase.copyFile(fileId, folderId),
      undoOperation: (token: string) => useCase.undoOperation(token),
      createFolder: (name: string, parentId?: string | null) => useCase.createFolder(name, parentId),
      searchFiles: (query: string) => useCase.searchFiles(query),
    }),
    [useCase],
  );
}
