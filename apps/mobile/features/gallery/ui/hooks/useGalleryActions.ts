import { useMemo } from "react";
import { BrowseGalleryUseCase } from "../../usecase/BrowseGallery";
import { HttpGalleryRepository } from "../../infrastructure/httpGalleryRepository";

export function useGalleryActions() {
  const useCase = useMemo(() => {
    return new BrowseGalleryUseCase(new HttpGalleryRepository());
  }, []);

  return useMemo(
    () => ({
      listMedia: () => useCase.listMedia(),
    }),
    [useCase],
  );
}
