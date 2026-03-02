import { useMemo } from "react";
import { BrowseCloudUseCase } from "../../usecase/BrowseCloud";
import { HttpCloudRepository } from "../../infrastructure/httpCloudRepository";

export function useCloudActions() {
  const useCase = useMemo(() => {
    return new BrowseCloudUseCase(new HttpCloudRepository());
  }, []);

  return {
    listItems: (folderId?: string | null) => useCase.listItems(folderId),
  };
}
